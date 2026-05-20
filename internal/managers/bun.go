package managers

import (
	"context"
	"path/filepath"
	"strconv"

	"github.com/xavier/smsc/internal/config"
)

type Bun struct{}

func (Bun) ID() string   { return "bun" }
func (Bun) Name() string { return "Bun" }

func (Bun) Plan(ctx context.Context, env Env, days int, allowLower bool) Status {
	env = env.withDefaults()
	exe, err := env.Runner.LookPath("bun")
	if err != nil {
		return missingStatus("bun", "Bun")
	}
	version := commandVersion(ctx, env.Runner, "bun", "--version")
	if !versionAtLeast(version, 1, 3, 0) {
		return unsupportedStatus("bun", "Bun", exe, version, "install.minimumReleaseAge requires Bun 1.3.0 or newer")
	}
	configPath := filepath.Join(env.HomeDir, ".bunfig.toml")
	before, _, readErr := readExisting(configPath)
	secondsTarget := config.DaysToSeconds(days)
	status := Status{
		ID:           "bun",
		Name:         "Bun",
		Executable:   exe,
		Version:      version,
		Installed:    true,
		Supported:    true,
		Configurable: true,
		ConfigPath:   configPath,
		TargetRaw:    "install.minimumReleaseAge=" + strconv.FormatInt(secondsTarget, 10),
	}
	if readErr != nil {
		status.Error = readErr.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	}
	if seconds, raw, ok, err := config.ReadBunMinimumReleaseAge(before); err != nil {
		status.Error = err.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	} else if ok {
		status.currentAgeSeconds = &seconds
		status.CurrentRaw = "install.minimumReleaseAge=" + raw
	}
	after, err := config.SetBunMinimumReleaseAge(before, secondsTarget)
	if err != nil {
		status.Error = err.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	}
	status.Changes = []config.Change{{
		ManagerID:   status.ID,
		ManagerName: status.Name,
		Path:        configPath,
		Description: "set Bun install.minimumReleaseAge",
		Before:      before,
		After:       after,
	}}
	return finalizeStatus(status, days, allowLower)
}
