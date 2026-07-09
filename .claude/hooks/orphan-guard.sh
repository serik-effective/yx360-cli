#!/usr/bin/env bash
# SessionStart + Stop — detect orphaned background processes (T3a / B4).
# Warn-only, never kills (ANTI-2). Branches on .hook_event_name.
set -euo pipefail

fail_open() { exit 0; }
trap fail_open ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
STDIN="$(cat)"

SESSION_ID="$(printf '%s' "$STDIN" | jq -r '.session_id // empty')"
EVENT="$(printf '%s' "$STDIN" | jq -r '.hook_event_name // empty')"
[ -n "$SESSION_ID" ] || exit 0

SESS_DIR="$ROOT/.claude/sessions"
mkdir -p "$SESS_DIR" 2>/dev/null || exit 0
PIDS="$SESS_DIR/${SESSION_ID}.pids"

case "$EVENT" in
    SessionStart)
        # Record this session's process-group leader so Stop can scope the scan.
        printf 'pgid=%s\n' "$(ps -o pgid= -p $$ 2>/dev/null | tr -d ' ' || echo)" > "$PIDS" 2>/dev/null || true
        exit 0
        ;;
    Stop|SubagentStop)
        : # fall through to scan
        ;;
    *)
        exit 0
        ;;
esac

SESS_PGID=""
if [ -f "$PIDS" ]; then
    SESS_PGID="$(grep -E '^pgid=' "$PIDS" 2>/dev/null | head -1 | cut -d= -f2 || echo)"
fi
# Without a recorded session PGID we cannot scope the scan; listing every system
# PPID=1 daemon would be pure noise, so stay silent.
[ -n "$SESS_PGID" ] || exit 0

ALLOW="${HARNESS_ORPHAN_ALLOW:-}"

# Best-effort: list processes reparented to init (PPID=1) that belong to this
# session's process group. Shared servers can be skipped via HARNESS_ORPHAN_ALLOW regex.
FOUND=""
while IFS= read -r line; do
    [ -n "$line" ] || continue
    pid="$(printf '%s' "$line" | awk '{print $1}')"
    ppid="$(printf '%s' "$line" | awk '{print $2}')"
    pgid="$(printf '%s' "$line" | awk '{print $3}')"
    cmd="$(printf '%s' "$line" | cut -d' ' -f4-)"
    [ "$ppid" = "1" ] || continue
    [ "$pid" != "$$" ] || continue
    [ "$pgid" = "$SESS_PGID" ] || continue
    if [ -n "$ALLOW" ] && printf '%s' "$cmd" | grep -qE "$ALLOW" 2>/dev/null; then
        continue
    fi
    FOUND="${FOUND}  ${pid}  ${cmd}"$'\n'
done < <(ps -ax -o pid=,ppid=,pgid=,command= 2>/dev/null || true)

if [ -n "$FOUND" ]; then
    PID_LIST="$(printf '%s' "$FOUND" | awk '{print $1}' | tr '\n' ' ' | sed 's/ *$//')"
    {
        echo "Orphaned background processes detected (PPID=1, this session):"
        printf '%s' "$FOUND"
        echo "Run \`kill ${PID_LIST}\` to stop them. (Not killed automatically — ANTI-2.)"
    } >&2
fi

exit 0
