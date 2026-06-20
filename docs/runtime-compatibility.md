# Runtime Compatibility

This matrix documents agent-doc compatibility for the current repository. It is not a claim that every runtime adapter has been smoke-tested.

## Matrix

| Runtime | Canonical project docs | Runtime-specific surface | Current status | Verification status |
|---------|------------------------|--------------------------|----------------|---------------------|
| Codex | `AGENTS.md` | Optional `.codex` hooks or plugin/skill packaging when provisioned by an installer | Compatible with committed project instructions | yx360 runtime adapter smoke pending unless run in the target Codex environment |
| Claude Code | `AGENTS.md` via `CLAUDE.md` shim | `.claude/settings.json`, `.claude/agents`, `.claude/skills`, hooks | Compatible with current repository layout | Existing Claude surface is present; per-target install smoke is still required |
| OpenCode | `AGENTS.md` | `opencode.json`, `.opencode`, or `.agents/skills` where provisioned | Docs-compatible instruction surface | Adapter smoke pending |
| OpenClaw | `AGENTS.md` where the project is mounted into an OpenClaw workspace/session | OpenClaw gateway config at `~/.openclaw/openclaw.json` | OpenClaw exists and is docs-verified as a self-hosted gateway | yx360 adapter smoke pending |

## OpenClaw Boundary

Owner clarified that OpenClaw means `https://openclaw.ai/`; canonical docs are at `https://docs.openclaw.ai/`.

Docs verified on 2026-06-20:

- OpenClaw is described as a self-hosted gateway connecting chat apps and channel surfaces to AI coding agents.
- The quick start installs the gateway with `npm install -g openclaw@latest`, runs onboarding with `openclaw onboard --install-daemon`, and opens the dashboard with `openclaw dashboard`.
- Optional configuration lives at `~/.openclaw/openclaw.json`.

This repository has not run an executable OpenClaw smoke test for yx360. Until that happens, say "docs-compatible" or "pending smoke", not "supported and verified".

## Installer Boundary

Installers for these runtimes may:

- Copy or render `AGENTS.md`, `CLAUDE.md`, runtime settings, hooks, and skill folders.
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

- Confirm the target project exposes `AGENTS.md`.
- If `.codex` hooks are provisioned, confirm hook paths are target-local and executable.

Claude Code:

- Confirm `CLAUDE.md` points to `AGENTS.md` and contains only current project deltas.
- Confirm `.claude/settings.json` and hook scripts are syntactically valid.

OpenCode:

- Confirm the runtime discovers `AGENTS.md`.
- Confirm any `opencode.json`, `.opencode`, or `.agents/skills` files are target-local and valid.

OpenClaw:

- Confirm OpenClaw is installed and can read its config at `~/.openclaw/openclaw.json`.
- Confirm the yx360 project workspace is mounted or otherwise visible to the OpenClaw agent session.
- Run a harmless process smoke such as `yx360 --help`; for authenticated smoke, use only a read-only Mail command against a test credential before claiming adapter smoke.

## Current Recommendation

Use `AGENTS.md` as the canonical cross-runtime instruction file. Keep `CLAUDE.md` as a short Claude Code shim. Treat OpenCode and OpenClaw as docs-compatible until adapter provisioning and executable smoke checks are implemented.
