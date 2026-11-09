// Package api serves the dependency graph over HTTP.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"example.com/bazelviewer/internal/graph"
)

const (
	defaultLimit = 50
	maxLimit     = 1000
)

// Source answers graph questions. Handlers depend only on this, so another
// implementation (for example, one that queries Bazel per request) can replace
// the in-memory graph.
type Source interface {
	Len() int
	Roots(opts graph.Options) graph.Page
	Children(label string, opts graph.Options) (graph.Page, bool)
}

// Loader produces a fresh Source. A non-empty warning marks it as incomplete.
type Loader func(ctx context.Context) (src Source, warning string, err error)

type snapshot struct {
	source   Source
	warning  string
	loadedAt time.Time
}

// Server holds the current graph and reloads it in the background. Requests
// keep using the previous graph until a reload succeeds.
type Server struct {
	ctx  context.Context
	load Loader
	cur  atomic.Pointer[snapshot]

	mu      sync.Mutex
	loading bool
	pending bool // A refresh arrived during a load, so load again afterwards.
	lastErr string
}

// New returns a Server that loads graphs until ctx is cancelled. Call Refresh
// to perform the first load.
func New(ctx context.Context, load Loader) *Server {
	return &Server{ctx: ctx, load: load}
}

// Refresh starts a reload in the background. If one is already running, another
// follows it, so that changes made after the running load began are picked up.
func (s *Server) Refresh() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loading {
		s.pending = true
		return
	}
	s.loading = true
	go s.runLoads()
}

func (s *Server) runLoads() {
	for {
		src, warning, err := s.load(s.ctx)

		s.mu.Lock()
		if err != nil {
			s.lastErr = err.Error()
			log.Printf("graph load failed: %v", err)
		} else {
			s.lastErr = ""
			s.cur.Store(&snapshot{source: src, warning: warning, loadedAt: time.Now()})
			log.Printf("graph loaded: %d targets", src.Len())
		}
		if !s.pending {
			s.loading = false
			s.mu.Unlock()
			return
		}
		s.pending = false
		s.mu.Unlock()
	}
}

// Handler returns the HTTP routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("POST /api/refresh", s.handleRefresh)
	mux.HandleFunc("GET /api/roots", s.handleRoots)
	mux.HandleFunc("GET /api/children", s.handleChildren)
	return mux
}

type statusResponse struct {
	State    string     `json:"state"` // "loading", "ready" or "error".
	Loading  bool       `json:"loading"`
	Targets  int        `json:"targets"`
	LoadedAt *time.Time `json:"loadedAt,omitempty"`
	Warning  string     `json:"warning,omitempty"`
	Error    string     `json:"error,omitempty"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	resp := statusResponse{Loading: s.loading, Error: s.lastErr}
	s.mu.Unlock()

	snap := s.cur.Load()
	switch {
	case snap != nil:
		resp.State = "ready"
		resp.Targets = snap.source.Len()
		resp.LoadedAt = &snap.loadedAt
		resp.Warning = snap.warning
	case resp.Error != "" && !resp.Loading:
		resp.State = "error"
	default:
		resp.State = "loading"
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	s.Refresh()
	writeJSON(w, http.StatusAccepted, map[string]bool{"loading": true})
}

type pageResponse struct {
	Nodes  []graph.Node `json:"nodes"`
	Total  int          `json:"total"`
	Offset int          `json:"offset"`
	Limit  int          `json:"limit"`
}

func (s *Server) handleRoots(w http.ResponseWriter, r *http.Request) {
	opts, err := parseOptions(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	src, ok := s.source(w)
	if !ok {
		return
	}
	writePage(w, src.Roots(opts), opts)
}

func (s *Server) handleChildren(w http.ResponseWriter, r *http.Request) {
	label := r.URL.Query().Get("label")
	if label == "" {
		writeError(w, http.StatusBadRequest, "missing label")
		return
	}
	opts, err := parseOptions(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	src, ok := s.source(w)
	if !ok {
		return
	}
	page, found := src.Children(label, opts)
	if !found {
		writeError(w, http.StatusNotFound, "unknown target: "+label)
		return
	}
	writePage(w, page, opts)
}

// source returns the current graph, or writes a 503 if none has loaded yet.
func (s *Server) source(w http.ResponseWriter) (Source, bool) {
	snap := s.cur.Load()
	if snap == nil {
		writeError(w, http.StatusServiceUnavailable, "graph is not loaded; see /api/status")
		return nil, false
	}
	return snap.source, true
}

func parseOptions(r *http.Request) (graph.Options, error) {
	q := r.URL.Query()
	opts := graph.Options{Limit: defaultLimit}

	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return opts, fmt.Errorf("invalid offset: %q", v)
		}
		opts.Offset = n
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxLimit {
			return opts, fmt.Errorf("invalid limit: %q (must be 1-%d)", v, maxLimit)
		}
		opts.Limit = n
	}

	var err error
	if opts.IncludeTests, err = parseBool(q.Get("tests")); err != nil {
		return opts, fmt.Errorf("invalid tests: %q", q.Get("tests"))
	}
	if opts.IncludeExternal, err = parseBool(q.Get("external")); err != nil {
		return opts, fmt.Errorf("invalid external: %q", q.Get("external"))
	}
	return opts, nil
}

func parseBool(v string) (bool, error) {
	if v == "" {
		return false, nil
	}
	return strconv.ParseBool(v)
}

func writePage(w http.ResponseWriter, p graph.Page, opts graph.Options) {
	writeJSON(w, http.StatusOK, pageResponse{
		Nodes:  p.Nodes,
		Total:  p.Total,
		Offset: opts.Offset,
		Limit:  opts.Limit,
	})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
