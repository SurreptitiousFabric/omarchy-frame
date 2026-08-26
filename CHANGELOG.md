# Changelog

All notable user-visible changes are recorded here. The project follows
Semantic Versioning once the first public `v1.0.0` release is tagged.

## Unreleased

### Security

- Replace pattern-only Samsung key validation with explicit click and hold
  allowlists; service, factory, reset, unknown, and unsupported hold operations
  are rejected before connecting to the TV.
- Correlate Art replies to generated request IDs and harden WebSocket framing,
  short writes, transfer bounds, and held-key release behavior.
- Bound thumbnail batches and cache storage to 64 MB and prune only
  plugin-generated hashed files.

### Fixed

- Distinguish an unreachable TV from a confirmed TV/Art state.
- Make the header Off action perform a deliberate power hold and label the
  short action as TV / Art.
- Allow discovery/configuration to recover from malformed state and make
  concurrent config writes collision-safe and durable.
- Report upload success separately from follow-up artwork selection.
- Generate portable release checksums with relative filenames.
- Make artwork and photo cards keyboard-focusable, expose accessible button
  semantics, and support Enter, Return, and Space activation.
- Use Omarchy's native panel lifecycle and controller so bar clicks, shell IPC,
  multi-monitor routing, focus, and popup switching share one state model;
  per-instance IPC is disabled so duplicate monitor copies cannot race.

### Changed

- Split the panel into focused Remote, Art, Photos, Setup, gallery-card, and TV
  icon QML components.
- Draw the plugin identity as a theme-aware retro antenna television so it is
  visually distinct from Omarchy's generic display widget.
- Expand backend coverage beyond 75% with race-tested in-memory protocol and
  local transfer integration tests.
- Pin release builds to mise-managed Go 1.27.0 and add CI/release automation.
- Add an offline capability query and a packaged-runtime gate that executes the
  native static launcher with Go absent from its environment.
