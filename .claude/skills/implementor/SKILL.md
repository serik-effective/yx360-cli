---
name: implementor
description: Execute an already-approved `/pre-feature` plan. Reads swarm-report/<slug>-plan-<date>.md, fans out to executing agents per file scope, runs the verify gate, and writes an implementation report. Use AFTER /pre-feature, never before.
---

# Skill: /implementor

Layered, agent-driven execution of an approved plan. Orchestrator only — no direct code edits in the main loop.

**When to invoke:** user types `/implementor <slug>` after `/pre-feature <slug>` produced a plan and the user said "proceed". The plan file must exist at `swarm-report/<slug>-plan-<date>.md`.

**When NOT to invoke:** there is no plan (run `/pre-feature` first); plan has unresolved HIGH-severity blockers (resolve / revise first); user asks a one-line fix (just do it).

## Invocation

```
/implementor <slug>
```

Optional second argument: `--continue` to resume after a verify-gate failure or hard-stop. Skip otherwise.

## Orchestrator workflow

### Step 1 — Validate inputs

Read in order:
1. `swarm-report/<slug>-plan-<date>.md` — locate by glob; latest date wins. Missing → abort: "Run `/pre-feature \"<feature>\"` first."
2. `.assistant/INVARIANTS.md`
3. `.memory-bank/product-overview/anti-stories.md`
4. `AGENTS.md` § Routing § Executing — the `Agent + Scope` table that maps file patterns to exec agents.
5. `.assistant/decisions.md` — any D-NNN that overturns part of the plan?

Parse the plan's **Blockers** section. If any item is still HIGH severity + `requires_human: true` and was not explicitly waived → abort. Tell the user which blocker.

Parse the plan's **Out-of-scope (declared)** section. Anything in there stays out of scope; do not implement opportunistically.

### Step 2 — Match plan tasks → file scope → exec agents

For each task in the plan:
- Resolve `affected_files` (or compute from `path:line` citations).
- Match each affected file against the `Executing` scope table → exec agent.
- Group tasks by `(layer, agent)`. Default layer order: schema/model → backend → frontend → infra → docs. Plan may override.

Multi-layer tasks → split into per-layer sub-tasks. Don't fan out one agent across layers in one Task call.

### Step 3 — Branch hygiene (ANTI-3)

`git status --porcelain` clean → suggest a feature branch: `git checkout -b feat/<slug>` (don't auto-create — print and proceed if user already on a non-default branch).

Dirty → list dirty files, ask: "Commit / stash first, then re-run, OR continue and accept the diff stacks?"

### Step 4 — Fan out by layer (serialized) + by agent (parallel within layer)

For each layer in order:
- Spawn the Task tool calls for every `(agent, sub-task)` in this layer **in parallel** (single message, N Task calls).
- Each Task prompt includes:
  - The plan file path + the specific task ID(s) the agent owns
  - Path references for the agent to read (INVARIANTS, the relevant memory-bank files)
  - Hard contract: "Edit > Write. No comments explaining WHAT. No error handlers for impossible cases. No feature flags. Output strict YAML per `.claude/agents/<role>.md` exec contract. End with `verify:` block listing every command run and its output."
- Wait for all layer-N agents to return before starting layer N+1.

If a Task agent fails mid-layer:
- Mark its sub-task `status: failed`
- Continue parallel siblings in the same layer
- After the layer completes, DO NOT auto-start layer N+1. Surface failures to user.

### Step 5 — Verify gate (ANTI-11)

After all layers complete (or after a partial-layer success):
- Run the project's verify script (lookup order: `/quickstart` SKILL output → `Makefile` `verify` target → `package.json` `scripts.verify` → project-type defaults).
- Default per project type: typecheck + lint + unit tests + a single smoke invocation.
- For UI changes: must include a real run (`/run` skill if present, else manual launch in the report's checklist for the user).
- **Visual critic is MANDATORY (not optional) when the diff touches a UI surface** — any changed file matching `*.tsx` / `*.jsx` / `*.vue` / `*.svelte` / `*.swift` / `*.css` / `components/`, or a project that declares a UI surface. Before the implementor may report `status: complete`, the verify gate MUST:
  1. Invoke an existing critic skill — `apple-design-critic` (Apple targets), or `frontend-design` / `visual-spec` / the `ui` consilium role (web). Do not hand-roll a parallel review.
  2. Capture before/after screenshots, reusing the project's `playwright-cli` named-session path.
  If a UI-touching diff ships with **no critic run**, `status` cannot be `complete` — write the report as `partial` and surface that the visual critic was skipped.

Verify failure → DO NOT claim done. Write the report with `status: verify-failed`, list the failing command + its stderr tail, surface to user.

### Step 6 — Write the implementation report

Append-only write to `swarm-report/<slug>-implementation-<YYYY-MM-DD>.md`. Sections:

- **Status:** `complete` | `verify-failed` | `partial` | `aborted-blocker`
- **Layers executed:** ordered list with timing
- **Files touched:** full list with line counts; cross-link to the plan task IDs
- **Per-agent verbatim YAML:** every exec agent's return, kept for audit
- **Verify results:** commands run + exit codes + stderr tails
- **Out-of-scope (declared):** carried over from the plan
- **Open issues raised during implementation:** new failures, agent-flagged tech debt, surfacing-on-completion items → these MAY need a new `/pre-feature` round
- **Suggested commit message + PR title:** drafts only, never auto-committed
- **Next:** point at `/post-feature <slug>` for memory-bank + decisions log updates, or at the next step if status is not `complete`

### Step 7 — Surface to user

Output to chat:
- Status line + counts (files changed / inserts / deletes / failing verify steps)
- Path to the implementation report
- Question: "Proceed to `/post-feature <slug>` (decisions + memory bank updates), or revise?"

**Do NOT** auto-spawn `/post-feature`. **Do NOT** commit. **Do NOT** push. Human gate is mandatory per `.memory-bank/steerings/project-types.md`.

## Loop guards

- Plan file missing → abort with explicit path tried.
- HIGH-severity blocker unresolved → abort; quote the blocker.
- Verify gate fails → write report with `verify-failed`, surface, stop. Re-run via `/implementor <slug> --continue` after the user fixes the root cause.
- Exec agent returns prose instead of YAML → re-spawn ONCE with stricter prompt. Second prose-return → record `agent-failed: <role>`, continue siblings, skip this sub-task.
- Layer-N partial failure → never auto-start layer N+1. User must decide.
- `/implementor` called twice on the same slug within 1h → emit "implementation already in flight; use `--continue` or revise the plan first".

## What this skill does NOT do

- Does not design — `/pre-feature` does that.
- Does not commit / push / open PR.
- Does not modify `.memory-bank/` or `.assistant/decisions.md` — `/post-feature` does that.
- Does not re-decide stack choices made in the plan.
- Does not opportunistically refactor adjacent code (ANTI-6).
- Does not silently fix lints unrelated to the diff (run linter in verify; surface, don't auto-edit).
- Does not skip the verify gate even when "obviously working".

## Example invocation

```
/implementor add-defrag-skill
```

Expected behavior: read `swarm-report/add-defrag-skill-plan-<date>.md`, fan out exec agents per file scope across schema → backend → docs layers, run verify, write `swarm-report/add-defrag-skill-implementation-<date>.md`, surface status + next step.
