#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

if command -v omarchy >/dev/null 2>&1; then
  omarchy plugin validate .
else
  jq -e '
    .schemaVersion == 1 and
    .id == "swa.frame" and
    (.version | test("^[0-9]+\\.[0-9]+\\.[0-9]+$")) and
    (.kinds | index("service")) and
    (.kinds | index("bar-widget")) and
    .entryPoints.service == "Service.qml" and
    .entryPoints.barWidget == "BarWidget.qml"
  ' manifest.json >/dev/null
fi

if find . -path ./.git -prune -o -type l -print -quit | grep -q .; then
  echo "validate-repository: symlinks are not allowed" >&2
  exit 1
fi

if find . -path ./.git -prune -o -type f -perm -0002 -print -quit | grep -q .; then
  echo "validate-repository: world-writable files are not allowed" >&2
  exit 1
fi

sh -n bin/frame-controller
bash -n scripts/build-release.sh scripts/validate-repository.sh

if awk 'NF >= 2 && $2 ~ /^\//' bin/SHA256SUMS | grep -q .; then
  echo "validate-repository: SHA256SUMS contains absolute paths" >&2
  exit 1
fi

(
  cd bin
  sha256sum -c SHA256SUMS
)

for required in README.md USER_MANUAL.md CAPABILITIES.md THREAT_MODEL.md SECURITY.md DEVELOPMENT.md LICENSE; do
  test -f "$required" || {
    echo "validate-repository: missing $required" >&2
    exit 1
  }
done
