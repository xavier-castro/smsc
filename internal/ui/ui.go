package ui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xavier/smsc/internal/config"
	"github.com/xavier/smsc/internal/managers"
)

type stage int

const (
	stageScanning stage = iota
	stageSelect
	stageAge
	stageReview
	stageApply
	stageDone
)

type scanDoneMsg []managers.Status

type applyDoneMsg struct {
	applied []config.AppliedChange
	err     error
}

type model struct {
	ctx        context.Context
	env        managers.Env
	stage      stage
	spinner    spinner.Model
	statuses   []managers.Status
	selected   map[string]bool
	cursor     int
	days       int
	allowLower bool
	removeMode bool
	applied    []config.AppliedChange
	err        error
}

var (
	titleStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	philosophyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mutedStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	missingManagerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Strikethrough(true).StrikethroughSpaces(true)
)

const (
	philosophyText         = "SMSC only adds release-age flags to your global package-manager config.\nIt is not a fix-all; it is one more prevention measure against current supply chain attacks."
	supplyChainQuote       = "A 7-day package delay would have blocked installs in most short-lived malicious\npublish attacks from the last 8 years"
	missingManagerLabel    = "package manager not installed"
	secureConfigAddedLabel = "secure configuration added"
	secureModeLabel        = "Secure configuration"
	removeModeLabel        = "Remove release-age configuration"
)

func Run(ctx context.Context, env managers.Env, days int, output io.Writer) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	m := model{
		ctx:      ctx,
		env:      env,
		stage:    stageScanning,
		spinner:  s,
		selected: map[string]bool{},
		days:     days,
	}
	_, err := tea.NewProgram(m, tea.WithOutput(output), tea.WithAltScreen()).Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.scanCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case scanDoneMsg:
		m.statuses = []managers.Status(msg)
		m.selected = map[string]bool{}
		for _, status := range m.statuses {
			m.selected[status.ID] = status.Selected
		}
		m.stage = stageSelect
		return m, nil
	case applyDoneMsg:
		m.applied = msg.applied
		m.err = msg.err
		m.stage = stageDone
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	switch m.stage {
	case stageScanning:
		return titleStyle.Render("SMSC") + "\n\n" + m.spinner.View() + " scanning package managers..."
	case stageSelect:
		return m.viewSelect()
	case stageAge:
		return m.viewAge()
	case stageReview:
		return m.viewReview()
	case stageApply:
		if m.removeMode {
			return titleStyle.Render("Removing policy") + "\n\n" + m.spinner.View() + " removing selected global config entries..."
		}
		return titleStyle.Render("Applying policy") + "\n\n" + m.spinner.View() + " writing selected global config files..."
	case stageDone:
		return m.viewDone()
	default:
		return ""
	}
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.stage != stageApply {
			return m, tea.Quit
		}
	}

	switch m.stage {
	case stageSelect:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.statuses)-1 {
				m.cursor++
			}
		case "r":
			m.removeMode = !m.removeMode
			m.refreshPlan(false)
		case " ", "space":
			if len(m.statuses) == 0 {
				break
			}
			status := m.statuses[m.cursor]
			if canToggleStatus(status, m.removeMode) {
				m.selected[status.ID] = !m.selected[status.ID]
			}
		case "enter":
			if m.removeMode {
				m.refreshPlan(true)
				m.stage = stageReview
			} else {
				m.stage = stageAge
			}
		}
	case stageAge:
		switch msg.String() {
		case "left", "-", "h":
			if m.days > 1 {
				m.days--
			}
		case "right", "+", "l":
			m.days++
		case "enter":
			m.refreshPlan(true)
			m.stage = stageReview
		case "b":
			m.stage = stageSelect
		}
	case stageReview:
		switch msg.String() {
		case "enter":
			m.stage = stageApply
			return m, tea.Batch(m.spinner.Tick, m.applyCmd())
		case "b":
			if m.removeMode {
				m.stage = stageSelect
			} else {
				m.stage = stageAge
			}
		}
	case stageDone:
		switch msg.String() {
		case "enter", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) scanCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(250 * time.Millisecond)
		return scanDoneMsg(m.scanStatuses())
	}
}

func (m model) applyCmd() tea.Cmd {
	return func() tea.Msg {
		selected := managers.SelectChanges(m.statuses, m.selected)
		applied, err := config.ApplyChanges(selected, m.env.ConfigHome, m.env.Now().UTC())
		return applyDoneMsg{applied: applied, err: err}
	}
}

func (m model) scanStatuses() []managers.Status {
	if m.removeMode {
		return managers.ScanRemove(m.ctx, m.env)
	}
	return managers.Scan(m.ctx, m.env, m.days, m.allowLower)
}

func (m *model) refreshPlan(preserveSelection bool) {
	old := m.selected
	if old == nil {
		old = map[string]bool{}
	}
	m.statuses = m.scanStatuses()
	if !preserveSelection {
		m.resetSelection()
		return
	}
	for _, status := range m.statuses {
		if _, ok := old[status.ID]; !ok {
			old[status.ID] = status.Selected
		}
	}
	m.selected = old
}

func (m *model) resetSelection() {
	m.selected = map[string]bool{}
	for _, status := range m.statuses {
		m.selected[status.ID] = status.Selected
	}
}

func (m model) viewSelect() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Secure My Supply Chain"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Mode: %s\n", m.renderMode()))
	b.WriteString(mutedStyle.Render(m.modeSwitchHint()))
	b.WriteString("\n\n")
	b.WriteString(philosophyStyle.Render(philosophyText))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render(supplyChainQuote))
	b.WriteString("\n\n")
	if m.removeMode {
		b.WriteString("Select package managers to clean up.\n\n")
	} else {
		b.WriteString("Select package managers to secure.\n\n")
	}
	for i, status := range m.statuses {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		check := " "
		if m.selected[status.ID] {
			check = "x"
		}
		b.WriteString(statusLine(cursor, check, status, m.removeMode))
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("r mode  space toggle  enter continue  q quit"))
	return b.String()
}

func (m model) renderMode() string {
	if m.removeMode {
		return warnStyle.Render(removeModeLabel)
	}
	return okStyle.Render(secureModeLabel)
}

func (m model) modeSwitchHint() string {
	if m.removeMode {
		return "r to switch to secure configuration"
	}
	return "r to switch to remove secure configuration (revert what this package has done)"
}

func statusLine(cursor, check string, status managers.Status, removeMode bool) string {
	current := status.CurrentAge
	if current == "" {
		current = "not configured"
	}
	if !status.Installed {
		line := fmt.Sprintf("%s [%s] %-13s current: %-16s target: %-8s %s", cursor, check, status.Name, current, status.TargetAge, missingManagerLabel)
		return missingManagerStyle.Render(line) + "\n"
	}
	return fmt.Sprintf("%s [%s] %-13s current: %-16s target: %-8s %s\n", cursor, check, status.Name, current, status.TargetAge, statusLineState(status, removeMode))
}

func canToggleStatus(status managers.Status, removeMode bool) bool {
	if removeMode {
		return status.Installed && status.Configurable
	}
	return status.Installed && status.Supported && status.Configurable
}

func (m model) viewAge() string {
	return titleStyle.Render("Release age") +
		"\n\n" +
		fmt.Sprintf("Minimum release age: %s\n\n", okStyle.Render(config.FormatDays(m.days))) +
		mutedStyle.Render("- / left decrease  + / right increase  enter review  b back")
}

func (m model) viewReview() string {
	changes := config.MergeChanges(managers.SelectChanges(m.statuses, m.selected))
	var b strings.Builder
	if m.removeMode {
		b.WriteString(titleStyle.Render("Review removals"))
	} else {
		b.WriteString(titleStyle.Render("Review changes"))
	}
	b.WriteString("\n\n")
	if len(changes) == 0 {
		if m.removeMode {
			b.WriteString("No release-age configuration was found.\n")
		} else {
			b.WriteString("No file changes selected.\n")
		}
	} else {
		for _, change := range changes {
			b.WriteString(fmt.Sprintf("- %s\n  %s\n", change.Path, mutedStyle.Render(change.Description)))
		}
	}
	b.WriteString("\n")
	if m.removeMode {
		b.WriteString(mutedStyle.Render("enter remove  b back  q quit"))
	} else {
		b.WriteString(mutedStyle.Render("enter apply  b back  q quit"))
	}
	return b.String()
}

func (m model) viewDone() string {
	var b strings.Builder
	if m.err != nil {
		b.WriteString(errStyle.Render("Apply failed"))
		b.WriteString("\n\n")
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
		b.WriteString(mutedStyle.Render("enter close"))
		return b.String()
	}
	if m.removeMode {
		b.WriteString(okStyle.Render("Release-age configuration removed"))
	} else {
		b.WriteString(okStyle.Render("Policy applied"))
	}
	b.WriteString("\n\n")
	if len(m.applied) == 0 {
		if m.removeMode {
			b.WriteString("No release-age configuration was found.\n")
		} else {
			b.WriteString("No changes were needed.\n")
		}
	} else {
		for _, item := range m.applied {
			if item.Changed {
				b.WriteString(fmt.Sprintf("- %s\n", item.Path))
				if item.BackupPath != "" {
					b.WriteString(fmt.Sprintf("  backup: %s\n", mutedStyle.Render(item.BackupPath)))
				}
			}
		}
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("enter close"))
	return b.String()
}

func statusLineState(status managers.Status, removeMode bool) string {
	if !status.Installed {
		return mutedStyle.Render(missingManagerLabel)
	}
	if !removeMode && !status.Supported {
		return warnStyle.Render("unsupported")
	}
	if status.Error != "" {
		return errStyle.Render(status.Error)
	}
	if removeMode {
		if status.NeedsChange {
			return warnStyle.Render("will remove")
		}
		return mutedStyle.Render("not configured")
	}
	if status.AlreadyStricter {
		return okStyle.Render("stricter")
	}
	if status.NeedsChange {
		return warnStyle.Render("will update")
	}
	return okStyle.Render(secureConfigAddedLabel)
}
