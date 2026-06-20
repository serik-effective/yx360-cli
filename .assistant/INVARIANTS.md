# Effective Harness — INVARIANTS

> Hard rules. Every subagent reads this. Breaking any of these is a bug, not a tradeoff. If a proposal violates an invariant — push back loudly, don't silently work around.

## §1. Harness = files

Effective Harness is a **drop-in file layout** (`.claude/`, `.memory-bank/`, `.assistant/`, `AGENTS.md`). After install, the developer uses Claude Code / OpenCode / Codex directly.

### §1.1. No CLI wrapper for the dev workflow
There is no `harness do "сделай оплату"` / `harness implement X` / `harness fix Y` command. Developers invoke Claude Code / OpenCode / Codex directly; harness activates through files (agents, hooks, skills, memory bank). Any feature that requires a custom CLI as the **runtime** entry point = anti-pattern, reject.

### §1.2. Bootstrap and update utilities are outside §1.1
A one-off installer (bash script, `npx` initializer, git submodule, SessionStart hook that pulls upstream, etc.) **provisions files** — it does not wrap features. Treat install/update tooling as **deployment**, not as part of the runtime. These tools are allowed and may include a CLI surface (e.g., `harness install`, `harness sync`) as long as they only manage files and never invoke dev-workflow features.

**Test for compliance with §1:** does the command modify project files / refresh the harness layout (allowed, §1.2)? Or does it invoke a development task like "build feature X" / "fix bug Y" (forbidden, §1.1)?

## §2. Skills are task-based, not project-based
Skills named by **task type**: `/pre-feature`, `/research`, `/implementor`, `/post-feature`, `/audit`, `/diagnose`, `/refactor`, `/defrag`. Never `/harness-consilium` / `/gift-card-consilium` / `/meetily-consilium`. Same skill works in every project where harness is installed.

## §3. Subagent output: strict YAML, no prose
Auditors and consilium subagents return structured findings — no markdown prose, no preamble. Schema (universal):
```
- severity: HIGH | MEDIUM | LOW
  category: <role-specific>
  file: path/or/n-a
  line: <int or n-a>
  problem: <one sentence>
  suggested_fix: <≤2 sentences>
  requires_human: true | false
  confidence: high | medium | low | corroborated | unverified
```
Orchestrator dedupes by `(file, line, category)`, aggregates, writes report. No raw prose-dump from subagents back to user.

## §4. Soft edit-guard: orchestrator doesn't write code
The main session (orchestrator) does **not** Edit/Write files under `.memory-bank/`, `.claude/agents/`, `.claude/skills/`, `.claude/hooks/` directly. Modifications go through exec-agents (per file scope) spawned via Task. Exception: `.assistant/decisions.md` (append-only) and `swarm-report/*` (orchestrator-owned outputs).

## §5. Internet required for research-class agents
`architect`, `security`, `researcher`, `api`, `devops` agents must have MCP access to web search (`mcp-omnisearch` or equivalent). Without internet they hallucinate. Narrow exec-agents from an approved plan can run offline.

## §6. 30-day re-verify for facts
Any fact older than 30 days about external systems (Claude Code APIs, MCP servers, LiteLLM, dae_codex, Kiro, etc.) is **stale by default**. Before recommending or basing a decision on it — re-verify via web search. Drift goes into `.assistant/decisions.md` with new dated entry, old entry stays.

## §7. Memory bank = source of truth
On conflict between agent's internal knowledge and `.memory-bank/` — trust memory bank, update internal model. On conflict between code and `.memory-bank/` — update memory bank (or surface the divergence for human review).

## §8. Append-only decisions log
`.assistant/decisions.md` is append-only. When a decision is overturned, add a revision entry with date + reason. Never edit or delete prior entries — they explain *why* current state exists.

## §9. Confidence flags on web research
All findings from web-search agents carry `confidence: high | medium | low | corroborated | unverified`. Single-source findings never auto-apply to memory bank — they go to swarm-report and require human review.

## §10. No push to main, no force-push, no `--no-verify`
Feature branches → PR → review → merge. Destructive git ops (`reset --hard`, `push --force`, `branch -D`) require explicit user confirmation. Hooks bypass (`--no-verify`, `--no-gpg-sign`) forbidden unless user asks verbatim.

## §11. Stack agnostic
Harness does not impose tech stack. Each consuming project declares its stack in its own `STACK.md` / `CLAUDE.md`. The harness defines **roles + pipeline + gates**, not technologies.

## §12. No secrets in memory bank or `.assistant/`
Only public facts, template configs with placeholders, source links. API keys, tokens, real IPs, internal URLs → never committed. Credential management is project-specific (env vars, secret manager).

---

Last updated: 2026-06-09
