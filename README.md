# Omarchy Samsung Frame

A keyboard-first Omarchy Quickshell plugin for local control of Samsung The
Frame TVs. It is developed against the European 2022 55-inch Frame
`QE55LS03BAUXXH` (`LS03B`) and Samsung's Auto Rotating Stand.

## Features

- SSDP discovery and manual IP setup
- One-time on-TV approval with a locally stored Samsung pairing token
- Local artwork gallery with thumbnails, current-selection marker, and
  click-to-select control when the Frame Art service responds
- Owner-selected JPEG/PNG upload to Samsung **My Photos**, confirmed deletion
  of My Photos, plus built-in sequential and shuffled slideshow controls
- Deliberate three-second power-off hold and Wake-on-LAN
- Live header status distinguishes TV, Art Mode, unreachable, and unavailable
  mode data
- Navigation, volume, mute, channel, media, menu, source, and other explicitly
  allowed Samsung remote keys
- Task-focused Remote, Art, Photos, and Setup tabs; Remote adds Navigate,
  Sound, Media, and More subtabs so only one compact control set is visible
- Font-independent QML-drawn TV identity; no private-use icon font is required
- Clearly labelled TV/Art toggle plus feature-detected Art WebSocket requests
- Portrait/landscape rotation using the LS03B `KEY_MULTI_VIEW` three-second hold
- A built-in capability reference and visible errors for firmware gaps

The TV and computer must share a trusted local network. Nothing is sent to a
cloud service.

## Install

This repository must include release binaries before installation. Then run:

```bash
omarchy plugin add https://github.com/OWNER/omarchy-frame.git --enable
```

For local development, build and commit the working tree, then use Omarchy's
normal Git-backed installer with an absolute local URL:

```bash
omarchy plugin add file:///absolute/path/to/omarchy-frame --enable
```

An existing plugin with the same ID must be removed or updated first. Omarchy
backs up an unmanaged plugin folder when it is removed.

On the first command, approve **Omarchy Frame** on the television. The pairing
token is saved as `~/.local/state/omarchy-frame/config.json` with mode `0600`.

## Rotating stand

Pair the stand to an LS03B television by holding **Settings/Number/Color** and
**Multi View** together on the Samsung remote for at least three seconds. The
tested firmware 1720.7 rejects that pairing chord from network remotes, so a
compatible physical Samsung Smart Remote is required for initial pairing. The
plugin's Rotate button can then emulate a three-second Multi View hold.

## Build

Go is required only to build release binaries, not to install or use the
plugin. On Omarchy, install it with:

```bash
omarchy install dev-env go
scripts/build-release.sh
```

The backend uses only Go's standard library. Discovery optionally calls
`avahi-browse`, which is part of a standard Omarchy installation, when Samsung
SSDP is unavailable. Release builds are static Linux AMD64 and ARM64
executables; the tracked POSIX launcher selects the matching binary at runtime,
so users do not need Go. Review the source and reproduce the hashes in
`bin/SHA256SUMS` before publishing.

## Remove

```bash
omarchy plugin remove swa.frame
```

The uninstall command removes the plugin checkout but deliberately preserves
the paired-TV settings. To forget the TV, delete only:

```text
~/.local/state/omarchy-frame/
```

Also remove **Omarchy Frame** from the television's authorized-device list if
you want to revoke its token.

## Security and limitations

Omarchy plugins run unsandboxed. This plugin spawns only its bundled backend.
The QML UI launches the installed Zenity file chooser as a separate process so
a desktop-portal fault cannot crash the shell. The backend accepts fixed
commands, validates IP addresses, enforces explicit click and hold key
allowlists, bounds hold durations, verifies image signatures and sizes, limits
network responses and caches, and never opens a listening port. Upload and
thumbnail transfer endpoints are pinned to the configured TV. Samsung uses a
self-signed certificate on the local secure TV WebSocket, so certificate-chain
verification is unavailable; the connection is restricted to the configured
numeric IP and Samsung port 8002.

Samsung's undocumented Art protocol and application REST endpoint vary across
firmware. Ordinary remote control remains usable when those endpoints are
absent; the header reports **ART / TV?** instead of guessing when the optional
Art-status request is unavailable or ambiguous. Tested LS03B firmware can
return `off` while visibly displaying artwork. Rotation toggles orientation
because the LS03B API does not report a reliable physical orientation state.

## Marketplace

The repository layout follows the Omarchy marketplace contract: one plugin,
root manifest, README, license, no symlinks or install hooks, documented
dependencies/removal, and an optional root `preview.png`. Validate with:

```bash
omarchy plugin validate .
```

License: MIT.

See [USER_MANUAL.md](USER_MANUAL.md) for operation and troubleshooting,
[THREAT_MODEL.md](THREAT_MODEL.md) for the security boundary, and
[DEVELOPMENT.md](DEVELOPMENT.md) for testing and release procedures.
