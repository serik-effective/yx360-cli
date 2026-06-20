# yx360-cli

`yx360` is a Go CLI for Yandex 360 automation. The implemented surface signs in with documented Yandex OAuth and uses documented Mail IMAP/SMTP for mailbox read/search/read-attachment/send workflows.

The harness and agent docs in this repository are provisioning artifacts. They help Codex, Claude Code, OpenCode, and OpenClaw-style runtimes find the same project instructions; they are not a workflow wrapper and must not run feature work on behalf of the developer.

## CLI Build And Install

Prerequisites:

- Go 1.26 with toolchain `go1.26.4`.
- A Yandex OAuth client id in `YX360_CLIENT_ID`.
- Mail access enabled in Yandex 360 Mail settings before Mail commands are used.

Build locally:

```bash
go build -o bin/yx360 ./cmd/yx360
```

Run checks:

```bash
go test ./...
go vet ./...
```

Sign in:

```bash
YX360_CLIENT_ID=<client-id> ./bin/yx360 login
YX360_CLIENT_ID=<client-id> ./bin/yx360 login --mail --mail-send
```

Mail examples:

```bash
./bin/yx360 mail list --limit 20
./bin/yx360 mail search --from user@example.com --since 2026-06-01 --json
./bin/yx360 mail read <uid> --json
./bin/yx360 mail attachment <uid> <attachment-id> --out ./downloads
./bin/yx360 mail send --to user@example.com --subject "Hello" --body "Text"
```

`mail send` shows a preview and asks for confirmation by default. Use `--yes` only when the caller has already obtained an explicit human approval for that send.

## Agent-Doc Installation Surfaces

Canonical project instructions live in `AGENTS.md`. Runtime-specific files should point at or preserve that source of truth instead of duplicating a second policy body.

Current surfaces:

| Runtime | Project entrypoint | Status |
|---------|--------------------|--------|
| Codex | `AGENTS.md` | Docs-compatible project instructions. Adapter smoke is separate from CLI smoke. |
| Claude Code | `CLAUDE.md` plus `AGENTS.md` | `CLAUDE.md` is a short current-state shim. |
| OpenCode | `AGENTS.md` | Docs-compatible instruction surface; runtime adapter smoke remains pending. |
| OpenClaw | `AGENTS.md` as project context where mounted into an OpenClaw workspace | OpenClaw is docs-verified as a self-hosted gateway, but yx360 adapter smoke remains pending. |

OpenClaw notes are based on the canonical docs at `https://docs.openclaw.ai/`, which describe OpenClaw as a self-hosted gateway and document configuration at `~/.openclaw/openclaw.json`. This repository does not yet claim a verified OpenClaw adapter for yx360.

Installer work must stay provisioning-only: copy, render, validate, sync, or doctor files. It must not implement features, run consilium workflows, or hide externally visible actions behind an install command.

## Verification

Docs-only verification:

```bash
sed -n '1,220p' CLAUDE.md
sed -n '1,220p' README.md
sed -n '1,240p' docs/agent-contract.md
sed -n '1,260p' docs/runtime-compatibility.md
```

CLI verification:

```bash
go test ./...
go vet ./...
go build -o bin/yx360 ./cmd/yx360
```

Live Mail verification requires a real Yandex OAuth client id and account consent. Do not claim live runtime or OpenClaw adapter smoke unless those commands have actually been run in the target runtime.
