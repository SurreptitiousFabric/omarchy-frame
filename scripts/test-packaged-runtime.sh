#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

case $(uname -m) in
  x86_64) native="$root/bin/frame-controller-linux-amd64" ;;
  aarch64 | arm64) native="$root/bin/frame-controller-linux-arm64" ;;
  *)
    echo "test-packaged-runtime: unsupported test architecture" >&2
    exit 1
    ;;
esac

test -x "$root/bin/frame-controller"
test -x "$native"

if command -v file >/dev/null 2>&1; then
  file "$native" | grep -q 'statically linked' || {
    echo "test-packaged-runtime: native backend is not static" >&2
    exit 1
  }
fi

runtime_root=$(mktemp -d "${TMPDIR:-/tmp}/omarchy-frame-runtime.XXXXXX")
case $runtime_root in
  "${TMPDIR:-/tmp}"/omarchy-frame-runtime.*) ;;
  *)
    echo "test-packaged-runtime: unsafe temporary path" >&2
    exit 1
    ;;
esac
trap 'rm -rf -- "$runtime_root"' EXIT
mkdir -p "$runtime_root/bin" "$runtime_root/home" "$runtime_root/state"
ln -s "$(command -v dirname)" "$runtime_root/bin/dirname"
ln -s "$(command -v uname)" "$runtime_root/bin/uname"

output=$(env -i \
  HOME="$runtime_root/home" \
  XDG_STATE_HOME="$runtime_root/state" \
  PATH="$runtime_root/bin" \
  "$root/bin/frame-controller" capabilities)

jq -e '
  .ok == true and
  (.capabilities | type == "array" and length >= 8) and
  (all(.capabilities[]; (.group | type == "string" and length > 0) and (.items | type == "string" and length > 0)))
' <<<"$output" >/dev/null

echo "Packaged runtime works without Go: $(basename "$native")"
