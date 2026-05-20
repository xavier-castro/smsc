package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xavier/smsc/internal/managers"
)

type Report struct {
	Statuses       []managers.Status `json:"statuses"`
	LocalOverrides []string          `json:"localOverrides"`
}

func Run(ctx context.Context, env managers.Env, out io.Writer, jsonOut bool) error {
	statuses := managers.Scan(ctx, env, managers.DefaultDays, false)
	report := Report{
		Statuses:       statuses,
		LocalOverrides: FindLocalOverrides(env.Cwd),
	}
	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	fmt.Fprintln(out, "SMSC doctor")
	fmt.Fprintln(out)
	for _, status := range statuses {
		state := "missing"
		if status.Installed && status.Supported {
			state = "supported"
		} else if status.Installed {
			state = "unsupported"
		}
		current := status.CurrentAge
		if current == "" {
			current = "not configured"
		}
		fmt.Fprintf(out, "- %-12s %-11s current: %s", status.Name, state, current)
		if status.Reason != "" {
			fmt.Fprintf(out, " (%s)", status.Reason)
		}
		fmt.Fprintln(out)
	}

	if len(report.LocalOverrides) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Possible project-local overrides:")
		for _, path := range report.LocalOverrides {
			fmt.Fprintf(out, "- %s\n", path)
		}
	}
	return nil
}

func FindLocalOverrides(cwd string) []string {
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
	for {
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
