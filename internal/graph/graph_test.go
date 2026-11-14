package graph

import (
	"reflect"
	"testing"
)

func labels(p Page) []string {
	out := make([]string, 0, len(p.Nodes))
	for _, n := range p.Nodes {
		out = append(out, n.Label)
	}
	return out
}

func sample() *Graph {
	b := NewBuilder()
	b.AddRule("//app:bin", "go_binary", []string{"//app:lib"})
	b.AddRule("//app:lib", "go_library", []string{
		"//app:lib.go", // source file: dropped
		"//util:b",
		"//util:a",
		"//util:a",              // duplicate: dropped
		"//gen:out.go",          // generated file: becomes //gen:gen
		"@dep//:ext",            // external
		"//app:lib",             // self edge: dropped
		"//missing:from_output", // not in the graph: dropped
	})
	b.AddRule("//app:lib_test", "go_test", []string{"//app:lib"})
	b.AddRule("//gen:gen", "genrule", nil)
	b.AddGeneratedFile("//gen:out.go", "//gen:gen")
	b.AddRule("//util:a", "go_library", nil)
	b.AddRule("//util:b", "go_library", []string{"//util:a"})
	b.AddRule("@dep//:ext", "go_library", nil)
	return b.Build()
}

func TestRootsAreBinaries(t *testing.T) {
	got := labels(sample().Roots(Options{}))
	if want := []string{"//app:bin"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Roots = %v, want %v", got, want)
	}
}

func TestRootsIncludeTests(t *testing.T) {
	got := labels(sample().Roots(Options{IncludeTests: true}))
	if want := []string{"//app:bin", "//app:lib_test"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Roots = %v, want %v", got, want)
	}
}

func TestRootsFallBackWhenNoBinaries(t *testing.T) {
	b := NewBuilder()
	b.AddRule("//a:lib", "go_library", []string{"//b:lib"})
	b.AddRule("//a:lib_test", "go_test", []string{"//a:lib"})
	b.AddRule("//b:lib", "go_library", nil)
	got := labels(b.Build().Roots(Options{}))
	// //a:lib is only used by its test, so it still counts as a root.
	if want := []string{"//a:lib"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Roots = %v, want %v", got, want)
	}
}

func TestChildrenFiltersAndResolvesEdges(t *testing.T) {
	g := sample()
	p, ok := g.Children("//app:lib", Options{})
	if !ok {
		t.Fatal("//app:lib not found")
	}
	want := []string{"//gen:gen", "//util:a", "//util:b"}
	if got := labels(p); !reflect.DeepEqual(got, want) {
		t.Errorf("Children = %v, want %v", got, want)
	}

	p, _ = g.Children("//app:lib", Options{IncludeExternal: true})
	want = append(want, "@dep//:ext")
	if got := labels(p); !reflect.DeepEqual(got, want) {
		t.Errorf("Children with external = %v, want %v", got, want)
	}
}

func TestChildrenUnknownLabel(t *testing.T) {
	if _, ok := sample().Children("//nope:nope", Options{}); ok {
		t.Error("expected unknown label to report false")
	}
}

func TestUnreportedExternalInputsBecomeLeaves(t *testing.T) {
	b := NewBuilder()
	b.AddRule("//app:lib", "go_library", []string{
		"@dep//:a",      // external, not in the output: becomes a leaf
		"//missing:lib", // main repo, not in the output: dropped
	})
	b.AddRule("//app:bin", "go_binary", []string{"//app:lib", "@dep//:a"})
	g := b.Build()

	p, _ := g.Children("//app:lib", Options{IncludeExternal: true})
	want := []Node{{Label: "@dep//:a", Kind: "", External: true, DepCount: 0, ParentCount: 2}}
	if !reflect.DeepEqual(p.Nodes, want) {
		t.Errorf("Children = %+v, want %+v", p.Nodes, want)
	}
	if got := labels(g.Roots(Options{})); !reflect.DeepEqual(got, []string{"//app:bin"}) {
		t.Errorf("Roots = %v, want [//app:bin]", got)
	}
}

func TestNodeCounts(t *testing.T) {
	g := sample()
	p := g.Roots(Options{IncludeTests: true})
	lib, _ := g.Children("//app:bin", Options{})
	got := lib.Nodes[0]
	if got.DepCount != 3 || got.ParentCount != 2 {
		t.Errorf("counts = %d deps, %d parents; want 3 and 2", got.DepCount, got.ParentCount)
	}
	if p.Nodes[0].DepCount != 1 {
		t.Errorf("//app:bin DepCount = %d, want 1", p.Nodes[0].DepCount)
	}

	lib, _ = g.Children("//app:bin", Options{IncludeExternal: true})
	if lib.Nodes[0].DepCount != 4 {
		t.Errorf("DepCount with external = %d, want 4", lib.Nodes[0].DepCount)
	}
}

func TestPaging(t *testing.T) {
	g := sample()
	tests := []struct {
		offset, limit int
		want          []string
	}{
		{0, 2, []string{"//gen:gen", "//util:a"}},
		{2, 2, []string{"//util:b"}},
		{9, 2, []string{}},
		{-1, 0, []string{"//gen:gen", "//util:a", "//util:b"}},
	}
	for _, tc := range tests {
		p, _ := g.Children("//app:lib", Options{Offset: tc.offset, Limit: tc.limit})
		if got := labels(p); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("offset=%d limit=%d: got %v, want %v", tc.offset, tc.limit, got, tc.want)
		}
		if p.Total != 3 {
			t.Errorf("Total = %d, want 3", p.Total)
		}
	}
}

// hotspotSample: //util:a is used by //app:lib, which //app:bin and
// //app:lib_test use; //util:a depends on @dep//:ext.
func hotspotSample() *Graph {
	b := NewBuilder()
	b.AddRule("//app:bin", "go_binary", []string{"//app:lib", "//app:main.go"})
	b.AddRule("//app:lib", "go_library", []string{"//util:a", "//app:lib.go"})
	b.AddRule("//app:lib_test", "go_test", []string{"//app:lib", "//app:lib.go", "//app:lib.go"})
	b.AddRule("//util:a", "go_library", []string{"//util:a.go", "@dep//:ext"})
	for _, f := range []string{"//app:main.go", "//app:lib.go", "//util:a.go", "//unused:x.go"} {
		b.AddSourceFile(f)
	}
	return b.Build()
}

type ranked struct {
	Label      string
	Dependents int
	Direct     int
}

func ranking(p Page) []ranked {
	out := make([]ranked, 0, len(p.Nodes))
	for _, n := range p.Nodes {
		out = append(out, ranked{n.Label, n.Dependents, n.ParentCount})
	}
	return out
}

func TestHotspotsRankTargetsByTransitiveDependents(t *testing.T) {
	g := hotspotSample()
	got := ranking(g.Hotspots(Options{}))
	want := []ranked{
		{"//util:a", 3, 1},
		{"//app:lib", 2, 2},
		{"//app:bin", 0, 0}, // Ties keep label order.
		{"//app:lib_test", 0, 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Hotspots = %v, want %v", got, want)
	}

	got = ranking(g.Hotspots(Options{IncludeExternal: true, Limit: 1}))
	if want := []ranked{{"@dep//:ext", 4, 1}}; !reflect.DeepEqual(got, want) {
		t.Errorf("Hotspots with external = %v, want %v", got, want)
	}

	p := g.Hotspots(Options{Offset: 1, Limit: 2})
	if got, want := ranking(p), want[1:3]; !reflect.DeepEqual(got, want) || p.Total != 4 {
		t.Errorf("Hotspots page = %v (total %d), want %v (total 4)", got, p.Total, want)
	}
}

func TestFileHotspotsCountTheirRulesDependents(t *testing.T) {
	p := hotspotSample().FileHotspots(Options{})
	want := []ranked{
		{"//util:a.go", 4, 1},  // //util:a and its 3 dependents.
		{"//app:lib.go", 3, 2}, // Listed twice by //app:lib_test, counted once.
		{"//app:main.go", 1, 1},
		{"//unused:x.go", 0, 0},
	}
	if got := ranking(p); !reflect.DeepEqual(got, want) {
		t.Errorf("FileHotspots = %v, want %v", got, want)
	}
	if p.Nodes[0].Kind != "source file" {
		t.Errorf("Kind = %q, want %q", p.Nodes[0].Kind, "source file")
	}
}
