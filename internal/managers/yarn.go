package managers

import (
	"context"
	"path/filepath"
	"strconv"

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
		if seconds, ok := config.ParseAgeDuration(raw); ok {
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
