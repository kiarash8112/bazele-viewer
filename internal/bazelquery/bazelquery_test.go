package bazelquery

import (
	"slices"
	"strings"
	"testing"

	"example.com/bazelviewer/internal/graph"
)

const sampleOutput = `{"type":"SOURCE_FILE","sourceFile":{"name":"//a:a.go"}}
{"type":"RULE","rule":{"name":"//a:bin","ruleClass":"go_binary","attribute":[{"name":"x"}],"ruleInput":["//a:a.go","//a:gen.go"]}}
{"type":"GENERATED_FILE","generatedFile":{"name":"//a:gen.go","generatingRule":"//a:gen"}}
{"type":"RULE","rule":{"name":"//a:gen","ruleClass":"genrule"}}
`

func TestParse(t *testing.T) {
	b := graph.NewBuilder()
	if err := Parse(strings.NewReader(sampleOutput), b); err != nil {
		t.Fatal(err)
	}
	g := b.Build()
	if g.Len() != 2 {
		t.Fatalf("Len = %d, want 2 rules", g.Len())
	}
	p, ok := g.Children("//a:bin", graph.Options{})
	if !ok || len(p.Nodes) != 1 || p.Nodes[0].Label != "//a:gen" || p.Nodes[0].Kind != "genrule" {
		t.Errorf("Children = %+v, %v; want //a:gen genrule", p, ok)
	}
	files := g.FileHotspots(graph.Options{})
	if len(files.Nodes) != 1 || files.Nodes[0].Label != "//a:a.go" || files.Nodes[0].ParentCount != 1 {
		t.Errorf("FileHotspots = %+v; want //a:a.go used by //a:bin", files)
	}
}

func TestParseRejectsTruncatedOutput(t *testing.T) {
	err := Parse(strings.NewReader(`{"type":"RULE","rule":{"name"`), graph.NewBuilder())
	if err == nil {
		t.Error("expected an error for truncated output")
	}
}

func TestQueryEnvDropsBazeliskSkipWrapper(t *testing.T) {
	got := queryEnv([]string{"PATH=/bin", "BAZELISK_SKIP_WRAPPER=true", "HOME=/h"})
	want := []string{"PATH=/bin", "HOME=/h"}
	if !slices.Equal(got, want) {
		t.Errorf("queryEnv = %q, want %q", got, want)
	}
}

func TestTailWriterKeepsEnd(t *testing.T) {
	w := &tailWriter{limit: 4}
	w.Write([]byte("abc"))
	w.Write([]byte("defg"))
	if got := w.String(); got != "defg" {
		t.Errorf("tail = %q, want %q", got, "defg")
	}
}
