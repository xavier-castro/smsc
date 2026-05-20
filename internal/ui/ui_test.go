package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xavier/smsc/internal/managers"
)

func TestViewSelectShowsSupplyChainQuote(t *testing.T) {
	m := model{stage: stageSelect}

	view := m.viewSelect()

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
