#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

qml_lint=${QMLLINT:-}
if [[ -z $qml_lint ]]; then
  if command -v qmllint >/dev/null 2>&1; then
    qml_lint=$(command -v qmllint)
  elif [[ -x /usr/lib/qt6/bin/qmllint ]]; then
    qml_lint=/usr/lib/qt6/bin/qmllint
  fi
fi

if [[ -z $qml_lint ]]; then
  if [[ ${FRAME_REQUIRE_QMLLINT:-0} == 1 ]]; then
    echo "test-qml-types: qmllint is required but unavailable" >&2
    exit 1
  fi
  echo "QML type validation skipped: qmllint is not installed"
  exit 0
fi

omarchy_qml_root=${OMARCHY_QML_ROOT:-/usr/share/omarchy/shell}
for module in Commons Ui; do
  if [[ ! -d $omarchy_qml_root/$module ]]; then
    echo "test-qml-types: missing Omarchy QML module $omarchy_qml_root/$module" >&2
    exit 1
  fi
done

work=$(mktemp -d "${TMPDIR:-/tmp}/omarchy-frame-qml-types.XXXXXX")
case $work in
  "${TMPDIR:-/tmp}"/omarchy-frame-qml-types.*) ;;
  *)
    echo "test-qml-types: unsafe temporary path" >&2
    exit 1
    ;;
esac
trap 'rm -rf -- "$work"' EXIT

mkdir -p "$work/imports/qs"
ln -s "$omarchy_qml_root/Commons" "$work/imports/qs/Commons"
ln -s "$omarchy_qml_root/Ui" "$work/imports/qs/Ui"

set +e
output=$("$qml_lint" -I "$work/imports" -I /usr/lib/qt6/qml BarWidget.qml components/*.qml 2>&1)
status=$?
set -e

if (( status != 0 )); then
  printf '%s\n' "$output" >&2
  echo "test-qml-types: qmllint failed" >&2
  exit 1
fi

if grep -E -q 'Failed to import (qs\.(Commons|Ui)|Quickshell(\.Io)?)|Warnings occurred while importing module "(qs\.(Commons|Ui)|Quickshell(\.Io)?)"' <<<"$output"; then
  printf '%s\n' "$output" >&2
  echo "test-qml-types: required QML modules did not resolve" >&2
  exit 1
fi

contract_errors=$(grep -E 'Could not find property ".*"\. \[missing-property\]|Component is missing required property .* from (ArtPage|FrameButton|FrameTabGroup|FrameTvIcon|GalleryCard|PhotosPage|RemotePage|SetupPage)' <<<"$output" || true)
if [[ -n $contract_errors ]]; then
  printf '%s\n' "$contract_errors" >&2
  echo "test-qml-types: local component contract is invalid" >&2
  exit 1
fi

echo "QML local component contracts are valid"
