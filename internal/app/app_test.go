package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xavier-castro/smsc/internal/managers"
)

type appFakeRunner struct {
	paths   map[string]string
	outputs map[string]string
}

func (f appFakeRunner) LookPath(file string) (string, error) {
	if path, ok := f.paths[file]; ok {
		return path, nil
	}
	return "", os.ErrNotExist
}

func (f appFakeRunner) Output(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	if out, ok := f.outputs[key]; ok {
		return out, nil
	}
	return "", nil
}

func TestVersionFlag(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"--version"}, &out, &err)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if out.String() == "" {
		t.Fatal("expected version output")
	}
}

func TestJSONDryRun(t *testing.T) {
	var out, stderr bytes.Buffer
	code := Run([]string{"--json", "--dry-run"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	if decoded["days"].(float64) != 8 {
		t.Fatalf("unexpected days: %#v", decoded["days"])
	}
}

func TestHelpFlagReturnsZero(t *testing.T) {
	var out, stderr bytes.Buffer
	code := Run([]string{"--help"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(out.String(), "global package-manager config") || stderr.Len() != 0 {
		t.Fatalf("expected global-config scope in stdout help:\nstdout=%s\nstderr=%s", out.String(), stderr.String())
	}
}

func TestUnknownManagerRejected(t *testing.T) {
	env, _ := removeTestEnv(t)
	var out, stderr bytes.Buffer
	code := run(context.Background(), env, []string{"--managers", "npm,not-a-manager", "--dry-run"}, &out, &stderr)
	if code != 2 {
		t.Fatalf("code=%d out=%s err=%s", code, out.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown manager") || !strings.Contains(stderr.String(), "npm") {
		t.Fatalf("expected manager validation error, got: %s", stderr.String())
	}
}

func TestRemoveDryRunAndJSON(t *testing.T) {
	env, _ := removeTestEnv(t)

	var out, stderr bytes.Buffer
	code := run(context.Background(), env, []string{"--remove", "--managers", "npm", "--dry-run"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	if !strings.Contains(out.String(), "SMSC dry run: remove release-age configuration") {
		t.Fatalf("expected removal dry-run header:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "will remove") || !strings.Contains(out.String(), "Planned removals:") {
		t.Fatalf("expected removal planning output:\n%s", out.String())
	}
	if strings.Contains(out.String(), "target release age") {
		t.Fatalf("did not expect secure-mode wording:\n%s", out.String())
	}

	out.Reset()
	stderr.Reset()
	code = run(context.Background(), env, []string{"--remove", "--managers", "npm", "--json"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	var decoded planOutput
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	if !decoded.Remove {
		t.Fatalf("expected remove=true in json: %#v", decoded)
	}
	if len(decoded.Changes) != 1 {
		t.Fatalf("expected one removal change, got %#v", decoded.Changes)
	}
}

func TestRemoveYesAppliesThroughBackupPath(t *testing.T) {
	env, npmrc := removeTestEnv(t)

	var out, stderr bytes.Buffer
	code := run(context.Background(), env, []string{"--remove", "--managers", "npm", "--yes"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	if !strings.Contains(out.String(), "Removed release-age configuration:") {
		t.Fatalf("expected removal applied output:\n%s", out.String())
	}
	got, err := os.ReadFile(npmrc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "min-release-age") {
		t.Fatalf("expected min-release-age to be removed from npmrc:\n%s", string(got))
	}
	if !strings.Contains(string(got), "registry=https://example.test") {
		t.Fatalf("expected unrelated npmrc setting to remain:\n%s", string(got))
	}
	backup := filepath.Join(env.ConfigHome, "smsc", "backups", "20260520T120000Z")
	if _, err := os.Stat(filepath.Join(backup, "manifest.json")); err != nil {
		t.Fatalf("expected backup manifest: %v", err)
	}
}

func TestApplyBackupsListAndRestoreEndToEnd(t *testing.T) {
	home := t.TempDir()
	configHome := filepath.Join(home, ".config")
	npmrc := filepath.Join(home, ".npmrc")
	original := "registry=https://example.test\n"
	if err := os.WriteFile(npmrc, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	env := managers.Env{
		HomeDir:    home,
		ConfigHome: configHome,
		Now:        func() time.Time { return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC) },
		Runner:     appFakeRunner{paths: map[string]string{"npm": "/bin/npm"}, outputs: map[string]string{"npm --version": "11.12.1", "npm config get userconfig": npmrc}},
	}

	var out, stderr bytes.Buffer
	code := run(context.Background(), env, []string{"--managers", "npm", "--yes"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("apply code=%d out=%s err=%s", code, out.String(), stderr.String())
	}
	got, err := os.ReadFile(npmrc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "min-release-age=8") {
		t.Fatalf("expected npm policy after apply:\n%s", string(got))
	}

	out.Reset()
	stderr.Reset()
	code = run(context.Background(), env, []string{"backups"}, &out, &stderr)
	if code != 0 || !strings.Contains(out.String(), "20260520T120000Z") {
		t.Fatalf("backups code=%d out=%s err=%s", code, out.String(), stderr.String())
	}

	out.Reset()
	stderr.Reset()
	code = run(context.Background(), env, []string{"restore", "latest", "--dry-run"}, &out, &stderr)
	if code != 0 || !strings.Contains(out.String(), "Files that would be restored") {
		t.Fatalf("restore dry-run code=%d out=%s err=%s", code, out.String(), stderr.String())
	}

	out.Reset()
	stderr.Reset()
	code = run(context.Background(), env, []string{"restore", "latest", "--yes"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("restore code=%d out=%s err=%s", code, out.String(), stderr.String())
	}
	got, err = os.ReadFile(npmrc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("expected original npmrc after restore, got:\n%s", string(got))
	}
}

func TestRestoreRemovesFileCreatedByApply(t *testing.T) {
	home := t.TempDir()
	configHome := filepath.Join(home, ".config")
	npmrc := filepath.Join(home, ".npmrc")
	env := managers.Env{
		HomeDir:    home,
		ConfigHome: configHome,
		Now:        func() time.Time { return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC) },
		Runner:     appFakeRunner{paths: map[string]string{"npm": "/bin/npm"}, outputs: map[string]string{"npm --version": "11.12.1", "npm config get userconfig": npmrc}},
	}

	var out, stderr bytes.Buffer
	code := run(context.Background(), env, []string{"--managers", "npm", "--yes"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("apply code=%d out=%s err=%s", code, out.String(), stderr.String())
	}
	if _, err := os.Stat(npmrc); err != nil {
		t.Fatalf("expected npmrc to be created: %v", err)
	}

	out.Reset()
	stderr.Reset()
	code = run(context.Background(), env, []string{"restore", "latest", "--yes"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("restore code=%d out=%s err=%s", code, out.String(), stderr.String())
	}
	if _, err := os.Stat(npmrc); !os.IsNotExist(err) {
		t.Fatalf("expected restore to remove created npmrc, stat err=%v", err)
	}
}

func removeTestEnv(t *testing.T) (managers.Env, string) {
	t.Helper()
	home := t.TempDir()
	configHome := filepath.Join(home, ".config")
	npmrc := filepath.Join(home, ".npmrc")
	if err := os.WriteFile(npmrc, []byte("registry=https://example.test\nmin-release-age=8\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := managers.Env{
		HomeDir:    home,
		ConfigHome: configHome,
		Now:        func() time.Time { return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC) },
		Runner: appFakeRunner{
			paths: map[string]string{"npm": "/bin/npm"},
			outputs: map[string]string{
				"npm --version":             "11.12.1",
				"npm config get userconfig": npmrc,
			},
		},
	}
	return env, npmrc
}
