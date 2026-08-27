#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

fail() {
  echo "check-release-readiness: $*" >&2
  exit 1
}

trim() {
  local value=$1
  value=${value#"${value%%[![:space:]]*}"}
  value=${value%"${value##*[![:space:]]}"}
  printf '%s' "$value"
}

if (( $# > 1 )); then
  fail "usage: scripts/check-release-readiness.sh <vMAJOR.MINOR.PATCH>"
fi

tag=${1:-${GITHUB_REF_NAME:-}}
[[ -n $tag ]] || fail "a release tag is required"

version=$(jq -er '
  .version
  | select(type == "string")
  | select(test("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"))
' manifest.json) || fail "manifest version is not stable semantic versioning"

[[ $version =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]] ||
  fail "manifest version is not stable semantic versioning"
(( 10#${BASH_REMATCH[1]} >= 1 )) ||
  fail "production releases require manifest version 1.0.0 or newer"
[[ $tag == "v$version" ]] ||
  fail "tag '$tag' does not match manifest version '$version'"

[[ -d .git ]] || fail "release readiness requires a Git checkout"
[[ -z $(git status --porcelain --untracked-files=all) ]] ||
  fail "working tree is not clean"

mapfile -t acceptance_versions < <(awk '
  /^- Plugin: / {
    for (field = 1; field <= NF; field++) {
      if ($field ~ /^[0-9]+\.[0-9]+\.[0-9]+$/) print $field
    }
  }
' ACCEPTANCE.md)
(( ${#acceptance_versions[@]} == 1 )) ||
  fail "acceptance record must contain exactly one plugin version"
[[ ${acceptance_versions[0]} == "$version" ]] ||
  fail "acceptance version '${acceptance_versions[0]}' does not match manifest '$version'"
if rg -q '^- Plugin: .*\(pre-1\.0\)$' ACCEPTANCE.md; then
  fail "acceptance record still marks the plugin pre-1.0"
fi

workflow_row="Tagged release workflow executes remotely"
workflow_rows=0
matrix_rows=0
while IFS='|' read -r _ raw_check raw_status _; do
  check=$(trim "$raw_check")
  status=$(trim "$raw_status")
  [[ -n $check && -n $status ]] || continue
  [[ $check == "Check" || $check =~ ^-+$ ]] && continue
  (( matrix_rows += 1 ))
  case $status in
    PASS | N/A) ;;
    PENDING)
      [[ $check == "$workflow_row" ]] ||
        fail "pending applicable acceptance check: $check"
      ;;
    *) fail "unknown acceptance status '$status' for '$check'" ;;
  esac
  [[ $check != "$workflow_row" ]] || (( workflow_rows += 1 ))
done <ACCEPTANCE.md
(( matrix_rows > 0 )) || fail "acceptance matrix has no checks"
(( workflow_rows == 1 )) ||
  fail "acceptance matrix must contain exactly one tagged-workflow row"

version_re=${version//./\\.}
rg -q "^## \\[?${version_re}\\]? - [0-9]{4}-[0-9]{2}-[0-9]{2}$" CHANGELOG.md ||
  fail "changelog has no dated section for $version"

mapfile -t candidates < <(awk '
  /^- Code and packaging candidate: public commit / {
    if (match($0, /[0-9a-f]{7,40}/)) print substr($0, RSTART, RLENGTH)
  }
' ACCEPTANCE.md)
(( ${#candidates[@]} == 1 )) ||
  fail "acceptance record must name exactly one hexadecimal code candidate"
candidate=${candidates[0]}
candidate_commit=$(git rev-parse --verify "${candidate}^{commit}" 2>/dev/null) ||
  fail "recorded code candidate '$candidate' is unavailable"
git merge-base --is-ancestor "$candidate_commit" HEAD ||
  fail "recorded code candidate is not an ancestor of the release"

while IFS= read -r changed; do
  case $changed in
    ACCEPTANCE.md | MARKETPLACE.md) ;;
    *) fail "release-sensitive path changed after candidate: $changed" ;;
  esac
done < <(git diff --name-only "$candidate_commit"..HEAD)

echo "Release readiness passed for $tag"
