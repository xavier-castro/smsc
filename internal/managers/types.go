package managers

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xavier-castro/smsc/internal/config"
)

type Runner interface {
	LookPath(file string) (string, error)
	Output(ctx context.Context, name string, args ...string) (string, error)
}

type OSRunner struct{}

func (OSRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (OSRunner) Output(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

type Env struct {
	HomeDir         string
	ConfigHome      string
	Runner          Runner
	Now             func() time.Time
	GOOS            string
	Cwd             string
	SavePrefixTilde bool
}

func DefaultEnv() Env {
	home, _ := os.UserHomeDir()
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	cwd, _ := os.Getwd()
	return Env{
		HomeDir:    home,
		ConfigHome: configHome,
		Runner:     OSRunner{},
		Now:        time.Now,
		GOOS:       runtime.GOOS,
		Cwd:        cwd,
	}
}

func (e Env) withDefaults() Env {
	if e.HomeDir == "" {
		home, _ := os.UserHomeDir()
		e.HomeDir = home
	}
	if e.ConfigHome == "" {
		e.ConfigHome = filepath.Join(e.HomeDir, ".config")
	}
	if e.Runner == nil {
		e.Runner = OSRunner{}
	}
	if e.Now == nil {
		e.Now = time.Now
	}
	if e.GOOS == "" {
		e.GOOS = runtime.GOOS
	}
	return e
}

type Manager interface {
	ID() string
	Name() string
	Plan(ctx context.Context, env Env, days int, allowLower bool) Status
	Remove(ctx context.Context, env Env) Status
}

type Status struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Executable        string          `json:"executable,omitempty"`
	Version           string          `json:"version,omitempty"`
	Installed         bool            `json:"installed"`
	Supported         bool            `json:"supported"`
	Configurable      bool            `json:"configurable"`
	Selected          bool            `json:"selected"`
	NeedsChange       bool            `json:"needsChange"`
	AlreadyStricter   bool            `json:"alreadyStricter"`
	Protected         bool            `json:"protected"`
	ConfigPath        string          `json:"configPath,omitempty"`
	CurrentRaw        string          `json:"currentRaw,omitempty"`
	CurrentAge        string          `json:"currentAge,omitempty"`
	TargetRaw         string          `json:"targetRaw,omitempty"`
	TargetAge         string          `json:"targetAge"`
	Reason            string          `json:"reason,omitempty"`
	Error             string          `json:"error,omitempty"`
	Changes           []config.Change `json:"changes,omitempty"`
	currentAgeSeconds *int64
	targetAgeSeconds  int64
	savePrefixChange  bool
}

func missingStatus(id, name string) Status {
	return Status{
		ID:        id,
		Name:      name,
		TargetAge: config.FormatDays(DefaultDays),
		Reason:    "package manager not installed",
	}
}

func missingRemoveStatus(id, name string) Status {
	status := missingStatus(id, name)
	status.TargetAge = removeTargetAge
	return status
}

func unsupportedStatus(id, name, exe, version, reason string) Status {
	return Status{
		ID:           id,
		Name:         name,
		Executable:   exe,
		Version:      version,
		Installed:    true,
		Supported:    false,
		Configurable: false,
		TargetAge:    config.FormatDays(DefaultDays),
		Reason:       reason,
	}
}

func finalizeStatus(status Status, days int, allowLower bool) Status {
	status.TargetAge = config.FormatDays(days)
	status.targetAgeSeconds = config.DaysToSeconds(days)
	if status.currentAgeSeconds != nil {
		status.CurrentAge = config.SecondsLabel(*status.currentAgeSeconds)
		status.Protected = *status.currentAgeSeconds >= status.targetAgeSeconds
		if *status.currentAgeSeconds > status.targetAgeSeconds && !allowLower {
			if !status.savePrefixChange {
				status.AlreadyStricter = true
				status.NeedsChange = false
				status.Selected = false
				status.Changes = nil
				status.Reason = "already stricter than requested; preserving existing policy"
				return status
			}
		}
		if *status.currentAgeSeconds == status.targetAgeSeconds {
			if status.savePrefixChange {
				status.NeedsChange = status.Configurable && len(status.Changes) > 0
				status.Selected = status.NeedsChange
				return status
			}
			status.NeedsChange = false
			status.Selected = false
			if status.Reason == "" {
				status.Reason = "already configured"
			}
			status.Changes = nil
			return status
		}
	}
	status.NeedsChange = status.Configurable && len(status.Changes) > 0
	status.Selected = status.NeedsChange
	return status
}

func finalizeRemoveStatus(status Status) Status {
	if status.TargetAge == "" {
		status.TargetAge = removeTargetAge
	}
	if !status.Installed {
		return status
	}
	status.NeedsChange = status.Configurable && len(status.Changes) > 0
	status.Selected = status.NeedsChange
	if status.Configurable && !status.NeedsChange && status.Reason == "" {
		status.Reason = "release-age configuration not found"
	}
	return status
}

func readExisting(path string) (string, bool, error) {
	return config.ReadFile(path)
}

func commandVersion(ctx context.Context, runner Runner, command string, args ...string) string {
	out, err := runner.Output(ctx, command, args...)
	if err != nil {
		return ""
	}
	return firstLine(out)
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	return value
}

const (
	DefaultDays     = 8
	removeTargetAge = "remove"
)
