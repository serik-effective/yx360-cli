#!/usr/bin/env bash
# Stop — density/slop gate (D-029, RC-3 from reflect 2026-06-29).
# Conservative by design (shared guardrail -> false positives are expensive):
# fires ONLY when the terminal human-facing answer is BOTH over the deep budget
# (>4000 chars) AND carries >=3 high-precision slop markers AND the preceding
# human turn did NOT ask for a long deliverable. Then it requires a de-slop pass.
# Density scorer only — never an LLM "is this good?" (LLM judges reward verbosity).
# Loop-safe: stop_hook_active short-circuit, <=1 block per HEAD, fail-open.
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

LAST_ANSWER="$(jq -rs 'map(select(.type=="assistant")) | last // empty
  | .message.content[]? | select(.type=="text") | .text' "$TRANSCRIPT" 2>/dev/null || echo)"
[ -n "$LAST_ANSWER" ] || exit 0

CHARS="$(printf '%s' "$LAST_ANSWER" | wc -m | tr -d ' ')"
[ "$CHARS" -gt 4000 ] || exit 0

# Did the human ask for length? Then a long answer is not slop.
LAST_USER="$(jq -rs 'map(select(.type=="user" and (.message.content|type=="string"))) | last // empty
  | .message.content' "$TRANSCRIPT" 2>/dev/null || echo)"
if printf '%s' "$LAST_USER" | grep -qiE 'подробн|детальн|разверн|полностью|по полной|\bfull\b|\bdraft\b|документ|長|длинн|весь текст|целиком'; then
    exit 0
fi

# High-precision slop markers (curated subset; full list:
# .claude/skills/anti-ai-slop-writing/references/banned-words.md).
MARKERS='delve|tapestry|moreover|furthermore|it'\''s important to note|it is important to note|navigating the|in the realm of|underscores the|a testament to|ever-evolving|seamless|robust solution|elevate your|in today'\''s|стоит отметить|важно понимать|в современном мире|не только.*но и|играет важную роль|следует учитывать|таким образом,'
BANNED="$(printf '%s' "$LAST_ANSWER" | grep -oiE "$MARKERS" | wc -l | tr -d ' ')"
[ "$BANNED" -ge 3 ] || exit 0

# Cap: <=1 block per HEAD (slop nudge needs only one shot).
SESS_DIR="$ROOT/.claude/sessions"; mkdir -p "$SESS_DIR" 2>/dev/null || exit 0
COUNTER="$SESS_DIR/${SESSION_ID}.slopcount"
HEAD="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || echo no-git)"
[ -f "$COUNTER" ] && [ "$(cat "$COUNTER" 2>/dev/null)" = "$HEAD" ] && exit 0
printf '%s\n' "$HEAD" > "$COUNTER" 2>/dev/null || true

REASON="Slop gate: the answer is ${CHARS} chars with ${BANNED} slop markers and the user did not ask for a long deliverable. Run /anti-ai-slop-writing on it: cut to the density the question warrants, drop the marker phrases. Lead with the answer."
jq -nc --arg reason "$REASON" '{decision:"block", reason:$reason}'
exit 0
