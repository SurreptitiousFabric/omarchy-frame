#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

if ! command -v quickshell >/dev/null 2>&1; then
  if [[ ${FRAME_REQUIRE_QUICKSHELL:-0} == 1 ]]; then
    echo "test-qml-runtime: quickshell is required" >&2
    exit 1
  fi
  echo "Quickshell runtime test skipped: quickshell is not installed"
  exit 0
fi

omarchy_qml_root=${OMARCHY_QML_ROOT:-/usr/share/omarchy/shell}
for module in Commons Ui; do
  [[ -d $omarchy_qml_root/$module ]] || {
    echo "test-qml-runtime: missing Omarchy QML module $omarchy_qml_root/$module" >&2
    exit 1
  }
done

work=$(mktemp -d "${TMPDIR:-/tmp}/omarchy-frame-qml.XXXXXX")
case $work in
  "${TMPDIR:-/tmp}"/omarchy-frame-qml.*) ;;
  *)
    echo "test-qml-runtime: unsafe temporary path" >&2
    exit 1
    ;;
esac
trap 'rm -rf -- "$work"' EXIT

cp tests/FrameTabGroupTest.qml "$work/FrameTabGroupTest.qml"
cp tests/BarWidgetTest.qml "$work/BarWidgetTest.qml"
cp BarWidget.qml "$work/BarWidget.qml"
cp -a components "$work/components"
ln -s "$omarchy_qml_root/Commons" "$work/Commons"
ln -s "$omarchy_qml_root/Ui" "$work/Ui"

run_test() {
  local file=$1
  local pass_marker=$2
  local failure_message=$3
  local output status

  cp "$work/$file" "$work/shell.qml"
  set +e
  output=$(timeout 10 quickshell --no-duplicate --no-color --path "$work/shell.qml" 2>&1)
  status=$?
  set -e

  printf '%s\n' "$output"
  if (( status != 0 )) || grep -Fq "${pass_marker%_PASS}_FAIL" <<<"$output" || ! grep -Fq "$pass_marker" <<<"$output"; then
    echo "test-qml-runtime: $failure_message" >&2
    exit 1
  fi
}

run_test FrameTabGroupTest.qml FRAME_TAB_TEST_PASS "Frame tab navigation failed"
run_test BarWidgetTest.qml FRAME_BAR_WIDGET_TEST_PASS "BarWidget component-load smoke test failed"

echo "Quickshell Frame tab navigation and BarWidget component-load smoke test passed"
