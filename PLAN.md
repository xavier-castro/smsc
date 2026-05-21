# SMSC Usefulness Improvement Plan

## Context

SMSC is currently a small Go CLI/TUI that applies minimum release-age policies to global package-manager configs for npm, pnpm, VP, Yarn, Bun, and uv. The core code already supports scanning, dry-run JSON, applying changes with backups, removal, and a basic `doctor` command.

Initial scan findings:

- Existing package-manager implementations live in `internal/managers/` and share `Status`, `Env`, `Runner`, and `config.Change` abstractions.
- Config mutation helpers exist in `internal/config/` for rc/key-value, TOML, YAML, age parsing, merging, backups, and manifest writing.
- CLI orchestration is in `internal/app/app.go`; TUI flow is in `internal/ui/ui.go`; diagnostics are in `internal/doctor/doctor.go`.
- Tests exist for app/config/manager/UI behavior and currently pass with `go test ./...`; rough package coverage is modest, especially UI and diagnostics.
- Distribution scaffolding exists via `.goreleaser.yaml` and `packaging/homebrew/smsc.rb`, but there are no GitHub workflow files, no `LICENSE` file despite the MIT metadata, and docs are limited to `README.md` plus one screenshot.

## Approach

Make SMSC useful for **individual developers** as a polished local safety tool. Keep scope intentionally limited to **global package-manager configuration only**; project-local files should be detected and explained, not modified. Prioritize polish and trust before broader package-manager coverage.

Recommended v1 direction:

1. Publish-ready packaging and repository trust signals.
2. Strong documentation and support matrix for what SMSC can and cannot protect.
3. First-class rollback UX using the existing backup manifest path.
4. Better diagnostics around local overrides, unsupported versions, and whether global config is protected.
5. Higher confidence through CI, security scanning, and fixture/e2e tests.

Defer team/CI enforcement, project-local fixing, and broad new package-manager support until after the individual-developer global-config workflow is polished.

## Files to modify

Likely files/areas for an implementation pass:

- `README.md` — support matrix, examples, limitations, quickstart, rollback instructions.
- `LICENSE` — add the MIT license matching release metadata.
- `.github/workflows/*.yml` — CI, release, govulncheck/CodeQL/Scorecard or equivalent checks.
- `.goreleaser.yaml` — finalize release automation/tap metadata.
- `packaging/homebrew/smsc.rb` — replace placeholder SHA after first release or rely on GoReleaser-generated formula.
- `internal/app/app.go` — improved help/UX, `backups`/`restore` commands, stricter manager validation.
- `internal/doctor/doctor.go` — deeper diagnostics and actionable remediation output without modifying project-local config.
- `internal/config/apply.go` — reuse backup manifest data and add restore/list-backup helpers.
- `internal/managers/*.go` — support matrix accuracy and edge-case handling for current managers; no new managers for v1 unless needed to fix correctness.
- `internal/**/*_test.go` — add fixtures and workflow-critical tests.

## Reuse

Reuse existing abstractions instead of adding parallel systems:

- `internal/managers.Manager`, `Status`, `Env`, and `Runner` for all manager detection/config planning.
- `managers.Scan`, `ScanRemove`, and `SelectChanges` for CLI, TUI, JSON, and doctor flows.
- `config.Change`, `MergeChanges`, and `ApplyChanges` for any apply/remove/restore path.
- `config.ReadFile`, rc/TOML/YAML helpers, and age parsing helpers for diagnostics and tests.
- Existing fake runner patterns in `internal/managers/managers_test.go` and temp-dir tests in `internal/app/app_test.go`.

## Steps

- [x] Lock v1 scope in docs and help text: SMSC is for individual developers and global config only.
- [x] Add repository trust basics: `LICENSE`, CI, vulnerability scanning, release workflow, badges, and a reproducible install path.
- [x] Expand README into a user guide: supported managers/versions/config keys, exact files written, backup/restore, local override caveats, examples, and limitations.
- [x] Add first-class rollback commands, e.g. `smsc backups` to list manifests and `smsc restore latest --yes` / `smsc restore <timestamp> --yes` to restore backed-up files.
- [x] Improve CLI polish: validate unknown `--managers` values, make help/version behavior conventional, add clearer dry-run/apply/remove/restore messages, and keep JSON stable.
- [x] Strengthen `doctor`: explain unsupported manager versions, identify project-local override files and relevant key values where possible, summarize global protection status, and emit stable JSON.
- [x] Add fixture-based tests for each manager/config format, including empty files, existing stricter policies, malformed configs, aliases, missing binaries, unsupported versions, and duplicate VP/pnpm path behavior.
- [x] Add end-to-end tests for dry-run/apply/remove/backup/list-backups/restore in temp home/config directories.
- [x] Prepare the first release: tag, GoReleaser artifacts, Homebrew tap/formula, checksum verification, changelog/release notes.
- [x] Defer expanded package-manager support, team policy, CI enforcement, and project-local fixing until after v1 polish.

## Verification

- Run `go test ./...` locally and in CI.
- Run coverage and ensure new tests exercise manager/config edge cases and doctor behavior.
- Manually run `smsc --dry-run`, `smsc --json --dry-run`, `smsc --days 8 --managers all --yes`, `smsc backups`, `smsc restore latest --yes`, `smsc --remove --managers all --yes`, and `smsc doctor` against temp home/config directories.
- Verify release artifacts with checksums and test Homebrew install/upgrade/uninstall on macOS; smoke-test Linux tarballs. Publishing the tag and artifacts remains a maintainer action at release time.
- Confirm README examples match actual CLI output and config files written.

## Decisions

- Target audience: individual developers.
- Scope: global package-manager config only; never modify project-local overrides.
- Rollback: first-class backup listing and restore commands are part of v1 polish.
- Priority: polish, trust, docs, release readiness, and recovery UX before scope expansion.
