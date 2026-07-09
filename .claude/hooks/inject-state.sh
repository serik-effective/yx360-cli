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

# 2) Last 5 decisions — full blocks (supports `## D-NNN` and `## ADR-N` styles;
#    the `[0-9]` guard skips literal template headers like `## ADR-N: <title>`)
if [ -f "$ROOT/.assistant/decisions.md" ]; then
    echo "=== RECENT DECISIONS (last 5, append-only log) ==="
    awk '
        /^## (D-|ADR-)[0-9]/ { n++; cur=n; blocks[cur]=$0 "\n"; next }
        /^## / && cur>0 { cur=0 }
        cur>0 { blocks[cur]=blocks[cur] $0 "\n" }
        END {
            start=n-N+1; if(start<1) start=1
            for(i=start;i<=n;i++) printf "%s", blocks[i]
        }
    ' N=5 "$ROOT/.assistant/decisions.md" | head -n 150
    echo ""
fi

# 3) Open questions — headers + priority groups (supports `## OQ-NNN` and `### Q<n>`
#    styles; resolved items written as `### ~~Q2 …~~` fall out naturally)
if [ -f "$ROOT/.assistant/open-questions.md" ]; then
    echo "=== OPEN QUESTIONS (unresolved design questions, see file for details) ==="
    { grep -E '^### (OQ-|Q)[0-9]|^## OQ-[0-9]|^## .*[Pp]riority|priority:' "$ROOT/.assistant/open-questions.md" || true; } | head -40
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
