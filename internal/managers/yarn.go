package managers

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xavier/smsc/internal/config"
)

type Yarn struct{}

func (Yarn) ID() string   { return "yarn" }
func (Yarn) Name() string { return "Yarn" }

func (Yarn) Plan(ctx context.Context, env Env, days int, allowLower bool) Status {
	env = env.withDefaults()
	exe, err := env.Runner.LookPath("yarn")
	if err != nil {
		return missingStatus("yarn", "Yarn")
	}
	version := commandVersion(ctx, env.Runner, "yarn", "--version")
	if !versionAtLeast(version, 4, 0, 0) {
		return unsupportedStatus("yarn", "Yarn", exe, version, "npmMinimalAgeGate requires Yarn Berry 4 or newer")
	}
	configPath := filepath.Join(env.HomeDir, ".yarnrc.yml")
	before, _, readErr := readExisting(configPath)
	status := Status{
		ID:           "yarn",
		Name:         "Yarn",
		Executable:   exe,
		Version:      version,
		Installed:    true,
		Supported:    true,
		Configurable: true,
		ConfigPath:   configPath,
		TargetRaw:    `npmMinimalAgeGate: "` + strconv.Itoa(days) + `d"`,
	}
	if readErr != nil {
		status.Error = readErr.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	}
	if raw, ok, err := config.ReadYAMLString(before, "npmMinimalAgeGate"); err == nil && ok {
		status.CurrentRaw = "npmMinimalAgeGate=" + raw
		if seconds, ok := parseYarnMinimalAge(raw); ok {
			status.currentAgeSeconds = &seconds
		}
	} else if err != nil {
		status.Error = err.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	}
	after, err := config.SetYAMLString(before, "npmMinimalAgeGate", strconv.Itoa(days)+"d")
	if err != nil {
		status.Error = err.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	}
	status.Changes = []config.Change{{
		ManagerID:   status.ID,
		ManagerName: status.Name,
		Path:        configPath,
		Description: "set Yarn npmMinimalAgeGate",
		Before:      before,
		After:       after,
	}}
	return finalizeStatus(status, days, allowLower)
}

func (Yarn) Remove(ctx context.Context, env Env) Status {
	env = env.withDefaults()
	exe, err := env.Runner.LookPath("yarn")
	if err != nil {
		return missingRemoveStatus("yarn", "Yarn")
	}
	version := commandVersion(ctx, env.Runner, "yarn", "--version")
	if version == "" {
		version = "detected"
	}
	configPath := filepath.Join(env.HomeDir, ".yarnrc.yml")
	before, _, readErr := readExisting(configPath)
	status := Status{
		ID:           "yarn",
		Name:         "Yarn",
		Executable:   exe,
		Version:      version,
		Installed:    true,
		Supported:    true,
		Configurable: true,
		ConfigPath:   configPath,
		TargetAge:    removeTargetAge,
		TargetRaw:    "remove npmMinimalAgeGate",
	}
	if readErr != nil {
		status.Error = readErr.Error()
		status.Configurable = false
		return finalizeRemoveStatus(status)
	}
	if raw, ok, err := config.ReadYAMLString(before, "npmMinimalAgeGate"); err != nil {
		status.Error = err.Error()
		status.Configurable = false
		return finalizeRemoveStatus(status)
	} else if ok {
		status.CurrentRaw = "npmMinimalAgeGate=" + raw
		if seconds, ok := parseYarnMinimalAge(raw); ok {
			status.currentAgeSeconds = &seconds
		}
	}
	after, err := config.RemoveYAMLTopKey(before, "npmMinimalAgeGate")
	if err != nil {
		status.Error = err.Error()
		status.Configurable = false
		return finalizeRemoveStatus(status)
	}
	if before != after {
		status.Changes = []config.Change{{
			ManagerID:   status.ID,
			ManagerName: status.Name,
			Path:        configPath,
			Description: "remove Yarn npmMinimalAgeGate",
			Before:      before,
			After:       after,
		}}
	}
	return finalizeRemoveStatus(status)
}

func parseYarnMinimalAge(raw string) (int64, bool) {
	if seconds, ok := config.ParseAgeDuration(raw); ok {
		return seconds, true
	}
	minutes, err := strconv.ParseInt(strings.Trim(raw, `"' `), 10, 64)
	if err != nil {
		return 0, false
	}
	return minutes * 60, true
}
