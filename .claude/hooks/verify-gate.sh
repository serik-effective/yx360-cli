#!/usr/bin/env bash
# Stop — enforce ANTI-11: cite verify output before declaring done (T1).
# Loop-safe (B2): short-circuits on stop_hook_active, caps blocks at ≤3 per git HEAD.
set -euo pipefail

fail_open() { exit 0; }
trap fail_open ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
STDIN="$(cat)"

# B2: never re-block once we are already inside a Stop-hook continuation.
STOP_ACTIVE="$(printf '%s' "$STDIN" | jq -r '.stop_hook_active // false')"
[ "$STOP_ACTIVE" = "true" ] && exit 0

SESSION_ID="$(printf '%s' "$STDIN" | jq -r '.session_id // empty')"
[ -n "$SESSION_ID" ] || exit 0

# Resolve the project's declared verify command — no-op if none declared (§11).
VERIFY_CMD=""
if [ -f "$ROOT/.assistant/verify-commands.json" ]; then
    VERIFY_CMD="$(jq -r '.verify // empty' "$ROOT/.assistant/verify-commands.json" 2>/dev/null || echo)"
fi
if [ -z "$VERIFY_CMD" ] && [ -f "$ROOT/Makefile" ]; then
    grep -qE '^verify:' "$ROOT/Makefile" 2>/dev/null && VERIFY_CMD="make verify"
fi
if [ -z "$VERIFY_CMD" ] && [ -f "$ROOT/package.json" ]; then
    NPM_VERIFY="$(jq -r '.scripts.verify // empty' "$ROOT/package.json" 2>/dev/null || echo)"
    [ -n "$NPM_VERIFY" ] && VERIFY_CMD="npm run verify"
fi
[ -n "$VERIFY_CMD" ] || exit 0

SESS_DIR="$ROOT/.claude/sessions"
mkdir -p "$SESS_DIR" 2>/dev/null || exit 0
COUNTER="$SESS_DIR/${SESSION_ID}.blockcount"

HEAD="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || echo no-git)"

# B2 cap: ≤3 blocks at the same HEAD, then give up.
PREV_HEAD=""
PREV_COUNT=0
if [ -f "$COUNTER" ]; then
    PREV_HEAD="$(cut -d' ' -f1 "$COUNTER" 2>/dev/null || echo)"
    PREV_COUNT="$(cut -d' ' -f2 "$COUNTER" 2>/dev/null || echo 0)"
fi
case "$PREV_COUNT" in ''|*[!0-9]*) PREV_COUNT=0 ;; esac

if [ "$PREV_HEAD" = "$HEAD" ]; then
    [ "$PREV_COUNT" -ge 3 ] && exit 0
    NEW_COUNT=$((PREV_COUNT + 1))
else
    NEW_COUNT=1
fi
printf '%s %s\n' "$HEAD" "$NEW_COUNT" > "$COUNTER" 2>/dev/null || true

REASON="Verify gate: run the project verify command (\`$VERIFY_CMD\`) and cite its output before declaring done (ANTI-11). Do not bypass with --no-verify."
CONTEXT="A verify command is declared for this project: \`$VERIFY_CMD\`. Before ending, run it and paste the result. If you already ran it this turn, restate the exact output. Block ${NEW_COUNT}/3 at this commit."

jq -nc --arg reason "$REASON" --arg ctx "$CONTEXT" \
    '{decision:"block", reason:$reason, hookSpecificOutput:{additionalContext:$ctx}}'

exit 0
