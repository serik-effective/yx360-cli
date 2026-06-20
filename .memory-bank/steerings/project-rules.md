# Project Rules

> These rules are specific to Effective Harness. They live alongside `AGENTS.md` (full working agreement) and `.assistant/INVARIANTS.md` (12 hard rules). All three are self-contained within this repository — no external `CLAUDE.md` is required.

## Philosophy (recap)

The full philosophy is in `AGENTS.md` under "Philosophy (NON-NEGOTIABLE)". Short list:

- Accuracy > speed
- Verify, don't assume
- Disagree loudly
- Push back on flaky premises
- Structure over chaos
- No bullshit
- Don't fake completion
- Eat your own dog food

## Harness-specific

### Runtime-agnostic

The harness runs on Claude Code / OpenCode / Codex. Any feature that requires binding to a specific runtime must have a fallback for the others or be explicitly documented as `claude-code-only` / `codex-only`.

### Memory bank — source of truth

The memory bank is the source of truth, above any internal agent knowledge. If an agent "remembers" one thing but the memory bank says another — trust the memory bank, update the internal model.

If you spot a divergence between code and memory bank — update the memory bank (or surface it for human review).

### Agents must have internet

Every agent making architectural / technological / legal decisions must have MCP access to web search. Without internet, only narrow executing agents working from an approved plan.

### Stack agnosticism

The harness does not dictate a tech stack. Each project declares its stack in `STACK.md` or in a `STACK` section of its own `CLAUDE.md`. Agents read it before working.

### Profile routing

The harness ships task-based skills (`.claude/skills/*`) and per-role agents (`.claude/agents/*`). The consuming project's `CLAUDE.md` may define additional routing (`keywords → role → agent`) on top of what the harness provides.

### Pipeline discipline

The pipeline in `pipeline-stages.md` defines **allowed transitions**, not a "ladder downward." Skipping stages is forbidden. If you feel a stage can be skipped (e.g., "no architecture needed for a small feature"), check `project-types.md` — it might be a Type 1 project where the gate is auto-pass.

### No fake completion

`Status: Done` is set **only** after stage 7 (Done). Intermediate "done" → `Status: stage-N-complete`. No "seems to work" in final reports.

### Reports always

Every feature in a Type 2 project ends with a report in `./swarm-report/<slug>-<YYYY-MM-DD>.md` (format defined in `AGENTS.md` → Validation pipeline). Without a report, Done is not Done.

### Caveman-aware

The user may activate caveman mode in their terminal (a token-saving compressed chat mode). All agent prompts must be robust to caveman-style messages from the user. Agents write technical content normally; conversational prose is terse.

### Profile selection

The harness uses task-based skills (`/pre-feature`, `/research`, `/implementor`, etc.) — never project-named profiles. The consuming project may declare its own profile routing in its `CLAUDE.md`, but routing always maps to harness skills + agents.

## Related

- [Project Types](project-types.md) — Type 1 / Type 2 distinction drives gates
- [Vision](../product-overview/vision.md)
- [Anti-Stories](../product-overview/anti-stories.md) — boundaries of the harness
