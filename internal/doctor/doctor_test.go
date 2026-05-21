package doctor

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

func TestDoctorJSONSummarizesGlobalProtectionAndLocalOverrides(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(home, "src", "example")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	globalNPMRC := filepath.Join(home, ".npmrc")
	if err := os.WriteFile(globalNPMRC, []byte("min-release-age=8\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	localNPMRC := filepath.Join(project, ".npmrc")
	if err := os.WriteFile(localNPMRC, []byte("min-release-age=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	env := managers.Env{
		HomeDir:    home,
		ConfigHome: filepath.Join(home, ".config"),
		Cwd:        project,
		Now:        func() time.Time { return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC) },
		Runner:     fakeRunner{paths: map[string]string{"npm": "/bin/npm"}, outputs: map[string]string{"npm --version": "11.12.1", "npm config get userconfig": globalNPMRC}},
	}

	var out bytes.Buffer
	if err := Run(context.Background(), env, &out, true); err != nil {
		t.Fatal(err)
	}
	var report Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	if !report.Protected || report.Summary.Installed != 1 || report.Summary.Protected != 1 {
		t.Fatalf("unexpected protection summary: %#v", report)
	}
	if len(report.LocalOverrides) != 1 || report.LocalOverrides[0].Path != localNPMRC {
		t.Fatalf("expected only project-local override, got %#v", report.LocalOverrides)
	}
	if len(report.LocalOverrides[0].Values) != 1 || report.LocalOverrides[0].Values[0].Age != "1 day" {
		t.Fatalf("expected parsed local npm age value: %#v", report.LocalOverrides[0].Values)
	}
}

func TestDoctorTextExplainsUnsupportedVersions(t *testing.T) {
	home := t.TempDir()
	env := managers.Env{
		HomeDir:    home,
		ConfigHome: filepath.Join(home, ".config"),
		Cwd:        filepath.Join(home, "project"),
		Runner:     fakeRunner{paths: map[string]string{"npm": "/bin/npm"}, outputs: map[string]string{"npm --version": "10.9.0", "npm config get userconfig": filepath.Join(home, ".npmrc")}},
	}
	var out bytes.Buffer
	if err := Run(context.Background(), env, &out, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "unsupported version") || !strings.Contains(out.String(), "npm min-release-age requires npm 11 or newer") {
		t.Fatalf("expected unsupported version explanation:\n%s", out.String())
	}
}
