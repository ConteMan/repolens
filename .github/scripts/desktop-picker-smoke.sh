#!/usr/bin/env bash
set -euo pipefail

if [[ "${GITHUB_ACTIONS:-}" != true || "${RUNNER_OS:-}" != macOS || "${RUNNER_ENVIRONMENT:-}" != github-hosted ]]; then
  echo "This smoke test requires an isolated GitHub-hosted macOS runner." >&2
  exit 1
fi

work="$(mktemp -d /tmp/repolens-picker.XXXXXX)"
trap 'if [[ -n "${app_pid:-}" ]]; then kill "$app_pid" 2>/dev/null || true; wait "$app_pid" 2>/dev/null || true; fi; rm -rf "$work"' EXIT
artifact_dir="$RUNNER_TEMP/native-picker-artifacts"
mkdir -p "$artifact_dir"
swiftc .github/scripts/desktop-picker.swift -o "$work/picker"

status=0
"$work/picker" || status=$?
if [[ "$status" == 77 ]]; then
  echo "::warning::System directory picker smoke skipped because macOS Accessibility or keyboard event access is unavailable."
  echo "System directory picker smoke skipped because macOS Accessibility or keyboard event access is unavailable." >>"$GITHUB_STEP_SUMMARY"
  exit 0
fi
if [[ "$status" != 0 ]]; then
  exit "$status"
fi

go build -tags e2e -o "$work/desktop" ./cmd/repolens-desktop
go build -o "$work/automation" github.com/egoist/mygo/plugins/automation/cmd/mygo-automation
fixture="$work/fixture repository"
mkdir "$fixture"
git -C "$fixture" init --quiet
socket="$work/automation.sock"
export MYGO_AUTOMATION_SESSION=repolens-picker
MYGO_AUTOMATION="$socket" "$work/desktop" -lang en >"$artifact_dir/desktop.log" 2>&1 &
app_pid=$!
for ((attempt = 0; attempt < 50; attempt++)); do
  [[ ! -S "$socket" ]] || break
  kill -0 "$app_pid"
  sleep 0.2
done

auto="$work/automation"
"$auto" open --socket "$socket"
"$auto" wait 'key=welcome-picker'
"$auto" click 'key=welcome-picker'
"$work/picker" "$app_pid" cancel
for ((attempt = 0; attempt < 30; attempt++)); do
  "$auto" snapshot >"$artifact_dir/after-cancel.txt"
  if rg -q 'key=welcome-picker( focused)?$' "$artifact_dir/after-cancel.txt"; then
    break
  fi
  sleep 0.2
done
if ! rg -q 'key=welcome-picker( focused)?$' "$artifact_dir/after-cancel.txt"; then
  echo "Welcome picker remained disabled after cancel" >&2
  exit 1
fi

"$auto" click 'key=welcome-picker'
"$work/picker" "$app_pid" open "$fixture"
"$auto" wait --text 'Paths and Rules'
"$auto" snapshot >"$artifact_dir/after-open.txt"
rg -F 'fixture repository/.repolens.yml' "$artifact_dir/after-open.txt"
"$auto" screenshot "$artifact_dir/opened-repository.png"
echo "System directory picker smoke passed for cancel and opening a repository with spaces in its path." >>"$GITHUB_STEP_SUMMARY"
