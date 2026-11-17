package bazelquery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"example.com/bazelviewer/internal/graph"
)

// Module kinds, as the graph reports them.
const (
	rootModuleKind = "root module"
	moduleKind     = "module"
)

// rootKey is the key `bazel mod graph` gives the root module.
const rootKey = "<root>"

// LoadModules runs `bazel mod graph` and builds the graph of Bazel modules:
// each module is a node, labelled name@version, with an edge to each module it
// depends on. The root module is the only root.
func LoadModules(ctx context.Context, opts Options) (*graph.Graph, error) {
	cmd := exec.CommandContext(ctx, opts.Bazel,
		"mod", "graph",
		"--output=json",
		"--noshow_progress",
		"--color=no",
		"--curses=no",
	)
	cmd.Dir = opts.Workspace
	cmd.Env = queryEnv(os.Environ())
	stderr := &tailWriter{limit: stderrTailSize}
	cmd.Stderr = stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bazel mod graph: %w: %s", err, stderr)
	}
	return ParseModules(bytes.NewReader(out))
}

// module is one node of `bazel mod graph --output=json`. A module that already
// appeared higher up is "unexpanded": it has a key and no dependency lists.
type module struct {
	Key                  string   `json:"key"`
	Name                 string   `json:"name"`
	Version              string   `json:"version"`
	Root                 bool     `json:"root"`
	Unexpanded           bool     `json:"unexpanded"`
	Dependencies         []module `json:"dependencies"`
	IndirectDependencies []module `json:"indirectDependencies"`
	Cycles               []module `json:"cycles"`
}

// ParseModules reads the JSON output of `bazel mod graph` into a graph.
func ParseModules(r io.Reader) (*graph.Graph, error) {
	var root module
	if err := json.NewDecoder(r).Decode(&root); err != nil {
		return nil, err
	}
	if root.Key == "" {
		return nil, fmt.Errorf("bazel mod graph: no root module in the output")
	}

	// The root module's key is "<root>"; show it as name@version, like the rest.
	rootLabel := root.Name
	if rootLabel == "" {
		rootLabel = rootKey
	} else if root.Version != "" {
		rootLabel += "@" + root.Version
	}
	label := func(m *module) string {
		if m.Key == rootKey {
			return rootLabel
		}
		return m.Key
	}

	b := graph.NewBuilder()
	added := make(map[string]bool)
	referenced := make(map[string]bool)
	var walk func(m *module)
	walk = func(m *module) {
		l := label(m)
		if m.Unexpanded || added[l] {
			return
		}
		added[l] = true
		var deps []string
		for _, list := range [][]module{m.Dependencies, m.IndirectDependencies, m.Cycles} {
			for i := range list {
				d := label(&list[i])
				deps = append(deps, d)
				referenced[d] = true
			}
		}
		kind := moduleKind
		if m.Key == rootKey {
			kind = rootModuleKind
		}
		b.AddRule(l, kind, deps)
		for _, list := range [][]module{m.Dependencies, m.IndirectDependencies, m.Cycles} {
			for i := range list {
				walk(&list[i])
			}
		}
	}
	walk(&root)

	// A module seen only as unexpanded still needs a node, or edges to it are
	// dropped. Bazel expands every module once, so this is only a safeguard.
	for l := range referenced {
		if !added[l] {
			b.AddRule(l, moduleKind, nil)
		}
	}
	return b.Build(), nil
}
