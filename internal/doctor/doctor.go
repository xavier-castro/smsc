package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xavier-castro/smsc/internal/config"
	"github.com/xavier-castro/smsc/internal/managers"
)

type Report struct {
	TargetAge      string            `json:"targetAge"`
	Protected      bool              `json:"protected"`
	Summary        Summary           `json:"summary"`
	Statuses       []managers.Status `json:"statuses"`
	LocalOverrides []LocalOverride   `json:"localOverrides"`
}

type Summary struct {
	Installed          int `json:"installed"`
	Supported          int `json:"supported"`
	Protected          int `json:"protected"`
	NeedsChange        int `json:"needsChange"`
	Unsupported        int `json:"unsupported"`
	Missing            int `json:"missing"`
	Errors             int `json:"errors"`
	LocalOverrideFiles int `json:"localOverrideFiles"`
}

type LocalOverride struct {
	Path   string               `json:"path"`
	Values []LocalOverrideValue `json:"values,omitempty"`
	Error  string               `json:"error,omitempty"`
}

type LocalOverrideValue struct {
	ManagerID string `json:"managerId"`
	Key       string `json:"key"`
	Raw       string `json:"raw"`
	Age       string `json:"age,omitempty"`
}

func Run(ctx context.Context, env managers.Env, out io.Writer, jsonOut bool) error {
	if env.Now == nil {
		env.Now = time.Now
	}
	statuses := managers.Scan(ctx, env, managers.DefaultDays, false)
	localOverrides := AnalyzeLocalOverrides(env.Cwd, env.HomeDir, env.Now())
	summary := summarize(statuses, len(localOverrides))
	report := Report{
		TargetAge:      config.FormatDays(managers.DefaultDays),
		Protected:      summary.Installed > 0 && summary.Protected == summary.Supported && summary.Unsupported == 0 && summary.Errors == 0,
		Summary:        summary,
		Statuses:       statuses,
		LocalOverrides: localOverrides,
	}
	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	fmt.Fprintln(out, "SMSC doctor")
	fmt.Fprintln(out, "Scope: individual developer, global package-manager config only")
	fmt.Fprintf(out, "Target release age: %s\n", report.TargetAge)
	if report.Protected {
		fmt.Fprintln(out, "Global protection: configured for detected supported managers")
	} else {
		fmt.Fprintln(out, "Global protection: action needed or no supported package managers detected")
	}
	fmt.Fprintln(out)

	for _, status := range statuses {
		current := status.CurrentAge
		if current == "" {
			current = "not configured"
		}
		fmt.Fprintf(out, "- %-12s %-34s current: %-16s target: %s", status.Name, doctorState(status), current, status.TargetAge)
		if status.ConfigPath != "" {
			fmt.Fprintf(out, " config: %s", status.ConfigPath)
		}
		if status.Reason != "" {
			fmt.Fprintf(out, " (%s)", status.Reason)
		}
		if status.Error != "" {
			fmt.Fprintf(out, " (%s)", status.Error)
		}
		fmt.Fprintln(out)
	}

	if len(report.LocalOverrides) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Project-local config files can override global settings. SMSC reports them but does not modify them:")
		for _, override := range report.LocalOverrides {
			fmt.Fprintf(out, "- %s", override.Path)
			if override.Error != "" {
				fmt.Fprintf(out, " (%s)", override.Error)
			}
			fmt.Fprintln(out)
			for _, value := range override.Values {
				age := value.Age
				if age == "" {
					age = "unparsed"
				}
				fmt.Fprintf(out, "  - %s %s=%s (%s)\n", value.ManagerID, value.Key, value.Raw, age)
			}
		}
	}
	return nil
}

func summarize(statuses []managers.Status, localOverrideFiles int) Summary {
	var summary Summary
	summary.LocalOverrideFiles = localOverrideFiles
	for _, status := range statuses {
		if !status.Installed {
			summary.Missing++
			continue
		}
		summary.Installed++
		if status.Error != "" {
			summary.Errors++
		}
		if !status.Supported {
			summary.Unsupported++
			continue
		}
		summary.Supported++
		if status.Protected {
			summary.Protected++
		}
		if status.NeedsChange {
			summary.NeedsChange++
		}
	}
	return summary
}

func doctorState(status managers.Status) string {
	if !status.Installed {
		return "package manager not installed"
	}
	if status.Error != "" {
		return "config read/write error"
	}
	if !status.Supported {
		return "unsupported version"
	}
	if status.Protected {
		return "protected by global config"
	}
	if status.NeedsChange {
		return "needs global config change"
	}
	return "supported"
}

func AnalyzeLocalOverrides(cwd, home string, now time.Time) []LocalOverride {
	paths := findLocalOverridePaths(cwd, home)
	overrides := make([]LocalOverride, 0, len(paths))
	for _, path := range paths {
		override := LocalOverride{Path: path}
		content, err := os.ReadFile(path)
		if err != nil {
			override.Error = err.Error()
			overrides = append(overrides, override)
			continue
		}
		values, err := localOverrideValues(path, string(content), now)
		if err != nil {
			override.Error = err.Error()
		}
		override.Values = values
		overrides = append(overrides, override)
	}
	return overrides
}

func FindLocalOverrides(cwd string) []string {
	return findLocalOverridePaths(cwd, "")
}

func findLocalOverridePaths(cwd, home string) []string {
	if cwd == "" {
		return nil
	}
	names := map[string]bool{
		".npmrc":              true,
		".pnpmrc":             true,
		"pnpm-workspace.yaml": true,
		".yarnrc.yml":         true,
		"bunfig.toml":         true,
		"uv.toml":             true,
	}
	var found []string
	current := filepath.Clean(cwd)
	home = filepath.Clean(home)
	for {
		if home != "." && home != "" && current == home {
			break
		}
		entries, err := os.ReadDir(current)
		if err == nil {
			for _, entry := range entries {
				if names[entry.Name()] {
					found = append(found, filepath.Join(current, entry.Name()))
				}
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return found
}

func localOverrideValues(path, content string, now time.Time) ([]LocalOverrideValue, error) {
	switch filepath.Base(path) {
	case ".npmrc":
		return npmLocalValues(content, now), nil
	case ".pnpmrc":
		return pnpmLocalValues(content), nil
	case "pnpm-workspace.yaml":
		return pnpmWorkspaceValues(content)
	case ".yarnrc.yml":
		return yarnLocalValues(content)
	case "bunfig.toml":
		return bunLocalValues(content)
	case "uv.toml":
		return uvLocalValues(content, now)
	default:
		return nil, nil
	}
}

func npmLocalValues(content string, now time.Time) []LocalOverrideValue {
	var values []LocalOverrideValue
	if raw, ok := config.ReadKeyValue(content, []string{"min-release-age"}); ok {
		value := LocalOverrideValue{ManagerID: "npm", Key: "min-release-age", Raw: raw}
		if days, _, ok := config.ReadKeyValueInt(content, []string{"min-release-age"}); ok {
			value.Age = config.SecondsLabel(config.DaysToSeconds(days))
		}
		values = append(values, value)
	}
	if raw, ok := config.ReadKeyValue(content, []string{"before"}); ok {
		value := LocalOverrideValue{ManagerID: "npm", Key: "before", Raw: raw}
		if seconds, ok := config.ParseRFC3339Cutoff(raw, now); ok {
			value.Age = config.SecondsLabel(seconds)
		}
		values = append(values, value)
	}
	return values
}

func pnpmLocalValues(content string) []LocalOverrideValue {
	aliases := []string{"minimum-release-age", "minimumReleaseAge"}
	raw, ok := config.ReadKeyValue(content, aliases)
	if !ok {
		return nil
	}
	value := LocalOverrideValue{ManagerID: "pnpm", Key: "minimumReleaseAge", Raw: raw}
	if minutes, _, ok := config.ReadKeyValueInt(content, aliases); ok {
		value.Age = config.SecondsLabel(int64(minutes) * 60)
	}
	return []LocalOverrideValue{value}
}

func pnpmWorkspaceValues(content string) ([]LocalOverrideValue, error) {
	for _, key := range []string{"minimumReleaseAge", "minimum-release-age"} {
		raw, ok, err := config.ReadYAMLString(content, key)
		if err != nil || !ok {
			if err != nil {
				return nil, err
			}
			continue
		}
		value := LocalOverrideValue{ManagerID: "pnpm", Key: key, Raw: raw}
		if minutes, err := strconv.Atoi(strings.Trim(raw, `"' `)); err == nil {
			value.Age = config.SecondsLabel(int64(minutes) * 60)
		}
		return []LocalOverrideValue{value}, nil
	}
	return nil, nil
}

func yarnLocalValues(content string) ([]LocalOverrideValue, error) {
	raw, ok, err := config.ReadYAMLString(content, "npmMinimalAgeGate")
	if err != nil || !ok {
		return nil, err
	}
	value := LocalOverrideValue{ManagerID: "yarn", Key: "npmMinimalAgeGate", Raw: raw}
	if seconds, ok := parseYarnAge(raw); ok {
		value.Age = config.SecondsLabel(seconds)
	}
	return []LocalOverrideValue{value}, nil
}

func bunLocalValues(content string) ([]LocalOverrideValue, error) {
	seconds, raw, ok, err := config.ReadBunMinimumReleaseAge(content)
	if err != nil || !ok {
		return nil, err
	}
	return []LocalOverrideValue{{ManagerID: "bun", Key: "install.minimumReleaseAge", Raw: raw, Age: config.SecondsLabel(seconds)}}, nil
}

func uvLocalValues(content string, now time.Time) ([]LocalOverrideValue, error) {
	raw, ok, err := config.ReadTOMLTopString(content, "exclude-newer")
	if err != nil || !ok {
		return nil, err
	}
	value := LocalOverrideValue{ManagerID: "uv", Key: "exclude-newer", Raw: raw}
	if seconds, ok := config.ParseAgeDuration(raw); ok {
		value.Age = config.SecondsLabel(seconds)
	} else if seconds, ok := config.ParseRFC3339Cutoff(raw, now); ok {
		value.Age = config.SecondsLabel(seconds)
	}
	return []LocalOverrideValue{value}, nil
}

func parseYarnAge(raw string) (int64, bool) {
	if seconds, ok := config.ParseAgeDuration(raw); ok {
		return seconds, true
	}
	minutes, err := strconv.ParseInt(strings.Trim(raw, `"' `), 10, 64)
	if err != nil {
		return 0, false
	}
	return minutes * 60, true
}
