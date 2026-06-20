# yx360-cli

Go CLI for Yandex 360 automation. Current production path is documented Yandex OAuth plus documented Mail IMAP/SMTP.

**PROJECT_TYPE: 2** (production, human-gated deploys).

## Read First

- Complete working agreement: `AGENTS.md`
- Hard invariants: `.assistant/INVARIANTS.md`
- Project knowledge: `.memory-bank/index.md`
- Decisions and open questions: `.assistant/decisions.md`, `.assistant/open-questions.md`

Follow `AGENTS.md` as the source of truth. This file is only a Claude Code entry point and current-state summary.

## Current Stack

- **Language:** Go, single static CLI binary.
- **Auth:** documented Yandex OAuth authorization-code + PKCE against `oauth.yandex.ru`; no embedded `client_secret`.
- **Token storage:** OS keychain by default. Plaintext file storage exists only behind explicit `--insecure-file-store`.
- **Mail:** documented IMAP/SMTP via OAuth scopes `mail:imap_full` and `mail:smtp`.
- **Network:** Yandex OAuth/account-info/IMAP/SMTP calls use IPv4 because the current deployment network has broken Yandex IPv6 routing.

## Current Product Surface

- `yx360 login` signs in with Yandex OAuth.
- `yx360 login --mail` requests Mail read scope.
- `yx360 login --mail --mail-send` requests Mail read + SMTP send scopes.
- `yx360 mail list`, `mail search`, `mail read`, `mail attachment`, and `mail send` operate through IMAP/SMTP.
- `mail send` is human-gated by default; `--yes` is explicit and non-default.
- Expired tokens require re-auth with `yx360 login`; refresh is intentionally unimplemented because the registered Yandex app requires a `client_secret` for refresh.

## Escalation Rule

Use documented Yandex APIs and protocols first. Private web/mobile surface research is allowed only for named gaps where documented APIs are missing or empirically fail, and must go through the scraping/surface consilium path before implementation.

## Agent Notes

- Use `--json` for machine-readable command output.
- Do not scrape human output when JSON exists.
- Never print, persist, or commit OAuth access tokens, refresh tokens, secrets, cookies, or captured browser sessions.
- See `docs/agent-contract.md` for the stable agent-facing CLI contract.
- See `docs/runtime-compatibility.md` for Codex, Claude Code, OpenCode, and OpenClaw compatibility status.
