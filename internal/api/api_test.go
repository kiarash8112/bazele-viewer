package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"example.com/bazelviewer/internal/graph"
)

func testGraph() *graph.Graph {
	b := graph.NewBuilder()
	b.AddRule("//a:bin", "go_binary", []string{"//a:lib"})
	b.AddRule("//a:lib", "go_library", []string{"//b:one", "//b:two"})
	b.AddRule("//b:one", "go_library", nil)
	b.AddRule("//b:two", "go_library", nil)
	return b.Build()
}

func testModules() *graph.Graph {
	b := graph.NewBuilder()
	b.AddRule("app@1.0", "root module", []string{"rules_go@0.57.0"})
	b.AddRule("rules_go@0.57.0", "module", nil)
	return b.Build()
}

func staticLoader(warning string, err error) Loader {
	return func(context.Context) (Graphs, string, error) {
		if err != nil {
			return Graphs{}, "", err
		}
		return Graphs{Targets: testGraph(), Modules: testModules()}, warning, nil
	}
}

func get(t *testing.T, s *Server, target string, out any) int {
	t.Helper()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %s: %v: %s", target, err, rec.Body)
		}
	}
	return rec.Code
}

func waitIdle(t *testing.T, s *Server) statusResponse {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var st statusResponse
		get(t, s, "/api/status", &st)
		if !st.Loading {
			return st
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("graph load did not finish")
	return statusResponse{}
}

func readyServer(t *testing.T) *Server {
	t.Helper()
	s := New(context.Background(), staticLoader("", nil))
	s.Refresh()
	waitIdle(t, s)
	return s
}

func TestRootsThenChildren(t *testing.T) {
	s := readyServer(t)

	var roots pageResponse
	if code := get(t, s, "/api/roots", &roots); code != http.StatusOK {
		t.Fatalf("roots status = %d", code)
	}
	if len(roots.Nodes) != 1 || roots.Nodes[0].Label != "//a:bin" || roots.Nodes[0].DepCount != 1 {
		t.Fatalf("roots = %+v", roots)
	}

	var children pageResponse
	if code := get(t, s, "/api/children?label=//a:lib&limit=1&offset=1", &children); code != http.StatusOK {
		t.Fatalf("children status = %d", code)
	}
	if children.Total != 2 || len(children.Nodes) != 1 || children.Nodes[0].Label != "//b:two" {
		t.Errorf("children = %+v", children)
	}
	if children.Target == nil || children.Target.Label != "//a:lib" || children.Target.DepCount != 2 {
		t.Errorf("children.Target = %+v", children.Target)
	}
}

func TestParents(t *testing.T) {
	s := readyServer(t)

	var parents pageResponse
	if code := get(t, s, "/api/parents?label=//b:one", &parents); code != http.StatusOK {
		t.Fatalf("parents status = %d", code)
	}
	if parents.Total != 1 || parents.Nodes[0].Label != "//a:lib" || parents.Nodes[0].ParentCount != 1 {
		t.Errorf("parents = %+v", parents)
	}
	if parents.Target == nil || parents.Target.Label != "//b:one" || parents.Target.ParentCount != 1 {
		t.Errorf("parents.Target = %+v", parents.Target)
	}
}

func TestHotspots(t *testing.T) {
	s := readyServer(t)

	var targets pageResponse
	if code := get(t, s, "/api/hotspots?limit=2", &targets); code != http.StatusOK {
		t.Fatalf("hotspots status = %d", code)
	}
	if targets.Total != 4 || len(targets.Nodes) != 2 ||
		targets.Nodes[0].Label != "//b:one" || targets.Nodes[0].Dependents != 2 {
		t.Errorf("hotspots = %+v", targets)
	}

	var found pageResponse
	if code := get(t, s, "/api/hotspots?q=two", &found); code != http.StatusOK {
		t.Fatalf("hotspots search status = %d", code)
	}
	if found.Total != 1 || found.Nodes[0].Label != "//b:two" || found.Nodes[0].Rank != 2 {
		t.Errorf("hotspots search = %+v, want //b:two at rank 2", found)
	}

	var files pageResponse
	if code := get(t, s, "/api/hotspots?of=files", &files); code != http.StatusOK {
		t.Fatalf("file hotspots status = %d", code)
	}
	if files.Total != 0 {
		t.Errorf("file hotspots = %+v, want none", files)
	}
}

func TestModules(t *testing.T) {
	s := readyServer(t)

	var st statusResponse
	get(t, s, "/api/status", &st)
	if st.Targets != 4 || st.Modules != 2 {
		t.Errorf("status = %+v, want 4 targets and 2 modules", st)
	}

	var roots pageResponse
	if code := get(t, s, "/api/roots?graph=modules", &roots); code != http.StatusOK {
		t.Fatalf("module roots status = %d", code)
	}
	if roots.Total != 1 || roots.Nodes[0].Label != "app@1.0" || roots.Nodes[0].DepCount != 1 {
		t.Errorf("module roots = %+v", roots)
	}

	var parents pageResponse
	if code := get(t, s, "/api/parents?graph=modules&label=rules_go@0.57.0", &parents); code != http.StatusOK {
		t.Fatalf("module parents status = %d", code)
	}
	if parents.Total != 1 || parents.Nodes[0].Label != "app@1.0" {
		t.Errorf("module parents = %+v", parents)
	}
}

func TestModulesUnavailable(t *testing.T) {
	s := New(context.Background(), func(context.Context) (Graphs, string, error) {
		return Graphs{Targets: testGraph(), ModulesError: "no MODULE.bazel"}, "", nil
	})
	s.Refresh()
	if st := waitIdle(t, s); st.State != "ready" || st.ModulesError != "no MODULE.bazel" {
		t.Errorf("status = %+v", st)
	}
	var body map[string]string
	code := get(t, s, "/api/roots?graph=modules", &body)
	if code != http.StatusServiceUnavailable || !strings.Contains(body["error"], "no MODULE.bazel") {
		t.Errorf("module roots: status %d, body %v; want 503 with the reason", code, body)
	}
	if code := get(t, s, "/api/roots", &pageResponse{}); code != http.StatusOK {
		t.Errorf("target roots status = %d, want 200", code)
	}
}

func TestBadRequests(t *testing.T) {
	s := readyServer(t)
	tests := []struct {
		target string
		want   int
	}{
		{"/api/children", http.StatusBadRequest},
		{"/api/children?label=//no:pe", http.StatusNotFound},
		{"/api/parents", http.StatusBadRequest},
		{"/api/parents?label=//no:pe", http.StatusNotFound},
		{"/api/roots?limit=0", http.StatusBadRequest},
		{"/api/roots?limit=5000", http.StatusBadRequest},
		{"/api/roots?offset=-1", http.StatusBadRequest},
		{"/api/roots?tests=maybe", http.StatusBadRequest},
		{"/api/hotspots?of=packages", http.StatusBadRequest},
		{"/api/hotspots?sort=size", http.StatusBadRequest},
		{"/api/roots?graph=files", http.StatusBadRequest},
	}
	for _, tc := range tests {
		var body map[string]string
		if code := get(t, s, tc.target, &body); code != tc.want || body["error"] == "" {
			t.Errorf("%s: status %d, body %v; want status %d with an error", tc.target, code, body, tc.want)
		}
	}
}

func TestNotLoadedYet(t *testing.T) {
	s := New(context.Background(), staticLoader("", nil))
	if code := get(t, s, "/api/roots", &map[string]string{}); code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", code)
	}
}

func TestFirstLoadFailure(t *testing.T) {
	s := New(context.Background(), staticLoader("", errors.New("boom")))
	s.Refresh()
	st := waitIdle(t, s)
	if st.State != "error" || st.Error != "boom" {
		t.Errorf("status = %+v", st)
	}
}

func TestWarningIsReported(t *testing.T) {
	s := New(context.Background(), staticLoader("partial", nil))
	s.Refresh()
	if st := waitIdle(t, s); st.State != "ready" || st.Warning != "partial" || st.Targets != 4 {
		t.Errorf("status = %+v", st)
	}
}

func TestFailedRefreshKeepsPreviousGraph(t *testing.T) {
	var fail atomic.Bool
	s := New(context.Background(), func(context.Context) (Graphs, string, error) {
		if fail.Load() {
			return Graphs{}, "", errors.New("broken")
		}
		return Graphs{Targets: testGraph()}, "", nil
	})
	s.Refresh()
	waitIdle(t, s)

	fail.Store(true)
	s.Refresh()
	st := waitIdle(t, s)
	if st.State != "ready" || st.Error != "broken" {
		t.Errorf("status = %+v", st)
	}
	if code := get(t, s, "/api/roots", &pageResponse{}); code != http.StatusOK {
		t.Errorf("roots status = %d, want 200", code)
	}
}

func TestRefreshDuringLoadRunsAgain(t *testing.T) {
	release := make(chan struct{})
	var loads atomic.Int32
	s := New(context.Background(), func(context.Context) (Graphs, string, error) {
		loads.Add(1)
		<-release
		return Graphs{Targets: testGraph()}, "", nil
	})
	s.Refresh()
	s.Refresh()
	s.Refresh()
	close(release)
	waitIdle(t, s)
	if got := loads.Load(); got != 2 {
		t.Errorf("loads = %d, want 2", got)
	}
}

func TestRefreshRequiresPost(t *testing.T) {
	s := readyServer(t)
	if code := get(t, s, "/api/refresh", nil); code != http.StatusMethodNotAllowed {
		t.Errorf("GET status = %d, want 405", code)
	}

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/refresh", strings.NewReader("")))
	if rec.Code != http.StatusAccepted {
		t.Errorf("POST status = %d, want 202", rec.Code)
	}
	waitIdle(t, s)
}
