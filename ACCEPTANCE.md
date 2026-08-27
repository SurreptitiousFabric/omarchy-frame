# Release acceptance

This matrix is the release record for the Samsung Frame plugin. A row is
**PASS** only when the cited automated or live evidence exercised that behavior.
Unsupported hardware behavior is **N/A**; an applicable unverified behavior is
**PENDING**. The release remains pre-1.0 while any applicable row is pending,
except that the tagged-workflow row may be the sole pending row while that
workflow is running because the run itself supplies its evidence.

## Candidate

- Code and packaging candidate: public commit `ebd38c8`
- Installed public checkout: commit `1d5d949` cloned from public GitHub; only
  this acceptance record and marketplace evidence differ from the code candidate
- Plugin: `io.github.surreptitiousfabric.omarchy-frame` 0.6.0 (pre-1.0)
- TV: Samsung `QE55LS03BAUXXH` / LS03B, firmware 1720.7
- Host: Omarchy on ARM64
- Date: 2026-08-27

No private address, MAC address, pairing token, content identifier, personal
image, or raw TV response belongs in this file.

## Automated and packaging gates

| Check | Status | Evidence |
|---|---|---|
| Go formatting, race detector, tests, vet | PASS | Local release-candidate run |
| Backend statement coverage at least 75% | PASS | 76.3% backend / 76.4% repository total on `ebd38c8` |
| Explicit remote click/hold allowlists and QML parity | PASS | Automated contract tests |
| QML parses and loads in Omarchy shell | PASS | `ebd38c8` passed native type contracts, fail-closed mutations, complete-widget creation, visible bar rendering, panel open/close, fresh-process journal inspection, and the required Arch CI job |
| Omarchy design-system contract | PASS | Native panel/control primitives, inherited bar font, scaled geometry, shared controls, native grouped tabs, negative policy tests, and runtime tab-state test |
| Static ARM64 and AMD64 release binaries | PASS | Reproducible builds and `file` inspection |
| Deterministic portable release archive | PASS | Mutation-safe packaging test proves identical repeated archives, normalized metadata, and an exact launcher/binaries/checksum allowlist |
| Relative checksums verify | PASS | `scripts/validate-repository.sh` |
| Launcher works with Go absent | PASS | `scripts/test-packaged-runtime.sh` on ARM64 from both the candidate and the installed public checkout |
| No symlink, world-writable file, unexpected executable, or secret finding | PASS | Repository validator and release audit |
| Official Go vulnerability analysis | PASS | `govulncheck` v1.7.0 reported no vulnerabilities for `ebd38c8` locally, in a clean clone, and in CI |
| Cross-process state safety and bounds | PASS | Race tests cover advisory locking, stale pairing rejection, unsafe lock/state files, 64 KiB config cap, and 4096-byte token cap |
| Clean Git checkout validates and runs packaged backend | PASS | Separate untrusted `--no-local` clone of `ebd38c8`; race, vet, vulnerability, type, QML policy, full-widget runtime, manifest, repository policy, checksum, and no-Go gates passed and left it clean |
| Permanent marketplace identity and listing metadata | PASS | Namespaced ID, author, module path, real repository URL, allowed category/tags, and optional-preview policy are validator-backed |
| Marketplace static security baseline | PASS | Baseline v4 on public `ebd38c8`: zero findings, non-blocking `review-required`, expected `bundled-executable-binary` capability only |
| CI workflow executes remotely | PASS | GitHub Actions run `33046553159` passed both required `qml-contract` and `verify` jobs on `ebd38c8` |
| Tagged release workflow executes remotely | PENDING | Requires an owner-approved release tag after the remaining live gates pass |
| Release guard rejects premature or incoherent tags | PASS | Mutation tests require stable v1+ tag/manifest/acceptance/changelog agreement, a clean tree, candidate ancestry, evidence-only post-candidate changes, and no applicable pending row except the workflow being exercised |

## Install and shell integration

| Check | Status | Evidence |
|---|---|---|
| Public GitHub install through `omarchy plugin add` | PASS | Removed the local-source clone and installed public commit `1d5d949` from the documented GitHub URL; a transient Omarchy discovery race required one rescan/retry |
| Enable in right bar section | PASS | Full right-bar order was unchanged after reinstall; the Frame remained between the tray and agents widgets |
| Panel opens without Frame-specific QML/runtime error | PASS | Installed public checkout passed complete-widget creation; live non-activating IPC summon/hide returned successfully with zero Frame errors in the resulting shell-journal window |
| Remove keeps unmanaged copy recoverable | PASS | Omarchy created a timestamped backup during migration |
| Reinstall and beta-ID migration preserve owner-only TV state | PASS | State stayed a real `0700` directory and `0600` file; removal left it byte-identical, and the public reinstall retained working authorization with sanitized online status |
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

Do not tag `v1.0.0` or submit to the marketplace until every applicable
**PENDING** row other than the tagged-workflow row is passed or the owner
explicitly removes that capability from the product scope. The release workflow
must then be the sole pending row and becomes PASS only after that exact tag run
succeeds. If a preview is added later, marketplace submission also requires
approval of that exact image.
