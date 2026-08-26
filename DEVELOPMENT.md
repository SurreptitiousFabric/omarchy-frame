# Development and release guide

## Design

Quickshell owns presentation and launches a short-lived Go command for each
operation. JSON is the only process interface. The backend opens no listening
port and uses only the Go standard library; Avahi is an optional Omarchy-provided
discovery fallback.

## Build and test

```bash
omarchy install dev-env go
go fmt ./...
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go vet ./...
scripts/build-release.sh
omarchy plugin validate .
```

Test protocol behavior with in-memory sockets or loopback test servers. Live-TV
tests must be non-destructive unless the operator explicitly approves the
action. Never put addresses, MACs, tokens, artwork identifiers, or raw TV dumps
in fixtures, logs, screenshots, issues, or commits.

The loopback WebSocket tests require permission to open a local listening
socket. Restricted sandboxes may need to run `go test` with that capability;
the tests never listen beyond `127.0.0.1`.

## Release checklist

- Working tree and `git diff --check` clean
- Tests, race detector, vet, manifest validation, and QML runtime load pass
- Coverage reviewed by function; security/protocol paths cannot regress silently
- ARM64 and AMD64 binaries rebuilt with CGO disabled and `-trimpath`
- Tracked launcher selects the native release binary on both supported architectures
- `SHA256SUMS` reproduced and verified
- No symlinks, unexpected executables, world-writable files, or secrets
- README/manual/capability limitations match the tested firmware
- Root license and optional user-approved `preview.png` present
- Install and removal tested from a clean checkout
- Publication and marketplace submission performed only with owner approval

## Versioning

Update `manifest.json` and documentation together. Protocol behavior changes
require tests. Security fixes require a clear changelog entry once releases are
public. Do not silently broaden accepted network targets or commands.
