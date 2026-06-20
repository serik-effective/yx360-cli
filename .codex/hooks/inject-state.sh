#!/usr/bin/env bash
# SessionStart hook — inject project state into agent context.
# Budget: ~3-5K tokens. Stays silent if files missing.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

echo "<effective-harness-state>"
echo ""

# 1) INVARIANTS — load-bearing rules
if [ -f "$ROOT/.assistant/INVARIANTS.md" ]; then
    echo "=== INVARIANTS (hard rules — every subagent must respect) ==="
    cat "$ROOT/.assistant/INVARIANTS.md"
    echo ""
fi

# 2) Last 5 decisions
if [ -f "$ROOT/.assistant/decisions.md" ]; then
    echo "=== RECENT DECISIONS (last 5, append-only log) ==="
    # grep decision headers, take last 5, then expand each
    awk '/^## D-/{c++; if(c>last5_count-5 && c<=last5_count) print}' last5_count=$(grep -c '^## D-' "$ROOT/.assistant/decisions.md") "$ROOT/.assistant/decisions.md" 2>/dev/null || true
    # fallback: just tail the file
    tail -n 60 "$ROOT/.assistant/decisions.md"
    echo ""
fi

# 3) Open questions — concise
if [ -f "$ROOT/.assistant/open-questions.md" ]; then
    echo "=== OPEN QUESTIONS (unresolved design questions, see file for details) ==="
    { grep -E '^## OQ-|priority:' "$ROOT/.assistant/open-questions.md" || true; } | head -40
    echo ""
fi

# 4) Memory bank index hint
if [ -f "$ROOT/.memory-bank/index.md" ]; then
    echo "=== MEMORY BANK INDEX (read .memory-bank/index.md for navigation) ==="
    { grep -E '^- \[' "$ROOT/.memory-bank/index.md" || true; } | head -30
    echo ""
fi

# 5) Git status (light)
if command -v git >/dev/null 2>&1 && [ -d "$ROOT/.git" ]; then
    echo "=== GIT STATUS ==="
    git -C "$ROOT" branch --show-current 2>/dev/null || true
    git -C "$ROOT" status --short 2>/dev/null | head -20 || true
    echo ""
    echo "=== RECENT COMMITS ==="
    git -C "$ROOT" log --oneline -5 2>/dev/null || true
    echo ""
fi

echo "</effective-harness-state>"

exit 0
