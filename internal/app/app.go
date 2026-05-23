package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime/debug"
	"strings"
	"time"

	"github.com/xavier-castro/smsc/internal/config"
	"github.com/xavier-castro/smsc/internal/doctor"
	"github.com/xavier-castro/smsc/internal/managers"
	"github.com/xavier-castro/smsc/internal/ui"
)

var version = "dev"

type planOutput struct {
	Days     int               `json:"days"`
	Remove   bool              `json:"remove"`
	Statuses []managers.Status `json:"statuses"`
	Changes  []config.Change   `json:"changes"`
}

type backupsOutput struct {
	Backups []config.Backup `json:"backups"`
}

type restoreOutput struct {
	Backup   config.Backup           `json:"backup"`
	Restored []config.RestoredChange `json:"restored,omitempty"`
	DryRun   bool                    `json:"dryRun"`
}

func Run(args []string, stdout, stderr io.Writer) int {
	ctx := context.Background()
	return run(ctx, managers.DefaultEnv(), args, stdout, stderr)
}

func run(ctx context.Context, env managers.Env, args []string, stdout, stderr io.Writer) int {
	if hasHelpFlag(args) && (len(args) == 1 || strings.HasPrefix(args[0], "-")) {
		printRootHelp(stdout)
		return 0
	}
	if len(args) > 0 {
		switch args[0] {
		case "doctor":
			return runDoctor(ctx, env, args[1:], stdout, stderr)
		case "backups":
			return runBackups(env, args[1:], stdout, stderr)
		case "restore":
			return runRestore(env, args[1:], stdout, stderr)
		case "version":
			printVersion(stdout)
			return 0
		case "help":
			return runHelp(args[1:], stdout, stderr)
		}
	}

	fs := flag.NewFlagSet("smsc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printRootHelp(fs.Output()) }
	days := fs.Int("days", managers.DefaultDays, "minimum package release age in days")
	managerList := fs.String("managers", "auto", "comma-separated managers, auto, or all")
	dryRun := fs.Bool("dry-run", false, "preview planned changes")
	yes := fs.Bool("yes", false, "apply without interactive confirmation")
	jsonOut := fs.Bool("json", false, "emit JSON")
	allowLower := fs.Bool("allow-lower", false, "allow replacing a stricter existing policy")
	saveTilde := fs.Bool("save-tilde", false, "set npm/pnpm save-prefix to ~ instead of ^ for new dependencies")
	remove := fs.Bool("remove", false, "remove SMSC-managed release-age configuration")
	showVersion := fs.Bool("version", false, "print version")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "unknown command or argument %q\n\n", fs.Arg(0))
		printRootHelp(stderr)
		return 2
	}

	if *showVersion {
		printVersion(stdout)
		return 0
	}
	if !*remove && *days <= 0 {
		fmt.Fprintln(stderr, "days must be greater than zero")
		return 2
	}
	env.SavePrefixTilde = *saveTilde

	noFlags := len(args) == 0
	if noFlags {
		if err := ui.Run(ctx, env, *days, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	var statuses []managers.Status
	if *remove {
		statuses = managers.ScanRemove(ctx, env)
	} else {
		statuses = managers.Scan(ctx, env, *days, *allowLower)
	}
	selected, err := selectedManagers(statuses, *managerList)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	changes := config.MergeChanges(managers.SelectChanges(statuses, selected))

	if *jsonOut {
		return writeJSON(stdout, planOutput{Days: *days, Remove: *remove, Statuses: statuses, Changes: changes})
	}

	if *dryRun {
		printPlan(stdout, *days, *remove, statuses, selected, changes)
		return 0
	}

	if !*yes {
		fmt.Fprintln(stderr, "refusing to apply without --yes; run smsc for the TUI, use --dry-run to preview, or add --yes")
		return 2
	}

	now := time.Now
	if env.Now != nil {
		now = env.Now
	}
	applied, err := config.ApplyChanges(changes, env.ConfigHome, now().UTC())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	printApplied(stdout, applied, *remove)
	return 0
}

func runDoctor(ctx context.Context, env managers.Env, args []string, stdout, stderr io.Writer) int {
	if hasHelpFlag(args) {
		printDoctorHelp(stdout)
		return 0
	}
	fs := flag.NewFlagSet("smsc doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printDoctorHelp(fs.Output()) }
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "unknown doctor argument %q\n\n", fs.Arg(0))
		printDoctorHelp(stderr)
		return 2
	}
	if err := doctor.Run(ctx, env, stdout, *jsonOut); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runBackups(env managers.Env, args []string, stdout, stderr io.Writer) int {
	if hasHelpFlag(args) {
		printBackupsHelp(stdout)
		return 0
	}
	fs := flag.NewFlagSet("smsc backups", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printBackupsHelp(fs.Output()) }
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "unknown backups argument %q\n\n", fs.Arg(0))
		printBackupsHelp(stderr)
		return 2
	}
	backups, err := config.ListBackups(env.ConfigHome)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *jsonOut {
		return writeJSON(stdout, backupsOutput{Backups: backups})
	}
	printBackups(stdout, env.ConfigHome, backups)
	return 0
}

func runRestore(env managers.Env, args []string, stdout, stderr io.Writer) int {
	if hasHelpFlag(args) {
		printRestoreHelp(stdout)
		return 0
	}
	fs := flag.NewFlagSet("smsc restore", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printRestoreHelp(fs.Output()) }
	yes := fs.Bool("yes", false, "restore without interactive confirmation")
	dryRun := fs.Bool("dry-run", false, "preview files that would be restored")
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(normalizeRestoreArgs(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "restore requires exactly one backup timestamp or 'latest'")
		fmt.Fprintln(stderr)
		printRestoreHelp(stderr)
		return 2
	}

	backup, err := config.ResolveBackup(env.ConfigHome, fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *dryRun {
		if *jsonOut {
			return writeJSON(stdout, restoreOutput{Backup: backup, DryRun: true})
		}
		printRestorePlan(stdout, backup)
		return 0
	}
	if !*yes {
		fmt.Fprintln(stderr, "refusing to restore without --yes; run 'smsc backups' to inspect backups or add --dry-run to preview")
		return 2
	}
	restored, err := config.RestoreBackup(backup)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *jsonOut {
		return writeJSON(stdout, restoreOutput{Backup: backup, Restored: restored})
	}
	printRestored(stdout, backup, restored)
	return 0
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "-help" {
			return true
		}
	}
	return false
}

func normalizeRestoreArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			continue
		}
		positionals = append(positionals, arg)
	}
	return append(flags, positionals...)
}

func runHelp(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printRootHelp(stdout)
		return 0
	}
	if len(args) > 1 {
		fmt.Fprintln(stderr, "help accepts at most one topic")
		return 2
	}
	switch args[0] {
	case "doctor":
		printDoctorHelp(stdout)
	case "backups":
		printBackupsHelp(stdout)
	case "restore":
		printRestoreHelp(stdout)
	default:
		fmt.Fprintf(stderr, "unknown help topic %q\n", args[0])
		return 2
	}
	return 0
}

func selectedManagers(statuses []managers.Status, value string) (map[string]bool, error) {
	selected := map[string]bool{}
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "auto" {
		for _, status := range statuses {
			selected[status.ID] = status.Selected
		}
		return selected, nil
	}
	if value == "all" {
		for _, status := range statuses {
			selected[status.ID] = status.Installed && status.Supported && status.Configurable && len(status.Changes) > 0
		}
		return selected, nil
	}

	valid := validManagerIDs(statuses)
	sawManager := false
	for _, part := range strings.Split(value, ",") {
		id := strings.TrimSpace(strings.ToLower(part))
		if id == "" {
			continue
		}
		sawManager = true
		if id == "vite+" || id == "vite-plus" {
			id = "vp"
		}
		if !valid[id] {
			return nil, fmt.Errorf("unknown manager %q; valid managers are: %s", id, strings.Join(managerChoices(statuses), ", "))
		}
		selected[id] = true
	}
	if !sawManager {
		return nil, errors.New("no managers specified; use auto, all, or a comma-separated manager list")
	}
	return selected, nil
}

func validManagerIDs(statuses []managers.Status) map[string]bool {
	valid := make(map[string]bool, len(statuses))
	for _, status := range statuses {
		valid[status.ID] = true
	}
	return valid
}

func managerChoices(statuses []managers.Status) []string {
	choices := []string{"auto", "all"}
	for _, status := range statuses {
		choices = append(choices, status.ID)
	}
	return choices
}

func printPlan(w io.Writer, days int, remove bool, statuses []managers.Status, selected map[string]bool, changes []config.Change) {
	if remove {
		fmt.Fprintln(w, "SMSC dry run: remove release-age configuration from global package-manager config")
	} else {
		fmt.Fprintf(w, "SMSC dry run: target global release age %s\n", config.FormatDays(days))
	}
	fmt.Fprintln(w)
	for _, status := range statuses {
		marker := " "
		if selected[status.ID] {
			marker = "x"
		}
		current := status.CurrentAge
		if current == "" {
			current = "not configured"
		}
		state := statusText(status, remove)
		fmt.Fprintf(w, "[%s] %-12s current: %-16s target: %-8s %s\n", marker, status.Name, current, status.TargetAge, state)
	}
	if len(changes) == 0 {
		if remove {
			fmt.Fprintln(w, "\nNo global release-age configuration was found.")
		} else {
			fmt.Fprintln(w, "\nNo global config changes planned.")
		}
		return
	}
	if remove {
		fmt.Fprintln(w, "\nPlanned removals:")
	} else {
		fmt.Fprintln(w, "\nPlanned global file changes:")
	}
	for _, change := range changes {
		fmt.Fprintf(w, "- %s: %s\n", change.Path, change.Description)
	}
}

func statusText(status managers.Status, remove bool) string {
	if !status.Installed {
		return "package manager not installed"
	}
	if status.Error != "" {
		return status.Error
	}
	if remove {
		if status.NeedsChange {
			return "will remove"
		}
		if status.Reason != "" {
			return status.Reason
		}
		return "release-age configuration not found"
	}
	if status.Reason != "" {
		return status.Reason
	}
	if status.NeedsChange {
		return "will update global config"
	}
	if status.Protected {
		return "protected by global config"
	}
	return "supported"
}

func printApplied(w io.Writer, applied []config.AppliedChange, remove bool) {
	if len(applied) == 0 {
		if remove {
			fmt.Fprintln(w, "No global release-age configuration was found.")
		} else {
			fmt.Fprintln(w, "Global release-age configuration already matches requested policy. No changes applied.")
		}
		return
	}
	if remove {
		fmt.Fprintln(w, "Removed release-age configuration:")
	} else {
		fmt.Fprintln(w, "Applied global release-age configuration:")
	}
	for _, item := range applied {
		if item.Changed {
			if item.BackupPath != "" {
				fmt.Fprintf(w, "- %s (backup: %s)\n", item.Path, item.BackupPath)
			} else {
				fmt.Fprintf(w, "- %s (new file; restore will remove it)\n", item.Path)
			}
			continue
		}
		fmt.Fprintf(w, "- %s unchanged\n", item.Path)
	}
}

func printBackups(w io.Writer, configHome string, backups []config.Backup) {
	if len(backups) == 0 {
		fmt.Fprintf(w, "No SMSC backups found in %s.\n", config.BackupRoot(configHome))
		return
	}
	fmt.Fprintln(w, "SMSC backups (newest first):")
	for _, backup := range backups {
		fmt.Fprintf(w, "- %s (%d file", backup.Timestamp, len(backup.Changes))
		if len(backup.Changes) != 1 {
			fmt.Fprint(w, "s")
		}
		fmt.Fprintf(w, ") %s\n", backup.Path)
	}
	fmt.Fprintln(w, "Restore with: smsc restore latest --yes")
}

func printRestorePlan(w io.Writer, backup config.Backup) {
	fmt.Fprintf(w, "SMSC restore dry run: %s\n\n", backup.Timestamp)
	if len(backup.Changes) == 0 {
		fmt.Fprintln(w, "Backup manifest has no file changes.")
		return
	}
	fmt.Fprintln(w, "Files that would be restored:")
	for _, change := range backup.Changes {
		if !change.Changed {
			continue
		}
		if change.BackupPath == "" {
			fmt.Fprintf(w, "- %s (would remove file created by SMSC)\n", change.Path)
			continue
		}
		fmt.Fprintf(w, "- %s (from %s)\n", change.Path, change.BackupPath)
	}
}

func printRestored(w io.Writer, backup config.Backup, restored []config.RestoredChange) {
	if len(restored) == 0 {
		fmt.Fprintf(w, "Backup %s had no changed files to restore.\n", backup.Timestamp)
		return
	}
	fmt.Fprintf(w, "Restored SMSC backup %s:\n", backup.Timestamp)
	for _, item := range restored {
		if item.Removed {
			fmt.Fprintf(w, "- %s removed\n", item.Path)
			continue
		}
		fmt.Fprintf(w, "- %s restored", item.Path)
		if item.BackupPath != "" {
			fmt.Fprintf(w, " (from %s)", item.BackupPath)
		}
		fmt.Fprintln(w)
	}
}

func printRootHelp(w io.Writer) {
	fmt.Fprintf(w, `SMSC secures global package-manager config for individual developers.
It only writes global release-age settings; project-local config is reported by doctor but never modified.

Usage:
  smsc                         open the interactive TUI
  smsc [flags]                 plan or apply global config changes
  smsc doctor [--json]         inspect global protection and local overrides
  smsc backups [--json]        list SMSC backup manifests
  smsc restore <backup> --yes  restore a backup (use "latest" or a timestamp)
  smsc version                 print version

Flags:
  --days int          minimum package release age in days (default %d)
  --managers string   comma-separated managers, auto, or all (default "auto")
                      valid managers: npm, pnpm, vp, yarn, bun, uv
  --dry-run           preview planned changes
  --yes               apply without interactive confirmation
  --json              emit JSON
  --allow-lower       allow replacing a stricter existing policy
  --save-tilde        set npm/pnpm save-prefix to ~ instead of ^
  --remove            remove SMSC-managed release-age configuration
  --version           print version
  -h, --help          show help

Examples:
  smsc --dry-run
  smsc --days 8 --managers all --yes
  smsc --days 8 --save-tilde --managers npm,pnpm --yes
  smsc backups
  smsc restore latest --yes
`, managers.DefaultDays)
}

func printDoctorHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: smsc doctor [--json]

Inspect detected package managers, global release-age settings, unsupported versions,
and project-local config files that may override global policy. Doctor never modifies files.

Flags:
  --json      emit JSON
  -h, --help  show help
`)
}

func printBackupsHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: smsc backups [--json]

List SMSC backup manifests created before global config changes.

Flags:
  --json      emit JSON
  -h, --help  show help
`)
}

func printRestoreHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: smsc restore <latest|timestamp> [--dry-run] [--yes] [--json]

Restore files from an SMSC backup manifest. If a file was created by SMSC and no
previous file existed, restore removes that file.

Flags:
  --dry-run   preview files that would be restored
  --yes       restore without interactive confirmation
  --json      emit JSON
  -h, --help  show help
`)
}

func printVersion(w io.Writer) {
	fmt.Fprintln(w, "smsc "+resolvedVersion())
}

func resolvedVersion() string {
	if version != "" && version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	if version != "" {
		return version
	}
	return time.Now().UTC().Format("dev-20060102")
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
