# Agent Docs Easy Install Implementation Report

**Status:** complete
**Date:** 2026-06-20
**Plan:** `swarm-report/agent-docs-easy-install-plan-2026-06-20.md`

## Layers executed

1. Validation — resolved B1 from owner clarification: `openclaw` means `https://openclaw.ai/`; canonical docs exist at `https://docs.openclaw.ai/`.
2. Docs — executed by worker `019ee445-3c95-77a1-a4a0-f9f52cac004d`.
3. Verify — docs smoke, config syntax checks, hook syntax checks, Go test/vet/build.

## Files touched

- `CLAUDE.md` — T1, 43 lines. Rewrote stale token-interception framing around documented OAuth, Mail IMAP/SMTP, and documented-first escalation.
- `README.md` — T2, 83 lines. Added onboarding, CLI build/install, agent-doc surfaces, and verification.
- `docs/agent-contract.md` — T3, 75 lines. Added stable agent-facing CLI contract for JSON, redaction, errors, send gate, and plaintext token warning.
- `docs/runtime-compatibility.md` — T4, 67 lines. Added Codex, Claude Code, OpenCode, and OpenClaw compatibility matrix with OpenClaw smoke still pending.
- `AGENTS.md` — T5, 204 lines total. Removed stale fixed repository-map counts.
- `swarm-report/agent-docs-easy-install-implementation-2026-06-20.md` — implementation report.

Tracked doc numstat:

```text
2	2	AGENTS.md
28	20	CLAUDE.md
```

`README.md` and `docs/*` are new untracked docs in the current dirty tree.

## Per-agent verbatim YAML

```yaml
status: complete
files_changed:
  - path: CLAUDE.md
    summary: Rewrote stale entrypoint around documented Yandex OAuth, Mail IMAP/SMTP, JSON usage, send gate, and private-surface escalation only for named gaps.
  - path: README.md
    summary: Added top-level onboarding with CLI build/install, agent-doc surfaces, provisioning-only boundary, and verification commands.
  - path: docs/agent-contract.md
    summary: Added agent-facing yx360 contract for auth prerequisites, --json usage, token redaction, error handling, mail send human gate, and plaintext token risk.
  - path: docs/runtime-compatibility.md
    summary: Added Codex, Claude Code, OpenCode, and OpenClaw compatibility matrix with OpenClaw docs-verified boundary and pending yx360 adapter smoke.
  - path: AGENTS.md
    summary: Removed stale fixed repository-map counts for tech-details and subagent files.
verify:
  - command: sed -n '1,220p' CLAUDE.md
    exit_code: 0
    output_tail: "Current path is documented OAuth plus Mail IMAP/SMTP; no default token-interception/private-endpoint framing remains."
  - command: sed -n '1,220p' README.md
    exit_code: 0
    output_tail: "Onboarding separates CLI build/install, agent-doc installation surfaces, and verification; OpenClaw support is marked pending smoke."
  - command: sed -n '1,240p' docs/agent-contract.md
    exit_code: 0
    output_tail: "Documents --json, redaction, actionable errors, mail send approval, and --insecure-file-store warning."
  - command: sed -n '1,260p' docs/runtime-compatibility.md
    exit_code: 0
    output_tail: "Matrix covers Codex, Claude Code, OpenCode, and OpenClaw; OpenClaw docs facts are bounded to docs verification."
  - command: rg -n "intercept|token-capture|private endpoints|OpenClaw|opencode|README|tech-details has 9|12 subagent" CLAUDE.md README.md docs AGENTS.md
    exit_code: 0
    output_tail: "Only expected OpenClaw/OpenCode/README references remain; no token-capture, private endpoints, tech-details count, or 12-subagent stale claims."
open_issues:
  - OpenClaw executable adapter smoke was not run; docs intentionally state it remains pending.
```

```yaml
status: complete
files_changed:
  - path: docs/runtime-compatibility.md
    summary: Replaced destructive logout smoke suggestion with harmless `yx360 --help` process smoke and read-only Mail command for authenticated smoke.
verify:
  - command: sed -n '52,66p' docs/runtime-compatibility.md
    exit_code: 0
    output_tail: "OpenClaw smoke now says to use `yx360 --help`, with authenticated smoke limited to read-only Mail against a test credential."
  - command: rg -n "yx360 --json logout|yx360 --help|read-only Mail" docs/runtime-compatibility.md
    exit_code: 0
    output_tail: "Only the corrected `yx360 --help` and read-only Mail wording remains."
open_issues: []
```

## Verify results

```text
sed -n '52,70p' docs/runtime-compatibility.md
exit 0
OpenClaw smoke uses `yx360 --help` for harmless process smoke and read-only Mail against a test credential for authenticated smoke.

rg -n "intercept|token-capture|private endpoints|yx360 --json logout|tech-details has 9|12 subagent" CLAUDE.md README.md docs AGENTS.md
exit 1
No matches.

make test
exit 0
go test ./...
ok github.com/effective-dev-os/yx360-cli/internal/auth (cached)
ok github.com/effective-dev-os/yx360-cli/internal/cli (cached)
ok github.com/effective-dev-os/yx360-cli/internal/mail (cached)
ok github.com/effective-dev-os/yx360-cli/internal/tokenstore (cached)

make vet
exit 0
go vet ./...

make build
exit 0
go build -o bin/yx360 ./cmd/yx360

bash -n .claude/hooks/inject-state.sh
exit 0

bash -n .codex/hooks/inject-state.sh
exit 0

jq empty .claude/settings.json .codex/hooks.json .harness-lock
exit 0
```

Note: the first sandboxed `make build` exited 0 but logged a Go module stat-cache write warning outside the workspace. Re-run unsandboxed exited 0 with clean output.

## Out-of-scope (declared)

- No command that asks harness/yx360 to perform development work (`harness fix`, `harness implement`, `yx360 agent run feature`).
- No global config writes without explicit approval.
- No automatic reading of prior agent session histories without opt-in and redaction.
- No hard claim of OpenClaw support until canonical OpenClaw docs and a real smoke test exist.
- No rewrite of the whole harness file layout in the same PR as docs/install fixes.

## Open issues raised during implementation

- OpenClaw executable adapter smoke remains pending. Docs state "docs-compatible" / "pending smoke", not verified support.
- The worktree was already dirty on `main`; this run intentionally stacked docs changes on top of existing modified/untracked work after `--continue`.

## Suggested commit message + PR title

- Commit: `docs: add agent install documentation`
- PR title: `Document agent install surfaces and runtime matrix`

## Next

Proceed to `/post-feature agent-docs-easy-install` for decisions + memory bank updates, or revise the docs before that gate.
