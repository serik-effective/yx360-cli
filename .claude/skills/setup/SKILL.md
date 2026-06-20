---
name: setup
description: Install Effective Harness into a target project. Orchestrates the file copy, interviews the user about the target project, and seeds the initial memory bank, decisions log, and harness-lock. Run this skill from a harness checkout; pass the target project path. Replaces the manual install prompt in README.md.
---

# Skill: /setup

Install Effective Harness into a target project. This skill IS the orchestrator — it does the work itself, no separate subagents required for the install. Use `/pre-feature` or `/research` later for design work in the target project.

## When to invoke

User just cloned this harness repo (or already has it locally) and wants to install it into another project. They run the AI-CLI inside the harness checkout, then type `/setup` with an optional target path.

## Invocation

```
/setup [<target-project-path>] [<short context, 1–2 sentences>]
```

- `<target-project-path>` — absolute path to the target project root. If omitted, the skill asks.
- `<short context>` — optional one-line description ("an iOS app for gift card scanning"); helps seed the memory bank. If omitted, the skill asks.

## Orchestrator workflow

### Step 1 — Verify we are running inside a harness checkout

The orchestrator checks the current working directory contains:
- `AGENTS.md`
- `.assistant/INVARIANTS.md`
- `.claude/agents/` with ≥10 agent files
- `.claude/skills/setup/SKILL.md` (this file)

If any of these are missing, abort with: "Run /setup from the root of a harness checkout (cloned from github.com/effective-dev-os/harness)."

Record the harness commit SHA: `git -C <harness-root> rev-parse HEAD` (for `.harness-lock`).

### Step 2 — Resolve target path

If no target path was given:
- Ask the user: "Absolute path to the target project (the project the harness will be installed into)?"

Validate the path:
- Must be an absolute path
- Must be a directory that exists
- Must not be the harness checkout itself (abort if equal)
- Should be a git repo (warn if not — harness expects branch workflow per ANTI-3)

### Step 3 — Inspect the target project

Read (best-effort; missing files are fine):
- `README.md`, `CLAUDE.md`, `AGENTS.md` if they exist
- `package.json`, `Cargo.toml`, `pyproject.toml`, `pubspec.yaml`, `Package.swift`, `build.gradle*`, `go.mod` — detect primary language(s)
- `.gitignore`, `.editorconfig` — detect existing conventions
- Top-level dir layout (1 level deep)

Detect:
- Primary language(s) and stack
- Whether `.memory-bank/` / `.assistant/` / `.claude/` already exist (any of these = previous install or conflict)
- Existing `CLAUDE.md` (back up to `CLAUDE.local.md` if it exists and is non-trivial — i.e., not a stub)

### Step 3.5 — Mine prior AI-CLI sessions (parallel)

Before interviewing the user, mine the prior Claude Code / OpenCode / Codex sessions for the target project. They typically contain a lot of user context the user won't think to repeat in the install interview: domain glossary, tech-stack details, friction patterns, recurring complaints, open questions already voiced, validated approaches.

**Detect session directories.** Slugify the target path into the AI-CLI's session-dir convention:

- **Claude Code:** `~/.claude/projects/<slug>/*.jsonl` — slug = target absolute path with `/` replaced by `-` (e.g., `/Users/ayusavin/Projects/jukte` → `-Users-ayusavin-Projects-jukte`).
- **OpenCode:** `~/.opencode/<slug>/...` *(format TBD — Phase 6 research; for now, best-effort search)*
- **Codex:** `~/.codex/<slug>/...` *(format TBD; best-effort search)*

If no session files exist in any of these locations → skip this step, proceed to Step 4 with no extra context.

If ≥3 session files exist → run the MapReduce flow below.

**Map phase: spawn 3 parallel `general-purpose` agents via `Task` tool, single message, 3 Task calls.**

Each agent gets a non-overlapping subset of recent sessions (split by date — agent A = newest third, B = middle third, C = oldest third). Each agent's prompt:

> You are a session-miner for the Effective Harness `/setup` skill. Read the following Claude Code / OpenCode / Codex session JSONL files in `<dir>` (your assigned subset: `<file1>`, `<file2>`, ...). **Do NOT read full files** (they can be 10–100MB) — use `head -300` + `tail -300` + a mid-sample per file. Look for:
>
> - **Tech stack signals** — frameworks, libraries, databases the user has mentioned working with
> - **Domain glossary** — project-specific terms ("portal", "card-balance", "tunnel", etc.) the user uses
> - **Friction signals** — places the user says "no", "нет", "не так", "стоп", "поправь" — quote 1–2 examples
> - **Validation signals** — places the user says "да", "отлично", "идеально", "продолжай" — quote 1–2 examples
> - **Recurring complaints / wishes** — anything the user has asked for repeatedly
> - **Open questions** — design questions the user has voiced but not resolved
> - **Implicit invariants** — rules the user has stated ("don't push to main", "don't mock the DB", "always check logs first")
>
> Output strict YAML per the `researcher` agent schema. Each finding carries `confidence: high | medium | low` and cites the session filename + line range.
> Keep it under 600 words.

**Reduce phase: orchestrator dedupes findings across the three agents.**

Group by category (stack / glossary / friction / validation / complaints / open-questions / invariants). Drop duplicates by `(category, finding-substring)`. Surface the highest-confidence findings.

**Use of findings:**
- Tech-stack signals → propose defaults for the **Primary stack** interview question (user can still override).
- Domain glossary → seed `.memory-bank/tech-details/glossary.md` with the terms.
- Implicit invariants → suggest entries for `.assistant/open-questions.md` as "OQ-INV-1: confirm <invariant> applies project-wide?" (don't auto-add to INVARIANTS.md — they're harness-wide, not project-wide).
- Open questions → seed `.assistant/open-questions.md` with OQ-2..N.
- Friction / validation signals → seed `.memory-bank/steerings/project-rules.md` extended notes.

**Show the user the mined summary** before the interview:

```
Mined N sessions across <date-range>:

Tech stack detected: <list>
Domain terms: <list>
Recurring user invariants: <quoted, with session refs>
Open questions you've voiced: <quoted>

I'll use this to pre-fill the interview defaults below. You can override anything.
```

**Loop guards:**
- Session files >100MB total combined → ask the user before spending the tokens.
- No useful findings (all agents returned empty arrays) → proceed silently to Step 4 without surfacing anything.
- User declines mining (privacy / time) → skip and proceed.

### Step 4 — Interview the user (5 questions max)

Use `AskUserQuestion` (single multi-question call) to collect:

1. **PROJECT_TYPE** — "Type 1 (MVP / pre-sale / experiment)" or "Type 2 (production, human-gated)". See `.memory-bank/steerings/project-types.md`.
2. **Primary stack** (multiSelect) — backend / web frontend / iOS / Android / Flutter / infra. Used to pick executing-agent scope defaults in the target's `CLAUDE.md`.
3. **One-line vision** — "What does this project do, in one sentence?" Seeds `.memory-bank/product-overview/vision.md`.
4. **Touch policy** — should `/setup` overwrite an existing `CLAUDE.md` (back up as `CLAUDE.local.md`) or refuse to overwrite (abort, ask user to merge manually)?
5. **Existing memory bank** — does the project already have `.memory-bank/` (skip seeding) or not (seed templates)?

If the user has skipped clarifying context in Step 2, also ask: "Anything important the harness should know before generating an initial memory bank? (Sensitive dirs to avoid, compliance notes, existing tooling we should respect.)"

Where Step 3.5 mined findings, use them as **pre-filled defaults** in the question options (e.g., the multi-select for Primary stack defaults to the detected stack; the one-line vision is pre-filled with a synthesized summary from session content; the user confirms or edits).

### Step 5 — Plan the copy (dry-run summary)

Before any file is written, print the plan:

```
About to install harness <commit-sha> into <target-path>:

  Will create:
    .claude/agents/                 (12 files)
    .claude/hooks/inject-state.sh
    .claude/skills/                 (7 skills: pre-feature, research, setup, anti-ai-slop-writing, frontend-design, swiftui-pro, swiftui-macos-26, memory-bank-defrag)
    .claude/settings.json
    .assistant/INVARIANTS.md
    AGENTS.md

  Will create if missing (templates):
    .memory-bank/index.md           (seed)
    .memory-bank/product-overview/vision.md  (from your one-line answer)
    .memory-bank/product-overview/anti-stories.md  (copy of harness's, mark as project-template)
    .memory-bank/steerings/project-rules.md  (copy of harness's, mark as project-template)
    .memory-bank/tech-details/stack.md  (skeleton — fill in)
    .assistant/decisions.md         (D-001 = installed from harness)
    .assistant/open-questions.md
    CLAUDE.md                       (short entry point)
    .harness-lock                   (version metadata)

  Will skip / preserve:
    Any file the project already owns and Touch policy says skip
    Existing CLAUDE.md → backed up to CLAUDE.local.md if Touch policy says overwrite

Proceed? (y / n / details)
```

Wait for explicit user approval. Abort cleanly if user says no.

### Step 6 — Execute the copy

Copy files using `Bash` (cp / rsync, never agent-driven Edit/Write for the bulk copy — that's slower and noisier in logs). Use `cp -R` with explicit source paths. Never copy:
- `.git/`
- `swarm-report/` (harness-specific historical record)
- `.harness-lock` from harness checkout (we generate a fresh one for the target)
- This harness checkout's `.memory-bank/` content (only the *structure* — empty dirs + index.md template)
- Anything matching `.gitignore` patterns the user listed

### Step 7 — Seed target-specific files

These are not bulk-copied; they are generated based on the interview answers.

**`<target>/.harness-lock`** (JSON):

`harness_source` MUST be a remote URL pinned to a commit, never a local filesystem path. Canonical form: `<git-remote-url>@<commit-sha>` (e.g. `git@github.com:effective-dev-os/harness.git@abc1234…`). Local checkout paths leak the installer's machine layout and are unreachable from CI / other developers / future installs.

```json
{
  "harness_version": "<commit-sha>",
  "harness_source": "<git-remote-url>@<commit-sha>",
  "installed_at": "<ISO 8601 UTC>",
  "install_method": "skill:/setup",
  "project_type": <1 or 2>,
  "primary_stack": ["<list from interview>"],
  "files": {
    ".claude/agents/architect.md": { "owner": "harness", "sha256": "<computed>" },
    ".claude/hooks/inject-state.sh": { "owner": "harness", "sha256": "<computed>" },
    ".memory-bank/product-overview/vision.md": { "owner": "project-template", "sha256": "<computed>" },
    "CLAUDE.md": { "owner": "project-template", "sha256": "<computed>" },
    ...
  }
}
```

**`<target>/.assistant/decisions.md`**:
```markdown
# Decisions Log

> Append-only chronological record. When a decision is overturned, add a new entry with date + reason. Never edit or delete prior entries.

---

## D-001 — Harness installed
**Date:** <YYYY-MM-DD>
**Status:** accepted
**Decision:** Effective Harness installed at commit `<sha>` via the `/setup` skill. `PROJECT_TYPE: <N>`. Primary stack: <list>.
**Source:** `<git-remote-url>@<sha>` (remote URL pinned to commit — never a local filesystem path)
**Touch policy chosen at install:** <overwrite | preserve>
```

**`<target>/.memory-bank/index.md`**: minimal table of contents pointing at the seeded files. Project fills in as work progresses.

**`<target>/.memory-bank/product-overview/vision.md`**: starts with the user's one-line answer; includes prompts for the project owner to expand "Target audience", "DoD", "What we don't do".

**`<target>/.memory-bank/tech-details/stack.md`**: a stub with detected language(s) + dependencies + framework hints from Step 3 inspection plus stack signals from Step 3.5 session mining. Marked TODO for the project owner.

**`<target>/.memory-bank/tech-details/glossary.md`**: seeded from Step 3.5 domain-term findings. Each term carries a one-line definition (best-effort, marked TODO if the agent couldn't infer one). Empty if Step 3.5 found nothing.

**`<target>/.assistant/open-questions.md`**: in addition to the seed OQ-001 about stack lock-in, append OQ-2..N for the open questions the user voiced in prior sessions (from Step 3.5 mining).

**`<target>/CLAUDE.md`**: a short entry point — points at `AGENTS.md`, `.memory-bank/index.md`, declares `PROJECT_TYPE`, declares stack. Backs up any existing `CLAUDE.md` to `CLAUDE.local.md` if Touch policy = overwrite.

**`<target>/.assistant/open-questions.md`**: seed empty file with a header comment ("OQ-001 — set this project's primary stack and tooling versions explicitly").

### Step 8 — Verify install

Run sanity checks on the target dir:
- `.claude/hooks/inject-state.sh` is executable (`chmod +x` if not)
- `.harness-lock` parses as JSON
- `.assistant/INVARIANTS.md` exists and is non-empty
- `AGENTS.md` exists and is non-empty
- The SessionStart hook script runs without error: `bash <target>/.claude/hooks/inject-state.sh` → exit 0

Any check fails → emit a warning, don't abort silently.

### Step 9 — Summary and next steps

Output to the user:

```
✓ Harness <commit-sha> installed in <target-path>

Created:
  - 12 agent files under .claude/agents/
  - 8 skills under .claude/skills/ (including /pre-feature, /research, /setup)
  - SessionStart hook
  - .memory-bank/ skeleton + vision.md seed
  - .assistant/INVARIANTS.md, decisions.md (D-001), open-questions.md
  - .harness-lock

Next steps:
  1. cd <target-path>
  2. Open the project in Claude Code / OpenCode / Codex
  3. Edit .memory-bank/product-overview/vision.md to fill out target audience and DoD
  4. Edit .memory-bank/tech-details/stack.md to lock the stack
  5. Open a PR labeled "harness: initial install"
  6. Run /pre-feature on your first real change to verify the consilium works
```

## Loop guards

- **Already installed.** If `<target>/.harness-lock` exists with same commit SHA → emit "Already installed at this version. Use `/sync` to update."
- **Older harness install.** If `.harness-lock` exists with a different SHA → refuse `/setup`; tell the user to run `/sync` instead (in-place update with drift detection).
- **Half-installed state.** If `.claude/` exists but `.harness-lock` is missing → ask the user to either clean up manually or accept the Touch policy and proceed.
- **Re-run after error.** If Step 6 failed mid-copy, every retry starts by checking `.harness-lock` (if it doesn't exist, copy is incomplete and safe to redo).

## What this skill does NOT do

- Does not push to the target project's git remote.
- Does not commit to the target project. The user opens the PR.
- Does not install MCP servers or third-party skills (out of scope per ANTI-12).
- Does not auto-update later. That's `/sync` — run it from a harness checkout against the target.
- Does not run `/pre-feature` for the first real change. User does that explicitly.
- Does not read full session JSONLs in Step 3.5 — only head/tail/mid samples (files can be 10–100MB).
- Does not store mined session content in the harness repo — findings are summarized and folded into the target project's `.memory-bank/` + `.assistant/`; raw quotes go to `swarm-report/setup-mining-<date>.md` in the target project (gitignored by default).

## Example flows

### Flow A — fresh project

```
$ git clone https://github.com/effective-dev-os/harness
$ cd harness
$ claude
> /setup ~/projects/my-new-app

[skill asks interview questions]
[skill prints plan]
[user confirms]
[copy + seed runs]
[summary printed]

$ cd ~/projects/my-new-app
$ claude
> /pre-feature "add user signup flow"
```

### Flow B — existing project, with existing CLAUDE.md

```
$ cd harness
$ claude
> /setup ~/projects/jukte "Government tax portal for RK; treat as Type 2 production"

[skill detects existing CLAUDE.md]
[asks Touch policy → user picks "back up to CLAUDE.local.md and overwrite"]
[plan + confirm + copy + seed]
[user manually merges any custom rules from CLAUDE.local.md into the new CLAUDE.md afterwards]
```
