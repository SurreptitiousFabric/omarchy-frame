#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
output=${1:-omarchy-frame-linux.tar.gz}

[[ $output =~ ^[A-Za-z0-9._-]+\.tar\.gz$ ]] || {
  echo "package-release: output must be a .tar.gz basename" >&2
  exit 1
}

for required in frame-controller frame-controller-linux-amd64 frame-controller-linux-arm64 SHA256SUMS; do
  [[ -f "$root/bin/$required" && ! -L "$root/bin/$required" ]] || {
    echo "package-release: missing regular bin/$required" >&2
    exit 1
  }
done

source_date_epoch=${SOURCE_DATE_EPOCH:-$(git -C "$root" log -1 --format=%ct -- \
  bin/frame-controller \
  bin/frame-controller-linux-amd64 \
  bin/frame-controller-linux-arm64 \
  bin/SHA256SUMS)}
[[ $source_date_epoch =~ ^[0-9]+$ ]] || {
  echo "package-release: SOURCE_DATE_EPOCH must be an integer" >&2
  exit 1
}

destination="$PWD/$output"
temporary=$(mktemp "$PWD/.omarchy-frame-package.XXXXXX")
case $temporary in
  "$PWD"/.omarchy-frame-package.*) ;;
  *)
    echo "package-release: unsafe temporary path" >&2
    exit 1
    ;;
esac
trap 'rm -f -- "$temporary"' EXIT

LC_ALL=C tar \
  --sort=name \
  --format=gnu \
  --mtime="@$source_date_epoch" \
  --owner=0 \
  --group=0 \
  --numeric-owner \
  -C "$root/bin" \
  -cf - \
  frame-controller \
  frame-controller-linux-amd64 \
  frame-controller-linux-arm64 \
  SHA256SUMS |
  gzip -n >"$temporary"

chmod 0644 "$temporary"
mv "$temporary" "$destination"
trap - EXIT
echo "Packaged $destination"
