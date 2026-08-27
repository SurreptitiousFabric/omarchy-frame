# Development and release guide

## Design

Quickshell owns presentation and launches a short-lived Go command for each
operation. JSON is the only process interface. The backend opens no listening
port and uses only the Go standard library; Avahi is an optional Omarchy-provided
discovery fallback.

`BarWidget.qml` composes the panel. Focused tab and card/icon implementations
live under `components/`; `Service.qml` is the only UI/backend boundary. The UI
inherits `bar.fontFamily`, uses `Style.space()` for fixed geometry, and composes
controls from `qs.Ui`; `scripts/test-ui-contract.sh` enforces those Omarchy
design-system boundaries.

## Build and test

```bash
omarchy install dev-env go
go fmt ./...
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
scripts/build-release.sh
git diff --exit-code -- bin
scripts/test-release-package.sh
scripts/test-packaged-runtime.sh
scripts/test-qml-policy.sh
scripts/test-qml-types.sh
scripts/test-qml-runtime.sh
scripts/test-repository-policy.sh
scripts/test-release-readiness.sh
scripts/test-ui-contract.sh
omarchy plugin validate .
```

The QML runtime script exercises both keyboard behavior in the real tab
component and creation of the complete `BarWidget.qml` entry point. The widget
harness supplies inert bar and service doubles, leaves the panel closed, and
cannot contact a television. It catches entry-point compilation, local
component contracts, object creation, and invalid bar geometry in the actual
installed Omarchy runtime.

The QML type script maps Omarchy's `qs.Commons` and `qs.Ui` directories into a
standard Qt import tree and rejects unknown local-component properties or
missing required properties. GitHub Actions runs it in an Arch container with
official Quickshell and Omarchy QML APIs pinned to commit
`dec29fa90afc3d16a7e0c487c1869c7e512282ca`. Repository-policy mutations prove
both contract failures are rejected by the companion QML policy script.

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
- Official Go vulnerability analysis reports no reachable vulnerability
- Coverage reviewed by function; security/protocol paths cannot regress silently
- Backend statement coverage remains at least 75%; hardware-only wrappers are
  exercised by the live acceptance matrix
- ARM64 and AMD64 binaries rebuilt with CGO disabled and `-trimpath`
- Tracked launcher selects the native release binary on both supported architectures
- `SHA256SUMS` covers the launcher and both binaries and verifies successfully
- Rebuilding leaves every tracked launcher, binary, and checksum byte-identical
- The compressed release bundle has an allowlisted payload and reproducible
  ordering, release-asset-derived timestamp, ownership, and gzip metadata, so
  evidence-only commits do not alter its bytes
- No symlinks, unexpected executables, world-writable files, or secrets
- Every remote GitHub Action uses an immutable full 40-character commit pin
- README/manual/capability limitations match the tested firmware
- Root license present; include `preview.png` only after owner approval of the
  exact image
- Install and removal tested from a clean checkout
- `ACCEPTANCE.md` matches the exact release candidate and has no unexplained
  pending applicable check
- `scripts/check-release-readiness.sh v<version>` accepts the clean tag tree;
  it permits only the tagged-workflow row to remain pending because that row is
  proven by the workflow currently running
- Publication and marketplace submission performed only with owner approval
- Current marketplace baseline result and any review capabilities are recorded
  against the exact candidate commit

## Versioning

Update `manifest.json`, `ACCEPTANCE.md`, and the dated `CHANGELOG.md` release
section together. Tagged releases must use stable `vMAJOR.MINOR.PATCH` versions
at or above 1.0.0. The release workflow fails closed on version drift, a dirty
tree, unavailable or non-ancestor candidates, post-candidate runtime/packaging
changes, and every applicable pending acceptance row other than its own
self-referential tagged-workflow check. Protocol behavior changes require tests.
Security fixes require a clear changelog entry once releases are public. Do not
silently broaden accepted network targets or commands.
