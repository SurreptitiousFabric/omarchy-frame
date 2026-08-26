# Release acceptance

This matrix is the release record for the Samsung Frame plugin. A row is
**PASS** only when the cited automated or live evidence exercised that behavior.
Unsupported hardware behavior is **N/A**; an applicable unverified behavior is
**PENDING**. The release remains pre-1.0 while any applicable row is pending.

## Candidate

- Code and packaging candidate: local commit `24dcc2b`
- Plugin: `swa.frame` 0.5.1 (pre-1.0)
- TV: Samsung `QE55LS03BAUXXH` / LS03B, firmware 1720.7
- Host: Omarchy on ARM64
- Date: 2026-08-26

No private address, MAC address, pairing token, content identifier, personal
image, or raw TV response belongs in this file.

## Automated and packaging gates

| Check | Status | Evidence |
|---|---|---|
| Go formatting, race detector, tests, vet | PASS | Local release-candidate run |
| Backend statement coverage at least 75% | PASS | 75.7% backend / 75.9% repository total on `5b3eb26` |
| Explicit remote click/hold allowlists and QML parity | PASS | Automated contract tests |
| QML parses and loads in Omarchy shell | PASS | Standalone lint plus live shell load |
| Static ARM64 and AMD64 release binaries | PASS | Reproducible builds and `file` inspection |
| Relative checksums verify | PASS | `scripts/validate-repository.sh` |
| Launcher works with Go absent | PASS | `scripts/test-packaged-runtime.sh` on ARM64 |
| No symlink, world-writable file, unexpected executable, or secret finding | PASS | Repository validator and release audit |
| Clean Git checkout validates and runs packaged backend | PASS | Separate `--no-local` clone of `5b3eb26`; validator and race suite passed |
| CI and tagged release workflows execute remotely | PENDING | Requires an owner-approved remote repository |

## Install and shell integration

| Check | Status | Evidence |
|---|---|---|
| Git-backed install through `omarchy plugin add` | PASS | Local absolute `file://` install |
| Enable in right bar section | PASS | Live shell layout and plugin catalog |
| Panel opens without Frame-specific QML/runtime error | PASS | Live shell IPC smoke test |
| Remove keeps unmanaged copy recoverable | PASS | Omarchy created a timestamped backup during migration |
| Reinstall preserves owner-only TV state | PASS | Existing authorization remained usable after reinstall |
| Keyboard-only traversal, visible focus, and dismissal | PASS | Live native key-catcher test showed themed focus, focused-monitor routing, and Escape dismissal after focus moved into a control |
| Optional marketplace preview approved for publication | PENDING | No image has been approved for a public listing |

## Live LS03B behavior

| Check | Status | Evidence |
|---|---|---|
| Discover expected television | PASS | Initial live setup |
| One-time on-TV authorization and `0600` token state | PASS | Initial live setup and permission check |
| Sanitized status contains no token or private endpoint | PASS | Live status and redaction regression tests |
| Ordinary remote and power/status control | PASS | Initial live acceptance |
| Wake-on-LAN after full power-off | PENDING | Requires an observer near the television |
| Confirmed Art Mode status | PASS | Repeated live Art `on` response |
| Ambiguous LS03B Art reply does not claim TV mode | PASS | Owner-visible Art Mode plus `off` response regression |
| Gallery, thumbnails, and current marker | PASS | Live Art and My Photos reads |
| Select artwork | PASS | Live re-selection of current artwork |
| Upload owner-selected JPEG/PNG without shell crash | PASS | Owner-selected upload displayed on the television after chooser isolation |
| Uploaded item appears in refreshed My Photos | PASS | Owner-visible gallery refresh during upload flow |
| Delete a disposable My Photos image and verify refresh | PENDING | Destructive content test requires explicit owner approval |
| Start/stop sequential and shuffled My Photos slideshow | PENDING | Mutating content setting requires explicit owner approval |
| Ordinary remote remains usable when Art service fails | PASS | Fallback tests and live firmware behavior |

## Auto Rotating Stand

| Check | Status | Evidence |
|---|---|---|
| Network command sends bounded three-second Multi View hold | PASS | Automated release/retry tests and live TV acceptance |
| Network-originated initial pairing chord | N/A | Firmware displays “Not available”; feature was removed |
| Stand paired once with compatible physical Smart Remote | PENDING | Physical remote is not currently available |
| Rotate once in each direction with cable clearance confirmed | PENDING | Depends on completed physical pairing and nearby observer |
| UI avoids claiming a known orientation | PASS | Control is labelled as a toggle; API exposes no reliable orientation |

## Release decision

Do not tag `v1.0.0`, publish a repository, submit to the marketplace, or mark
this matrix complete until every applicable **PENDING** row is passed or the
owner explicitly removes that capability from the product scope. Publication
also requires replacing placeholder repository links and approving the exact
preview image, if one is included.
