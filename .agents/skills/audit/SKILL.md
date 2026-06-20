---
name: audit
description: Multi-agent review of existing code against INVARIANTS, anti-stories, project-rules, and prior decisions. Diff-scoped (current branch / a path / a PR) or full-scoped. Returns a severity-tagged findings list. Use when reviewing a PR, sanity-checking a refactor, or auditing a module before a release.
---

# Skill: /audit

Multi-agent review consilium for code that already exists. Sibling to `/pre-feature` (designs code that doesn't exist yet). Orchestrator only.

**When to invoke:** user asks "review this PR / branch / module", "audit X before release", "is this still following INVARIANTS", "find what's wrong with <path>". Also: pre-release sanity pass on a high-risk path.

**When NOT to invoke:** debugging a specific failing test (`/diagnose`); designing a new feature (`/pre-feature`); implementing an approved plan (`/implementor`).

## Invocation

```
/audit <scope>
```

Scope forms:
- `branch` — current branch vs default (`git merge-base main HEAD`).
- `pr <N>` — GitHub PR diff (via `gh pr diff <N>`).
- `path <glob>` — specific files / directories.
- `commits <a>..<b>` — explicit range.
- `full` — whole repo (slow, fan-out is wider).

Optional trailing arg: `--severity HIGH` to filter findings.

## Orchestrator workflow

### Step 1 — Resolve scope + collect context

Read:
1. `.assistant/INVARIANTS.md`
2. `.memory-bank/product-overview/anti-stories.md`
3. `.memory-bank/steerings/project-rules.md` (if present)
4. `.assistant/decisions.md` (last 10 D-NNN — recent context window)
5. `.memory-bank/tech-details/stack.md`

Compute the diff scope (the actual file list + line ranges to review).
Empty diff → abort: "No changes in scope `<scope>`. Did you mean `path` or `full`?"

### Step 2 — Fan out subagents in parallel (single message, N Task calls)

Default crew (skip any whose `.Codex/agents/<role>.md` doesn't exist in the project):
- `reviewer` — INVARIANTS + anti-stories + decisions parity.
- `security` — OWASP, auth flow, secret handling, input validation at boundaries.
- `architect` — module boundaries, layering, SOLID drift, dead abstractions.
- `skeptic` — devil's advocate; over-engineering, scope creep, hidden costs.
- `diagnostics` — only if scope contains failing tests or error paths.

If scope is Apple, add `swiftui-pro` + (where applicable) `apple-design-critic`.
If scope is heavy on scraping, add `scraping-diagnostician` + `anti-bot-evasion` + `proxy-strategist`.

Each Task prompt includes:
- The diff (compressed if >50k chars: per-file hunks + cited file:line)
- Path references to the steerings the agent must enforce
- Hard contract: "Output strict YAML per `.Codex/agents/<role>.md`. Severity-tag every finding (HIGH/MEDIUM/LOW). Cite file:line. No prose. No praise."

### Step 3 — Aggregate (orchestrator only)

Collect all YAML. Dedupe by `(file, line, category, problem-similarity)`. Sort by severity, then by file.

Group:
- **TL;DR** — counts per severity, top 3 must-fix.
- **HIGH (blockers)** — reject the change set.
- **MEDIUM (concerns)** — recommend fix before merge.
- **LOW (notes)** — informational, do not block.
- **Cross-cutting patterns** — multiple files hit by the same finding type.
- **Out-of-scope (declared)** — anything an agent explicitly excluded.
- **Per-agent verbatim YAML** — for audit trail.

### Step 4 — Write the audit report

Append-only write to `swarm-report/audit-<scope-slug>-<YYYY-MM-DD>.md`. Status: `audit-complete`.

### Step 5 — Surface to user

Output to chat:
- TL;DR + counts.
- HIGH findings list verbatim.
- Path to the full report.
- Question: "Apply fixes via `/implementor audit-<slug>`, open a follow-up `/pre-feature` for the architectural concerns, or close as informational?"

**Do NOT** auto-fix. **Do NOT** post comments on the PR. (Use a dedicated `/code-review --comment` skill for that, if installed.)

## Loop guards

- Empty diff → abort cleanly.
- Subagent returns prose → re-spawn once with stricter prompt; second prose → record `agent-failed: <role>` and continue with the rest.
- Total findings count >150 → emit `audit-too-large; narrow the scope`. The report is still written for the record.
- Same audit re-run within 1h on the same scope → emit a one-line diff against the prior report, do not respawn all agents.

## What this skill does NOT do

- Does not modify code.
- Does not commit / push / open PR comments.
- Does not run tests (use `/diagnose` for failing tests, or `/implementor`'s verify gate for full coverage).
- Does not auto-close findings as `accepted-risk`. The user must mark explicitly in their reply.
- Does not skip HIGH-severity findings as "stylistic" — severity is the agent's call.

## Example invocation

```
/audit branch
/audit pr 142
/audit path .Codex/agents/scraping-* --severity HIGH
```

Expected behavior: pick scope, fan out 4–6 reviewers, dedupe + group + severity-sort, write report at `swarm-report/audit-branch-<date>.md`, surface TL;DR + HIGH list + next-step prompt.
