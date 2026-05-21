# Changelog

All notable SMSC changes are documented here.

## Unreleased

## v0.1.0 - 2026-05-21

Initial public release:

- Lock v1 scope to individual developers and global package-manager config only.
- Add first-class backup listing and restore commands.
- Expand `doctor` diagnostics for unsupported versions, global protection status, and project-local override reporting.
- Add CI, security scanning, release workflow, MIT license, and release documentation.
- Expand tests for manager fixtures, malformed configs, aliases, missing binaries, and end-to-end backup/restore flows.
- TUI and CLI for applying release-age policies to npm, pnpm, VP, Yarn, Bun, and uv global config.
- Dry-run and JSON output.
- Backup manifests before writes.
- Remove mode for SMSC-managed release-age keys.
