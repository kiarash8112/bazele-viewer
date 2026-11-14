// Package gitchanges finds the commits that changed each file in a workspace's
// git history.
package gitchanges

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// commitMark starts each commit in the log output (an ASCII record separator).
const commitMark = "\x1e"

// Load returns, for each file under workspace that a commit changed since
// `since` (a git date such as "6 months ago"), the commits that changed it.
// Commits are numbered from 0, newest first; paths are relative to workspace.
// Merge commits are skipped, so a change is counted once.
func Load(ctx context.Context, workspace, since string) (map[string][]int32, error) {
	cmd := exec.CommandContext(ctx, "git",
		"-C", workspace,
		"-c", "core.quotePath=false",
		"log",
		"--since="+since,
		"--no-merges",
		"--relative",
		"--name-only",
		"--format=%x1e", // git prints commitMark; it rejects the raw byte here.
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start git: %w", err)
	}
	commits, parseErr := Parse(stdout)
	if parseErr != nil {
		_, _ = io.Copy(io.Discard, stdout) // Let git finish writing, or Wait blocks.
	}
	waitErr := cmd.Wait()
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return nil, fmt.Errorf("git log: %w: %s", waitErr, strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("git log: %w", waitErr)
	}
	if parseErr != nil {
		return nil, fmt.Errorf("read git log: %w", parseErr)
	}
	return commits, nil
}

// Parse reads `git log --name-only --format=%x1e` output: a commitMark line
// for each commit, followed by the paths it changed.
func Parse(r io.Reader) (map[string][]int32, error) {
	commits := make(map[string][]int32)
	commit := int32(-1)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == commitMark:
			commit++
		case line != "" && commit >= 0:
			commits[line] = append(commits[line], commit)
		}
	}
	return commits, sc.Err()
}
