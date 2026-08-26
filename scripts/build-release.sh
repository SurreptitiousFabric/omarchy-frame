#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
mkdir -p "$root/bin"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$root/bin/frame-controller-linux-amd64" "$root/cmd/frame-controller"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$root/bin/frame-controller-linux-arm64" "$root/cmd/frame-controller"

chmod 0755 "$root/bin/frame-controller" "$root/bin/frame-controller-linux-"*
sha256sum "$root/bin/frame-controller-linux-"* > "$root/bin/SHA256SUMS"
