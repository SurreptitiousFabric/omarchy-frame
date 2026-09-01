#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

if command -v omarchy >/dev/null 2>&1; then
  omarchy plugin validate .
fi

jq -e '
  .schemaVersion == 1 and
  .id == "io.github.surreptitiousfabric.omarchy-frame" and
  .author == "SurreptitiousFabric" and
  (.version | test("^[0-9]+\\.[0-9]+\\.[0-9]+$")) and
  (.kinds | index("service")) and
  (.kinds | index("bar-widget")) and
  .entryPoints.service == "Service.qml" and
  .entryPoints.barWidget == "BarWidget.qml"
' manifest.json >/dev/null

if find . -path ./.git -prune -o -type l -print -quit | grep -q .; then
  echo "validate-repository: symlinks are not allowed" >&2
  exit 1
fi

if find . -path ./.git -prune -o -type f -perm -0002 -print -quit | grep -q .; then
  echo "validate-repository: world-writable files are not allowed" >&2
  exit 1
fi

mapfile -d '' -t repository_scripts < <(find scripts -mindepth 1 -maxdepth 1 -type f -name '*.sh' -print0)
(( ${#repository_scripts[@]} > 0 )) || {
  echo "validate-repository: no validation scripts found" >&2
  exit 1
}

declare -A allowed_executables=(
  [./bin/frame-controller]=1
  [./bin/frame-controller-linux-amd64]=1
  [./bin/frame-controller-linux-arm64]=1
)
for script in "${repository_scripts[@]}"; do
  allowed_executables["./$script"]=1
done

while IFS= read -r -d '' executable; do
  if [[ -z ${allowed_executables[$executable]+allowed} ]]; then
    echo "validate-repository: unexpected executable $executable" >&2
    exit 1
  fi
done < <(find . -path ./.git -prune -o -type f -perm /0111 -print0)

if grep -R -n -E 'github\.com/(OWNER|YOUR[-_A-Z]*|your[_-]?github)|BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY' \
  README.md SECURITY.md MARKETPLACE.md 2>/dev/null; then
  echo "validate-repository: placeholder URL or private key marker found" >&2
  exit 1
fi

while IFS= read -r action; do
  if [[ $action == ./* ]]; then
    continue
  fi
  if [[ ! $action =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+@[0-9a-f]{40}$ ]]; then
    echo "validate-repository: remote action is not pinned to a full commit: $action" >&2
    exit 1
  fi
done < <(awk '
  /^[[:space:]]*uses:[[:space:]]*/ {
    action = $0
    sub(/^[[:space:]]*uses:[[:space:]]*/, "", action)
    sub(/[[:space:]]+#.*$/, "", action)
    print action
  }
' .github/workflows/*.yml)

sh -n bin/frame-controller
bash -n "${repository_scripts[@]}"

scripts/test-ui-contract.sh
scripts/test-release-package.sh
scripts/test-release-readiness.sh

if awk 'NF >= 2 && $2 ~ /^\//' bin/SHA256SUMS | grep -q .; then
  echo "validate-repository: SHA256SUMS contains absolute paths" >&2
  exit 1
fi

(
  cd bin
  sha256sum -c SHA256SUMS
)

scripts/test-packaged-runtime.sh

for required in README.md USER_MANUAL.md CAPABILITIES.md THREAT_MODEL.md SECURITY.md DEVELOPMENT.md ACCEPTANCE.md CHANGELOG.md MARKETPLACE.md LICENSE; do
  test -f "$required" || {
    echo "validate-repository: missing $required" >&2
    exit 1
  }
done
