#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/omarchy-frame-release-policy.XXXXXX")
case $work in
  "${TMPDIR:-/tmp}"/omarchy-frame-release-policy.*) ;;
  *)
    echo "test-release-readiness: unsafe temporary path" >&2
    exit 1
    ;;
esac
trap 'rm -rf -- "$work"' EXIT

new_case() {
  local name=$1
  local target="$work/$name"
  mkdir -p "$target"
  cp -a "$root/." "$target/"
  git -C "$target" remote remove origin >/dev/null 2>&1 || true
  git -C "$target" config user.name "Release Policy Test"
  git -C "$target" config user.email "release-policy@example.invalid"
  printf '%s\n' "$target"
}

commit_case() {
  local target=$1 message=$2
  git -C "$target" add -A
  git -C "$target" commit -q -m "$message"
}

prepare_ready() {
  local name=$1 target candidate
  target=$(new_case "$name")
  jq '.version = "1.0.0"' "$target/manifest.json" >"$target/manifest.json.new"
  mv "$target/manifest.json.new" "$target/manifest.json"
  awk '
    /^- Plugin: / {
      print "- Plugin: `io.github.surreptitiousfabric.omarchy-frame` 1.0.0"
      next
    }
    { print }
  ' "$target/ACCEPTANCE.md" >"$target/ACCEPTANCE.md.new"
  mv "$target/ACCEPTANCE.md.new" "$target/ACCEPTANCE.md"
  sed -i \
    '/Tagged release workflow executes remotely/! s/| PENDING |/| PASS |/' \
    "$target/ACCEPTANCE.md"
  awk '{ print; if ($0 == "## Unreleased") print "\n## 1.0.0 - 2026-08-27" }' \
    "$target/CHANGELOG.md" >"$target/CHANGELOG.md.new"
  mv "$target/CHANGELOG.md.new" "$target/CHANGELOG.md"
  commit_case "$target" "Prepare release candidate"
  candidate=$(git -C "$target" rev-parse HEAD)
  awk -v candidate="$candidate" '
    /^- Code and packaging candidate: public commit `/ {
      print "- Code and packaging candidate: public commit `" candidate "`"
      next
    }
    { print }
  ' "$target/ACCEPTANCE.md" >"$target/ACCEPTANCE.md.new"
  mv "$target/ACCEPTANCE.md.new" "$target/ACCEPTANCE.md"
  commit_case "$target" "Record candidate evidence"
  printf '%s\n' "$target"
}

clone_case() {
  local source=$1 name=$2
  local target="$work/$name"
  mkdir -p "$target"
  cp -a "$source/." "$target/"
  printf '%s\n' "$target"
}

expect_rejected() {
  local name=$1 pattern=$2 target=$3 tag=${4:-v1.0.0} output
  if output=$(cd "$target" && ./scripts/check-release-readiness.sh "$tag" 2>&1); then
    echo "test-release-readiness: accepted $name" >&2
    exit 1
  fi
  grep -Fq "$pattern" <<<"$output" || {
    echo "test-release-readiness: $name failed for the wrong reason" >&2
    echo "$output" >&2
    exit 1
  }
}

ready=$(prepare_ready ready)
(cd "$ready" && ./scripts/check-release-readiness.sh v1.0.0 >/dev/null)

restricted_bin="$work/restricted-bin"
mkdir -p "$restricted_bin"
for command in awk basename bash dirname git grep jq sed; do
  command_path=$(command -v "$command")
  ln -s "$command_path" "$restricted_bin/$command"
done
(
  cd "$ready"
  env PATH="$restricted_bin" ./scripts/check-release-readiness.sh v1.0.0 >/dev/null
)

target=$(clone_case "$ready" pending-check)
sed -i '0,/| PASS |/s//| PENDING |/' "$target/ACCEPTANCE.md"
commit_case "$target" "Restore pending acceptance"
expect_rejected "pending acceptance" "pending applicable acceptance check" "$target"

target=$(clone_case "$ready" wrong-tag)
expect_rejected "wrong tag" "does not match manifest version" "$target" v1.0.1

target=$(clone_case "$ready" acceptance-version)
awk '
  /^- Plugin: / {
    print "- Plugin: `io.github.surreptitiousfabric.omarchy-frame` 1.0.1"
    next
  }
  { print }
' "$target/ACCEPTANCE.md" >"$target/ACCEPTANCE.md.new"
mv "$target/ACCEPTANCE.md.new" "$target/ACCEPTANCE.md"
commit_case "$target" "Drift acceptance version"
expect_rejected "acceptance version drift" "does not match manifest" "$target"

target=$(clone_case "$ready" missing-changelog)
sed -i '/^## 1\.0\.0 - 2026-08-27$/d' "$target/CHANGELOG.md"
commit_case "$target" "Remove release changelog"
expect_rejected "missing changelog" "changelog has no dated section" "$target"

target=$(clone_case "$ready" runtime-drift)
printf '\n// release-policy mutation\n' >>"$target/Service.qml"
commit_case "$target" "Mutate runtime after candidate"
expect_rejected "post-candidate runtime drift" "release-sensitive path changed after candidate" "$target"

target=$(clone_case "$ready" dirty-tree)
printf '\nrelease-policy mutation\n' >>"$target/README.md"
expect_rejected "dirty release tree" "working tree is not clean" "$target"

target=$(clone_case "$ready" pre-v1)
jq '.version = "0.6.0"' "$target/manifest.json" >"$target/manifest.json.new"
mv "$target/manifest.json.new" "$target/manifest.json"
awk '
  /^- Plugin: / {
    print "- Plugin: `io.github.surreptitiousfabric.omarchy-frame` 0.6.0"
    next
  }
  { print }
' "$target/ACCEPTANCE.md" >"$target/ACCEPTANCE.md.new"
mv "$target/ACCEPTANCE.md.new" "$target/ACCEPTANCE.md"
commit_case "$target" "Attempt beta release"
expect_rejected "pre-v1 release" "production releases require" "$target" v0.6.0

echo "Release readiness fails closed"
