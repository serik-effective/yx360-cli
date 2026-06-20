# Implementation Report — yx360 OAuth login scaffold (PR-1)

**Slug:** yx360-oauth-login-scaffold
**Date:** 2026-06-20
**Status:** complete
**Branch:** `feat/login-oauth`
**Plan:** `swarm-report/yx360-oauth-login-scaffold-plan-2026-06-20.md`
**Exec agent:** backend (opus), single layer (all Go).

---

## Layers executed

1. **backend** (Go) — one cohesive scaffold (parallel agents would collide on `go.mod`/shared packages). ~5.5 min.

## Files touched (21 created)

| File | Lines | Note |
|---|---|---|
| `go.mod` | 19 | go 1.26 + toolchain go1.26.4 |
| `Makefile` | 18 | build / test / vet / fmt / lint |
| `cmd/yx360/main.go` | 20 | entrypoint |
| `internal/config/config.go` | 31 | OAuth config; `YX360_CLIENT_ID`; scopes = `login:info` only |
| `internal/auth/credential.go` | 32 | Credential + GrantKind + `Valid()` 60s skew |
| `internal/auth/provider.go` | 28 | Provider + AuthOptions; **Refresher declared, unimplemented** (B2 WHY-comment) |
| `internal/auth/ladder.go` | 38 | loopback→device; advance on `errRungUnavailable`, abort on real OAuth error |
| `internal/auth/oauth.go` | 84 | oauth2.Config (DeviceAuthURL in Endpoint); account GET to login.yandex.ru/info; B1 error |
| `internal/auth/flow_loopback.go` | 152 | 127.0.0.1:8899, EADDRINUSE→rung-unavailable *before* browser, PKCE S256, 32-byte state, close-tab page, 3-min timeout |
| `internal/auth/flow_device.go` | 41 | DeviceAuth→stderr prompt→DeviceAccessToken |
| `internal/tokenstore/store.go` | 16 | TokenStore iface + `ErrNoCredential` |
| `internal/tokenstore/keyring.go` | 63 | default; service `yx360`; headless error → `--insecure-file-store` guidance |
| `internal/tokenstore/file.go` | 59 | plaintext `~/.config/yx360/credential.json` mode 0600, flag-gated only |
| `internal/cli/root.go` | 34 | persistent `--json` / `--insecure-file-store`; store selection |
| `internal/cli/output.go` | 17 | `emit(cmd, human, payload)` |
| `internal/cli/login.go` | 78 | ladder + persist; payload = status/account/scopes/expiry, **never token** |
| `internal/cli/logout.go` | 31 | idempotent Clear (ErrNoCredential = success) |
| `*_test.go` (4) | 205 | Credential.Valid; ladder advance-vs-abort; store round-trip + ErrNoCredential; **§12 no-token-in-JSON** |

## Verify results (re-run independently by orchestrator, not just agent-reported)

| Command | Exit | Result |
|---|---|---|
| `gofmt -l .` | 0 | clean |
| `go vet ./...` | 0 | no findings |
| `go build ./...` + `-o bin/yx360` | 0 | built (go1.26.4) |
| `go test ./...` | 0 | auth / cli / tokenstore PASS; config / cmd no tests |
| `yx360 --help` | 0 | command tree renders (login, logout, --json, --insecure-file-store) |
| `YX360_CLIENT_ID= yx360 login --device` | 1 | exact B1 guidance message |
| `grep -ri client_secret cmd internal` | — | none in code (only the B2 WHY-comment) — §12 clean |

## OQ-006 resolution applied

Owner chose **flag-gated file store**: `tokenstore/file.go` (plaintext, 0600) used only behind `--insecure-file-store`; keyring default; headless keyring error points the user at the flag. No silent plaintext.

## Out-of-scope (declared, confirmed absent)

`--paste` rung · `Refresher` impl · API/CalDAV/Disk/Mail/Telemost commands · GoReleaser/brew tap · agent skill.

## Open issues raised during implementation

- **Dep versions drifted from the plan.** `go mod tidy` resolved cobra 1.10.2 / x/oauth2 0.36.0 / go-keyring 0.2.8 (plan named cobra 1.8.x etc.). Plan specified *libraries*, not exact patches; all compile + test clean. Accept, or constrain to the older pins — owner call. (Not a blocker.)
- **Live login round-trip DEFERRED** — needs **B1** (register Yandex OAuth app + `YX360_CLIENT_ID`) and **B2** (empirical secretless-refresh test). The binary compiles and the command tree works without a client_id; `login` fails with B1 guidance. The ANTI-11 real-login verify happens after B1/B2.

## Suggested commit (draft — NOT committed)

```
feat(login): scaffold CLI + Yandex OAuth login (PKCE, loopback+device)

Greenfield Go scaffold: cobra CLI, internal/{cli,auth,tokenstore,config}.
`yx360 login` runs a loopback(8899)->device flow ladder with PKCE/S256
public-client OAuth (no secret), persists the credential to the OS keychain
(or a flag-gated 0600 file via --insecure-file-store), and `yx360 logout`
clears it. Refresher seam declared but unimplemented pending the secretless-
refresh test. Closes OQ-001 (Go 1.26 + cobra + x/oauth2 + go-keyring).
```
PR title: `feat: yx360 login via Yandex OAuth (PKCE loopback + device) — PR-1 scaffold`

## Next

- **B1 + B2** are yours (register app, refresh test). Then the live-login verify.
- `/post-feature yx360-oauth-login-scaffold` — append D-003 (closes OQ-001), update stack.md, draft the PR. (Do NOT run until you're ready; nothing committed yet.)
