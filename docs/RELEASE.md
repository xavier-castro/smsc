# Release checklist

SMSC releases are intended to be reproducible and verifiable.

## Before tagging

1. Confirm `README.md` examples match the current CLI.
2. Run:

   ```sh
   gofmt -w .
   go test ./...
   go vet ./...
   go run golang.org/x/vuln/cmd/govulncheck@latest ./...
   goreleaser check
   ```

3. Update `CHANGELOG.md` and replace the planned date for the release.
4. Confirm the Homebrew tap in `.goreleaser.yaml` points to the correct repository.

## Tag and release

```sh
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

The GitHub release workflow runs GoReleaser, publishes archives, writes `checksums.txt`, and opens/updates the Homebrew tap formula.

## Verify artifacts

Download the archive for your platform and verify it against `checksums.txt`:

```sh
sha256sum -c checksums.txt --ignore-missing
```

Smoke test the binary:

```sh
smsc --version
smsc --dry-run
smsc doctor
```

## Homebrew smoke test

```sh
brew tap xavier-castro/smsc
brew install smsc
smsc --version
brew uninstall smsc
```
