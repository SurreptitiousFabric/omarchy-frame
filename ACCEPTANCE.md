# Release acceptance

This matrix is the release record for the Samsung Frame plugin. A row is
**PASS** only when the cited automated or live evidence exercised that behavior.
Unsupported hardware behavior is **N/A**; an applicable unverified behavior is
**PENDING**. The release remains pre-1.0 while any applicable row is pending.

## Candidate

- Code and packaging candidate: local commit `55e95e7`
- Plugin: `io.github.surreptitiousfabric.omarchy-frame` 0.6.0 (pre-1.0)
- TV: Samsung `QE55LS03BAUXXH` / LS03B, firmware 1720.7
- Host: Omarchy on ARM64
- Date: 2026-08-26

No private address, MAC address, pairing token, content identifier, personal
image, or raw TV response belongs in this file.

## Automated and packaging gates

| Check | Status | Evidence |
|---|---|---|
| Go formatting, race detector, tests, vet | PASS | Local release-candidate run |
| Backend statement coverage at least 75% | PASS | 76.3% backend / 76.4% repository total on `55e95e7` |
| Explicit remote click/hold allowlists and QML parity | PASS | Automated contract tests |
| QML parses and loads in Omarchy shell | PENDING | Prior tab-only harness missed a complete-entry-point type error; requires corrected exact-candidate type, full-widget, and live-shell evidence |
| Omarchy design-system contract | PASS | Native panel/control primitives, inherited bar font, scaled geometry, shared controls, native grouped tabs, negative policy tests, and runtime tab-state test |
| Static ARM64 and AMD64 release binaries | PASS | Reproducible builds and `file` inspection |
| Relative checksums verify | PASS | `scripts/validate-repository.sh` |
| Launcher works with Go absent | PASS | `scripts/test-packaged-runtime.sh` on ARM64 |
| No symlink, world-writable file, unexpected executable, or secret finding | PASS | Repository validator and release audit |
| Official Go vulnerability analysis | PASS | `govulncheck` v1.7.0 reported no vulnerabilities for `55e95e7` |
| Cross-process state safety and bounds | PASS | Race tests cover advisory locking, stale pairing rejection, unsafe lock/state files, 64 KiB config cap, and 4096-byte token cap |
| Clean Git checkout validates and runs packaged backend | PENDING | Prior `55e95e7` clone used the incomplete tab-only QML harness; requires a corrected clean-clone rerun |
| Permanent marketplace identity and listing metadata | PASS | Namespaced ID, author, module path, real repository URL, allowed category/tags, and optional-preview policy are validator-backed |
| Marketplace static security baseline | PASS | Current engine on exact `55e95e7`: zero findings, non-blocking `review-required`, expected bundled-binary capability only |
| CI workflow executes remotely | PASS | GitHub Actions run `33003748420` passed on public `main` at `de6903b` |
| Tagged release workflow executes remotely | PENDING | Requires an owner-approved release tag after the remaining live gates pass |

## Install and shell integration

| Check | Status | Evidence |
|---|---|---|
| Git-backed install through `omarchy plugin add` | PASS | Exact `55e95e7` local `file://` install/update under the permanent ID |
| Enable in right bar section | PASS | Live shell layout and plugin catalog |
| Panel opens without Frame-specific QML/runtime error | PENDING | Installed `55e95e7` is rejected by Quickshell; requires corrected exact-candidate live reload and journal inspection |
| Remove keeps unmanaged copy recoverable | PASS | Omarchy created a timestamped backup during migration |
| Beta-ID migration preserves owner-only TV state | PASS | State file remained byte-identical and existing authorization returned sanitized online/Art status |
| Keyboard-only traversal, visible focus, and dismissal | PASS | Live native key-catcher focus/dismissal acceptance plus exact-candidate windowless execution of selected-tab landing, bounded movement, and activation |
| Optional marketplace preview | N/A | Intentionally omitted; the marketplace permits repositories without one |

## Live LS03B behavior

| Check | Status | Evidence |
|---|---|---|
| Discover expected television | PASS | Initial live setup |
| One-time on-TV authorization and `0600` token state | PASS | Initial live setup and permission check |
| Sanitized status contains no token or private endpoint | PASS | Exact-candidate raw offline JSON allowlist, stored IP/MAC/token non-occurrence, unchanged state hash, and IPv4/IPv6/token redaction tests |
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

Do not tag `v1.0.0`, submit to the marketplace, or mark this matrix complete
until every applicable **PENDING** row is passed or the owner explicitly removes
that capability from the product scope. If a preview is added later,
marketplace submission also requires approval of that exact image.
