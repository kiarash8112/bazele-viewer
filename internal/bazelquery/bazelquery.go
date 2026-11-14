// Package bazelquery loads a workspace's dependency graph from `bazel query`.
package bazelquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"

	"example.com/bazelviewer/internal/graph"
)

// exitPartial is Bazel's exit code when --keep_going produced partial results.
const exitPartial = 3

const stderrTailSize = 4096

// queryExpr is every target in the main repository, each rule with its direct
// deps. Unlike deps(//...), //...:* does not load external repositories, which
// would fetch every toolchain they register (for example a Python interpreter
// per platform), so @external targets appear only as labels.
//
// It leaves out the targets rules_js's npm_link_all_packages creates: a
// node_modules/<package> link per npm dependency and four hidden
// .aspect_rules_js/... targets per package in the lockfile, hundreds in all.
// They are third-party packages, like @external targets, and edges to them
// are dropped with them.
const queryExpr = `//...:* except attr(generator_function, "^npm_link_all_packages$", //...:*)`

// Options configures Load.
type Options struct {
	Bazel     string // Path to the bazel binary.
	Workspace string // Directory to run bazel in.
}

// Load runs the query and builds the graph. A non-empty warning means the graph
// is usable but incomplete, for example because a BUILD file is broken.
func Load(ctx context.Context, opts Options) (g *graph.Graph, warning string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, opts.Bazel,
		"query", queryExpr,
		"--keep_going",
		"--output=streamed_jsonproto",
		"--noshow_progress",
		"--color=no",
		"--curses=no",
	)
	cmd.Dir = opts.Workspace
	cmd.Env = queryEnv(os.Environ())
	stderr := &tailWriter{limit: stderrTailSize}
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", err
	}
	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("start bazel: %w", err)
	}

	b := graph.NewBuilder()
	parseErr := Parse(stdout, b)
	if parseErr != nil {
		cancel() // Stop bazel: nothing is reading its output any more.
	}
	waitErr := cmd.Wait()

	if parseErr != nil {
		return nil, "", fmt.Errorf("read query output: %w (bazel: %v; %s)", parseErr, waitErr, stderr)
	}
	if waitErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(waitErr, &exitErr) || exitErr.ExitCode() != exitPartial {
			return nil, "", fmt.Errorf("bazel query: %w: %s", waitErr, stderr)
		}
		warning = "graph is incomplete: " + stderr.String()
	}
	return b.Build(), warning, nil
}

// queryEnv returns environ without BAZELISK_SKIP_WRAPPER. Bazelisk sets it for
// the viewer under `bazel run`; dropping it lets the inspected workspace's own
// tools/bazel wrapper run, as it does from a terminal.
func queryEnv(environ []string) []string {
	return slices.DeleteFunc(slices.Clone(environ), func(kv string) bool {
		return strings.HasPrefix(kv, "BAZELISK_SKIP_WRAPPER=")
	})
}

// record holds the only fields kept from each query result. The decoder skips
// everything else, including the large attribute lists.
type record struct {
	Rule *struct {
		Name      string   `json:"name"`
		RuleClass string   `json:"ruleClass"`
		RuleInput []string `json:"ruleInput"`
	} `json:"rule"`
	GeneratedFile *struct {
		Name           string `json:"name"`
		GeneratingRule string `json:"generatingRule"`
	} `json:"generatedFile"`
	SourceFile *struct {
		Name string `json:"name"`
	} `json:"sourceFile"`
}

// Parse reads --output=streamed_jsonproto results from r into b, one target at
// a time.
func Parse(r io.Reader, b *graph.Builder) error {
	dec := json.NewDecoder(r)
	for {
		var rec record
		if err := dec.Decode(&rec); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		switch {
		case rec.Rule != nil:
			b.AddRule(rec.Rule.Name, rec.Rule.RuleClass, rec.Rule.RuleInput)
		case rec.GeneratedFile != nil:
			b.AddGeneratedFile(rec.GeneratedFile.Name, rec.GeneratedFile.GeneratingRule)
		case rec.SourceFile != nil:
			b.AddSourceFile(rec.SourceFile.Name)
		}
	}
}

// tailWriter keeps the last limit bytes written to it.
type tailWriter struct {
	limit int
	buf   []byte
}

func (w *tailWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	if extra := len(w.buf) - w.limit; extra > 0 {
		w.buf = append(w.buf[:0], w.buf[extra:]...)
	}
	return len(p), nil
}

func (w *tailWriter) String() string {
	return strings.TrimSpace(string(w.buf))
}
