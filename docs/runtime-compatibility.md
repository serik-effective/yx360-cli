# Runtime Compatibility

This matrix documents how a target agent runtime should consume `yx360` as a CLI-backed skill/tool. It is not a claim that every runtime adapter has been smoke-tested.

The install-facing contract is `docs/agent-contract.md`. The root `AGENTS.md` belongs to this repository's own development workflow and must not be copied as the yx360 skill contract into unrelated projects.

## Matrix

| Runtime | yx360 skill contract | Runtime-specific surface | Current status | Verification status |
|---------|------------------------|--------------------------|----------------|---------------------|
| Codex | `docs/agent-contract.md` | Optional `.codex` hooks or plugin/skill packaging when provisioned by an installer | Contract-compatible | yx360 runtime adapter smoke pending unless run in the target Codex environment |
| Claude Code | `docs/agent-contract.md` rendered into a `.claude/skills/.../SKILL.md` or referenced by a target-local `CLAUDE.md` | `.claude/settings.json`, `.claude/agents`, `.claude/skills`, hooks | Contract-compatible | Existing Claude surface is present; per-target install smoke is still required |
| OpenCode | `docs/agent-contract.md` rendered into target-local OpenCode instructions/skill files | `opencode.json`, `.opencode`, or `.agents/skills` where provisioned | Docs-compatible instruction surface | Adapter smoke pending |
| OpenClaw | `docs/agent-contract.md` visible to the OpenClaw agent session | OpenClaw gateway config at `~/.openclaw/openclaw.json` | OpenClaw exists and is docs-verified as a self-hosted gateway | yx360 adapter smoke pending |

## OpenClaw Boundary

Owner clarified that OpenClaw means `https://openclaw.ai/`; canonical docs are at `https://docs.openclaw.ai/`.

Docs verified on 2026-06-20:

- OpenClaw is described as a self-hosted gateway connecting chat apps and channel surfaces to AI coding agents.
- The quick start installs the gateway with `npm install -g openclaw@latest`, runs onboarding with `openclaw onboard --install-daemon`, and opens the dashboard with `openclaw dashboard`.
- Optional configuration lives at `~/.openclaw/openclaw.json`.

This repository has not run an executable OpenClaw smoke test for yx360. Until that happens, say "docs-compatible" or "pending smoke", not "supported and verified".

## Installer Boundary

Installers for these runtimes may:

- Copy or render `docs/agent-contract.md`, runtime settings, hooks, and skill folders.
- Validate syntax.
- Report drift and conflicts.
- Run doctor checks that prove a runtime can discover the expected files.

Installers must not:

- Run `/pre-feature`, `/implementor`, `/research`, or any other development workflow.
- Hide feature work behind commands like `install and fix`.
- Write global runtime config without explicit user approval.
- Claim a runtime is verified unless the matching executable smoke test ran.

## Minimum Smoke Checks

Codex:

- Confirm the target project or Codex skill can read the rendered `yx360` contract.
- If `.codex` hooks are provisioned, confirm hook paths are target-local and executable.

Claude Code:

- Confirm the target `.claude/skills/.../SKILL.md` or `CLAUDE.md` references the `yx360` contract, not this repository's root `AGENTS.md`.
- Confirm `.claude/settings.json` and hook scripts are syntactically valid.

OpenCode:

- Confirm the runtime discovers the rendered `yx360` contract.
- Confirm any `opencode.json`, `.opencode`, or `.agents/skills` files are target-local and valid.

OpenClaw:

- Confirm OpenClaw is installed and can read its config at `~/.openclaw/openclaw.json`.
- Confirm the yx360 project workspace is mounted or otherwise visible to the OpenClaw agent session.
- Run a harmless process smoke such as `yx360 --help`; for authenticated smoke, use only a read-only Mail command against a test credential before claiming adapter smoke.

## Current Recommendation

Use `docs/agent-contract.md` as the canonical cross-runtime yx360 skill contract. Treat OpenCode and OpenClaw as docs-compatible until adapter provisioning and executable smoke checks are implemented.
