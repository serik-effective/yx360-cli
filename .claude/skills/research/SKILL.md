---
name: research
description: Deep web research via researcher + skeptic + reviewer agents. Confidence-flagged findings, auto-applies safe drift to memory-bank, surfaces risky drift for human review. Use when user asks "how does X work" / "what are best practices for Y" / "поресерчи".
---

# Skill: /research

Research consilium that produces confidence-flagged findings on external topics (libraries, protocols, providers, prior art, best practices). Triggered by request only — no cron.

**When to invoke:** user asks "what's best for X", "how do Y / Kiro / dae_codex work", "поресерчи / what's the current state of Z", "compare A vs B". Or before any architecture decision that depends on external facts.

**When NOT to invoke:** when answer is already in `.memory-bank/` AND `last_updated` < 30 days. Re-read memory bank first.

## Invocation

```
/research "<research question, one sentence>"
```

Optional second arg: depth (`shallow | normal | deep`). Default `normal`.

## Orchestrator workflow

### Step 1 — Check memory bank first

Search `.memory-bank/` for the topic. If found AND `Last updated:` line is within 30 days → return the memory bank answer with note "answered from memory, no web research needed; re-run with `--force` to web-research anyway". Skip rest of workflow.

If found but >30 days → carry forward as "prior context" into Step 2 prompt for researcher.

### Step 2 — Spawn 3 subagents in parallel

1. `researcher` — primary web research via `mcp-omnisearch`. MUST emit ≥1 finding per major sub-question, with confidence flags. Quote sources verbatim.
2. `skeptic` — second pass on researcher's findings AFTER they arrive (so this is actually a 2-phase fan-out, see Step 3). Looks for: stale sources, single-source claims dressed as `high` confidence, anecdotal blogs treated as authoritative, contradictions with prior decisions.
3. `reviewer` — cross-check findings against INVARIANTS §6 (30-day rule) and §9 (confidence flags).

### Step 3 — Two-phase fan-out

Phase A: spawn `researcher` first, wait for completion.
Phase B: spawn `skeptic` and `reviewer` in parallel, each given researcher's YAML output as input.

This ordering is intentional: skeptic and reviewer need real findings to critique, not the question.

### Step 4 — Aggregate

Group findings by sub-topic. For each finding:
- If `confidence: high` AND no skeptic/reviewer objection AND no contradiction with prior decision → **safe drift**, eligible for auto-apply to memory bank.
- If `confidence: medium` OR single-source → **risky drift**, requires human review.
- If `confidence: low | unverified` → **note only**, never auto-apply.
- If contradicts a prior decision (D-NNN) → **conflict**, surface explicitly with both perspectives.

### Step 5 — Auto-apply safe drift (orchestrator-only edits)

This is the **one place** orchestrator IS allowed to write into memory bank (overrides INVARIANT §4 soft edit-guard, by skill design):
- Update `last_updated:` dates in `.memory-bank/tech-details/` files
- Append source URLs to relevant files
- NEVER change a decision in `.assistant/decisions.md` — additions only, via Step 6

Each auto-apply gets a one-line note in the swarm-report indicating what was written and why.

### Step 6 — Surface to user

Write report to `swarm-report/research-<slug>-<YYYY-MM-DD>.md` with:
- TL;DR (top 3 findings)
- Safe drift applied (what got written into memory bank, with diff)
- Risky drift (findings needing human approval — ask user)
- Conflicts with prior decisions (if any — both sides shown)
- Full researcher YAML
- Full skeptic + reviewer critiques

Surface to chat:
- Report path
- TL;DR verbatim
- Question (if any risky drift or conflict): `AskUserQuestion` with specific drift to apply/reject

## Loop guards

- Max 2 research passes per topic per session. If `/research "X"` invoked 3rd time → emit "topic over-researched; consult human or check `.assistant/open-questions.md`".
- Researcher returns 0 findings → emit "no results" report; do not re-spawn (zero results IS a finding).
- Skeptic flags all of researcher's findings as low-confidence → emit "research-inconclusive", surface explicitly, do not auto-apply anything.

## Example invocations

```
/research "best practices for auto-updating dotfile-style developer setups in 2026"
```
Closes OQ-001 / OQ-002.

```
/research "compare AWS Kiro three-file spec vs GitHub spec-kit for Effective Harness adoption"
```
Updates `.memory-bank/tech-details/existing-solutions.md`.

```
/research "current rate-limit and pricing for Anthropic API on Opus 4.7 tier"
```
Surfaces cost-watcher data.

## What this skill does NOT do

- Does not execute code or make API calls beyond web search
- Does not modify `.assistant/decisions.md` (orchestrator can only append decisions after explicit user approval — Step 6 question)
- Does not pull in MCP server changes / install dependencies
