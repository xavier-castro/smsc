# SMSC: Secure My Supply Chain

`smsc` is a terminal UI for applying minimum release-age policies across package managers. It helps reduce exposure to freshly published malicious packages by delaying installation of new versions.

SMSC is deliberately small: it adds release-age flags to your global package-manager configuration. It is not a fix-all for software supply-chain security, but it is another prevention measure against current supply-chain attacks.

V1 secures global package-manager configuration for:

- npm
- pnpm
- Vite+ / `vp` through the pnpm policy it uses
- Yarn
- Bun
- uv

The default recommendation is 8 days.

## Install

Initial distribution is intended through a custom Homebrew tap:

```sh
brew tap <owner>/smsc
brew install smsc
```

Until a release exists, build locally:

```sh
go build ./cmd/smsc
./smsc --dry-run
```

## Usage

Open the TUI:

```sh
smsc
```

Preview planned changes:

```sh
smsc --dry-run
```

Apply non-interactively:

```sh
smsc --days 8 --managers all --yes
```

Remove SMSC-managed release-age configuration:

```sh
smsc --remove --managers all --yes
```

Emit machine-readable dry-run output:

```sh
smsc --json --dry-run
```

Print diagnostics:

```sh
smsc doctor
```

## What SMSC Writes

SMSC only edits the specific release-age keys it owns and creates backups before applying changes:

```text
~/.config/smsc/backups/<timestamp>/
```

Package-local config can override global settings. `smsc doctor` warns when it sees likely local overrides.
