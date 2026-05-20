package managers

import (
	"context"
	"path/filepath"
	"strconv"

	"github.com/xavier/smsc/internal/config"
)

type UV struct{}

func (UV) ID() string   { return "uv" }
func (UV) Name() string { return "uv" }

func (UV) Plan(ctx context.Context, env Env, days int, allowLower bool) Status {
	env = env.withDefaults()
	exe, err := env.Runner.LookPath("uv")
	if err != nil {
		return missingStatus("uv", "uv")
	}
	version := commandVersion(ctx, env.Runner, "uv", "--version")
	if !versionAtLeast(version, 0, 5, 0) {
		return unsupportedStatus("uv", "uv", exe, version, "exclude-newer duration config requires a recent uv release")
	}
	configPath := filepath.Join(env.ConfigHome, "uv", "uv.toml")
	before, _, readErr := readExisting(configPath)
	target := strconv.Itoa(days) + " days"
	status := Status{
		ID:           "uv",
		Name:         "uv",
		Executable:   exe,
		Version:      version,
		Installed:    true,
		Supported:    true,
		Configurable: true,
		ConfigPath:   configPath,
		TargetRaw:    `exclude-newer="` + target + `"`,
	}
	if readErr != nil {
		status.Error = readErr.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	}
	if raw, ok, err := config.ReadTOMLTopString(before, "exclude-newer"); err != nil {
		status.Error = err.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	} else if ok {
		status.CurrentRaw = "exclude-newer=" + raw
		if seconds, ok := config.ParseAgeDuration(raw); ok {
			status.currentAgeSeconds = &seconds
		} else if seconds, ok := config.ParseRFC3339Cutoff(raw, env.Now()); ok {
			status.currentAgeSeconds = &seconds
		}
	}
	after, err := config.SetTOMLTopString(before, "exclude-newer", target)
	if err != nil {
		status.Error = err.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	}
	status.Changes = []config.Change{{
		ManagerID:   status.ID,
		ManagerName: status.Name,
		Path:        configPath,
		Description: "set uv exclude-newer",
		Before:      before,
		After:       after,
	}}
	return finalizeStatus(status, days, allowLower)
}
