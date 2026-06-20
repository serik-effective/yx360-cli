Base directory for this skill: /Users/ayusavin/Projects/effective/harness/.claude/skills/sync

# Skill: /sync

Update an existing Effective Harness install in a target project to the current harness checkout's commit. Pair to `/setup`: setup installs from scratch, sync updates in-place with drift detection.

This skill IS the orchestrator. It runs from the harness checkout, reads the target project's `.harness-lock`, diffs the file tree, and applies harness-owned changes only.

## When to invoke

- Target project has `.harness-lock` (already installed).
- Harness checkout HEAD is ahead of the SHA in `.harness-lock`.
- User wants the new agents / skills / hooks / invariants without rewriting their own memory bank.

If `.harness-lock` is missing → tell the user to run `/setup` instead.

## Invocation

```
/sync [<target-project-path>] [<optional note for the decisions log>]
```

- `<target-project-path>` — absolute path to the target. If omitted, the skill asks.
- `<optional note>` — short reason for this sync, recorded in `.assistant/decisions.md`. Optional.

## Ownership model (the contract)

`.harness-lock` records every managed file with an `owner` field:

| Owner              | Semantics                                                                                                | Sync behavior                                                                                                                |
|--------------------|----------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------|
| `harness`          | Framework file. Source of truth = harness checkout. Examples: `.claude/agents/*`, `.claude/hooks/*`, `.claude/skills/*` (skill bodies, not project edits), `.assistant/INVARIANTS.md`, `AGENTS.md`. | Auto-overwrite when upstream changed AND target hash equals lock hash (no drift). Conflict if both changed. |
| `project-template` | Seeded once from harness, then owned by the project. Examples: `.memory-bank/**`, `.assistant/decisions.md`, `.assistant/open-questions.md`, `CLAUDE.md`, `.harness-lock` itself (generated). | Never overwritten. If upstream template changed, surface a one-line diff hint; user merges manually if they want. |
| `project`          | Pure project content created post-install. Lock doesn't track these.                                       | Never touched.                                                                                                              |

Rules:
- A file appears in lock iff it was created by the harness (either `harness` or `project-template`).
- `/sync` never reads or writes files outside the lock's keyspace plus newly added harness-owned paths discovered in the harness checkout.

## Orchestrator workflow

### Step 1 — Verify we are running inside a harness checkout

Same checks as `/setup` Step 1:
- `AGENTS.md`, `.assistant/INVARIANTS.md`, `.claude/agents/` (≥10 files), `.claude/skills/sync/SKILL.md` all present.
- If missing → abort: "Run /sync from the root of a harness checkout."

Record:
- `HARNESS_NEW_SHA = git -C <harness-root> rev-parse HEAD`
- `HARNESS_REMOTE = git -C <harness-root> remote get-url origin`
- Working tree status: if `git -C <harness-root> status --porcelain` returns anything, warn: "Harness checkout has uncommitted changes. Sync will reflect your local edits, not the published commit. Continue?" Wait for explicit y/n.

### Step 2 — Resolve target path

If no target was given, ask. Validate: absolute, exists, directory, not the harness checkout itself, is a git repo (warn if not).

### Step 3 — Read and validate `.harness-lock`

- `<target>/.harness-lock` must exist. If missing → "No harness install detected. Use /setup."
- Parse as JSON. Bail with the parse error if invalid; tell the user to fix manually.
- Extract: `harness_version` (= `HARNESS_OLD_SHA`), `harness_source`, `project_type`, `primary_stack`, `touch_policy`, `files`.

Compare SHAs:
- `HARNESS_OLD_SHA == HARNESS_NEW_SHA` → "Already at `<sha>`. No-op." Exit 0.
- `HARNESS_OLD_SHA` not reachable from `HARNESS_NEW_SHA` (i.e., `git -C <harness-root> merge-base --is-ancestor <old> <new>` fails) → warn: "Lock SHA `<old>` is not an ancestor of harness HEAD `<new>` — harness history was rewritten or you're on a divergent branch. Continue at your own risk?" Wait for explicit y/n.

Compare remotes:
- Lock's `harness_source` host/path ≠ `HARNESS_REMOTE` host/path → ask: "Lock recorded source = `<lock-source>`. Current harness remote = `<HARNESS_REMOTE>`. Switching upstream — confirm?" Wait for y/n.

### Step 4 — Target git hygiene

Run `git -C <target> status --porcelain`. If any tracked file under managed paths (paths in lock + `.claude/` + `.assistant/` + `.memory-bank/`) is dirty → list them, ask: "Target has uncommitted changes in harness-managed paths. /sync writes to those paths. Stash / commit first, then re-run, OR continue and accept overwrites?" Wait.

Current branch: if it's the default branch (`main` / `master` / `trunk`) → suggest `git checkout -b harness-sync-<short-new-sha>` before applying. Don't auto-create; just print the suggestion and proceed if user confirms.

### Step 5 — Build the plan

For each path `p`:

**5a. Harness-owned paths (lock says `harness`, or newly added in harness checkout under managed roots).**

Compute:
- `harness_sha = sha256(<harness-root>/p)` (or `MISSING` if upstream deleted it)
- `target_sha  = sha256(<target>/p)` (or `MISSING` if target doesn't have it)
- `lock_sha    = lock.files[p].sha256` (or `MISSING` if newly added in upstream)

Classify:

| harness_sha | target_sha | lock_sha    | Action                  |
|-------------|------------|-------------|-------------------------|
| ≠ lock      | == lock    | present     | **overwrite** (auto)    |
| ≠ lock      | ≠ lock     | present     | **conflict** (both changed; ask) |
| MISSING     | == lock    | present     | **delete** (auto, with one batch confirm) |
| MISSING     | ≠ lock     | present     | **conflict-delete** (target diverged; ask keep or delete) |
| MISSING     | MISSING    | present     | clean-lock-entry (already gone) |
| present     | MISSING    | present     | **restore** (auto; user removed harness file) |
| present     | any        | MISSING     | **add** (auto; new in upstream) |
| == lock     | == lock    | present     | no-op                   |
| == lock     | ≠ lock     | present     | drift-warn (target edited; upstream didn't; leave alone, surface in summary) |

**5b. Project-template paths (lock says `project-template`).**

- `upstream_sha = sha256(<harness-root>/p)` (if the upstream template still exists)
- If file removed in upstream → no-op (project owns it now).
- If `upstream_sha == lock.files[p].sha256` → no-op (template didn't change upstream).
- If `upstream_sha ≠ lock.files[p].sha256` → emit a **template-update-hint**: `path → harness updated the seed template. Target file is project-owned; run \`diff <(harness-show p) <target>/p\` to compare.` Do **not** touch the file. Do **not** update the lock's hash for this entry (lock hash records what was seeded, not the current upstream).

**5c. Newly added project-template paths (in upstream, missing in target).**

If the upstream now ships a new `project-template` file (e.g., a new seed under `.memory-bank/` that didn't exist when the project was installed) → seed it with the same logic as `/setup` Step 7, and add the lock entry with `owner: project-template`.

### Step 6 — Show the plan, collect conflict decisions

Print to user:

```
Sync plan: <HARNESS_OLD_SHA[:8]> → <HARNESS_NEW_SHA[:8]>  (<N> commits)

Auto (no conflicts):
  + add:       <count> files
  ~ overwrite: <count> files
  - delete:    <count> files
  ↻ restore:   <count> files

Templates updated upstream (not touched — yours to merge):
  <list of project-template files where upstream changed>

Local drift on harness files (you edited, upstream didn't — left alone):
  <list>

Conflicts (both you and upstream changed — need decision):
  <count> files
    <path>
    <path>
    ...
```

**Conflict resolution.**

- 0 conflicts → skip ahead.
- 1 conflict → ask with `AskUserQuestion` (4 options): `Keep target / Use upstream / View diff and decide later (abort sync) / Skip this file`.
- 2..N conflicts → batch decision with `AskUserQuestion`:
  - "Resolve all `<N>` conflicts the same way?"
  - Options: `Use upstream for all` / `Keep target for all` / `Write conflict report and abort` (writes `<target>/.harness-sync-conflicts.md` with file paths + a 3-way diff hint, user resolves manually, re-runs `/sync`).

No per-file mixed decisions in one run — re-run after manual edits if you need granularity. (Keeps the skill simple and predictable; manual cherry-pick is one `cp` away.)

**Final confirmation:**

```
Proceed?  (y / n / show-diff <path>)
```

`show-diff <path>` prints `git -C <harness-root> show <HARNESS_NEW_SHA>:<p>` next to the target file, then re-asks.

### Step 7 — Apply

Order:
1. Adds (new harness files, new project-template seeds).
2. Overwrites (auto + conflict-resolved upstream).
3. Restores.
4. Deletes (with one final "delete N files" confirm).
5. Make all `.sh` under `.claude/hooks/` executable (`chmod +x`).

Use `cp` / `rm` via `Bash`, not Edit/Write (faster, cleaner logs). Never touch paths outside the lock's keyspace plus the newly-added harness-managed paths.

### Step 8 — Regenerate `.harness-lock`

Preserve from old lock: `installed_at`, `project_type`, `primary_stack`, `touch_policy`.

Update:
- `harness_version` = `HARNESS_NEW_SHA`
- `harness_source` = `<HARNESS_REMOTE>@<HARNESS_NEW_SHA>` (remote URL pinned to commit, never a local path — same rule as `/setup` Step 7)
- `last_synced_at` = ISO 8601 UTC now
- `files` = recompute for every harness-owned + project-template path (use the file's current sha256 in the target)
- `sync_history` = append `{ from: <OLD_SHA>, to: <NEW_SHA>, at: <iso>, conflicts: <count>, decisions: <"keep-target" | "use-upstream" | "none"> }`, cap at last 10 entries

Write atomically: write to `.harness-lock.tmp`, then `mv` over `.harness-lock`.

### Step 9 — Append decision entry

Add to `<target>/.assistant/decisions.md` (next D-NNN, auto-numbered):

```markdown
## D-NNN — Harness synced

**Date:** <YYYY-MM-DD>
**Status:** accepted
**Decision:** Harness updated `<OLD_SHA[:8]>` → `<NEW_SHA[:8]>`.
**Counts:** +<add> ~<overwrite> -<delete> ↻<restore>; <conflicts> conflicts resolved (<keep-target | use-upstream>).
**Template updates pending manual merge:** <list-or-"none">
**Note:** <user-supplied note or omit line>
**Source:** `<HARNESS_REMOTE>@<NEW_SHA>`
```

### Step 10 — Verify

Same sanity checks as `/setup` Step 8:
- `.claude/hooks/inject-state.sh` exists and is executable.
- `.harness-lock` parses as JSON; required keys present.
- `.assistant/INVARIANTS.md` non-empty.
- `AGENTS.md` non-empty.
- SessionStart hook smoke test: `bash <target>/.claude/hooks/inject-state.sh` exits 0.

Failure → warn, don't auto-rollback. The user has a git checkpoint (Step 4 suggested a branch).

### Step 11 — Summary

```
✓ Harness <OLD[:8]> → <NEW[:8]> in <target>

Applied:
  + <N> new files
  ~ <N> updated
  - <N> deleted
  ↻ <N> restored
  <N> conflicts resolved (<decision>)

Templates upstream updated, project-owned — merge manually if you want them:
  <list>

Next steps:
  1. cd <target>
  2. git status                       # review the diff
  3. git diff .harness-lock           # confirm metadata
  4. git commit -m "harness: sync to <NEW[:8]>"
  5. Run /pre-feature on the next real change — confirms new agents/skills work.
```

## Loop guards

- **No `.harness-lock`** → reject; point at `/setup`.
- **Same SHA** → no-op exit.
- **Lock SHA not ancestor of HEAD** → warn, require explicit y/n (history rewrite or divergent branch).
- **Different `harness_source`** → ask before treating it as the same project.
- **Target git dirty in managed paths** → require explicit y/n.
- **Harness checkout dirty** → warn, require explicit y/n.
- **Lock JSON parse error** → abort; tell user to fix manually.
- **Conflict count > 0 and user picked "write report"** → write `.harness-sync-conflicts.md` and exit 0 without applying anything else (atomicity: partial sync is worse than no sync).
- **Step 7 mid-failure** → leave `.harness-lock` untouched (it's the last write). Re-run `/sync` continues cleanly.
- **User declined at Step 6** → no files written, no lock change. Clean abort.

## What this skill does NOT do

- Does not `git fetch` / `git pull` the harness checkout. User controls the harness HEAD.
- Does not push, commit, or open a PR in the target project.
- Does not merge `project-template` upstream changes into the project's current copy. Surfaces the path; user merges manually.
- Does not migrate lock schema. If lock version drifts incompatibly in the future, a separate `/migrate-lock` skill handles it (OQ).
- Does not touch files outside the lock's keyspace + newly added harness-managed paths. The project's own code is untouched.
- Does not install MCP servers, third-party skills, or modify global `~/.claude/` state.
- Does not rename-detect (upstream rename → recorded as delete+add; user re-applies any local edits manually).
- Does not roll back on Step 10 verify failure. Rollback = `git checkout .` on the sync branch.

## Example flows

### Flow A — clean update, no drift

```
$ cd ~/projects/harness
$ git pull
$ claude
> /sync ~/projects/usmint

[skill reads usmint/.harness-lock → 77cdc03]
[harness HEAD = 99a4ae4]
[plan: +5 files (new agents/skills), ~3 files (skill updates), 0 conflicts]
[user confirms]
[apply + lock regen + decision D-007 appended]
[summary]
```

### Flow B — conflict on harness-owned file

User locally edited `.claude/agents/architect.md` in the target. Upstream also changed it.

```
> /sync ~/projects/usmint

[plan shows 1 conflict: .claude/agents/architect.md]
[AskUserQuestion → user picks "Write conflict report and abort"]
[writes ~/projects/usmint/.harness-sync-conflicts.md with diff hint]
[no other files touched, lock untouched]

# user merges manually, removes report, re-runs /sync
```

### Flow C — template updated upstream

Upstream changed `.memory-bank/steerings/project-rules.md` template. Project's copy is project-owned (per ownership model).

```
[plan output includes:]
  Templates updated upstream (not touched):
    .memory-bank/steerings/project-rules.md
       diff: git show 99a4ae4:.memory-bank/steerings/project-rules.md | diff - .memory-bank/steerings/project-rules.md

[apply runs as normal for harness-owned files]
[user decides off-band whether to merge the template change]
```

## Open questions for this skill

- **OQ-SYNC-1:** Should `project-template` upstream changes be surfaced in the decision log too, or only the summary? (Current: summary only; decision log records "templates pending merge: <list>".)
- **OQ-SYNC-2:** Lock schema migration policy when we add new fields (e.g., a future `signature` field). Probably: tolerant read, strict write — `/sync` accepts older lock shapes and rewrites them in the new shape.
- **OQ-SYNC-3:** Should `/sync` ever pull harness upstream itself? Current answer: no, keep concerns separate. User runs `git pull` in the harness checkout when they want.
- **OQ-SYNC-4:** Rename detection (upstream rename → currently delete+add, loses any local edits to the renamed file). Worth adding `git log --follow` heuristic? Defer until the first real-world rename hurts.
