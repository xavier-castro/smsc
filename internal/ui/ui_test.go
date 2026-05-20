package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xavier/smsc/internal/managers"
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

func TestViewSelectShowsSupplyChainQuote(t *testing.T) {
	m := model{stage: stageSelect}

	view := m.viewSelect()

	if strings.Contains(view, "LOCK") || strings.Contains(view, "==[ ]==") {
		t.Fatalf("did not expect ASCII logo in select view:\n%s", view)
	}
	if !strings.Contains(view, "r to switch to remove secure configuration (revert what this package has done)") {
		t.Fatalf("expected explicit remove-mode switch hint:\n%s", view)
	}
	if !strings.Contains(view, "SMSC only adds release-age flags to your global package-manager config.") {
		t.Fatalf("expected philosophy text in select view:\n%s", view)
	}
	if !strings.Contains(view, "A 7-day package delay would have blocked installs") {
		t.Fatalf("expected supply chain quote in select view:\n%s", view)
	}
	if !strings.Contains(view, "publish attacks from the last 8 years") {
		t.Fatalf("expected full supply chain quote in select view:\n%s", view)
	}
}

func TestSpaceTogglesConfigurableManagerWithoutPendingChanges(t *testing.T) {
	m := model{
		stage: stageSelect,
		statuses: []managers.Status{{
			ID:           "npm",
			Name:         "npm",
			Installed:    true,
			Supported:    true,
			Configurable: true,
			TargetAge:    "8 days",
			CurrentAge:   "8 days",
		}},
		selected: map[string]bool{"npm": false},
	}

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	got := updated.(model)

	if !got.selected["npm"] {
		t.Fatal("expected space to toggle a configurable manager with no pending changes")
	}
}

func TestRemoveModeToggleSelectsCleanupRowsAndSkipsAge(t *testing.T) {
	home := t.TempDir()
	npmrc := filepath.Join(home, ".npmrc")
	if err := os.WriteFile(npmrc, []byte("registry=https://example.test\nmin-release-age=8\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := model{
		ctx:      context.Background(),
		stage:    stageSelect,
		selected: map[string]bool{},
		days:     8,
		env: managers.Env{
			HomeDir:    home,
			ConfigHome: filepath.Join(home, ".config"),
			Runner: fakeRunner{
				paths: map[string]string{"npm": "/bin/npm"},
				outputs: map[string]string{
					"npm --version":             "11.12.1",
					"npm config get userconfig": npmrc,
				},
			},
		},
	}

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	got := updated.(model)
	if !got.removeMode {
		t.Fatal("expected r to enable remove mode")
	}
	if !got.selected["npm"] {
		t.Fatalf("expected npm cleanup row to be selected: %#v", got.selected)
	}
	if view := got.viewSelect(); !strings.Contains(view, "will remove") {
		t.Fatalf("expected remove mode view to label planned cleanup:\n%s", view)
	}
	if view := got.viewSelect(); !strings.Contains(view, "r to switch to secure configuration") {
		t.Fatalf("expected secure-mode switch hint in remove mode:\n%s", view)
	}

	updated, _ = got.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	got = updated.(model)
	if got.stage != stageReview {
		t.Fatalf("expected remove mode enter to skip age and go to review, got stage %d", got.stage)
	}
}

func TestSelectViewLabelsMissingAndConfiguredManagers(t *testing.T) {
	m := model{
		stage: stageSelect,
		statuses: []managers.Status{
			{
				ID:        "yarn",
				Name:      "Yarn",
				TargetAge: "8 days",
			},
			{
				ID:           "npm",
				Name:         "npm",
				Installed:    true,
				Supported:    true,
				Configurable: true,
				TargetAge:    "8 days",
				CurrentAge:   "8 days",
			},
		},
		selected: map[string]bool{},
	}

	view := m.viewSelect()

	if !strings.Contains(view, missingManagerLabel) {
		t.Fatalf("expected missing manager label in select view:\n%s", view)
	}
	if strings.Contains(view, " missing") {
		t.Fatalf("did not expect old missing label in select view:\n%s", view)
	}
	if !strings.Contains(view, secureConfigAddedLabel) {
		t.Fatalf("expected configured manager label in select view:\n%s", view)
	}
}
