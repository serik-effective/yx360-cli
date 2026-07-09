#!/usr/bin/env bash
# Stop — scope-keyed evidence gate (D-028, RC-1/RC-2 from reflect 2026-06-29).
# A "done" on a UI diff must cite an outcome artifact (a screenshot of the
# RUNNING result), not a build-green proxy. SwiftUI views additionally require
# that apple-design-critic ran this session (creator != critic).
# Loop-safe: short-circuits on stop_hook_active, caps blocks at <=2 per git HEAD,
# fail-open on any error.
set -euo pipefail

fail_open() { exit 0; }
trap fail_open ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
STDIN="$(cat)"

STOP_ACTIVE="$(printf '%s' "$STDIN" | jq -r '.stop_hook_active // false')"
[ "$STOP_ACTIVE" = "true" ] && exit 0

SESSION_ID="$(printf '%s' "$STDIN" | jq -r '.session_id // empty')"
TRANSCRIPT="$(printf '%s' "$STDIN" | jq -r '.transcript_path // empty')"
[ -n "$SESSION_ID" ] || exit 0
[ -n "$TRANSCRIPT" ] && [ -f "$TRANSCRIPT" ] || exit 0

# Files edited this session (Edit/Write/MultiEdit tool_use file_path).
EDITED="$(jq -r 'select(.type=="assistant") | .message.content[]?
  | select(.type=="tool_use")
  | select(.name=="Edit" or .name=="Write" or .name=="MultiEdit")
  | .input.file_path // empty' "$TRANSCRIPT" 2>/dev/null || echo)"
[ -n "$EDITED" ] || exit 0

WEB_UI="$(printf '%s\n' "$EDITED" | grep -iE '\.(tsx|jsx|vue|svelte|css|scss|html)$' | head -1 || true)"
SWIFT_UI="$(printf '%s\n' "$EDITED" | grep -iE 'View\.swift$|Screen[A-Za-z]*\.swift$' | head -1 || true)"
[ -n "$WEB_UI$SWIFT_UI" ] || exit 0

# Evidence present in the transcript this session.
HAS_SHOT="$(grep -cE 'take_screenshot|xcodebuildmcp__screenshot|record_sim_video' "$TRANSCRIPT" 2>/dev/null || true)"; HAS_SHOT="${HAS_SHOT:-0}"
HAS_CRITIC="$(grep -cE 'apple-design-critic|frontend-design|visual-spec' "$TRANSCRIPT" 2>/dev/null || true)"; HAS_CRITIC="${HAS_CRITIC:-0}"

MSG=""
if [ -n "$SWIFT_UI" ]; then
    if [ "$HAS_SHOT" -eq 0 ]; then
        MSG="You edited a SwiftUI view (\`$(basename "$SWIFT_UI")\`) but captured no simulator screenshot this session. Render it on a simulator and LOOK at the result before declaring done (RC-1). build-green != tested (ANTI-11)."
    elif [ "$HAS_CRITIC" -eq 0 ]; then
        MSG="You edited a SwiftUI view but never ran a design critic this session. Run \`/apple-design-critic\` (separate context — creator is not a credible self-critic) before done (RC-2). Clipped cards / notch collisions pass the build."
    fi
elif [ -n "$WEB_UI" ] && [ "$HAS_SHOT" -eq 0 ]; then
    MSG="You edited web UI (\`$(basename "$WEB_UI")\`) but captured no screenshot of the RUNNING dev server this session. Snapshot the rendered page (playwright/camoufox) and look at it before declaring done (RC-1). A build log is not the running thing."
fi
[ -n "$MSG" ] || exit 0

# Cap: <=2 blocks at the same HEAD, then give up (loop-safety, D-025 lineage).
SESS_DIR="$ROOT/.claude/sessions"
mkdir -p "$SESS_DIR" 2>/dev/null || exit 0
COUNTER="$SESS_DIR/${SESSION_ID}.evidencecount"
HEAD="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || echo no-git)"
PREV_HEAD=""; PREV_COUNT=0
if [ -f "$COUNTER" ]; then
    PREV_HEAD="$(cut -d' ' -f1 "$COUNTER" 2>/dev/null || echo)"
    PREV_COUNT="$(cut -d' ' -f2 "$COUNTER" 2>/dev/null || echo 0)"
fi
case "$PREV_COUNT" in ''|*[!0-9]*) PREV_COUNT=0 ;; esac
if [ "$PREV_HEAD" = "$HEAD" ]; then
    [ "$PREV_COUNT" -ge 2 ] && exit 0
    NEW_COUNT=$((PREV_COUNT + 1))
else
    NEW_COUNT=1
fi
printf '%s %s\n' "$HEAD" "$NEW_COUNT" > "$COUNTER" 2>/dev/null || true

REASON="Evidence gate: $MSG"
CONTEXT="$MSG  (Evidence block ${NEW_COUNT}/2 at this commit. Capture the artifact, cite it, then end.)"
jq -nc --arg reason "$REASON" --arg ctx "$CONTEXT" \
    '{decision:"block", reason:$reason, hookSpecificOutput:{additionalContext:$ctx}}'
exit 0
