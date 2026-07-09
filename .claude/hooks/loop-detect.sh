#!/usr/bin/env bash
# PostToolUse — detect repeating-error / identical-action loops (T2).
# Soft notice only: PostToolUse cannot block. Writes to .claude/sessions/<id>.loopstate.
set -euo pipefail

fail_open() { exit 0; }
trap fail_open ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
STDIN="$(cat)"

SESSION_ID="$(printf '%s' "$STDIN" | jq -r '.session_id // empty')"
TOOL_NAME="$(printf '%s' "$STDIN" | jq -r '.tool_name // empty')"
[ -n "$SESSION_ID" ] || exit 0
[ -n "$TOOL_NAME" ] || exit 0

SESS_DIR="$ROOT/.claude/sessions"
mkdir -p "$SESS_DIR" 2>/dev/null || exit 0
STATE="$SESS_DIR/${SESSION_ID}.loopstate"

sha256() {
    if command -v sha256sum >/dev/null 2>&1; then sha256sum | awk '{print $1}'
    else shasum -a 256 | awk '{print $1}'; fi
}

# Normalize tool_input (compact, key-sorted) so whitespace-only diffs don't dodge detection.
NORM_INPUT="$(printf '%s' "$STDIN" | jq -cS '.tool_input // {}' 2>/dev/null || printf '{}')"

# Error indicator from tool_response: presence of error + a short error string.
ERR_FLAG="$(printf '%s' "$STDIN" | jq -r '
  (.tool_response // {}) as $r
  | if ($r|type)=="object" then
      ((($r.is_error // $r.isError // false)|tostring) + "|" + (($r.error // $r.stderr // "")|tostring))
    else ($r|tostring) end' 2>/dev/null || printf 'false|')"
# Truncate error string so transient detail (timestamps, pids) noise is bounded.
ERR_FLAG="$(printf '%s' "$ERR_FLAG" | cut -c1-200)"

SIG="$(printf '%s\n%s\n%s' "$TOOL_NAME" "$NORM_INPUT" "$ERR_FLAG" | sha256)"
# Separate error-string signature for the "same error N×" rule (input may vary, error same).
ERR_ONLY="$(printf '%s' "$ERR_FLAG" | cut -d'|' -f2-)"
ERR_SIG=""
if [ -n "$ERR_ONLY" ] && [ "$ERR_ONLY" != "" ]; then
    ERR_SIG="$(printf '%s' "$ERR_ONLY" | sha256)"
fi

# Ring buffer: keep last 12 lines "<sig> <errsig>".
printf '%s %s\n' "$SIG" "$ERR_SIG" >> "$STATE"
tail -n 12 "$STATE" > "$STATE.tmp" 2>/dev/null && mv "$STATE.tmp" "$STATE" 2>/dev/null || true

SAME_SIG="$(awk -v s="$SIG" '$1==s{c++} END{print c+0}' "$STATE")"
SAME_ERR=0
if [ -n "$ERR_SIG" ]; then
    SAME_ERR="$(awk -v e="$ERR_SIG" '$2==e{c++} END{print c+0}' "$STATE")"
fi

if [ "$SAME_SIG" -ge 3 ]; then
    echo "LOOP DETECTED: tool '$TOOL_NAME' ran ${SAME_SIG}x with the same input and same result. Stop retrying the identical action — change approach or ask the user." >&2
elif [ "$SAME_ERR" -ge 5 ]; then
    echo "LOOP DETECTED: the same error has recurred ${SAME_ERR}x across recent tool calls. Stop retrying — change approach or ask the user." >&2
fi

exit 0
