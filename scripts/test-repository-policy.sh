#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/omarchy-frame-policy.XXXXXX")
case $work in
  "${TMPDIR:-/tmp}"/omarchy-frame-policy.*) ;;
  *)
    echo "test-repository-policy: unsafe temporary path" >&2
    exit 1
    ;;
esac
trap 'rm -rf -- "$work"' EXIT

new_case() {
  local name=$1
  local target="$work/$name"
  mkdir -p "$target"
  cp -a "$root/." "$target/"
  printf '%s\n' "$target"
}

expect_rejected() {
  local name=$1
  local target=$2
  if (cd "$target" && ./scripts/validate-repository.sh >/dev/null 2>&1); then
    echo "test-repository-policy: accepted $name" >&2
    exit 1
  fi
}

expect_accepted() {
  local name=$1
  local target=$2
  if ! (cd "$target" && ./scripts/validate-repository.sh >/dev/null 2>&1); then
    echo "test-repository-policy: rejected $name" >&2
    exit 1
  fi
}

target=$(new_case wrong-id)
jq '.id = "weak.id"' "$target/manifest.json" >"$target/manifest.json.new"
mv "$target/manifest.json.new" "$target/manifest.json"
expect_rejected "wrong plugin id" "$target"

target=$(new_case unexpected-executable)
install -m 0755 /dev/null "$target/unexpected-tool"
expect_rejected "unexpected executable" "$target"

target=$(new_case valid-top-level-script)
printf '#!/bin/bash\nset -euo pipefail\n' >"$target/scripts/valid-new-check.sh"
chmod 0755 "$target/scripts/valid-new-check.sh"
expect_accepted "valid top-level script" "$target"

target=$(new_case broken-top-level-script)
printf '#!/bin/bash\nif then\n' >"$target/scripts/broken-check.sh"
chmod 0755 "$target/scripts/broken-check.sh"
expect_rejected "broken top-level script syntax" "$target"

target=$(new_case nested-executable-script)
mkdir -p "$target/scripts/nested"
printf '#!/bin/bash\nif then\n' >"$target/scripts/nested/broken.sh"
chmod 0755 "$target/scripts/nested/broken.sh"
expect_rejected "nested executable script" "$target"

target=$(new_case nested-valid-executable-script)
mkdir -p "$target/scripts/nested"
printf '#!/bin/bash\nset -euo pipefail\n' >"$target/scripts/nested/valid.sh"
chmod 0755 "$target/scripts/nested/valid.sh"
expect_rejected "nested valid executable script" "$target"

target=$(new_case world-writable)
chmod o+w "$target/README.md"
expect_rejected "world-writable file" "$target"

target=$(new_case symlink)
ln -s README.md "$target/linked-readme"
expect_rejected "symlink" "$target"

target=$(new_case placeholder-url)
printf '\nhttps://github.com/OWNER/omarchy-frame\n' >>"$target/README.md"
expect_rejected "placeholder repository URL" "$target"

target=$(new_case mutable-workflow-action)
sed -i -E '0,/actions\/checkout@[0-9a-f]{40}/s//actions\/checkout@v7/' \
  "$target/.github/workflows/ci.yml"
expect_rejected "mutable workflow action" "$target"

echo "Repository policy rejects unsafe release shapes"
