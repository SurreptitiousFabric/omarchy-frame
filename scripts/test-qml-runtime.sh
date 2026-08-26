#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

if ! command -v quickshell >/dev/null 2>&1; then
  echo "Quickshell runtime test skipped: quickshell is not installed"
  exit 0
fi

work=$(mktemp -d "${TMPDIR:-/tmp}/omarchy-frame-qml.XXXXXX")
case $work in
  "${TMPDIR:-/tmp}"/omarchy-frame-qml.*) ;;
  *)
    echo "test-qml-runtime: unsafe temporary path" >&2
    exit 1
    ;;
esac
trap 'rm -rf -- "$work"' EXIT

cp tests/FrameTabGroupTest.qml "$work/shell.qml"
cp -a components "$work/components"
ln -s /usr/share/omarchy/shell/Commons "$work/Commons"
ln -s /usr/share/omarchy/shell/Ui "$work/Ui"

set +e
output=$(timeout 10 quickshell --no-duplicate --no-color --path "$work/shell.qml" 2>&1)
status=$?
set -e

printf '%s\n' "$output"
if (( status != 0 )) || grep -Fq 'FRAME_TAB_TEST_FAIL' <<<"$output" || ! grep -Fq 'FRAME_TAB_TEST_PASS' <<<"$output"; then
  echo "test-qml-runtime: Frame tab navigation failed" >&2
  exit 1
fi

echo "Quickshell Frame tab navigation passed"
