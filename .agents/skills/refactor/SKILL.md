---
name: refactor
description: Consilium for behavior-preserving code restructuring — extract module, untangle dependency, replace pattern, rename, dedupe. Mandates a coverage check BEFORE any change, freezes external behavior as a fixture, then plans the rewrite. Sibling to `/pre-feature` (designs new behavior) — `/refactor` adds none. Use when scope is "make this cleaner / smaller / better-factored" without changing what it does.
---

# Skill: /refactor

Behavior-preserving change consilium. Orchestrator only. The contract: external behavior is identical before and after; only internal shape changes.

**When to invoke:** user says "refactor this", "extract X", "clean up Y", "untangle Z", "rename across the codebase", "this module is too big", "we're duplicating logic in A and B". Scope is internal — no new features, no new APIs, no new flags.

**When NOT to invoke:** the change adds or removes a capability (`/pre-feature`); the goal is reviewing existing code (`/audit`); the user wants to fix a bug (`/diagnose` then `/implementor`).

## Invocation

```
/refactor "<goal in plain language>" [--scope <path|module>]
```

Examples: `/refactor "extract proxy selection into its own module" --scope worker/proxy`, `/refactor "rename ExceptionWithPageInfo → PageContextError everywhere"`, `/refactor "dedupe retry logic in handlers/*.py"`.

## Orchestrator workflow

### Step 1 — Validate the premise

Read:
1. `.assistant/INVARIANTS.md`
2. `.memory-bank/product-overview/anti-stories.md` — refactor-flavored anti-stories (ANTI-6: no opportunistic refactoring; never wrap a refactor inside a feature change).
3. `.assistant/decisions.md` (last 10) — has a prior D-NNN explicitly accepted the current shape?
4. `.memory-bank/tech-details/stack.md`

If the user's "refactor" actually changes behavior (new flag, new endpoint, new field), reject: "This is a feature change; use `/pre-feature`. Refactors preserve external behavior."

If a prior D-NNN locked the current shape and the user did not cite it, surface the D-NNN and ask: "D-NNN ratified this shape on <date>. Override?" Wait for explicit y/n.

### Step 2 — Coverage check (gate)

Compute the test coverage of the scope-defined files:
- Per-file line coverage from the project's coverage tool (look up `Makefile coverage`, `pyproject.toml [tool.coverage]`, `package.json scripts.coverage`).
- Per-file branch coverage if available.
- Per-function untested-branch list.

Threshold: scope coverage must be ≥ 70% line + the affected public functions must each have ≥ 1 happy-path test. Below threshold → emit a **coverage gap report** and abort the refactor:

```
swarm-report/refactor-<slug>-coverage-gap-<date>.md
Coverage gap blocks refactor. Add tests first:
  - <file>:<func> — 0 tests
  - <file>:<func> — happy path only, no error branches
Re-run /refactor after coverage ≥ 70%.
```

Refactor without test coverage is a bug factory. This gate is non-negotiable for Type 2 projects; Type 1 projects may waive with `--type-1-waiver`, but the report still records the gap.

### Step 3 — Capture the behavioral fixture

Before any restructuring, freeze a behavioral baseline:
- For pure functions: golden-file fixture with N input/output pairs covering the public surface.
- For services / handlers: VCR / cassette of real requests covering the happy path + each error class.
- For UI: a snapshot test (image / DOM tree).
- For CLI: stdout/stderr captures for representative invocations.

The fixture lives in `tests/fixtures/refactor-<slug>/` and MUST pass against the pre-refactor code before the refactor is allowed to start.

### Step 4 — Fan out the consilium

Single message, parallel Task calls:
- `architect` — proposes the new shape (module boundaries, names, layering, public API) AND lists every public symbol that must not change.
- `skeptic` — finds what the refactor breaks that you didn't notice (transitive dependents, callers outside scope, public-API consumers, build/CI impacts).
- `reviewer` — checks the plan against INVARIANTS + anti-stories + the D-NNN history.

Output contract: strict YAML per `.Codex/agents/<role>.md`. Refactor-specific fields each agent emits: `preserved_public_api: [...]`, `breaks_if: [...]`, `migration_steps: [...]`.

### Step 5 — Aggregate into a refactor plan

Write `swarm-report/refactor-<slug>-plan-<YYYY-MM-DD>.md`. Sections:
- **TL;DR** — one-sentence shape change.
- **Behavioral invariants** — pulled from the fixture; every assertion that must continue to pass.
- **Migration steps** — ordered list, each step labeled `mechanical` (pure rename / move) or `semantic` (logic touched). Bias hard toward mechanical-only steps; flag any semantic step for extra review.
- **Files touched** — full list with reason per file.
- **Breaks if** — every callsite or consumer that would break, with a citation.
- **Rollback plan** — if mid-way, how to revert (must be a single `git revert` of one commit per migration step).
- **Out-of-scope (declared)** — opportunistic cleanups the agents wanted but were rejected (ANTI-6).
- **Per-agent verbatim YAML** — for audit trail.

### Step 6 — Surface to user

Output:
- TL;DR.
- Coverage gate result (passed / waived).
- File touch count + step count.
- Path to the plan.
- Question: "Proceed to `/implementor refactor-<slug>` or revise?"

**Do NOT** auto-execute. `/implementor` runs each migration step as its own atomic commit, with the fixture re-run after each step as the verify gate.

## Loop guards

- Coverage gate failure → emit coverage-gap report, abort. Don't proceed even on user "force" unless `--type-1-waiver` was passed.
- Agent proposes a `semantic` step that the user didn't approve → surface as HIGH-severity blocker; user must explicitly accept.
- Plan touches > 25 files → emit `refactor-too-large; split into smaller refactors`. Plan is still written.
- Same refactor slug re-run within 24h → re-attach to prior plan (append revisions); don't rerun fixture capture if files unchanged.
- ANTI-6 trip (proposal bundles unrelated cleanups) → orchestrator strips them and lists them in Out-of-scope; agents may not insist.

## What this skill does NOT do

- Does not write code.
- Does not commit / push.
- Does not change external behavior (per definition; enforced by the fixture).
- Does not add features, flags, endpoints, fields.
- Does not opportunistically fix lint issues unrelated to the refactor (ANTI-6).
- Does not skip the coverage gate to "move fast".
- Does not collapse multiple migration steps into one commit (each step = one atomic commit, so any single step can revert).

## Example invocation

```
/refactor "split scraping-architect.md anti-bot section into per-vendor files" --scope .Codex/agents
/refactor "rename ExceptionWithPageInfo → PageContextError everywhere"
/refactor "extract retry/backoff helper used in handlers/*.py" --type-1-waiver
```

Expected behavior: coverage check → behavioral fixture → 3-agent consilium → refactor plan with per-step labels → surface coverage + step count + next-step prompt.
