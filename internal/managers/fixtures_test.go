package managers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xavier/smsc/internal/config"
)

func TestFixtureExistingPoliciesAndAliases(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC) }

	t.Run("npm stricter days", func(t *testing.T) {
		home := t.TempDir()
		path := copyManagerFixture(t, "npm-stricter.npmrc", filepath.Join(home, ".npmrc"))
		status := NPM{}.Plan(context.Background(), Env{HomeDir: home, ConfigHome: filepath.Join(home, ".config"), Now: now, Runner: fakeRunner{paths: map[string]string{"npm": "/bin/npm"}, outputs: map[string]string{"npm --version": "11.12.1", "npm config get userconfig": path}}}, 8, false)
		assertProtectedNoChange(t, status)
	})

	t.Run("pnpm camelCase alias", func(t *testing.T) {
		home := t.TempDir()
		path := copyManagerFixture(t, "pnpm-alias.rc", filepath.Join(home, ".config", "pnpm", "rc"))
		status := PNPM{}.Plan(context.Background(), Env{HomeDir: home, ConfigHome: filepath.Join(home, ".config"), Now: now, Runner: fakeRunner{paths: map[string]string{"pnpm": "/bin/pnpm"}, outputs: map[string]string{"pnpm --version": "10.33.0", "pnpm config get globalconfig --location=global": path}}}, 8, false)
		assertProtectedNoChange(t, status)
	})

	t.Run("yarn numeric minutes stricter", func(t *testing.T) {
		home := t.TempDir()
		copyManagerFixture(t, "yarn-numeric-stricter.yml", filepath.Join(home, ".yarnrc.yml"))
		status := Yarn{}.Plan(context.Background(), Env{HomeDir: home, ConfigHome: filepath.Join(home, ".config"), Now: now, Runner: fakeRunner{paths: map[string]string{"yarn": "/bin/yarn"}, outputs: map[string]string{"yarn --version": "4.10.0"}}}, 8, false)
		assertProtectedNoChange(t, status)
	})

	t.Run("bun stricter seconds", func(t *testing.T) {
		home := t.TempDir()
		copyManagerFixture(t, "bun-existing.toml", filepath.Join(home, ".bunfig.toml"))
		status := Bun{}.Plan(context.Background(), Env{HomeDir: home, ConfigHome: filepath.Join(home, ".config"), Now: now, Runner: fakeRunner{paths: map[string]string{"bun": "/bin/bun"}, outputs: map[string]string{"bun --version": "1.3.0"}}}, 8, false)
		assertProtectedNoChange(t, status)
	})

	t.Run("uv rfc3339 cutoff", func(t *testing.T) {
		home := t.TempDir()
		copyManagerFixture(t, "uv-rfc3339.toml", filepath.Join(home, ".config", "uv", "uv.toml"))
		status := UV{}.Plan(context.Background(), Env{HomeDir: home, ConfigHome: filepath.Join(home, ".config"), Now: now, Runner: fakeRunner{paths: map[string]string{"uv": "/bin/uv"}, outputs: map[string]string{"uv --version": "uv 0.11.13"}}}, 8, false)
		assertProtectedNoChange(t, status)
	})
}

func TestFixtureMalformedConfigsDisableManager(t *testing.T) {
	cases := []struct {
		name    string
		fixture string
		path    func(home string) string
		status  func(env Env) Status
	}{
		{
			name:    "yarn malformed yaml",
			fixture: "yarn-malformed.yml",
			path:    func(home string) string { return filepath.Join(home, ".yarnrc.yml") },
			status: func(env Env) Status {
				env.Runner = fakeRunner{paths: map[string]string{"yarn": "/bin/yarn"}, outputs: map[string]string{"yarn --version": "4.10.0"}}
				return Yarn{}.Plan(context.Background(), env, 8, false)
			},
		},
		{
			name:    "bun malformed toml",
			fixture: "bun-malformed.toml",
			path:    func(home string) string { return filepath.Join(home, ".bunfig.toml") },
			status: func(env Env) Status {
				env.Runner = fakeRunner{paths: map[string]string{"bun": "/bin/bun"}, outputs: map[string]string{"bun --version": "1.3.0"}}
				return Bun{}.Plan(context.Background(), env, 8, false)
			},
		},
		{
			name:    "uv malformed toml",
			fixture: "uv-malformed.toml",
			path:    func(home string) string { return filepath.Join(home, ".config", "uv", "uv.toml") },
			status: func(env Env) Status {
				env.Runner = fakeRunner{paths: map[string]string{"uv": "/bin/uv"}, outputs: map[string]string{"uv --version": "uv 0.11.13"}}
				return UV{}.Plan(context.Background(), env, 8, false)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			copyManagerFixture(t, tc.fixture, tc.path(home))
			status := tc.status(Env{HomeDir: home, ConfigHome: filepath.Join(home, ".config"), Now: func() time.Time { return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC) }})
			if status.Error == "" || status.Configurable || status.NeedsChange || status.Selected {
				t.Fatalf("expected malformed config to disable manager: %#v", status)
			}
		})
	}
}

func TestEmptyGlobalConfigFilesPlanWrites(t *testing.T) {
	home := t.TempDir()
	configHome := filepath.Join(home, ".config")
	pnpmRC := filepath.Join(configHome, "pnpm", "rc")
	for _, path := range []string{filepath.Join(home, ".npmrc"), pnpmRC, filepath.Join(home, ".yarnrc.yml"), filepath.Join(home, ".bunfig.toml"), filepath.Join(configHome, "uv", "uv.toml")} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	env := Env{
		HomeDir:    home,
		ConfigHome: configHome,
		Runner:     fakeRunner{paths: map[string]string{"npm": "/bin/npm", "pnpm": "/bin/pnpm", "yarn": "/bin/yarn", "bun": "/bin/bun", "uv": "/bin/uv"}, outputs: map[string]string{"npm --version": "11.12.1", "npm config get userconfig": filepath.Join(home, ".npmrc"), "pnpm --version": "10.33.0", "pnpm config get globalconfig --location=global": pnpmRC, "yarn --version": "4.10.0", "bun --version": "1.3.0", "uv --version": "uv 0.11.13"}},
	}
	statuses := Scan(context.Background(), env, 8, false)
	changes := SelectChanges(statuses, map[string]bool{"npm": true, "pnpm": true, "yarn": true, "bun": true, "uv": true})
	if len(changes) != 5 {
		t.Fatalf("got %d changes, want 5: %#v", len(changes), changes)
	}
	joined := ""
	for _, change := range changes {
		joined += change.After
	}
	for _, want := range []string{"min-release-age=8", "minimum-release-age=11520", "npmMinimalAgeGate: 8d", "minimumReleaseAge = 691200", "exclude-newer = \"8 days\""} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q in empty-file planned writes:\n%s", want, joined)
		}
	}
}

func TestUnsupportedVersionsAndMissingBinaries(t *testing.T) {
	missing := NPM{}.Plan(context.Background(), Env{Runner: fakeRunner{}}, 8, false)
	if missing.Installed || missing.Reason != "package manager not installed" {
		t.Fatalf("unexpected missing npm status: %#v", missing)
	}

	cases := []struct {
		name   string
		status func() Status
	}{
		{"npm", func() Status { return NPM{}.Plan(context.Background(), unsupportedEnv("npm", "10.9.0"), 8, false) }},
		{"pnpm", func() Status { return PNPM{}.Plan(context.Background(), unsupportedEnv("pnpm", "10.15.0"), 8, false) }},
		{"yarn", func() Status { return Yarn{}.Plan(context.Background(), unsupportedEnv("yarn", "3.8.7"), 8, false) }},
		{"bun", func() Status { return Bun{}.Plan(context.Background(), unsupportedEnv("bun", "1.2.22"), 8, false) }},
		{"uv", func() Status { return UV{}.Plan(context.Background(), unsupportedEnv("uv", "uv 0.4.30"), 8, false) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := tc.status()
			if !status.Installed || status.Supported || status.Configurable || status.Reason == "" {
				t.Fatalf("expected unsupported status with reason: %#v", status)
			}
		})
	}
}

func TestDuplicateVPAndPNPMPathMergesChanges(t *testing.T) {
	home := t.TempDir()
	pnpmRC := filepath.Join(home, ".config", "pnpm", "rc")
	env := Env{
		HomeDir:    home,
		ConfigHome: filepath.Join(home, ".config"),
		Runner:     fakeRunner{paths: map[string]string{"pnpm": "/bin/pnpm", "vp": "/bin/vp"}, outputs: map[string]string{"pnpm --version": "10.33.0", "pnpm config get globalconfig --location=global": pnpmRC, "vp --version": "0.1.0"}},
	}
	statuses := Scan(context.Background(), env, 8, false)
	changes := config.MergeChanges(SelectChanges(statuses, map[string]bool{"pnpm": true, "vp": true}))
	if len(changes) != 1 {
		t.Fatalf("got %d merged changes, want 1: %#v", len(changes), changes)
	}
	if changes[0].Path != pnpmRC {
		t.Fatalf("expected merged change to target pnpm rc %q, got %q", pnpmRC, changes[0].Path)
	}
	if !strings.Contains(changes[0].Description, "pnpm") || !strings.Contains(changes[0].Description, "VP") {
		t.Fatalf("expected merged description to mention pnpm and VP: %q", changes[0].Description)
	}
}

func assertProtectedNoChange(t *testing.T, status Status) {
	t.Helper()
	if !status.Protected || status.NeedsChange || status.Selected || len(status.Changes) != 0 {
		t.Fatalf("expected protected no-change status: %#v", status)
	}
}

func unsupportedEnv(command, version string) Env {
	home := os.TempDir()
	outputs := map[string]string{command + " --version": version}
	if command == "npm" {
		outputs["npm config get userconfig"] = filepath.Join(home, ".npmrc")
	}
	if command == "pnpm" {
		outputs["pnpm config get globalconfig --location=global"] = filepath.Join(home, ".config", "pnpm", "rc")
	}
	return Env{HomeDir: home, ConfigHome: filepath.Join(home, ".config"), Runner: fakeRunner{paths: map[string]string{command: "/bin/" + command}, outputs: outputs}}
}

func copyManagerFixture(t *testing.T, name, dst string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dst
}
