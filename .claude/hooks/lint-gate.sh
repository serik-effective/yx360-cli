#!/usr/bin/env bash
# PostToolUse (Edit|Write|MultiEdit) — surface the declared lint command for the
# file just edited (T1/T6). Detect-and-surface only: never auto-runs lint, O(1).
set -euo pipefail

fail_open() { exit 0; }
trap fail_open ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
STDIN="$(cat)"

# Read the declared lint command; no-op if absent (§11 — no hardcoded eslint/ruff/tsc).
[ -f "$ROOT/.assistant/verify-commands.json" ] || exit 0
LINT_CMD="$(jq -r '.lint // empty' "$ROOT/.assistant/verify-commands.json" 2>/dev/null || echo)"
[ -n "$LINT_CMD" ] || exit 0

FILE="$(printf '%s' "$STDIN" | jq -r '.tool_input.file_path // empty')"
[ -n "$FILE" ] || exit 0

REL="${FILE#$ROOT/}"
echo "Lint reminder: you edited '$REL'. Run the declared lint command (\`$LINT_CMD\`) before declaring done — do not skip it or use --no-verify." >&2

exit 0
