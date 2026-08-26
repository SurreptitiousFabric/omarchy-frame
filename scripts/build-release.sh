#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
mkdir -p "$root/bin"

required_go="go1.27.0"
actual_go=$(go env GOVERSION)
if [[ $actual_go != "$required_go" ]]; then
  echo "build-release: require $required_go, found $actual_go; run 'mise install'" >&2
  exit 1
fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$root/bin/frame-controller-linux-amd64" "$root/cmd/frame-controller"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$root/bin/frame-controller-linux-arm64" "$root/cmd/frame-controller"

chmod 0755 "$root/bin/frame-controller" "$root/bin/frame-controller-linux-"*
(
  cd "$root/bin"
  sha256sum frame-controller-linux-* > SHA256SUMS
)
