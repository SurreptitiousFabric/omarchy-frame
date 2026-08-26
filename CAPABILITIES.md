# Capability reference

Samsung does not publish the consumer-TV remote WebSocket and Frame Art
protocols as a stable public API. Capabilities are therefore split into:

- Stable on LS03B: discovery metadata, token pairing, remote key clicks,
  press/release, power off, Wake-on-LAN, and Multi View hold rotation.
- Firmware-dependent: applications REST listing/launching and direct HDMI keys.
  Firmware 1720.7 on the tested QE55LS03BAUXXH does not answer installed-app
  enumeration, so the plugin reports it as unsupported instead of timing out.
- Feature-detected/experimental: Art status, current artwork, categories, and
  slideshow status through `com.samsung.art-app`.

The UI exposes errors instead of claiming success when an optional endpoint is
unavailable. Destructive TV administration and service-menu keys are excluded.
