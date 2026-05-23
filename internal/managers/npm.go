package managers

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xavier-castro/smsc/internal/config"
)

type NPM struct{}

func (NPM) ID() string   { return "npm" }
func (NPM) Name() string { return "npm" }

func (NPM) Plan(ctx context.Context, env Env, days int, allowLower bool) Status {
	env = env.withDefaults()
	exe, err := env.Runner.LookPath("npm")
	if err != nil {
		return missingStatus("npm", "npm")
	}

	version := commandVersion(ctx, env.Runner, "npm", "--version")
	if !versionAtLeast(version, 11, 0, 0) {
		return unsupportedStatus("npm", "npm", exe, version, "npm min-release-age requires npm 11 or newer")
	}

	configPath := npmUserConfigPath(ctx, env)
	before, _, readErr := readExisting(configPath)
	status := Status{
		ID:           "npm",
		Name:         "npm",
		Executable:   exe,
		Version:      version,
		Installed:    true,
		Supported:    true,
		Configurable: true,
		ConfigPath:   configPath,
		TargetRaw:    "min-release-age=" + strconv.Itoa(days),
	}
	if readErr != nil {
		status.Error = readErr.Error()
		status.Configurable = false
		return finalizeStatus(status, days, allowLower)
	}
	if current, raw, ok := config.ReadKeyValueInt(before, []string{"min-release-age"}); ok {
		seconds := config.DaysToSeconds(current)
		status.currentAgeSeconds = &seconds
		status.CurrentRaw = "min-release-age=" + raw
	}
	after := config.UpsertKeyValue(before, "min-release-age", strconv.Itoa(days), nil, []string{"before"})
	if env.SavePrefixTilde {
		currentPrefix, ok := config.ReadKeyValue(after, []string{"save-prefix"})
		if !ok || strings.Trim(currentPrefix, `"' `) != "~" {
			after = config.UpsertKeyValue(after, "save-prefix", "~", nil, nil)
			status.savePrefixChange = true
		}
	}
	description := "set npm min-release-age"
	if env.SavePrefixTilde {
		description = "set npm min-release-age and save-prefix"
	}
	status.Changes = []config.Change{{
		ManagerID:   status.ID,
		ManagerName: status.Name,
		Path:        configPath,
		Description: description,
		Before:      before,
		After:       after,
	}}
	return finalizeStatus(status, days, allowLower)
}

func (NPM) Remove(ctx context.Context, env Env) Status {
	env = env.withDefaults()
	exe, err := env.Runner.LookPath("npm")
	if err != nil {
		return missingRemoveStatus("npm", "npm")
	}

	version := commandVersion(ctx, env.Runner, "npm", "--version")
	if version == "" {
		version = "detected"
	}
	configPath := npmUserConfigPath(ctx, env)
	before, _, readErr := readExisting(configPath)
	status := Status{
		ID:           "npm",
		Name:         "npm",
		Executable:   exe,
		Version:      version,
		Installed:    true,
		Supported:    true,
		Configurable: true,
		ConfigPath:   configPath,
		TargetAge:    removeTargetAge,
	}
	aliases := []string{"min-release-age"}
	targetRaw := "remove min-release-age"
	description := "remove npm min-release-age"
	if env.SavePrefixTilde {
		aliases = append(aliases, "save-prefix")
		targetRaw = "remove min-release-age and save-prefix"
		description = "remove npm min-release-age and save-prefix"
	}
	status.TargetRaw = targetRaw
	if readErr != nil {
		status.Error = readErr.Error()
		status.Configurable = false
		return finalizeRemoveStatus(status)
	}
	if raw, ok := config.ReadKeyValue(before, []string{"min-release-age"}); ok {
		status.CurrentRaw = "min-release-age=" + raw
		if current, _, ok := config.ReadKeyValueInt(before, []string{"min-release-age"}); ok {
			seconds := config.DaysToSeconds(current)
			status.currentAgeSeconds = &seconds
		}
	}
	after := config.RemoveKeyValue(before, aliases)
	if before != after {
		status.Changes = []config.Change{{
			ManagerID:   status.ID,
			ManagerName: status.Name,
			Path:        configPath,
			Description: description,
			Before:      before,
			After:       after,
		}}
	}
	return finalizeRemoveStatus(status)
}

func npmUserConfigPath(ctx context.Context, env Env) string {
	out, err := env.Runner.Output(ctx, "npm", "config", "get", "userconfig")
	if err == nil && out != "" && out != "undefined" && out != "null" {
		return out
	}
	return filepath.Join(env.HomeDir, ".npmrc")
}
