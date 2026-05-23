package managers

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xavier-castro/smsc/internal/config"
)

type PNPM struct{}

func (PNPM) ID() string   { return "pnpm" }
func (PNPM) Name() string { return "pnpm" }

func (PNPM) Plan(ctx context.Context, env Env, days int, allowLower bool) Status {
	env = env.withDefaults()
	exe, err := env.Runner.LookPath("pnpm")
	if err != nil {
		return missingStatus("pnpm", "pnpm")
	}
	version := commandVersion(ctx, env.Runner, "pnpm", "--version")
	if version != "" && !versionAtLeast(version, 10, 16, 0) {
		return unsupportedStatus("pnpm", "pnpm", exe, version, "pnpm minimumReleaseAge requires pnpm 10.16.0 or newer")
	}
	if version == "" {
		version = "unknown"
	}
	return pnpmStatus(ctx, env, days, allowLower, "pnpm", "pnpm", exe, version)
}

func (PNPM) Remove(ctx context.Context, env Env) Status {
	env = env.withDefaults()
	exe, err := env.Runner.LookPath("pnpm")
	if err != nil {
		return missingRemoveStatus("pnpm", "pnpm")
	}
	version := commandVersion(ctx, env.Runner, "pnpm", "--version")
	if version == "" {
		version = "detected"
	}
	return pnpmRemoveStatus(ctx, env, "pnpm", "pnpm", exe, version)
}

func pnpmStatus(ctx context.Context, env Env, days int, allowLower bool, id, name, exe, version string) Status {
	configPath := pnpmGlobalConfigPath(ctx, env)
	before, _, readErr := readExisting(configPath)
	targetMinutes := days * 24 * 60
	status := Status{
		ID:           id,
		Name:         name,
		Executable:   exe,
		Version:      version,
		Installed:    true,
		Supported:    true,
		Configurable: true,
		ConfigPath:   configPath,
		TargetRaw:    "minimumReleaseAge=" + strconv.Itoa(targetMinutes),
	}
	if readErr != nil {
		status.Error = readErr.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	}
	if current, raw, ok := config.ReadKeyValueInt(before, []string{"minimum-release-age", "minimumReleaseAge"}); ok {
		seconds := int64(current) * 60
		status.currentAgeSeconds = &seconds
		status.CurrentRaw = "minimumReleaseAge=" + raw
	}
	after := config.UpsertKeyValue(before, "minimum-release-age", strconv.Itoa(targetMinutes), []string{"minimumReleaseAge"}, nil)
	if env.SavePrefixTilde {
		currentPrefix, ok := config.ReadKeyValue(after, []string{"save-prefix"})
		if !ok || strings.Trim(currentPrefix, `"' `) != "~" {
			after = config.UpsertKeyValue(after, "save-prefix", "~", nil, nil)
			status.savePrefixChange = true
		}
	}
	description := "set pnpm minimumReleaseAge"
	if env.SavePrefixTilde {
		description = "set pnpm minimumReleaseAge and save-prefix"
	}
	status.Changes = []config.Change{{
		ManagerID:   id,
		ManagerName: name,
		Path:        configPath,
		Description: description,
		Before:      before,
		After:       after,
	}}
	return finalizeStatus(status, days, allowLower)
}

func pnpmRemoveStatus(ctx context.Context, env Env, id, name, exe, version string) Status {
	configPath := pnpmGlobalConfigPath(ctx, env)
	before, _, readErr := readExisting(configPath)
	status := Status{
		ID:           id,
		Name:         name,
		Executable:   exe,
		Version:      version,
		Installed:    true,
		Supported:    true,
		Configurable: true,
		ConfigPath:   configPath,
		TargetAge:    removeTargetAge,
		TargetRaw:    "remove minimumReleaseAge",
	}
	if readErr != nil {
		status.Error = readErr.Error()
		status.Configurable = false
		return finalizeRemoveStatus(status)
	}
	aliases := []string{"minimum-release-age", "minimumReleaseAge"}
	targetRaw := "remove minimumReleaseAge"
	description := "remove pnpm minimumReleaseAge"
	if env.SavePrefixTilde {
		aliases = append(aliases, "save-prefix")
		targetRaw = "remove minimumReleaseAge and save-prefix"
		description = "remove pnpm minimumReleaseAge and save-prefix"
	}
	status.TargetRaw = targetRaw
	if raw, ok := config.ReadKeyValue(before, aliases); ok {
		status.CurrentRaw = "minimumReleaseAge=" + raw
		if current, _, ok := config.ReadKeyValueInt(before, aliases); ok {
			seconds := int64(current) * 60
			status.currentAgeSeconds = &seconds
		}
	}
	after := config.RemoveKeyValue(before, aliases)
	if before != after {
		status.Changes = []config.Change{{
			ManagerID:   id,
			ManagerName: name,
			Path:        configPath,
			Description: description,
			Before:      before,
			After:       after,
		}}
	}
	return finalizeRemoveStatus(status)
}

func pnpmGlobalConfigPath(ctx context.Context, env Env) string {
	out, err := env.Runner.Output(ctx, "pnpm", "config", "get", "globalconfig", "--location=global")
	if err == nil && out != "" && out != "undefined" && out != "null" {
		return out
	}
	if env.GOOS == "darwin" {
		return filepath.Join(env.HomeDir, "Library", "Preferences", "pnpm", "rc")
	}
	return filepath.Join(env.ConfigHome, "pnpm", "rc")
}
