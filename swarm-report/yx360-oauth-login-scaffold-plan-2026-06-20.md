# Consilium Plan — yx360 OAuth login scaffold (PR-1)

**Slug:** yx360-oauth-login-scaffold
**Date:** 2026-06-20
**Status:** consilium-complete
**Feature:** Scaffold the Go CLI + `yx360 login` via documented Yandex OAuth (auth-code + PKCE, public client). First real feature. Closes OQ-001.
**Decided basis:** D-002 (OAuth, not interception). PROJECT_TYPE 2.
**Agents:** architect (opus), skeptic (opus), researcher, reviewer.

> This document IS the Type-2 hash-locked plan (ANTI-4). It needs owner sign-off before `/implementor`.

---

## TL;DR

- The plan is concrete and buildable. **Severity (dedup):** HIGH 5 (4 human-gated) · MEDIUM 6 · LOW 3.
- **3 hard prerequisites before any auth code:**
  1. **Register a Yandex OAuth app** → public `client_id` + redirect `http://localhost:8899`. Human task, blocks the slice. (`client_id` is public, §12-safe to commit; there is NO secret.)
  2. **Empirically test secretless refresh** (`grant_type=refresh_token`, no secret) — Yandex docs contradict. If it fails → re-auth-at-expiry, store access+expiry only, **never embed a client_secret** (§12).
  3. **Device-flow fallback is mandatory in PR-1**, not deferred — fixed port 8899 hard-fails when occupied/headless; loopback-only login is a day-1 regression.

## Architecture (from architect, verified by researcher)

**Package layout** (one-way deps: `cli → auth → {tokenstore, config}`):
```
cmd/yx360/main.go                      thin cobra entrypoint
internal/cli/{root,login,logout,output}.go
internal/auth/{provider,credential,ladder,flow_loopback,flow_device}.go
internal/tokenstore/{store,keyring}.go
internal/config/config.go
```

**Seams (keep the open refresh question isolated):**
- `auth.Provider interface { Authenticate(ctx, AuthOptions) (*Credential, error) }`
- `auth.Refresher interface { Refresh(ctx, *Credential) (*Credential, error) }` — **UNIMPLEMENTED in PR-1**, behind a type-assertion, pending the refresh test.
- `auth.Credential struct { AccessToken, RefreshToken, TokenType string; Expiry time.Time; Scope, Account string; ObtainedVia GrantKind }` — persisted as ONE JSON blob (one keychain item); `RefreshToken` may be empty.
- `tokenstore.TokenStore interface { Save, Load, Clear }` + sentinel `ErrNoCredential` (distinguishes logged-out from broken).
- `auth.Ladder` — rungs loopback → device → paste; advances ONLY on a classified `errRungUnavailable` (EADDRINUSE on `net.Listen("127.0.0.1:8899")` *before* opening browser / `--no-browser` / 3-min timeout); **aborts immediately on a real OAuth error** (consent denied, invalid_grant) — no pointless fall-through.

**PKCE:** use x/oauth2 native `GenerateVerifier()` + `S256ChallengeOption()` + `VerifierOption()`. Set `DeviceAuthURL` inside the same `oauth2.Endpoint` as AuthURL/TokenURL (researcher-confirmed field). No hand-rolled S256, no third-party PKCE lib.

**Loopback detail:** redirect string must byte-match the registered `http://localhost:8899` (host `localhost`, even though we `net.Listen` on `127.0.0.1`); validate `state` (32-byte, constant-time); serve "you may close this tab"; `Shutdown` in defer.

## OQ-001 closure (record as D-003 in /post-feature)

| Decision | Value | Source |
|---|---|---|
| Go toolchain | `go 1.26` + `toolchain go1.26.4` (latest stable 2026-06-02; 1.25.11 conservative alt) | go.dev/doc/devel/release |
| CLI framework | **cobra** v1.8.x (no urfave/cli; no cobra-cli generator) | architect |
| OAuth lib | `golang.org/x/oauth2` (native PKCE + device flow, stable) | pkg.go.dev/golang.org/x/oauth2 |
| Keychain | `github.com/zalando/go-keyring` (canonical, maintained) | github.com/zalando/go-keyring |
| Distribution | GoReleaser → Homebrew tap — **decided, DEFERRED** to a post-login PR | goreleaser.com |

## PR slicing

- **PR-1 (this slice):** scaffold + `yx360 login` (loopback → device fallback) + `yx360 logout` (idempotent, clears keychain) + token→keychain + a single login-validation call (`login.yandex.ru/info`) to populate `Credential.Account` and prove the token is live. `--json` persistent root flag wired (token NEVER in JSON/logs). Minimal scope set only.
- **OUT of PR-1:** manual-paste rung, `Refresher` impl, any API/Disk/Mail/Telemost command, CalDAV/CardDAV, GoReleaser/brew tap, agent skill.
- **PR-2:** `--paste` rung + `yx360 whoami`.
- **PR-3+:** first API command, refresh decision (post empirical test), distribution.

## Blockers (HIGH, requires_human)

- **B1 — Register OAuth app.** Human prerequisite: `client_id` + `http://localhost:8899` redirect. Code reads `client_id` from config/const (override via `YX360_CLIENT_ID` env for test apps). Blocks the slice.
- **B2 — Secretless-refresh empirical test** is a gating task *inside* PR-1, before storage/refresh code. Failure → re-auth-at-expiry; never embed a secret (§12).
- **B3 — Device-flow fallback in PR-1** (not deferred). Detect EADDRINUSE → fall to device flow.
- **B4 — Branch → PR.** First feature lands on `feat/login-oauth` → PR → review (ANTI-3/§10). The install commit to `main` was a one-off owner waiver; it does NOT extend to feature work.
- **B5 — Type-2 architecture sign-off (ANTI-4).** This plan is the hash-locked plan; owner must approve before `/implementor`.

## Concerns (MEDIUM)

- **C1 — Minimal scopes only.** Request the smallest set that yields a token (e.g. `login:info`), verified once against the live consent screen. Do NOT hardcode disk/mail/telemost/directory speculatively (scope strings are medium-confidence, fragmented docs).
- **C2 — Headless keychain failure.** `zalando/go-keyring` **errors** on no-D-Bus/Secret-Service (researcher-confirmed; its only non-OS provider is in-memory `MockInit`, test-only — no persistent pure-Go fallback). PR-1 must catch keyring errors and fail with guidance, OR offer a file-store behind an explicit flag — **never silently write the token to a plaintext file** (§12). Test login on one headless Linux target before "done".
- **C3 — Personal-account scope only.** Slice assumes personal accounts (12-mo token). Org/Directory (1-hr service-app tokens, admin consent) deferred — OQ-INV-1.
- **C4 — Device-flow S256 unverified against Yandex.** x/oauth2 supporting it ≠ Yandex honoring it. Keep the S256-on-device toggle in config; integration-test. If Yandex rejects, device rung runs without S256 (user-code is its own PoP).
- **C5 — §12 token redaction** in all log/error/`--json` paths; add a test asserting the bearer is absent from JSON output.
- **C6 — Verify gate = real login round-trip** via `/verify` (ANTI-11), not unit-tests-only.

## Notes (LOW)

- **N1** — Live integration test (real authorize→token S256 round-trip against `oauth.yandex.com`) in PR-1's verify gate.
- **N2** — D-003 closes OQ-001; write in `/post-feature`, not mid-implementation. Record that `auth.Refresher` is intentionally unimplemented pending the refresh test (so the empty seam isn't read as incomplete).
- **N3** — GoReleaser cross-repo formula push needs a PAT (`HOMEBREW_TAP_GITHUB_TOKEN`) with contents:write on the tap repo — default GITHUB_TOKEN can't push cross-repo. (later slice)

## Open questions raised

- **OQ-006 — Headless/CI token storage.** What is the §12-clean fallback when no OS secret service exists? (error-and-guide vs flag-gated file store vs `99designs/keyring` file backend). Decide in C2.

## Process note

Reviewer flagged "no plan file provided" (HIGH) — a process artifact: the architect produces this plan in the same consilium round, so the reviewer critiqued the prompt summary, not the assembled spec. Its substantive findings (branch→PR, §12 no-secret, ANTI-4 sign-off, ANTI-11 verify gate) are folded into B2/B4/B5/C5/C6 above. Not an open blocker.

## Agent IDs (re-query via SendMessage)
architect=ac76df81e7a0b8dc8 · skeptic=a1744c324386d6133 · researcher=ab8fac235aa566f7a · reviewer=a5ad25e493f3536ea
