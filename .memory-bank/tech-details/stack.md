# Stack

> Last updated: 2026-06-20 (research: `swarm-report/research-yandex-360-oauth-cli-login-2026-06-20.md`; login build: `swarm-report/yx360-oauth-login-scaffold-implementation-2026-06-20.md`; Mail build: `swarm-report/mail-inbox-search-attachments-send-implementation-2026-06-20.md` + `swarm-report/mail-send-implementation-2026-06-20.md`)

## Language

- **Go** — single static binary, Homebrew-friendly. Pinned: `go 1.26` + `toolchain go1.26.4` (decided, OQ-001 → D-003).

## Auth — Sign in with Yandex 360 via documented OAuth

Decided 2026-06-20: documented OAuth, **not** token interception / private-endpoint reverse-engineering (supersedes the original scraping framing — see D-001 Notes; pending D-002).

| Concern | Approach | Confidence | Source |
|---------|----------|-----------|--------|
| Grant | authorization-code + PKCE/S256, **public client (no client_secret)** | high | yandex.com/dev/id/doc/en/codes/code-url |
| Login transport (default) | loopback `http://localhost:8899` — system browser + local listener captures `?code=` | medium — fixed port, register 8899 | …/doc/en/register-client |
| Fallback (headless / port busy) | device-authorization flow (`oauth.yandex.com/device/code` → `ya.ru/device`) | high | …/oauth/doc/dg/concepts/device-token |
| Fallback (last resort) | manual-paste `verification_code` behind `--paste` | medium | …/register-client |
| OAuth library | `golang.org/x/oauth2` — native PKCE (`GenerateVerifier`/`S256ChallengeOption`/`VerifierOption`) + `DeviceAuth`; hand-set `Endpoint{authorize,token}` | high | pkg.go.dev/golang.org/x/oauth2 |
| Token storage | OS keychain (`go-keyring`) only — never repo/logs (§12) | high | — |
| Token lifetime | ~12-month access token (personal account); refresh returns new refresh_token | high (personal only) | …/tokens/refresh-client |
| Mail | OAuth bearer via IMAP/SMTP (`mail:imap_full`, `mail:smtp`), no app password | high | live verification + Yandex OAuth app UI |

**Resolved dep versions** (`go mod tidy`, latest-compatible as of 2026-06-20; pinning revisitable):

- `github.com/spf13/cobra` v1.10.2
- `golang.org/x/oauth2` v0.36.0
- `github.com/zalando/go-keyring` v0.2.8
- `github.com/emersion/go-imap/v2` v2.0.0-beta.8
- `github.com/emersion/go-message` v0.18.2
- `github.com/emersion/go-sasl` v0.0.0-20241020182733-b788ff22d5a6

**Live-verified 2026-06-20** (D-004): `yx360 login` round-trips against a real Yandex 360 account. OAuth host is **`oauth.yandex.ru`** — `.com` does not show RU accounts. PKCE code-exchange works with no secret.

**Resolved (was "verify empirically"):**
- **Secretless REFRESH — NOT supported** for the registered confidential app (D-004): refresh without `client_secret` → `invalid_client`; with secret → works. §12 forbids shipping the secret, so `auth.Refresher` stays unimplemented and the strategy is **re-auth at expiry** (~yearly, ~12-month token). Possible unverified lever: a *native/public* app registration might allow secretless refresh.
- **Mail scopes** — `mail:imap_full` for IMAP read/search/read/attachments (D-005) and `mail:smtp` for SMTP send (D-007), both from the Yandex OAuth app UI and live-verified.
- **Yandex IPv6 route** — broken in the current deployment network; Yandex OAuth/account-info/IMAP/SMTP calls use IPv4 `tcp4` until IPv6 is proven reliable (D-006).

**Still verify empirically before code depends on it:**
- **Calendar/Contacts (CalDAV/CardDAV) for personal accounts** — contradictory sources on OAuth-bearer vs app-password. Integration-test against caldav/carddav.yandex.ru via XOAUTH2 before scoping into v1; else out-of-scope.
- **Exact non-Mail scope strings** (`cloud_api:disk.*`, `telemost-api:*`, `directory:*`) — verify each against the live consent screen.
- **Org / Directory scopes** — require Yandex 360 org + admin-enabled service app + written user consent. Personal accounts: Mail/Disk/Telemost self-scope only.

## Other components

| Concern | Candidate approach | Status |
|---------|--------------------|--------|
| CLI framework | `cobra` (actual: v1.10.2) | DECIDED (OQ-001 → D-003) |
| API client | plain Go `net/http` + bearer token against documented APIs (Telemost API, Disk API, Directory); ref `essentialkaos/telemost` | TODO (PR-3+) |
| Distribution | Homebrew tap + GoReleaser | DECIDED, DEFERRED (post-login PR) |
| Agent skill | `.claude/skills/`-style drop-in wrapping the CLI; reserve `--json` output convention at scaffold | DEFERRED |

## Built — PR-1 (`feat/login-oauth`)

Greenfield scaffold + `yx360 login` / `yx360 logout`. Compiles, vets, tests clean on go1.26.4. Live login round-trip still pending B1 (register Yandex OAuth app) + B2 (secretless-refresh test).

**Package layout** (one-way deps `cli → auth → {tokenstore, config}`):
```
cmd/yx360/main.go
internal/cli/{root,login,logout,output}.go
internal/auth/{provider,credential,ladder,oauth,flow_loopback,flow_device}.go
internal/tokenstore/{store,keyring,file}.go
internal/config/config.go
```

- **Flow ladder:** loopback `127.0.0.1:8899` → device. Loopback fails fast to the device rung on EADDRINUSE detected *before* the browser opens; aborts on a real OAuth error. PKCE/S256, **public client (no client_secret)**, 32-byte `state`, 3-min timeout. Redirect string byte-matches registered `http://localhost:8899`. Scopes = `login:info` only.
- **Token storage:** OS keychain (`go-keyring`, service `yx360`) is the default; one JSON blob per credential. Headless keychain failure errors with guidance toward `--insecure-file-store` — a flag-gated plaintext fallback (`os.UserConfigDir()/yx360/credential.json`, mode 0600 — macOS `~/Library/Application Support`, Linux `~/.config`). Never silent plaintext (OQ-006 → flag-gated file store).
- **`auth.Refresher`** is declared but **unimplemented** — intentional seam pending the B2 secretless-refresh test (empty seam, not incomplete work).
- **OQ-001 closed by D-003.**

## Built — Mail IMAP/SMTP

Mail v1 is implemented through documented Yandex Mail protocols, not a private Mail REST API.

**Commands:**
- `yx360 login --mail` requests `mail:imap_full`.
- `yx360 login --mail --mail-send` requests read + SMTP send scopes.
- `yx360 mail list --folder INBOX --limit 20`
- `yx360 mail search --from ... --subject ... --since YYYY-MM-DD --limit 20`
- `yx360 mail read <uid>`
- `yx360 mail attachment <uid> <attachment-id> --out <dir>`
- `yx360 mail send --to ... --subject ... --body ... [--attach file] [--yes]`

**Protocol choices:**
- IMAP: `imap.yandex.ru:993` over TLS, configurable via `YX360_IMAP_HOST`.
- SMTP: `smtp.yandex.ru:465` over TLS, configurable via `YX360_SMTP_HOST`.
- OAuth SASL: XOAUTH2 first, OAUTHBEARER fallback.
- Network: forced IPv4 `tcp4` for Yandex calls (D-006).

**Safety:**
- Mail send defaults to a preview and interactive confirmation. `--yes` is explicit and non-default.
- Bcc recipients are passed to SMTP but omitted from MIME headers.
- Attachment downloads require `--out`, sanitize filenames, write mode 0600, and never auto-open files.
- `auth.Refresher` remains unimplemented; expired tokens require `yx360 login` again.

**Live-verified 2026-06-20:**
- OAuth re-consent with Mail scopes.
- Inbox list, bounded search, message read, attachment download.
- SMTP self-send to the authenticated account and IMAP read-back.

**Known operational note:**
- One combined IMAP search (`--from` + `--subject`) hit a transient Yandex `NO [UNAVAILABLE] UID SEARCH Backend error`; list/read verified delivery. If it repeats, run `/diagnose mail search`.

## Detected at install

- Empty repository at install time — no `go.mod`, no source yet. Greenfield.
- `git init` done at install (branch `main`). Future feature work lands on a branch → PR (ANTI-3 / §10).
