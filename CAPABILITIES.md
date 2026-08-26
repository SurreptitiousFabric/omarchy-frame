# Capability reference

Samsung does not publish the consumer-TV remote WebSocket and Frame Art
protocols as a stable public API. Capabilities are therefore split into:

- Stable on LS03B: discovery metadata, token pairing, remote key clicks,
  press/release, power off, Wake-on-LAN, and Multi View hold rotation.
- Firmware-dependent: direct HDMI keys.
- Feature-detected: artwork listing, thumbnails, current selection, and
  selection through `com.samsung.art-app`. The TV omits title/artist metadata.

The UI exposes errors instead of claiming success when an optional endpoint is
unavailable. Destructive TV administration and service-menu keys are excluded.
