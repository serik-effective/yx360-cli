#!/usr/bin/env bash
# UserPromptSubmit — runaway-cost guard (T3b / B4).
# Observe-mode by default: blocks the next prompt only when a threshold is declared
# in .assistant/limits.json and session spend has reached it. Human decides.
set -euo pipefail

fail_open() { exit 0; }
trap fail_open ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
STDIN="$(cat)"

# Threshold: absent file or absent key → observe-mode (never block).
[ -f "$ROOT/.assistant/limits.json" ] || exit 0
THRESHOLD="$(jq -r '.max_session_usd // empty' "$ROOT/.assistant/limits.json" 2>/dev/null || echo)"
[ -n "$THRESHOLD" ] || exit 0

SPENT="$(printf '%s' "$STDIN" | jq -r '.cost.total_cost_usd // empty')"
[ -n "$SPENT" ] || exit 0

# Numeric compare via awk (bash has no float math). Block when spent >= threshold.
OVER="$(awk -v s="$SPENT" -v t="$THRESHOLD" 'BEGIN{print (s+0 >= t+0) ? 1 : 0}')"
if [ "$OVER" = "1" ]; then
    printf 'Cost cap $%s reached (spent $%s). Confirm to continue.\n' "$THRESHOLD" "$SPENT" >&2
    exit 2
fi

exit 0
