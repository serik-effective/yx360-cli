---
name: pre-feature
description: Run a consilium (architect + skeptic + researcher + reviewer) on a proposed feature or design change. Writes a deduped plan to swarm-report/. Use BEFORE writing code, when scope is unclear, or for any non-trivial design decision.
---

# Skill: /pre-feature

Multi-agent consilium that validates a feature proposal before code is written. Returns a structured plan with findings from four independent perspectives, dedup'd at orchestrator level.

**When to invoke:** user asks for a new feature, refactor, or design decision that touches ≥2 files OR violates an existing pattern OR requires external research. Skip for trivial changes (typo, single-line fix, renames).

**When NOT to invoke:** debugging (`/diagnose`), reviewing existing code (`/audit`), implementing an already-approved plan (`/implementor`).

## Invocation

```
/pre-feature "<one-sentence feature description>"
```

Optional second argument: scope hint (`harness | infra | ui | research`). If omitted, orchestrator infers from description.

## Orchestrator workflow (this skill IS the orchestrator)

### Step 1 — Validate inputs

Read in order:
1. `.assistant/INVARIANTS.md`
2. `.memory-bank/product-overview/anti-stories.md`
3. `.memory-bank/product-overview/pipeline-stages.md`
4. `.assistant/decisions.md` (full file, append-only is short)
5. `.assistant/open-questions.md` — does proposal touch any OQ?

If the proposal description itself violates an INVARIANT or hard-stop in AGENTS.md → abort with explanation; don't spawn consilium.

### Step 2 — Spawn 4 subagents in parallel (single message, 4 Task calls)

Use the `Task` tool with `subagent_type: general-purpose` (until dedicated agent types exist). In each prompt include:
- The feature description verbatim
- Path references for the subagent to read (INVARIANTS, anti-stories, decisions, relevant memory-bank files)
- Explicit reminder: "Output strict YAML per `.Codex/agents/<role>.md` schema. No prose. Cite file:line for every finding. Empty array allowed."

The four subagents:
1. `architect` — read `.Codex/agents/architect.md`, focus on module boundaries, pipeline stage fit, SOLID, migration path
2. `skeptic` — read `.Codex/agents/skeptic.md`, find flaws / hidden costs / scope creep / invariant violations
3. `researcher` — read `.Codex/agents/researcher.md`, find external prior art and best practices via `mcp-omnisearch`. Confidence flags mandatory.
4. `reviewer` — read `.Codex/agents/reviewer.md`, cross-check against INVARIANTS / anti-stories / prior decisions

### Step 3 — Aggregate (orchestrator only)

Collect all four YAML outputs. Dedupe by `(file, line, category, problem-similarity)`. Sort by severity (HIGH first), then category.

Group into report sections:
- **TL;DR** — counts per severity, top 3 must-fix items
- **Blockers** (HIGH severity, requires_human: true)
- **Concerns** (MEDIUM)
- **Notes** (LOW)
- **Research findings** (from `researcher`, with confidence flags)
- **Out-of-scope (declared)** — anything subagents explicitly excluded
- **Open questions raised** — new OQs to add to `.assistant/open-questions.md`
- **Per-agent verbatim sections** (architect findings, skeptic findings, researcher findings, reviewer findings — for audit trail)

### Step 4 — Write report

Append-only write to `swarm-report/<slug>-plan-<YYYY-MM-DD>.md`. Slug from feature description (lowercase, hyphen-joined, ≤6 words).

Report has no Status: Done. Status is `consilium-complete` or `consilium-rejected` (if Step 1 aborted).

### Step 5 — Surface to user

Output to chat:
- One-line slug + path to report
- TL;DR section verbatim
- Blockers list (if any)
- Question: "Proceed to `/implementor <slug>` or revise scope?"

**Do NOT** auto-spawn `/implementor`. Human gate is mandatory (per `.memory-bank/steerings/project-types.md` — even Type 1 prototypes get a review checkpoint at plan-time).

## Loop guards

- Max 1 replan per feature. If user calls `/pre-feature` twice with the same slug within 24h, orchestrator emits a single finding: `feature over-scoped, split into smaller features`. No third spawn.
- Subagent prose-output detected → orchestrator re-spawns that subagent ONCE with stricter prompt. If second attempt still produces prose → emit `agent-failed: <role>` in report and proceed without that subagent's contribution.
- Empty findings from all four subagents → emit `consilium-found-nothing` with explicit note that user must decide whether plan is trivially-safe or scope was too narrow.

## What this skill does NOT do

- Does not write code (use `/implementor`)
- Does not modify `.memory-bank/` or `.assistant/` directly (orchestrator soft edit-guard, INVARIANT §4)
- Does not auto-merge findings into prior decisions (human review only)
- Does not search the codebase for "all files that might be affected" — `architect` does that within its task scope

## Example invocation

```
/pre-feature "add /defrag skill for memory-bank-defrag cron integration"
```

Expected output: report at `swarm-report/add-defrag-skill-plan-2026-06-04.md` with 4-agent findings, TL;DR, blockers if any, question about next step.
