package gitchanges

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestLoad runs real git on a temporary repository whose workspace is a
// subdirectory, so paths must come back relative to it.
func TestLoad(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		cmd.Env = append(os.Environ(), "HOME="+repo, "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(repo, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	git("init", "-q")
	write("ws/pkg/a.go", "1")
	write("outside.txt", "1")
	git("add", "-A")
	git("commit", "-q", "-m", "first")
	write("ws/pkg/a.go", "2")
	write("ws/b.go", "1")
	git("add", "-A")
	git("commit", "-q", "-m", "second")

	got, err := Load(context.Background(), filepath.Join(repo, "ws"), "1 year ago")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]int32{"pkg/a.go": {0, 1}, "b.go": {0}} // Newest commit is 0.
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load = %v, want %v", got, want)
	}
}

func TestParse(t *testing.T) {
	log := "\x1e\n\na/x.go\nb/y.go\n" + // commit 0
		"\x1e\n\n" + // commit 1 changed no files under the workspace
		"\x1e\n\na/x.go\n" // commit 2
	got, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]int32{"a/x.go": {0, 2}, "b/y.go": {0}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse = %v, want %v", got, want)
	}
}

func TestParseIgnoresPathsBeforeTheFirstCommit(t *testing.T) {
	got, err := Parse(strings.NewReader("stray\n\x1e\nc.go\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string][]int32{"c.go": {0}}; !reflect.DeepEqual(got, want) {
		t.Errorf("Parse = %v, want %v", got, want)
	}
}
