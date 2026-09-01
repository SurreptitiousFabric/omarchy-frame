#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

# A negative mutation suite is meaningful only after the validator proves it
# can accept the valid source tree. Missing validators must not make every
# mutation look successfully rejected.
FRAME_REQUIRE_QMLLINT=1 "$root/scripts/test-qml-types.sh" >/dev/null

work=$(mktemp -d "${TMPDIR:-/tmp}/omarchy-frame-qml-policy.XXXXXX")
case $work in
  "${TMPDIR:-/tmp}"/omarchy-frame-qml-policy.*) ;;
  *)
    echo "test-qml-policy: unsafe temporary path" >&2
    exit 1
    ;;
esac
trap 'rm -rf -- "$work"' EXIT

new_case() {
  local name=$1
  local target="$work/$name"
  mkdir -p "$target"
  cp -a "$root/." "$target/"
  printf '%s\n' "$target"
}

expect_rejected() {
  local name=$1
  local target=$2
  shift 2
  if (cd "$target" && "$@" >/dev/null 2>&1); then
    echo "test-qml-policy: accepted $name" >&2
    exit 1
  fi
}

if QMLLINT="$work/missing-qmllint" FRAME_REQUIRE_QMLLINT=1 \
  "$root/scripts/test-qml-types.sh" >/dev/null 2>&1; then
  echo "test-qml-policy: missing required qmllint was treated as success" >&2
  exit 1
fi

target=$(new_case missing-required-page-property)
sed -i '/^[[:space:]]*ArtPage {/,/^[[:space:]]*}/ { /softFill: root.softFill/d; }' "$target/BarWidget.qml"
expect_rejected "missing required page property" "$target" env FRAME_REQUIRE_QMLLINT=1 ./scripts/test-qml-types.sh

target=$(new_case fixed-ui-geometry)
sed -i 's/width: Style.space(222)/width: 222/' "$target/components/RemotePage.qml"
expect_rejected "fixed UI geometry" "$target" ./scripts/test-ui-contract.sh

echo "QML policy rejects invalid local component contracts"
