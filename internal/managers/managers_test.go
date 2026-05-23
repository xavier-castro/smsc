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

func TestScanRemovePlansManagedPolicyRemoval(t *testing.T) {
	home := t.TempDir()
	configHome := filepath.Join(home, ".config")
	pnpmRC := filepath.Join(home, "Library", "Preferences", "pnpm", "rc")
	uvConfig := filepath.Join(configHome, "uv", "uv.toml")
	for _, path := range []string{pnpmRC, uvConfig} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(home, ".npmrc"):       "registry=https://example.test\nmin-release-age=8\nbefore=2026-05-12T00:00:00Z\n",
		pnpmRC:                              "minimumReleaseAge=11520\nregistry=https://example.test\n",
		filepath.Join(home, ".yarnrc.yml"):  "nodeLinker: node-modules\nnpmMinimalAgeGate: 8d\n",
		filepath.Join(home, ".bunfig.toml"): "telemetry = false\n\n[install]\nregistry = \"https://registry.npmjs.org\"\nminimumReleaseAge = 691200\n",
		uvConfig:                            "exclude-newer = \"8 days\"\n[pip]\nindex-url = \"https://example.test\"\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
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
				"vp":   "/bin/vp",
				"yarn": "/bin/yarn",
				"bun":  "/bin/bun",
				"uv":   "/bin/uv",
			},
			outputs: map[string]string{
				"npm --version":             "11.12.1",
				"npm config get userconfig": filepath.Join(home, ".npmrc"),
				"pnpm --version":            "10.33.0",
				"pnpm config get globalconfig --location=global": pnpmRC,
				"vp --version":   "0.1.0",
				"yarn --version": "4.10.0",
				"bun --version":  "1.3.0",
				"uv --version":   "uv 0.11.13",
			},
		},
	}
	statuses := ScanRemove(context.Background(), env)
	byID := map[string]Status{}
	selected := map[string]bool{}
	for _, status := range statuses {
		byID[status.ID] = status
		if status.ID != "vp" {
			selected[status.ID] = status.Selected
		}
	}
	for _, id := range []string{"npm", "pnpm", "yarn", "bun", "uv"} {
		status := byID[id]
		if !status.Selected || !status.NeedsChange {
			t.Fatalf("expected %s to be selected for removal: %#v", id, status)
		}
		if status.TargetAge != "remove" {
			t.Fatalf("expected %s target remove, got %q", id, status.TargetAge)
		}
	}
	if byID["vp"].ConfigPath != byID["pnpm"].ConfigPath {
		t.Fatalf("expected VP cleanup to target pnpm global config: vp=%q pnpm=%q", byID["vp"].ConfigPath, byID["pnpm"].ConfigPath)
	}
	if !byID["vp"].Selected {
		t.Fatalf("expected VP cleanup to be selected: %#v", byID["vp"])
	}

	changes := SelectChanges(statuses, selected)
	if len(changes) != 5 {
		t.Fatalf("got %d removal changes, want 5: %#v", len(changes), changes)
	}
	joinedAfter := ""
	for _, change := range changes {
		joinedAfter += change.After
	}
	for _, removed := range []string{"min-release-age", "minimumReleaseAge", "npmMinimalAgeGate", "minimumReleaseAge = 691200", "exclude-newer"} {
		if strings.Contains(joinedAfter, removed) {
			t.Fatalf("expected %q to be removed from planned output:\n%s", removed, joinedAfter)
		}
	}
	for _, preserved := range []string{"registry=https://example.test", "before=2026-05-12T00:00:00Z", "nodeLinker: node-modules", "telemetry = false", "[pip]"} {
		if !strings.Contains(joinedAfter, preserved) {
			t.Fatalf("expected %q to be preserved in planned output:\n%s", preserved, joinedAfter)
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

func TestRemoveHonorsSavePrefixToggle(t *testing.T) {
	home := t.TempDir()
	npmrc := filepath.Join(home, ".npmrc")
	pnpmRC := filepath.Join(home, "Library", "Preferences", "pnpm", "rc")
	if err := os.WriteFile(npmrc, []byte("min-release-age=8\nsave-prefix=~\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(pnpmRC), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pnpmRC, []byte("minimum-release-age=11520\nsave-prefix=~\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{
		HomeDir:         home,
		ConfigHome:      filepath.Join(home, ".config"),
		GOOS:            "darwin",
		SavePrefixTilde: true,
		Runner: fakeRunner{
			paths: map[string]string{
				"npm":  "/bin/npm",
				"pnpm": "/bin/pnpm",
			},
			outputs: map[string]string{
				"npm --version":             "11.12.1",
				"npm config get userconfig": npmrc,
				"pnpm --version":            "10.33.0",
				"pnpm config get globalconfig --location=global": pnpmRC,
			},
		},
	}
	npm := NPM{}.Remove(context.Background(), env)
	if !npm.NeedsChange || strings.Contains(npm.Changes[0].After, "save-prefix") || strings.Contains(npm.Changes[0].After, "min-release-age") {
		t.Fatalf("expected npm remove to clear managed keys: %#v", npm.Changes)
	}
	pnpm := PNPM{}.Remove(context.Background(), env)
	if !pnpm.NeedsChange || strings.Contains(pnpm.Changes[0].After, "save-prefix") || strings.Contains(pnpm.Changes[0].After, "minimumReleaseAge") {
		t.Fatalf("expected pnpm remove to clear managed keys: %#v", pnpm.Changes)
	}
}
