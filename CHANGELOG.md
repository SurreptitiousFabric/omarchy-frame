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

### Changed

- Split the panel into focused Remote, Art, Photos, Setup, gallery-card, and TV
  icon QML components.
- Expand backend coverage beyond 75% with race-tested in-memory protocol and
  local transfer integration tests.
- Pin release builds to mise-managed Go 1.27.0 and add CI/release automation.
