package bazelquery

import (
	"strings"
	"testing"

	"example.com/bazelviewer/internal/graph"
)

// sampleModules is trimmed `bazel mod graph --output=json` output. skylib
// appears twice; the second time it is unexpanded.
const sampleModules = `{
  "key": "<root>", "name": "app", "version": "1.0", "root": true,
  "dependencies": [
    {"key": "rules_go@0.57.0", "name": "rules_go", "version": "0.57.0",
     "dependencies": [
       {"key": "bazel_skylib@1.9.2", "name": "bazel_skylib", "version": "1.9.2",
        "dependencies": [], "indirectDependencies": [], "cycles": []}
     ],
     "indirectDependencies": [], "cycles": []},
    {"key": "bazel_skylib@1.9.2", "name": "bazel_skylib", "version": "1.9.2", "unexpanded": true}
  ],
  "indirectDependencies": [], "cycles": []
}`

func TestParseModules(t *testing.T) {
	g, err := ParseModules(strings.NewReader(sampleModules))
	if err != nil {
		t.Fatal(err)
	}
	if g.Len() != 3 {
		t.Fatalf("Len = %d, want 3 modules", g.Len())
	}
	roots := g.Roots(graph.Options{})
	if roots.Total != 1 || roots.Nodes[0].Label != "app@1.0" || roots.Nodes[0].Kind != "root module" || roots.Nodes[0].DepCount != 2 {
		t.Errorf("Roots = %+v; want app@1.0 with 2 deps", roots)
	}
	p, ok := g.Parents("bazel_skylib@1.9.2", graph.Options{})
	if !ok || p.Total != 2 {
		t.Errorf("Parents(bazel_skylib) = %+v, %v; want app and rules_go", p, ok)
	}
}

func TestParseModulesRejectsBadOutput(t *testing.T) {
	for _, in := range []string{`{"key":`, `{}`} {
		if _, err := ParseModules(strings.NewReader(in)); err == nil {
			t.Errorf("ParseModules(%q): expected an error", in)
		}
	}
}
