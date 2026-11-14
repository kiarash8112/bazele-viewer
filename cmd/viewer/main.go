// Command viewer serves a Bazel workspace's dependency graph over HTTP.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"example.com/bazelviewer/internal/api"
	"example.com/bazelviewer/internal/bazelquery"
	"example.com/bazelviewer/internal/gitchanges"
	"example.com/bazelviewer/web"
)

// changesSince is how far back hotspots count changes, in git's date syntax.
// The hotspots view labels the column with the same window.
const changesSince = "6 months ago"

func main() {
	// `bazel run` changes the working directory; this variable has the original.
	defaultWorkspace := os.Getenv("BUILD_WORKING_DIRECTORY")
	if defaultWorkspace == "" {
		defaultWorkspace = "."
	}
	// Under `bazel run`, Bazelisk puts the viewer's own Bazel version first on
	// PATH, so "bazel" would query the inspected workspace with the wrong
	// version (and restart its server). Bazelisk picks the version from the
	// inspected workspace's .bazelversion instead.
	defaultBazel := "bazel"
	if _, err := exec.LookPath("bazelisk"); err == nil {
		defaultBazel = "bazelisk"
	}
	addr := flag.String("addr", "127.0.0.1:8090", "listen address")
	workspace := flag.String("workspace", defaultWorkspace, "Bazel workspace to inspect")
	bazel := flag.String("bazel", defaultBazel, "path to the bazel binary")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opts := bazelquery.Options{Bazel: *bazel, Workspace: *workspace}
	server := api.New(ctx, func(ctx context.Context) (api.Source, string, error) {
		// Read the git history while bazel runs; it takes about a second.
		type history struct {
			commits map[string][]int32
			err     error
		}
		historyCh := make(chan history, 1)
		go func() {
			commits, err := gitchanges.Load(ctx, *workspace, changesSince)
			historyCh <- history{commits, err}
		}()

		g, warning, err := bazelquery.Load(ctx, opts)
		h := <-historyCh
		if err != nil {
			return nil, "", err
		}
		if h.err != nil {
			// The graph is still useful without change counts.
			log.Printf("change counts unavailable: %v", h.err)
			warning = strings.TrimSpace(warning + "\nchange counts unavailable: " + h.err.Error())
		} else {
			g.SetChanges(h.commits)
		}
		return g, warning, nil
	})
	server.Refresh()

	mux := http.NewServeMux()
	mux.Handle("/api/", server.Handler())
	mux.Handle("/", http.FileServerFS(web.FS()))

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("listening on http://%s (workspace %s)", *addr, *workspace)
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
