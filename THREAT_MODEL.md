# Threat model

## Protected assets

- Samsung pairing token
- TV availability and control authority
- User filesystem and long-lived Omarchy shell process
- User-selected local photos
- Private LAN topology and device metadata

## Trust boundaries

The QML plugin runs unsandboxed as the user. Its bundled backend is trusted
code. The LAN and TV responses are untrusted input. Samsung's port 8002 uses a
self-signed certificate, so confidentiality is opportunistic and the plugin
cannot authenticate a public CA chain.

## Controls

- Only numeric private/link-local TV addresses are accepted.
- HTTP redirects are refused; responses and WebSocket frames are bounded.
- Remote clicks and holds use separate explicit allowlists. Holds are limited to
  Power and Multi View; service, factory, reset, and unknown keys are rejected
  before a TV connection is opened.
- Pairing state is atomically written in an owner-only directory/file.
- Public JSON never serializes the token; regression tests enforce this.
- Public error text redacts LAN endpoints and token query parameters.
- TLS requires 1.2+, despite unavoidable certificate verification bypass.
- Discovery accepts only local IPv4 Samsung records and re-verifies TV metadata.
- Thumbnail transfer endpoints must resolve to the configured TV address;
  ports, headers, IDs, file sizes, and item counts are validated. One thumbnail
  batch and the generated cache are each bounded to 64 MB; cache pruning ignores
  unrelated filenames and preserves the current gallery batch.
- Artwork thumbnails are cached under owner-only local state with hashed names
  and owner-only file permissions.
- Uploads accept only an explicitly selected absolute local file URL/path, open
  it without invoking a shell, require a regular JPEG/PNG by signature, and cap
  it at 20 MB. The TV-provided upload endpoint is pinned to the configured TV.
- The image picker executes the fixed `/usr/bin/zenity` path with a fixed
  argument vector and reads only its selected-path output; no shell is invoked.
- Deletion accepts one validated content ID, re-reads the TV's My Photos list,
  and refuses IDs outside that category. The UI requires a second confirmation.
- Art replies are correlated to generated request IDs, so unsolicited mode
  broadcasts cannot be mistaken for a query result.
- WebSocket messages are size-bounded, handle fragmented text and short writes,
  and reject masked server frames, unsupported opcodes, and malformed controls.
- Held keys retry Release before the connection closes if the first release
  write fails.
- No listener, cloud service, shell interpolation, privileged command, install
  hook, telemetry, or automatic deletion is used.

## Residual risks

A hostile device on the same LAN may impersonate a TV during first pairing or
intercept self-signed TLS traffic. A compromised plugin can access everything
the user can access because Omarchy plugins are unsandboxed. Samsung firmware
may alter undocumented commands. Users should run the plugin only on a trusted
LAN, review updates, approve the expected on-TV client name, and revoke the TV
authorization after suspected compromise.
