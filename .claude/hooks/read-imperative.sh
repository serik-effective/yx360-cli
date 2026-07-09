#!/usr/bin/env bash
# UserPromptSubmit — when the user tells you to read/research a specific file or
# source, remind the model to actually Read it instead of answering from memory
# (T5 / OQ-010). Injects one line via stdout. Never blocks.
set -euo pipefail

fail_open() { exit 0; }
trap fail_open ERR

command -v jq >/dev/null 2>&1 || exit 0

STDIN="$(cat)"
PROMPT="$(printf '%s' "$STDIN" | jq -r '.prompt // empty')"
[ -n "$PROMPT" ] || exit 0

# Imperative to read/research (RU + EN).
IMP='читай|почитай|прочитай|поресёрч|поресерч|поресёрчь|поресёрчи|[Rr]ead the|[Rr]ead this|[Rr]ead [^ ]*\.(file|doc)|look at'
# Reference to a concrete path or @file.
REF='@[A-Za-z0-9._/-]+|[A-Za-z0-9._/-]+\.[A-Za-z0-9]+|[A-Za-z0-9_./-]+/[A-Za-z0-9_.-]+'

if printf '%s' "$PROMPT" | grep -qE "$IMP" 2>/dev/null \
   && printf '%s' "$PROMPT" | grep -qE "$REF" 2>/dev/null; then
    echo "Reminder: Read the referenced file(s) with the Read tool before answering — do not answer from memory (OQ-010/T5)."
fi

exit 0
