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
	applied    []config.AppliedChange
	err        error
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
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
		case " ":
			if len(m.statuses) == 0 {
				break
			}
			status := m.statuses[m.cursor]
			if status.Configurable && status.Supported && len(status.Changes) > 0 {
				m.selected[status.ID] = !m.selected[status.ID]
			}
		case "enter":
			m.stage = stageAge
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
			m.refreshPlan()
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
			m.stage = stageAge
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
		return scanDoneMsg(managers.Scan(m.ctx, m.env, m.days, m.allowLower))
	}
}

func (m model) applyCmd() tea.Cmd {
	return func() tea.Msg {
		selected := managers.SelectChanges(m.statuses, m.selected)
		applied, err := config.ApplyChanges(selected, m.env.ConfigHome, m.env.Now().UTC())
		return applyDoneMsg{applied: applied, err: err}
	}
}

func (m *model) refreshPlan() {
	old := m.selected
	m.statuses = managers.Scan(m.ctx, m.env, m.days, m.allowLower)
	for _, status := range m.statuses {
		if _, ok := old[status.ID]; !ok {
			old[status.ID] = status.Selected
		}
	}
	m.selected = old
}

func (m model) viewSelect() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Secure My Supply Chain"))
	b.WriteString("\n\nSelect package managers to secure.\n\n")
	for i, status := range m.statuses {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		check := " "
		if m.selected[status.ID] {
			check = "x"
		}
		current := status.CurrentAge
		if current == "" {
			current = "not configured"
		}
		state := statusLineState(status)
		b.WriteString(fmt.Sprintf("%s [%s] %-13s current: %-16s target: %-8s %s\n", cursor, check, status.Name, current, status.TargetAge, state))
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("space toggle  enter continue  q quit"))
	return b.String()
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
	b.WriteString(titleStyle.Render("Review changes"))
	b.WriteString("\n\n")
	if len(changes) == 0 {
		b.WriteString("No file changes selected.\n")
	} else {
		for _, change := range changes {
			b.WriteString(fmt.Sprintf("- %s\n  %s\n", change.Path, mutedStyle.Render(change.Description)))
		}
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("enter apply  b back  q quit"))
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
	b.WriteString(okStyle.Render("Policy applied"))
	b.WriteString("\n\n")
	if len(m.applied) == 0 {
		b.WriteString("No changes were needed.\n")
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

func statusLineState(status managers.Status) string {
	if !status.Installed {
		return mutedStyle.Render("missing")
	}
	if !status.Supported {
		return warnStyle.Render("unsupported")
	}
	if status.Error != "" {
		return errStyle.Render(status.Error)
	}
	if status.AlreadyStricter {
		return okStyle.Render("stricter")
	}
	if status.NeedsChange {
		return warnStyle.Render("will update")
	}
	return okStyle.Render("ok")
}
