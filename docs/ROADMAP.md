# Roadmap

## v1 polish scope

SMSC v1 is focused on individual developers and global package-manager configuration:

- reliable dry-run/apply/remove flows
- clear diagnostics for unsupported versions and local override caveats
- first-class backups and restore
- trustworthy release artifacts and installation docs
- tests for supported managers and config formats

## Deferred until after v1

The following are intentionally out of scope for v1 polish:

- team policy management
- CI enforcement or repository compliance gates
- automatic project-local config modification
- organization-wide rollout tooling
- broad package-manager expansion beyond managers with native release-age support

Project-local files may override global config. SMSC will continue to report likely overrides, but v1 will not edit them.
