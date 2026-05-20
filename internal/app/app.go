package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xavier/smsc/internal/config"
	"github.com/xavier/smsc/internal/doctor"
	"github.com/xavier/smsc/internal/managers"
	"github.com/xavier/smsc/internal/ui"
)

var version = "dev"

type planOutput struct {
	Days     int               `json:"days"`
	Statuses []managers.Status `json:"statuses"`
	Changes  []config.Change   `json:"changes"`
}

func Run(args []string, stdout, stderr io.Writer) int {
	ctx := context.Background()
	env := managers.DefaultEnv()

	if len(args) > 0 && args[0] == "doctor" {
		return runDoctor(ctx, env, args[1:], stdout, stderr)
	}

	fs := flag.NewFlagSet("smsc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	days := fs.Int("days", managers.DefaultDays, "minimum package release age in days")
	managerList := fs.String("managers", "auto", "comma-separated managers or all")
	dryRun := fs.Bool("dry-run", false, "preview planned changes")
	yes := fs.Bool("yes", false, "apply without interactive confirmation")
	jsonOut := fs.Bool("json", false, "emit JSON")
	allowLower := fs.Bool("allow-lower", false, "allow replacing a stricter existing policy")
	showVersion := fs.Bool("version", false, "print version")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "smsc "+version)
		return 0
	}
	if *days <= 0 {
		fmt.Fprintln(stderr, "days must be greater than zero")
		return 2
	}

	noFlags := len(args) == 0
	if noFlags {
		if err := ui.Run(ctx, env, *days, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	statuses := managers.Scan(ctx, env, *days, *allowLower)
	selected := selectedManagers(statuses, *managerList)
	changes := config.MergeChanges(managers.SelectChanges(statuses, selected))

	if *jsonOut {
		return writeJSON(stdout, planOutput{Days: *days, Statuses: statuses, Changes: changes})
	}

	if *dryRun {
		printPlan(stdout, *days, statuses, selected, changes)
		return 0
	}

	if !*yes {
		fmt.Fprintln(stderr, "refusing to apply without --yes; run smsc for the TUI or add --yes")
		return 2
	}

	applied, err := config.ApplyChanges(changes, env.ConfigHome, env.Now().UTC())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	printApplied(stdout, applied)
	return 0
}

func runDoctor(ctx context.Context, env managers.Env, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("smsc doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := doctor.Run(ctx, env, stdout, *jsonOut); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func selectedManagers(statuses []managers.Status, value string) map[string]bool {
	selected := map[string]bool{}
	value = strings.TrimSpace(value)
	if value == "" || value == "auto" {
		for _, status := range statuses {
			selected[status.ID] = status.Selected
		}
		return selected
	}
	if value == "all" {
		for _, status := range statuses {
			selected[status.ID] = status.Installed && status.Supported && status.Configurable && len(status.Changes) > 0
		}
		return selected
	}
	for _, part := range strings.Split(value, ",") {
		id := strings.TrimSpace(strings.ToLower(part))
		if id != "" {
			selected[id] = true
		}
	}
	return selected
}

func printPlan(w io.Writer, days int, statuses []managers.Status, selected map[string]bool, changes []config.Change) {
	fmt.Fprintf(w, "SMSC dry run: target release age %s\n\n", config.FormatDays(days))
	for _, status := range statuses {
		marker := " "
		if selected[status.ID] {
			marker = "x"
		}
		current := status.CurrentAge
		if current == "" {
			current = "not configured"
		}
		state := status.Reason
		if state == "" && !status.Installed {
			state = "package manager not installed"
		}
		if state == "" && status.NeedsChange {
			state = "will update"
		}
		if state == "" {
			state = "secure configuration added"
		}
		fmt.Fprintf(w, "[%s] %-12s current: %-16s target: %-8s %s\n", marker, status.Name, current, status.TargetAge, state)
	}
	if len(changes) == 0 {
		fmt.Fprintln(w, "\nNo changes planned.")
		return
	}
	fmt.Fprintln(w, "\nPlanned file changes:")
	for _, change := range changes {
		fmt.Fprintf(w, "- %s: %s\n", change.Path, change.Description)
	}
}

func printApplied(w io.Writer, applied []config.AppliedChange) {
	if len(applied) == 0 {
		fmt.Fprintln(w, "No changes applied.")
		return
	}
	fmt.Fprintln(w, "Applied changes:")
	for _, item := range applied {
		if item.Changed {
			if item.BackupPath != "" {
				fmt.Fprintf(w, "- %s (backup: %s)\n", item.Path, item.BackupPath)
			} else {
				fmt.Fprintf(w, "- %s\n", item.Path)
			}
			continue
		}
		fmt.Fprintf(w, "- %s unchanged\n", item.Path)
	}
}

func writeJSON(w io.Writer, value any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return 1
	}
	return 0
}

func init() {
	if version == "" {
		version = time.Now().UTC().Format("dev-20060102")
	}
}
