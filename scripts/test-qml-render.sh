#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

runner=${QMLTESTRUNNER:-}
if [[ -z $runner ]]; then
  if command -v qmltestrunner >/dev/null 2>&1; then
    runner=$(command -v qmltestrunner)
  elif [[ -x /usr/lib/qt6/bin/qmltestrunner ]]; then
    runner=/usr/lib/qt6/bin/qmltestrunner
  fi
fi

if [[ -z $runner || ! -x $runner ]]; then
  if [[ ${FRAME_REQUIRE_QML_RENDER:-0} == 1 ]]; then
    echo "test-qml-render: qmltestrunner is required" >&2
    exit 1
  fi
  echo "QML pixel rendering skipped: qmltestrunner is not installed"
  exit 0
fi

QT_QPA_PLATFORM=${FRAME_QML_PLATFORM:-offscreen} \
QSG_RHI_BACKEND=${FRAME_QML_RHI_BACKEND:-software} \
QT_SCALE_FACTOR=1 \
GDK_SCALE=1 \
  "$runner" -input tests/tst_FrameTvIcon.qml

echo "QML TV icon pixels render at supported bar sizes"
