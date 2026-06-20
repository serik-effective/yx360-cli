---
name: post-feature
description: Close out an implemented feature — append D-NNN to decisions log, update affected memory-bank files, close resolved open questions, draft commit message + PR body, surface anything that became a new open question. Use AFTER /implementor reports `status: complete`.
---

# Skill: /post-feature

Memory-bank + decisions-log curator that runs AFTER `/implementor`. Orchestrator only — fan out to docs / architect / reviewer agents for the actual writes.

**When to invoke:** `/implementor <slug>` returned `status: complete` (or `verify-failed` if the user explicitly waived). Plan + implementation reports exist in `swarm-report/`.

**When NOT to invoke:** implementation isn't done; verify still failing without waiver; trivial one-line fix (commit normally, skip the skill).

## Invocation

```
/post-feature <slug>
```

Optional second arg: `--no-commit-draft` if the user prefers to write commit / PR text by hand.

## Orchestrator workflow

### Step 1 — Locate the artifacts

Glob `swarm-report/<slug>-plan-*.md` and `swarm-report/<slug>-implementation-*.md` — latest dates win. Both missing → abort with explicit path tried.

Read both. Capture:
- Plan: chosen design, alternatives rejected, raised OQs.
- Implementation: status, files touched, verify results, open issues raised.

Read `.assistant/decisions.md` (last 5 D-NNN entries) for numbering and style parity.
Read `.assistant/open-questions.md` — which OQs did this feature resolve, partially address, or raise?

### Step 2 — Compute the deltas

For each affected `.memory-bank/` area, decide one of:
- **append** — the new feature adds a node; add a section.
- **rewrite** — implementation invalidated the prior wording; rewrite the section.
- **link** — only a cross-link from an existing doc.
- **none** — implementation is purely internal, no doc change.

Mark every memory-bank file that the implementation touched conceptually (not just by file path). Pipeline-stage docs / glossary / tech-details/stack.md / architecture-decisions/ are the usual suspects.

### Step 3 — Fan out updates (single layer, parallel)

Spawn Task tool calls in parallel for every `(memory-bank-file, action)` pair:
- Each agent reads the plan + implementation + the file it owns.
- Each agent edits ONLY its assigned file. No cross-file writes.
- Output contract: strict YAML — `path`, `action`, `lines_changed`, `summary`, `cross_links_added`.

Default agent: `general-purpose` (until dedicated `docs` agent ships). Use `architect` for architecture-decisions/, `diagnostics` for runbooks/.

### Step 4 — Append D-NNN to `.assistant/decisions.md`

Orchestrator writes directly (D-NNN append is an INVARIANT §4 exception). Required fields:
- `## D-NNN — <one-line headline>`
- `**Date:**` (project timezone date)
- `**Status:** accepted`
- `**Decision:**` — what was actually built, in 2–4 sentences.
- `**Why now:**` — the motivation, including any user-stated constraint.
- `**Alternatives rejected:**` — pull from plan; one bullet per alternative + why.
- `**Source:**` — cite plan + implementation report paths, and any consilium workflow IDs.
- `**Closes:**` — OQ IDs resolved (use exact `OQ-NNN`).
- `**Raises:**` — new OQ IDs (if any) — and append the matching `OQ-NNN` skeletons to `.assistant/open-questions.md`.

### Step 5 — Close / amend open questions

For every OQ this feature touched:
- **Fully closed** — mark `**Status:** closed by D-NNN on <date>`. Do not delete the entry; keep history.
- **Partially closed** — append `**Update <date>:** <summary>`; leave status as `open` with revised next step.
- **Newly raised** — create OQ skeleton with `priority`, `question`, `why it matters`, `linked`, `status: open`.

### Step 6 — Draft commit + PR text (skip if `--no-commit-draft`)

Write to chat (not to disk):

```
SUGGESTED COMMIT (conventional commits, short subject, body with WHY):
<type>(<scope>): <subject>

<body — references plan + impl report paths, lists files touched, calls out any open issues>

SUGGESTED PR TITLE:
<short, <70 chars>

SUGGESTED PR BODY:
## Summary
- <bullets>

## Test plan
- <bullets from the implementation report's verify section>
```

### Step 7 — Surface to user

Output:
- Status line: D-NNN appended, N OQs closed, M raised, K memory-bank files updated.
- One-line diff stat from each updated file.
- The commit / PR drafts (unless `--no-commit-draft`).
- Question: "Commit + push, open PR, or revise?"

**Do NOT** `git add` / `git commit` / `git push` / `gh pr create`. Human gate is mandatory.

## Loop guards

- Plan or implementation report missing → abort.
- Implementation reported `verify-failed` and no waiver flag → abort; tell the user to either fix verify or pass an explicit `--waive-verify`.
- An agent tries to edit a file outside its assigned `(memory-bank-file)` → reject its output, do not apply, surface as `agent-failed: <role>`.
- D-NNN number collision (rare: parallel `/post-feature` runs) → bump to next free number, log the collision in the entry.
- Two OQs claim the same resolution from this feature → keep both close-marks but cross-link.

## What this skill does NOT do

- Does not run code / tests — that was `/implementor`'s job.
- Does not commit / push / open PR.
- Does not touch source files (only `.memory-bank/` + `.assistant/`).
- Does not auto-write a release-notes file or changelog (out of scope).
- Does not retroactively edit prior D-NNN entries — append-only per INVARIANT §4.

## Example invocation

```
/post-feature add-defrag-skill
```

Expected behavior: read both swarm-report files for the slug, fan out memory-bank updates, append D-NNN, close OQ-006 partial / raise OQ-NEW-1, surface commit + PR drafts, ask for human approval.
