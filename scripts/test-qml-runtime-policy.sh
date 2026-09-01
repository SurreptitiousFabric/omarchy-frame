#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

# Prove the runtime is present and accepts the valid tree before interpreting
# rejection of a mutation as policy coverage.
(cd "$root" && FRAME_REQUIRE_QUICKSHELL=1 ./scripts/test-qml-runtime.sh >/dev/null)

work=$(mktemp -d "${TMPDIR:-/tmp}/omarchy-frame-qml-runtime-policy.XXXXXX")
case $work in
  "${TMPDIR:-/tmp}"/omarchy-frame-qml-runtime-policy.*) ;;
  *)
    echo "test-qml-runtime-policy: unsafe temporary path" >&2
    exit 1
    ;;
esac
trap 'rm -rf -- "$work"' EXIT

fixture=$work/hardcoded-ui-font
mkdir -p "$fixture"
cp -a "$root/." "$fixture/"
sed -i 's/bar ? bar.fontFamily : Style.font.family/"sans-serif"/' "$fixture/BarWidget.qml"

if (cd "$fixture" && FRAME_REQUIRE_QUICKSHELL=1 ./scripts/test-qml-runtime.sh >/dev/null 2>&1); then
  echo "test-qml-runtime-policy: accepted hard-coded UI font" >&2
  exit 1
fi

echo "QML runtime policy rejects hard-coded font inheritance"
