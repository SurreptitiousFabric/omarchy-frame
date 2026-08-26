# Samsung Frame Controller — User Manual

## Before you start

The computer and TV must be on the same trusted LAN. On first use the TV shows
an approval dialog for **Omarchy Frame**; choose Allow. The plugin does not use
Samsung Cloud or require a Samsung account.

Open the TV icon in the Omarchy bar. The header shows local online/offline
state. Middle-click the bar icon or press the refresh button to recheck.

## Setup

1. Open **Setup** and choose **Discover Frame TVs**.
2. If one Frame is found it is selected automatically. Otherwise select the TV
   or enter its numeric private IP address.
3. Send a harmless command such as Info and approve the on-TV prompt.
4. Reserve the TV's address in the router if it changes frequently.

The approval token is stored at
`~/.local/state/omarchy-frame/config.json` with owner-only permissions. Treat
it as a credential.

## Pages

- **Remote:** navigation, Home/Back, volume, mute, channels, guide, playback,
  source, info/tools/menu, and rotation.
- **Apps:** asks the TV for installed apps. Some firmware, including tested
  LS03B firmware 1720.7, does not expose the list.
- **Art:** enter Art Mode and inspect Art status, current work, categories, and
  slideshow capability. Samsung changes this undocumented protocol by firmware.
- **Setup:** discovery, manual address, approval, and stand instructions.
- **API:** plain-language capability groups.

## Auto Rotating Stand

Initial Bluetooth pairing cannot be performed by the network remote on tested
firmware 1720.7. Use a compatible physical Samsung Smart Remote once:

1. Power the stand and TV.
2. If needed, hold the stand's rear Reset button for three seconds.
3. Hold **Settings/Number/Color + Multi View** together for at least three
   seconds.
4. Confirm physical rotation using a long Multi View press.
5. Thereafter use **Rotate portrait / landscape** in this plugin.

Rotation is a toggle because the consumer API does not reliably expose the
stand's physical orientation. Keep cables and objects clear before rotating.

## Troubleshooting

| Symptom | Meaning | Action |
|---|---|---|
| No TV discovered | SSDP/mDNS blocked or TV on another LAN | Enter its private IP; check Wi-Fi isolation |
| Approval never appears | Existing authorization or network block | Remove old authorized device on TV, then retry |
| TV offline | TV asleep or address changed | Use Wake; rediscover; reserve its address |
| Wake fails | MAC was never learned or broadcast blocked | Discover once while on; check router Wi-Fi broadcast policy |
| Apps unavailable | Firmware omits enumeration API | Use Source/Home navigation; this is expected on 1720.7 |
| Art request fails | Firmware changed/disabled Art socket | Ordinary remote control remains available |
| Rotate says sent but does not move | Stand not paired | Follow the physical-remote pairing steps above |
| “Not available” during pairing | Network remote chord rejected | A compatible physical Smart Remote is required |

## Remove and revoke

Run `omarchy plugin remove swa.frame`. To forget local state, remove only
`~/.local/state/omarchy-frame/`. Also revoke **Omarchy Frame** in the TV's
authorized-device settings. Removing the plugin does not deliberately delete
credentials without asking.
