---
name: diagnose
description: Bug-hunting orchestrator. Given an error, failing test, stuck order, or "X stopped working", drives a hypothesis-→-repro-→-evidence-→-root-cause loop. Fans out to `diagnostics` + domain-specialist agents (e.g. `scraping-diagnostician`, `apple-platform-debugger`) based on symptom. Returns a diagnosis report with hypothesis ladder and the minimum fix scope — does NOT auto-fix.
---

# Skill: /diagnose

Bug-hunting consilium. Sibling to `/audit` (looks at code) and `/pre-feature` (looks at proposals). Orchestrator only.

**When to invoke:** user describes a specific failure — error message, stack trace, failing test, "X used to work and stopped", stuck order / pending state, log entry that shouldn't exist. Anything where root cause is the question.

**When NOT to invoke:** the user wants to design a new feature (`/pre-feature`); the user wants to review unchanged code (`/audit`); the user wants to implement a known fix (`/implementor`).

## Invocation

```
/diagnose "<symptom in plain language>"
```

Optional second arg: `--scope <path|service|component>` to narrow the suspected blast area.

Optional `--no-repro` if reproduction is impossible (production-only intermittent) — diagnostician falls back to log archaeology.

## Orchestrator workflow

### Step 1 — Capture the symptom precisely

Read the user's invocation and ask AT MOST one clarifying question via `AskUserQuestion` if the symptom is ambiguous (e.g. "which environment? prod / staging / local"). Otherwise proceed.

Read:
1. `.assistant/INVARIANTS.md`
2. `.memory-bank/runbooks/` (if present — prior incidents may already document this class)
3. `.memory-bank/tech-details/stack.md`
4. The last 3 D-NNN entries (recent changes are the most likely cause)
5. `git log --since='7 days ago' --oneline` (recent commits in scope)

### Step 2 — Classify symptom → pick the agent crew

Default lead: `diagnostics` (read `.claude/agents/diagnostics.md`).

Add domain specialists by symptom keywords:
- scraping / blocked / Cloudflare / DataDome / Akamai / "stuck pending" → `scraping-diagnostician` (and `anti-bot-evasion` standby).
- iOS / macOS / simulator / crash log → `apple-platform-debugger`.
- API contract / 500 / 4xx repeating → `api`.
- Auth / secrets / token leak → `security`.
- Slow query / N+1 / OOM → `architect` + `diagnostics`.
- Failing deploy / Inactive Lambda / build pipeline → `devops`.

If `--no-repro`: skip the repro step; rely entirely on log evidence + the architect / domain agent.

### Step 3 — Drive the hypothesis loop

Per iteration (cap at 3 rounds):
1. **Hypothesis** — diagnostics agent proposes the single most likely cause based on current evidence. Strict YAML: `hypothesis`, `evidence_so_far`, `disproof_test`, `confidence`.
2. **Disproof test** — orchestrator runs the agent's disproof command (read-only by default — log query, file read, grep). For mutating diagnostics (toggle a flag, replay a request), surface to user and require explicit "y" first.
3. **Outcome** — evidence updated. If hypothesis is refuted → next round with new hypothesis. Confirmed → exit loop.

Stop conditions:
- Hypothesis confirmed with ≥ 2 independent evidences.
- 3 rounds elapsed without confirmation → emit `inconclusive` and document the surviving 1–2 candidate hypotheses.
- An agent flags an INVARIANT violation as root cause → exit immediately, that's the answer.

### Step 4 — Distinguish similar failure modes

Common confusables — always check both before declaring root cause:
- Soft block (anti-bot) vs real backend pending (use a control-account probe).
- Token expired vs token revoked vs token-issuer down.
- Test flake vs real regression (re-run N times with `--seed` or matrix).
- "Service down" vs "we can't reach it" (probe from a clean network).
- New code bug vs old data row that violates a new invariant.

### Step 5 — Output the diagnosis report

Append-only write to `swarm-report/diagnose-<symptom-slug>-<YYYY-MM-DD>.md`. Sections:
- **TL;DR** — root cause in 1–2 sentences + the file:line or service responsible.
- **Hypothesis ladder** — every hypothesis tried, with evidence and verdict.
- **Minimum fix scope** — list of files / lines / config keys that need to change. Cite each.
- **What this is NOT** — explicitly rule out the misleading hypotheses, so future you doesn't re-investigate.
- **Suggested next step** — `/pre-feature` (if scope is non-trivial / architectural) OR `/implementor` (if a clean approved fix exists in an existing plan) OR direct edit (only for ≤2-file mechanical fix).
- **Open issues raised** — anything that becomes a new OQ.

### Step 6 — Surface to user

Output:
- TL;DR root cause.
- Minimum fix scope (cited).
- Path to the report.
- Question: "Apply the fix via `/pre-feature` (design first) or `/implementor` (use existing plan) or direct edit?"

**Do NOT** auto-fix even when the fix is one line. Diagnose is read + reason, never write.

## Loop guards

- Symptom too vague after one clarification → emit "need a repro / stack trace / failing test name; cannot diagnose without it".
- 3 rounds with no confirmed hypothesis → write `inconclusive` report, surface candidate list.
- Mutating disproof test requested → require explicit user "y" before executing.
- Same symptom slug re-run within 1h → re-attach to the prior report (append a new round), don't re-spawn the full crew.
- Agent suggests "just restart it" without explaining WHY it would work → reject; loop another round.

## What this skill does NOT do

- Does not write the fix.
- Does not commit / push / open PR.
- Does not modify the running system (logs queries only; mutations require explicit user "y").
- Does not auto-close the symptom — user marks resolved.
- Does not skip log archaeology even if the failure is intermittent — log evidence is the floor.

## Example invocation

```
/diagnose "order d34e1a9f stuck PENDING for 3h, checks_count not incrementing"
/diagnose "ToolUseError: schema validation in workflow wf_3f5c5d1c-3a0"
/diagnose "simulator crashes immediately after launch on iOS 26" --scope ios
```

Expected behavior: pick lead + specialist agents, run hypothesis-ladder ≤ 3 rounds, write `swarm-report/diagnose-<slug>-<date>.md`, surface root cause + minimum fix scope + next-step question.
