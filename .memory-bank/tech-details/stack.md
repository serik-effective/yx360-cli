# Stack

> Last updated: 2026-06-20 (research: `swarm-report/research-yandex-360-oauth-cli-login-2026-06-20.md`; PR-1 build: `swarm-report/yx360-oauth-login-scaffold-implementation-2026-06-20.md`)

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
| Mail | OAuth bearer via XOAUTH2 (imap/smtp.yandex.com), no app password | high | tech.yandex.com/oauth/doc/imap |

**Resolved dep versions** (`go mod tidy`, latest-compatible as of 2026-06-20; pinning revisitable):

- `github.com/spf13/cobra` v1.10.2
- `golang.org/x/oauth2` v0.36.0
- `github.com/zalando/go-keyring` v0.2.8

**Live-verified 2026-06-20** (D-004): `yx360 login` round-trips against a real Yandex 360 account. OAuth host is **`oauth.yandex.ru`** — `.com` does not show RU accounts. PKCE code-exchange works with no secret.

**Resolved (was "verify empirically"):**
- **Secretless REFRESH — NOT supported** for the registered confidential app (D-004): refresh without `client_secret` → `invalid_client`; with secret → works. §12 forbids shipping the secret, so `auth.Refresher` stays unimplemented and the strategy is **re-auth at expiry** (~yearly, ~12-month token). Possible unverified lever: a *native/public* app registration might allow secretless refresh.

**Still verify empirically before code depends on it:**
- **Calendar/Contacts (CalDAV/CardDAV) for personal accounts** — contradictory sources on OAuth-bearer vs app-password. Integration-test against caldav/carddav.yandex.ru via XOAUTH2 before scoping into v1; else out-of-scope.
- **Exact scope strings** (`cloud_api:disk.*`, `mail:imap_*`, `telemost-api:*`, `directory:*`) — verify each against the live consent screen.
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

## Detected at install

- Empty repository at install time — no `go.mod`, no source yet. Greenfield.
- `git init` done at install (branch `main`). Future feature work lands on a branch → PR (ANTI-3 / §10).
