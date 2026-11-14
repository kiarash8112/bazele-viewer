// Package graph holds a compact, read-only Bazel dependency graph.
//
// Targets are identified internally by int32 IDs assigned in label order, so
// dependency lists are small and already sorted by label. A Graph is immutable
// once built and safe for concurrent use.
package graph

import (
	"cmp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// sourceFileKind is the kind of a source file node, as `bazel query
// --output=label_kind` prints it.
const sourceFileKind = "source file"

// Node is a target as exposed to callers.
type Node struct {
	Label       string `json:"label"`
	Kind        string `json:"kind"`
	External    bool   `json:"external"`
	DepCount    int    `json:"depCount"`
	ParentCount int    `json:"parentCount"`
	// Dependents is the number of targets that depend on this one, directly or
	// transitively. Only hotspot pages set it.
	Dependents int `json:"dependents,omitempty"`
	// Changes is the number of commits that changed this file, or this target's
	// source files, in the recent history given to SetChanges. Only hotspot
	// pages set it, and only when that history is known.
	Changes *int `json:"changes,omitempty"`
	// Rank is the 1-based position in the full hotspot ranking, which a search
	// keeps. Only hotspot pages set it.
	Rank int `json:"rank,omitempty"`
}

// Page is one window of a larger result.
type Page struct {
	Nodes []Node `json:"nodes"`
	Total int    `json:"total"`
}

// Options controls which targets a query returns and which window of them.
type Options struct {
	IncludeTests    bool   // Roots only: also return test targets.
	IncludeExternal bool   // Show targets from @external repositories.
	Query           string // Hotspots only: keep labels that contain it, ignoring case.
	SortByChanges   bool   // Hotspots only: rank by changes, then dependents, not by dependents.
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

	files     []string  // file id -> source file label, sorted
	fileUsers [][]int32 // file id -> rules that list the file as an input, sorted

	// Commit counts from SetChanges; nil until it is called.
	changes     []int32 // target id -> distinct commits that changed its source files
	fileChanges []int32 // file id -> commits that changed the file

	hotOnce sync.Once
	hot     hotspots
}

// hotspots ranks targets and source files by how much of the graph depends on
// them. It is computed on first use, so loading the graph stays fast.
type hotspots struct {
	dependents       []int32 // target id -> transitive dependents
	fileDependents   []int32 // file id -> transitive dependents
	targets          []int32 // target ids, most dependents first
	files            []int32 // file ids, most dependents first
	targetsByChanges []int32 // target ids, most changes first, then most dependents
	filesByChanges   []int32 // file ids, most changes first, then most dependents
}

// Len returns the number of targets in the graph.
func (g *Graph) Len() int { return len(g.labels) }

// Roots returns the main entry points of the workspace: the binaries that no
// non-test target depends on, or, if there are none, all the targets that no
// non-test target depends on.
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

// Target returns the node for label itself. The boolean is false if the label
// is not in the graph.
func (g *Graph) Target(label string, opts Options) (Node, bool) {
	id, ok := g.lookup(label)
	if !ok {
		return Node{}, false
	}
	return g.node(id, opts.IncludeExternal), true
}

// Hotspots returns targets ranked by their transitive dependents: the targets
// that a change to them affects.
func (g *Graph) Hotspots(opts Options) Page {
	hot := g.hotspots()
	ids := hot.targets
	if opts.SortByChanges {
		ids = hot.targetsByChanges
	}
	if !opts.IncludeExternal {
		ids = g.withoutExternal(ids)
	}
	ids, ranks := search(ids, func(id int32) string { return g.labels[id] }, opts.Query)
	start, end := window(len(ids), opts)
	nodes := make([]Node, 0, end-start)
	for i := start; i < end; i++ {
		n := g.node(ids[i], opts.IncludeExternal)
		n.Dependents = int(hot.dependents[ids[i]])
		n.Changes = count(g.changes, ids[i])
		n.Rank = int(ranks[i])
		nodes = append(nodes, n)
	}
	return Page{Nodes: nodes, Total: len(ids)}
}

// search keeps the ids whose label contains query, ignoring case, in order,
// with each one's 1-based rank in ids. An empty query keeps every id.
func search(ids []int32, label func(id int32) string, query string) (kept, ranks []int32) {
	query = strings.ToLower(strings.TrimSpace(query))
	kept = make([]int32, 0, len(ids))
	ranks = make([]int32, 0, len(ids))
	for i, id := range ids {
		if query == "" || strings.Contains(strings.ToLower(label(id)), query) {
			kept = append(kept, id)
			ranks = append(ranks, int32(i+1))
		}
	}
	return kept, ranks
}

// window returns the [start, end) slice of n results that opts asks for.
func window(n int, opts Options) (start, end int) {
	start = min(max(opts.Offset, 0), n)
	end = n
	if opts.Limit > 0 {
		end = min(start+opts.Limit, n)
	}
	return start, end
}

// count returns counts[id], or nil if the counts are unknown.
func count(counts []int32, id int32) *int {
	if counts == nil {
		return nil
	}
	n := int(counts[id])
	return &n
}

// SetChanges records the commits that changed each source file, keyed by path
// relative to the workspace (see gitchanges.Load). A target's count is the
// number of distinct commits that changed any of its own source files. Call it
// before the graph is shared: the hotspot rankings, computed on first use, sort
// by these counts.
func (g *Graph) SetChanges(commitsByPath map[string][]int32) {
	g.fileChanges = make([]int32, len(g.files))
	ruleCommits := make([][]int32, len(g.labels))
	for file, label := range g.files {
		commits := commitsByPath[labelPath(label)]
		g.fileChanges[file] = int32(len(commits))
		for _, rule := range g.fileUsers[file] {
			ruleCommits[rule] = append(ruleCommits[rule], commits...)
		}
	}
	g.changes = make([]int32, len(g.labels))
	for rule, commits := range ruleCommits {
		slices.Sort(commits)
		g.changes[rule] = int32(len(slices.Compact(commits)))
	}
}

// labelPath turns a main-repository file label into its path relative to the
// workspace: "//pkg/dir:sub/file.go" is "pkg/dir/sub/file.go".
func labelPath(label string) string {
	pkg, name, _ := strings.Cut(strings.TrimPrefix(label, "//"), ":")
	if pkg == "" {
		return name
	}
	return pkg + "/" + name
}

// FileHotspots returns source files ranked by the transitive dependents of the
// rules that use them.
func (g *Graph) FileHotspots(opts Options) Page {
	hot := g.hotspots()
	ids := hot.files
	if opts.SortByChanges {
		ids = hot.filesByChanges
	}
	ids, ranks := search(ids, func(id int32) string { return g.files[id] }, opts.Query)
	start, end := window(len(ids), opts)
	nodes := make([]Node, 0, end-start)
	for i := start; i < end; i++ {
		id := ids[i]
		nodes = append(nodes, Node{
			Label:       g.files[id],
			Kind:        sourceFileKind,
			ParentCount: len(g.fileUsers[id]),
			Dependents:  int(hot.fileDependents[id]),
			Changes:     count(g.fileChanges, id),
			Rank:        int(ranks[i]),
		})
	}
	return Page{Nodes: nodes, Total: len(ids)}
}

func (g *Graph) hotspots() *hotspots {
	g.hotOnce.Do(g.computeHotspots)
	return &g.hot
}

// computeHotspots walks the reverse dependencies of every target and file. The
// walks are independent, so they run on all CPUs, each worker with its own
// visited marks.
func (g *Graph) computeHotspots() {
	n := len(g.labels)
	jobs := n + len(g.files)
	dependents := make([]int32, jobs) // targets first, then files
	var next atomic.Int64
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			seen := make([]uint32, n)
			var stamp uint32
			var queue []int32
			for {
				job := int(next.Add(1) - 1)
				if job >= jobs {
					return
				}
				stamp++
				queue = queue[:0]
				var self int
				if job < n {
					seen[job] = stamp
					queue = append(queue, int32(job))
					self = 1 // A target is not its own dependent.
				} else {
					for _, user := range g.fileUsers[job-n] {
						seen[user] = stamp
						queue = append(queue, user)
					}
				}
				for i := 0; i < len(queue); i++ {
					for _, parent := range g.parents[queue[i]] {
						if seen[parent] != stamp {
							seen[parent] = stamp
							queue = append(queue, parent)
						}
					}
				}
				dependents[job] = int32(len(queue) - self)
			}
		}()
	}
	wg.Wait()

	g.hot.dependents = dependents[:n]
	g.hot.fileDependents = dependents[n:]

	targetDependents := func(id int32) int { return int(g.hot.dependents[id]) }
	targetDirect := func(id int32) int { return len(g.parents[id]) }
	targetChanges := func(id int32) int { return countAt(g.changes, id) }
	fileDependents := func(id int32) int { return int(g.hot.fileDependents[id]) }
	fileDirect := func(id int32) int { return len(g.fileUsers[id]) }
	fileChanges := func(id int32) int { return countAt(g.fileChanges, id) }

	g.hot.targets = rankBy(n, targetDependents, targetDirect)
	g.hot.files = rankBy(len(g.files), fileDependents, fileDirect)
	g.hot.targetsByChanges = rankBy(n, targetChanges, targetDependents, targetDirect)
	g.hot.filesByChanges = rankBy(len(g.files), fileChanges, fileDependents, fileDirect)
}

// countAt returns counts[id], or 0 if the counts are unknown.
func countAt(counts []int32, id int32) int {
	if counts == nil {
		return 0
	}
	return int(counts[id])
}

// rankBy returns ids 0..n-1 ordered by each key in turn, all descending. Ids
// are in label order, so full ties keep it.
func rankBy(n int, keys ...func(id int32) int) []int32 {
	ids := make([]int32, n)
	for i := range ids {
		ids[i] = int32(i)
	}
	slices.SortStableFunc(ids, func(x, y int32) int {
		for _, key := range keys {
			if c := cmp.Compare(key(y), key(x)); c != 0 {
				return c
			}
		}
		return 0
	})
	return ids
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
	sources    map[int32]bool  // source file ids
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
		sources:    make(map[int32]bool),
		kindIDs:    make(map[string]uint16),
	}
}

// AddSourceFile records a source file, so that the rules using it can be
// counted for file hotspots.
func (b *Builder) AddSourceFile(label string) {
	b.sources[b.intern(label)] = true
}

// AddRule records a rule target and its inputs. Inputs may be rules, generated
// files or source files; only edges that lead to a rule, or to an @external
// target, are kept.
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
	b.addExternalLeaves()
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

	b.buildFiles(g)
	g.computeRoots()
	return g
}

// buildFiles records each source file and the rules that use it. Rules are
// already in g, in dense id order.
func (b *Builder) buildFiles(g *Graph) {
	raws := make([]int32, 0, len(b.sources))
	for raw := range b.sources {
		raws = append(raws, raw)
	}
	slices.SortFunc(raws, func(x, y int32) int { return strings.Compare(b.labels[x], b.labels[y]) })

	fileIDs := make(map[int32]int32, len(raws))
	g.files = make([]string, len(raws))
	g.fileUsers = make([][]int32, len(raws))
	for i, raw := range raws {
		fileIDs[raw] = int32(i)
		g.files[i] = b.labels[raw]
	}
	for i, r := range b.rules {
		for _, raw := range r.inputs {
			if file, ok := fileIDs[raw]; ok {
				g.fileUsers[file] = append(g.fileUsers[file], int32(i))
			}
		}
	}
	for i, users := range g.fileUsers {
		g.fileUsers[i] = slices.Compact(users) // Rules are visited in order, so duplicates are adjacent.
	}
}

// addExternalLeaves adds a target with no kind and no deps for each @external
// input that the query did not report. A query that does not load external
// repositories, such as //...:*, gives only their labels.
func (b *Builder) addExternalLeaves() {
	known := make([]bool, len(b.labels))
	for _, r := range b.rules {
		known[r.id] = true
	}
	for raw := range b.generating {
		known[raw] = true
	}
	noKind := b.internKind("")
	for i := range len(b.rules) {
		for _, raw := range b.rules[i].inputs {
			if !known[raw] && isExternal(b.labels[raw]) {
				known[raw] = true
				b.rules = append(b.rules, rawRule{id: raw, kind: noKind})
			}
		}
	}
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
		case g.hasNonTestParent(id):
			// Not an entry point, even if it is a binary: a code generator
			// or other tool another target runs, such as rules_js's
			// _run_binary and js_binary.
		case strings.HasSuffix(g.kindNames[g.kinds[id]], "_binary"):
			binaries = append(binaries, id)
		default:
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
