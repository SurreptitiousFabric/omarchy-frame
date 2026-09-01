#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

if ! command -v qmllint >/dev/null 2>&1 && [[ ! -x /usr/lib/qt6/bin/qmllint ]] && [[ -z ${QMLLINT:-} ]]; then
  if [[ ${FRAME_REQUIRE_QMLLINT:-0} == 1 ]]; then
    echo "test-qml-policy: qmllint is required but unavailable" >&2
    exit 1
  fi
  echo "QML negative contract tests skipped: qmllint is not installed"
  exit 0
fi

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
  if (cd "$target" && ./scripts/test-qml-types.sh >/dev/null 2>&1); then
    echo "test-qml-policy: accepted $name" >&2
    exit 1
  fi
}

target=$(new_case missing-required-page-property)
sed -i '/^[[:space:]]*ArtPage {/,/^[[:space:]]*}/ { /softFill: root.softFill/d; }' "$target/BarWidget.qml"
expect_rejected "missing required page property" "$target"

echo "QML policy rejects invalid local component contracts"
