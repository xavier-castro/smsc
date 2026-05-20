package managers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	paths   map[string]string
	outputs map[string]string
}

func (f fakeRunner) LookPath(file string) (string, error) {
	if path, ok := f.paths[file]; ok {
		return path, nil
	}
	return "", os.ErrNotExist
}

func (f fakeRunner) Output(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	if out, ok := f.outputs[key]; ok {
		return out, nil
	}
	return "", nil
}

func TestScanPlansEightDayPolicies(t *testing.T) {
	home := t.TempDir()
	configHome := filepath.Join(home, ".config")
	pnpmRC := filepath.Join(home, "Library", "Preferences", "pnpm", "rc")
	if err := os.MkdirAll(filepath.Dir(pnpmRC), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".npmrc"), []byte("registry=https://example.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{
		HomeDir:    home,
		ConfigHome: configHome,
		GOOS:       "darwin",
		Now:        func() time.Time { return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC) },
		Runner: fakeRunner{
			paths: map[string]string{
				"npm":  "/bin/npm",
				"pnpm": "/bin/pnpm",
				"yarn": "/bin/yarn",
				"bun":  "/bin/bun",
				"uv":   "/bin/uv",
			},
			outputs: map[string]string{
				"npm --version":             "11.12.1",
				"npm config get userconfig": filepath.Join(home, ".npmrc"),
				"pnpm --version":            "10.33.0",
				"pnpm config get globalconfig --location=global": pnpmRC,
				"yarn --version": "4.10.0",
				"bun --version":  "1.3.0",
				"uv --version":   "uv 0.11.13",
			},
		},
	}
	statuses := Scan(context.Background(), env, 8, false)
	selected := map[string]bool{}
	for _, status := range statuses {
		if status.ID != "vp" {
			selected[status.ID] = status.Selected
		}
	}
	changes := SelectChanges(statuses, selected)
	if len(changes) != 5 {
		t.Fatalf("got %d changes, want 5: %#v", len(changes), changes)
	}
	joined := ""
	for _, change := range changes {
		joined += change.After
	}
	for _, want := range []string{"min-release-age=8", "minimum-release-age=11520", "npmMinimalAgeGate: 8d", "minimumReleaseAge = 691200", "exclude-newer = \"8 days\""} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q in planned changes:\n%s", want, joined)
		}
	}
}

func TestStricterPolicyIsPreserved(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".npmrc")
	if err := os.WriteFile(path, []byte("min-release-age=30\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{
		HomeDir:    home,
		ConfigHome: filepath.Join(home, ".config"),
		Runner: fakeRunner{
			paths: map[string]string{"npm": "/bin/npm"},
			outputs: map[string]string{
				"npm --version":             "11.12.1",
				"npm config get userconfig": path,
			},
		},
	}
	status := NPM{}.Plan(context.Background(), env, 8, false)
	if !status.AlreadyStricter {
		t.Fatalf("expected stricter policy preservation: %#v", status)
	}
	if len(status.Changes) != 0 {
		t.Fatalf("expected no changes for stricter policy: %#v", status.Changes)
	}
}
