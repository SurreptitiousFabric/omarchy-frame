#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

if grep -R -n -E 'font(Family|\.family):[[:space:]]*"(sans-serif|serif|monospace)"' \
  BarWidget.qml components --include='*.qml'; then
  echo "test-ui-contract: QML must inherit the Omarchy font family" >&2
  exit 1
fi

layout_files=(BarWidget.qml components/ArtPage.qml components/FrameButton.qml components/FrameTabGroup.qml components/GalleryCard.qml components/PhotosPage.qml components/RemotePage.qml components/SetupPage.qml)
if grep -n -E '^[[:space:]]*(width|height|implicitWidth|implicitHeight|spacing|rowSpacing|columnSpacing|size|radius|anchors\.margins):[[:space:]]*[1-9][0-9]*(\.[0-9]+)?[[:space:]]*$' "${layout_files[@]}"; then
  echo "test-ui-contract: fixed QML geometry must use Omarchy Style spacing" >&2
  exit 1
fi

if grep -R -n -E 'component[[:space:]]+(SoftButton|QuietButton):' components --include='*.qml'; then
  echo "test-ui-contract: page-local button styles must use FrameButton" >&2
  exit 1
fi

echo "Omarchy UI contract is satisfied"
