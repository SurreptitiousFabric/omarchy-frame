#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/omarchy-frame-package-test.XXXXXX")
case $work in
  "${TMPDIR:-/tmp}"/omarchy-frame-package-test.*) ;;
  *)
    echo "test-release-package: unsafe temporary path" >&2
    exit 1
    ;;
esac
trap 'rm -rf -- "$work"' EXIT

(
  cd "$work"
  "$root/scripts/package-release.sh" first.tar.gz >/dev/null
  "$root/scripts/package-release.sh" second.tar.gz >/dev/null
)

cmp -s "$work/first.tar.gz" "$work/second.tar.gz" || {
  echo "test-release-package: repeated archives differ" >&2
  exit 1
}

expected=$(printf '%s\n' \
  frame-controller \
  frame-controller-linux-amd64 \
  frame-controller-linux-arm64 \
  SHA256SUMS |
  sort)
actual=$(tar -tzf "$work/first.tar.gz" | sort)
[[ $actual == "$expected" ]] || {
  echo "test-release-package: archive contents are not the release allowlist" >&2
  exit 1
}

if (cd "$work" && "$root/scripts/package-release.sh" ../escape.tar.gz >/dev/null 2>&1); then
  echo "test-release-package: accepted an output path" >&2
  exit 1
fi

fixture="$work/repository"
mkdir -p "$fixture"
cp -a "$root/." "$fixture/"
git -C "$fixture" remote remove origin >/dev/null 2>&1 || true
git -C "$fixture" config user.name "Release Package Test"
git -C "$fixture" config user.email "release-package@example.invalid"
git -C "$fixture" add -A
git -C "$fixture" commit -q --allow-empty -m "Package candidate"
(
  cd "$work"
  "$fixture/scripts/package-release.sh" candidate.tar.gz >/dev/null
)
printf '\nEvidence-only package test.\n' >>"$fixture/ACCEPTANCE.md"
git -C "$fixture" add ACCEPTANCE.md
git -C "$fixture" commit -q -m "Record package evidence"
(
  cd "$work"
  "$fixture/scripts/package-release.sh" evidence.tar.gz >/dev/null
)
cmp -s "$work/candidate.tar.gz" "$work/evidence.tar.gz" || {
  echo "test-release-package: evidence-only commit changed archive bytes" >&2
  exit 1
}

echo "Release archive is deterministic and allowlisted"
