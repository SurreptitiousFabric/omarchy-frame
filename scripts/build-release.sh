#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
mkdir -p "$root/bin"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$root/bin/frame-controller-linux-amd64" "$root/cmd/frame-controller"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$root/bin/frame-controller-linux-arm64" "$root/cmd/frame-controller"

arch=$(uname -m)
case "$arch" in
  x86_64) cp "$root/bin/frame-controller-linux-amd64" "$root/bin/frame-controller" ;;
  aarch64|arm64) cp "$root/bin/frame-controller-linux-arm64" "$root/bin/frame-controller" ;;
  *) echo "Unsupported local architecture: $arch" >&2; exit 1 ;;
esac
chmod 0755 "$root/bin/frame-controller" "$root/bin/frame-controller-linux-"*
sha256sum "$root/bin/frame-controller-linux-"* > "$root/bin/SHA256SUMS"
