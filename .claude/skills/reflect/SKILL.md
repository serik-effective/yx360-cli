---
name: reflect
description: Weekly/periodic reflection over the agent loop's own sessions. Extracts failure signals from Claude Code .jsonl transcripts (slop / no-pretest / ugly-ui / skipped-skill / redo-churn / unverified-facts), deep-reads the high-signal sessions, and proposes concrete harness fixes mapped to mechanisms (hook / skill-trigger / agent-prompt / decision). Carry-over stage verifies whether last cycle's shipped fixes actually moved their target signal. Use when the user says "проведи рефлексию", "reflect on this week's sessions", "weekly retro", or runs /reflect.
---

# Skill: /reflect

Periodic self-audit of the agent loop, run over its own past sessions. The ritual the hooks cannot replace: hooks change behaviour at the moment of action; `/reflect` catches the residue, instruments skill-skip rate, forces a periodic human raw-read, and verifies that shipped fixes actually worked.

Orchestrator only — fans out via Task into separate contexts (a scorer never sees another scorer's verdict; the critic context is never the loop that produced the work).

**When to invoke:** weekly/monthly retro over sessions; "проведи рефлексию по сессиям", "audit the agent loop", "did last week's fixes work".

**When NOT to invoke:** reviewing a specific PR/diff (`/audit`); debugging one failure (`/diagnose`); designing a feature (`/pre-feature`).

## Invocation

```
/reflect [window] [--scope <projects>]
```

- `window` — `7d` (default), `30d`, or `<a>..<b>` ISO dates.
- `--scope` — comma project substrings. Default = product work; always excludes harness self-dev and non-product dirs (leva, sample, tmp).

## Cadence (project decision — D-027)

**Weekly, full run.** The noise of a small weekly n (~23 flagged product sessions) is controlled NOT by reading a time-series (forbidden at this n) but by the **2-point carry-over check**: did last cycle's targeted signal drop, yes/no. The only longitudinal claim allowed.

## Signal taxonomy (frozen — the harness MAST). `anger` is a severity weight ∈{1.0,1.5,2.0}, never a signal.

| Signal | Failure (the act, not the feeling) | Tier |
|---|---|---|
| slop | human-facing answer exceeds density budget for question complexity | code |
| no_pretest | done-claim with no outcome artifact for the touched scope | code |
| ugly_ui | UI edit with no screenshot artifact, OR a visual complaint | code |
| skipped_skill | trigger present, matching skill never invoked | code |
| redo_churn | correction ≤2 turns after a done-claim | code + model confirm |
| unverified_facts | checkable external claim with no lookup/source | model, advisory |

## The load-bearing extraction primitive

Most `type:"user"` records are `tool_result`, not human turns. Every "what the user said" scorer runs ONLY on:

```
is_human_turn = type=="user" AND message.content is STRING
  AND not content.startswith("<command-") AND not a system-reminder-only payload
```

Session = one `.jsonl` (`sessionId`). Window = filter by `timestamp`. `gitBranch` + `cwd` give per-project + per-worktree attribution. **Hard rule:** no evidence → no finding. A `fail` cites ≥1 record `uuid` or auto-downgrades to `unknown` and is excluded from the backlog. The reference extractor is `scripts/reflect_extract.py` (pure code, no model calls).

## Orchestrator workflow (6 stages)

### Stage 1 — Carry-over (run FIRST, no new fixes until this is done)
Read the last `swarm-report/reflect-*.md` + `.assistant/decisions.md` + `.assistant/open-questions.md`. For each prior `D-NNN` reflection fix: did it land (mechanism present in repo)? did its predicted signal-count delta materialise this cycle? Mark landed / not-landed / no-move. A fix whose signal did not move in **2 cycles** → escalate or kill. No new proposals are generated until last cycle's are reconciled.

### Stage 2 — Threshold gate
Run `scripts/reflect_extract.py <window>`. Drop projects under N flags (default N=5). usmint:2 never fires; sales-vpo:184 always does.

### Stage 3 — Code extraction (no model)
The extractor emits `{session → {signal: verdict, evidence:[uuid], severity}}` using the `is_human_turn` primitive + the 4 code scorers (slop / no_pretest / ugly_ui / skipped_skill). Build trimmed digests of the top sessions for Stage 4.

### Stage 4 — Human raw-read (NOT delegable)
The orchestrator presents ~12–15 trimmed digests of the highest-signal sessions to the user (or reads raw traces itself) to sanity-check counts and catch what the scorers missed. This replaces any κ / confusion-matrix apparatus — a solo file-harness does not need it.

### Stage 5 — Model scorers (gated)
Fan out one Task per top session (separate critic context) for `redo_churn` confirm + `unverified_facts` advisory only, ONLY on code-pre-filtered candidates, each with a mandatory `Unknown` escape. Schema-forced structured output.

### Stage 6 — Pareto cut + write
Rank signals by `count × anger_weight`. Mandate **exactly 1–2 concrete harness mechanisms for the #1 signal only**; the rest go to backlog. Write the run report + fold the current-state taxonomy.

## Feedback-loop rule (the only rule that stops reflection theater)

> A `/reflect` run that produces only prose is rejected. Every finding must cite (a) an external artifact (git diff / test result / screenshot / signal-count with uuid), (b) a violated INVARIANT / H-rule ID, and (c) exactly one harness mechanism to change. Missing any of the three → finding auto-downgraded to `unverified`, excluded from the backlog.

Each finding routes to **exactly one** of four mechanism types (the executing agent never reads the retro — only the mechanism edit changes its next-session behaviour):

- **A** ugly_ui + no_pretest → extend `.claude/hooks/evidence-gate.sh`
- **B** slop + skipped_skill → `.claude/hooks/slop-gate.sh` / Stop+UserPromptSubmit triggers
- **C** unverified_facts → PostToolUse claim/dependency gate (extend `lint-gate.sh`)
- **D** redo_churn / trigger-tightening → skill-frontmatter `description` edits

Each shipped fix = one `D-NNN` carrying `{root_cause (5-whys), failure_signal, mechanism, verification step, predicted count delta}`. Next cycle's Stage 1 checks the prediction.

## Weekly metrics (auditor read only — NEVER exposed to executing agents as targets; Goodhart)

Denominator = sessions that did the *relevant* work.

| Cell | Metric | Formula |
|---|---|---|
| Effectiveness (headline) | rework_rate | (redo_churn + ugly_ui + no_pretest sessions) / sessions_that_wrote_code |
| Autonomy | intervention_rate | (redo_churn + factcheck + skipped_skill, anger-weighted) / human_turns |
| Quality | untested_ship_rate / ugly_ui_rate | no_pretest / wrote_code ; ugly_ui / touched_UI |
| Slop | slop_rate | slop turns / terminal_human_facing_turns |
| Process | skill_skip_rate (per skill) | skipped / sessions_with_that_trigger |

Report framing = a 4-cell balanced scorecard (Speed | Quality | Effectiveness | Slop). A green Speed cell next to a red Quality cell IS the finding.

## Outputs

- `swarm-report/reflect-<date>.md` — run report.
- `.assistant/failure-taxonomy.md` — one current-state doc, defrag-folded each cycle, never append-patched.
- `D-NNN` per shipped mechanism.

## Explicitly NOT produced (cut as theater for a solo file-harness)
Cohen's κ files, confusion-matrix corpus, per-failure regression fixtures, pass^k, a standing metrics time-series, an N-subagent judge swarm. Revisit only if the harness goes multi-dev / OSS.

## MVP vs full
- **MVP** = the moment-of-action hooks (`evidence-gate.sh`, `slop-gate.sh`) — they change behaviour next session with zero retro infrastructure. Primary.
- **Full** = this 6-stage skill + `reflect_extract.py` + the two gated model scorers. `/reflect` exists to catch the residue and force the human raw-read — not to re-discover a taxonomy that is already known.
