// Package graph holds a compact, read-only Bazel dependency graph.
//
// Targets are identified internally by int32 IDs assigned in label order, so
// dependency lists are small and already sorted by label. A Graph is immutable
// once built and safe for concurrent use.
package graph

import (
	"slices"
	"strings"
)

// Node is a target as exposed to callers.
type Node struct {
	Label       string `json:"label"`
	Kind        string `json:"kind"`
	External    bool   `json:"external"`
	DepCount    int    `json:"depCount"`
	ParentCount int    `json:"parentCount"`
}

// Page is one window of a larger result.
type Page struct {
	Nodes []Node `json:"nodes"`
	Total int    `json:"total"`
}

// Options controls which targets a query returns and which window of them.
type Options struct {
	IncludeTests    bool // Roots only: also return test targets.
	IncludeExternal bool // Show targets from @external repositories.
	Offset          int
	Limit           int
}

// Graph is an immutable dependency graph of rule targets.
type Graph struct {
	labels        []string // id -> label, sorted
	kinds         []uint16 // id -> index into kindNames
	kindNames     []string
	deps          [][]int32 // id -> direct deps, sorted
	parents       [][]int32 // id -> reverse deps, sorted
	roots         []int32
	rootsAndTests []int32
}

// Len returns the number of targets in the graph.
func (g *Graph) Len() int { return len(g.labels) }

// Roots returns the main entry points of the workspace: binaries, or, if the
// workspace has none, the targets that no non-test target depends on.
func (g *Graph) Roots(opts Options) Page {
	ids := g.roots
	if opts.IncludeTests {
		ids = g.rootsAndTests
	}
	return g.page(ids, opts)
}

// Children returns the direct dependencies of label. The boolean is false if
// the label is not in the graph.
func (g *Graph) Children(label string, opts Options) (Page, bool) {
	id, ok := g.lookup(label)
	if !ok {
		return Page{}, false
	}
	ids := g.deps[id]
	if !opts.IncludeExternal {
		ids = g.withoutExternal(ids)
	}
	return g.page(ids, opts), true
}

func (g *Graph) lookup(label string) (int32, bool) {
	i, ok := slices.BinarySearch(g.labels, label)
	return int32(i), ok
}

func (g *Graph) withoutExternal(ids []int32) []int32 {
	kept := make([]int32, 0, len(ids))
	for _, id := range ids {
		if !isExternal(g.labels[id]) {
			kept = append(kept, id)
		}
	}
	return kept
}

func (g *Graph) page(ids []int32, opts Options) Page {
	start := min(max(opts.Offset, 0), len(ids))
	end := len(ids)
	if opts.Limit > 0 {
		end = min(start+opts.Limit, len(ids))
	}
	nodes := make([]Node, 0, end-start)
	for _, id := range ids[start:end] {
		nodes = append(nodes, g.node(id, opts.IncludeExternal))
	}
	return Page{Nodes: nodes, Total: len(ids)}
}

func (g *Graph) node(id int32, includeExternal bool) Node {
	depCount := len(g.deps[id])
	if !includeExternal {
		depCount = 0
		for _, dep := range g.deps[id] {
			if !isExternal(g.labels[dep]) {
				depCount++
			}
		}
	}
	return Node{
		Label:       g.labels[id],
		Kind:        g.kindNames[g.kinds[id]],
		External:    isExternal(g.labels[id]),
		DepCount:    depCount,
		ParentCount: len(g.parents[id]),
	}
}

func isExternal(label string) bool {
	return strings.HasPrefix(label, "@") &&
		!strings.HasPrefix(label, "@//") &&
		!strings.HasPrefix(label, "@@//")
}

// Builder accumulates query results and produces a Graph.
type Builder struct {
	ids        map[string]int32
	labels     []string
	rules      []rawRule
	generating map[int32]int32 // generated file id -> generating rule id
	kindIDs    map[string]uint16
	kindNames  []string
}

type rawRule struct {
	id     int32
	kind   uint16
	inputs []int32
}

// NewBuilder returns an empty Builder.
func NewBuilder() *Builder {
	return &Builder{
		ids:        make(map[string]int32),
		generating: make(map[int32]int32),
		kindIDs:    make(map[string]uint16),
	}
}

// AddRule records a rule target and its inputs. Inputs may be rules, generated
// files or source files; only edges that lead to a rule are kept.
func (b *Builder) AddRule(label, kind string, inputs []string) {
	r := rawRule{id: b.intern(label), kind: b.internKind(kind)}
	r.inputs = make([]int32, len(inputs))
	for i, in := range inputs {
		r.inputs[i] = b.intern(in)
	}
	b.rules = append(b.rules, r)
}

// AddGeneratedFile records the rule that produces a generated file, so that a
// dependency on the file becomes a dependency on that rule.
func (b *Builder) AddGeneratedFile(label, generatingRule string) {
	b.generating[b.intern(label)] = b.intern(generatingRule)
}

func (b *Builder) intern(label string) int32 {
	if id, ok := b.ids[label]; ok {
		return id
	}
	id := int32(len(b.labels))
	b.ids[label] = id
	b.labels = append(b.labels, label)
	return id
}

func (b *Builder) internKind(kind string) uint16 {
	if id, ok := b.kindIDs[kind]; ok {
		return id
	}
	id := uint16(len(b.kindNames))
	b.kindIDs[kind] = id
	b.kindNames = append(b.kindNames, kind)
	return id
}

// Build produces the Graph. The Builder must not be used afterwards.
func (b *Builder) Build() *Graph {
	slices.SortFunc(b.rules, func(x, y rawRule) int {
		return strings.Compare(b.labels[x.id], b.labels[y.id])
	})

	// Map raw IDs to dense rule IDs, which follow label order.
	dense := make([]int32, len(b.labels))
	for i := range dense {
		dense[i] = -1
	}
	for i, r := range b.rules {
		dense[r.id] = int32(i)
	}
	resolve := func(raw int32) int32 {
		if rule, ok := b.generating[raw]; ok {
			return dense[rule]
		}
		return dense[raw]
	}

	n := len(b.rules)
	g := &Graph{
		labels:    make([]string, n),
		kinds:     make([]uint16, n),
		kindNames: b.kindNames,
		deps:      make([][]int32, n),
		parents:   make([][]int32, n),
	}
	for i, r := range b.rules {
		g.labels[i] = b.labels[r.id]
		g.kinds[i] = r.kind

		deps := make([]int32, 0, len(r.inputs))
		for _, raw := range r.inputs {
			if dep := resolve(raw); dep >= 0 && dep != int32(i) {
				deps = append(deps, dep)
			}
		}
		slices.Sort(deps)
		g.deps[i] = slices.Clip(slices.Compact(deps))
	}

	for i, deps := range g.deps {
		for _, dep := range deps {
			g.parents[dep] = append(g.parents[dep], int32(i))
		}
	}

	g.computeRoots()
	return g
}

func (g *Graph) computeRoots() {
	var binaries, tests, unused []int32
	for i := range g.labels {
		id := int32(i)
		if isExternal(g.labels[id]) {
			continue
		}
		switch {
		case g.isTest(id):
			tests = append(tests, id)
		case strings.HasSuffix(g.kindNames[g.kinds[id]], "_binary"):
			binaries = append(binaries, id)
		case !g.hasNonTestParent(id):
			unused = append(unused, id)
		}
	}

	g.roots = binaries
	if len(binaries) == 0 {
		g.roots = unused
	}
	g.rootsAndTests = append(slices.Clone(g.roots), tests...)
	slices.Sort(g.rootsAndTests)
}

func (g *Graph) isTest(id int32) bool {
	return strings.HasSuffix(g.kindNames[g.kinds[id]], "_test")
}

func (g *Graph) hasNonTestParent(id int32) bool {
	for _, parent := range g.parents[id] {
		if !g.isTest(parent) {
			return true
		}
	}
	return false
}
