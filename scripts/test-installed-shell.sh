#!/bin/bash
set -euo pipefail

plugin_id=io.github.surreptitiousfabric.omarchy-frame
exercise_panel=0
expected_commit=

usage() {
  cat <<'EOF'
Usage: scripts/test-installed-shell.sh [--exercise-panel] [--expect-commit COMMIT]

Validate the plugin through the running Omarchy shell. The default checks are
read-only. --exercise-panel summons and hides the panel; it performs a status
refresh but sends no control command to the television.
EOF
}

while (( $# > 0 )); do
  case $1 in
    --exercise-panel)
      exercise_panel=1
      shift
      ;;
    --expect-commit)
      (( $# >= 2 )) || { usage >&2; exit 2; }
      expected_commit=$2
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -n $expected_commit && ! $expected_commit =~ ^[0-9A-Fa-f]{7,40}$ ]]; then
  echo "test-installed-shell: --expect-commit requires a 7-40 digit hexadecimal commit" >&2
  exit 2
fi

for command in jq pgrep quickshell; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "test-installed-shell: missing command: $command" >&2
    exit 1
  }
done

mapfile -t shell_pids < <(pgrep -f '^quickshell .* -p /usr/share/omarchy/shell(/shell\.qml)?( |$)' || true)
if (( ${#shell_pids[@]} != 1 )); then
  echo "test-installed-shell: expected one running Omarchy shell, found ${#shell_pids[@]}" >&2
  exit 1
fi
shell_pid=${shell_pids[0]}

ipc() {
  quickshell ipc --pid "$shell_pid" call shell "$@"
}

[[ $(ipc ping) == ok ]] || {
  echo "test-installed-shell: shell IPC did not answer" >&2
  exit 1
}

plugins=$(ipc listPlugins)
jq -e --arg id "$plugin_id" '
  [.[] | select(.id == $id)] as $matches |
  ($matches | length) == 1 and
  $matches[0].enabled == true and
  $matches[0].firstParty == false and
  ($matches[0].kinds | index("service")) != null and
  ($matches[0].kinds | index("bar-widget")) != null
' <<<"$plugins" >/dev/null || {
  echo "test-installed-shell: plugin is not uniquely enabled as a third-party service and bar widget" >&2
  exit 1
}

config=$(ipc listShellConfig)
layout_entries=$(jq -c --arg id "$plugin_id" '
  [.bar.layout | to_entries[] | .key as $section | .value[] |
    select((if type == "object" then .id else . end) == $id) |
    {section: $section, entry: .}]
' <<<"$config")
if [[ $(jq 'length' <<<"$layout_entries") != 1 ]]; then
  echo "test-installed-shell: plugin does not have exactly one effective bar entry" >&2
  exit 1
fi
layout_section=$(jq -r '.[0].section' <<<"$layout_entries")

geometry=$(ipc debugBarGeometry)
frame_geometry=$(jq -c --arg id "$plugin_id" '[.[] | select(.id == $id)]' <<<"$geometry")
jq -e --arg section "$layout_section" '
  length > 0 and
  all(.[]; .section == $section and .visible == true and .itemVisible == true and
           .width > 0 and .height > 0 and .itemWidth > 0 and .itemHeight > 0)
' <<<"$frame_geometry" >/dev/null || {
  echo "test-installed-shell: live Frame widget geometry is absent or invisible" >&2
  exit 1
}

if command -v hyprctl >/dev/null 2>&1; then
  monitor_count=$(hyprctl monitors -j | jq 'length')
  widget_count=$(jq 'length' <<<"$frame_geometry")
  if (( monitor_count > 0 && widget_count != monitor_count )); then
    echo "test-installed-shell: expected $monitor_count live widget instances, found $widget_count" >&2
    exit 1
  fi
fi

if [[ -n $expected_commit ]]; then
  command -v git >/dev/null 2>&1 || {
    echo "test-installed-shell: missing command: git" >&2
    exit 1
  }
  plugin_dir=${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/$plugin_id
  [[ -d $plugin_dir/.git ]] || {
    echo "test-installed-shell: installed plugin is not a Git checkout" >&2
    exit 1
  }
  installed_commit=$(git -C "$plugin_dir" rev-parse HEAD)
  expected_commit=$(git -C "$plugin_dir" rev-parse "$expected_commit^{commit}")
  [[ $installed_commit == "$expected_commit" ]] || {
    echo "test-installed-shell: installed commit does not match expected commit" >&2
    exit 1
  }
  [[ -z $(git -C "$plugin_dir" status --short) ]] || {
    echo "test-installed-shell: installed plugin checkout is dirty" >&2
    exit 1
  }
fi

if (( exercise_panel )); then
  [[ $(ipc summon "$plugin_id" '{}') == ok ]] || {
    echo "test-installed-shell: live shell could not summon the Frame panel" >&2
    exit 1
  }
  ipc hide "$plugin_id" >/dev/null
fi

printf 'Installed Omarchy shell has %s visible Frame widget instance(s)' "$(jq 'length' <<<"$frame_geometry")"
if (( exercise_panel )); then
  printf ', and panel summon/hide passed'
fi
printf '\n'
