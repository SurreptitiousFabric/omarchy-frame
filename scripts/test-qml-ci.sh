#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

for command in quickshell sway; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "test-qml-ci: missing command: $command" >&2
    exit 1
  }
done

FRAME_REQUIRE_QMLLINT=1 scripts/test-qml-types.sh
FRAME_REQUIRE_QMLLINT=1 scripts/test-qml-policy.sh

runtime=$(mktemp -d "${TMPDIR:-/tmp}/omarchy-frame-sway.XXXXXX")
case $runtime in
  "${TMPDIR:-/tmp}"/omarchy-frame-sway.*) ;;
  *)
    echo "test-qml-ci: unsafe temporary path" >&2
    exit 1
    ;;
esac
chmod 0700 "$runtime"
sway_log=$runtime/sway.log
compositor_pid=

cleanup() {
  if [[ -n $compositor_pid ]]; then
    kill "$compositor_pid" 2>/dev/null || true
    wait "$compositor_pid" 2>/dev/null || true
  fi
  rm -rf -- "$runtime"
}
trap cleanup EXIT

XDG_RUNTIME_DIR=$runtime \
WLR_BACKENDS=headless \
WLR_HEADLESS_OUTPUTS=1 \
WLR_LIBINPUT_NO_DEVICES=1 \
WLR_RENDERER=pixman \
  sway --config /dev/null >"$sway_log" 2>&1 &
compositor_pid=$!

wayland_socket=
for _ in $(seq 1 50); do
  wayland_socket=$(find "$runtime" -maxdepth 1 -type s -name 'wayland-*' -printf '%f\n' | head -n 1)
  [[ -n $wayland_socket ]] && break
  if ! kill -0 "$compositor_pid" 2>/dev/null; then
    echo "test-qml-ci: headless Sway exited before creating a Wayland socket" >&2
    cat "$sway_log" >&2
    exit 1
  fi
  sleep 0.1
done

if [[ -z $wayland_socket ]]; then
  echo "test-qml-ci: headless Sway did not create a Wayland socket" >&2
  cat "$sway_log" >&2
  exit 1
fi

if ! XDG_RUNTIME_DIR=$runtime \
  WAYLAND_DISPLAY=$wayland_socket \
  QT_QPA_PLATFORM=wayland \
  QSG_RHI_BACKEND=software \
  FRAME_REQUIRE_QUICKSHELL=1 \
  scripts/test-qml-runtime.sh; then
  echo "test-qml-ci: Quickshell runtime test failed under headless Sway" >&2
  cat "$sway_log" >&2
  exit 1
fi

FRAME_REQUIRE_QML_RENDER=1 scripts/test-qml-render.sh

echo "QML contracts, layer-shell runtime loading, and icon pixels passed in CI"
