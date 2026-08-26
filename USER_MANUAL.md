# Samsung Frame Controller — User Manual

## Before you start

The computer and TV must be on the same trusted LAN. On first use the TV shows
an approval dialog for **Omarchy Frame**; choose Allow. The plugin does not use
Samsung Cloud or require a Samsung account.

Open the TV icon in the Omarchy bar. The header shows **TV**, **ART**, **OFF**,
or **ART / TV?**. The last label means the TV is reachable but its firmware did
not provide a trustworthy Art Mode answer; it does not mean the TV is off.
Middle-click the bar icon or press the refresh button to recheck. The status is
refreshed every 15 seconds while the panel is open by default.

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

- **Remote:** choose Navigate, Sound, Media, or More. Only that task's controls
  are shown. Navigate opens by default whenever the panel is reopened, and
  short pages shrink so the panel does not leave a large empty area.
- **Art:** enter Art Mode, browse artwork previews, and select artwork. Entries
  without a usable TV-provided preview are omitted.
- **Photos:** upload a local JPEG/PNG into **My Photos**, select personal
  photos, control the My Photos slideshow, or delete a photo after a separate
  confirmation. Samsung/store artwork cannot be deleted by the plugin.
  The upload chooser runs separately from the bar so a desktop file-dialog
  failure cannot terminate the Omarchy shell.
- **Setup:** discovery, manual address, approval, and stand instructions.
- **Technical capabilities:** expandable under Setup.

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
| Art request fails | Firmware changed/disabled Art socket | Ordinary remote control remains available |
| Header says ART / TV? | TV is online but Art status was unavailable or ambiguous | Remote controls remain usable; LS03B firmware can report `off` while displaying art |
| Upload fails | Unsupported image, TV storage full, or Art service declined transfer | Try a JPEG/PNG under 20 MB; ensure Art Mode has completed its first-use setup |
| Uploaded photo is missing | Gallery still reflects the previous TV list | Wait for the automatic refresh or press Refresh gallery |
| Rotate says sent but does not move | Stand not paired | Follow the physical-remote pairing steps above |
| “Not available” during pairing | Network remote chord rejected | A compatible physical Smart Remote is required |

## Remove and revoke

Run `omarchy plugin remove swa.frame`. To forget local state, remove only
`~/.local/state/omarchy-frame/`. Also revoke **Omarchy Frame** in the TV's
authorized-device settings. Removing the plugin does not deliberately delete
credentials without asking.
