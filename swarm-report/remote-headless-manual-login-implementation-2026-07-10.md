# Implementation — Remote/Headless Login (manual-paste authorization-code + PKCE)

**Slug:** `remote-headless-manual-login`
**Date:** 2026-07-10
**Plan:** `swarm-report/remote-headless-manual-login-plan-2026-06-20.md`
**Branch:** `feat/remote-headless-manual-login` (off `feat/forms-question-types` @ c6fe019)
**Status:** `complete`

---

## Blockers (all human-waived before implement)

- **B-1** — register `https://oauth.yandex.ru/verification_code` redirect in each OAuth app. Owner: "это ок, другого варианта нет". Human B-task, tracked OQ-018. Not code — must be done in Yandex app UI before live use.
- **B-2** — headless token store stays explicit `--insecure-file-store`, never silent plaintext. Owner: "ОК на явный plaintext на VDS". Satisfied: `--complete` calls `selectStoreFor(profile)` → keyring errors on headless with the existing "re-run with --insecure-file-store" hint; no silent fallback.
- **B-3** — PKCE-verifier-at-rest store. Owner: "согласен с консилиумом". Satisfied: `0600` file `UserConfigDir/yx360/manual-login.json`, dir `0700`, 10-min TTL, deleted on complete **and** failure, gitignored, verifier never printed.

## Layers executed

1. **backend** (`internal/auth` + `internal/cli` + `internal/config`) — single exec agent (one cohesive security-sensitive change, same package = no parallel split). ~10.5m.
2. **docs** (`docs/agent-contract.md` + `README.md`) — single exec agent. ~3.4m.
3. **verify** — orchestrator. build/vet/test + `--begin` smoke.

## Files touched

New:
- `internal/auth/flow_manual.go` — `ManualProvider` Begin/Complete, pending-state file (`pendingManualLogin`, write/load/delete), `parseManualInput`, `LoadPendingProfile`.
- `internal/auth/flow_manual_test.go` — unit tests: parseManualInput (bare code / URL / error-param / state-mismatch), Begin pending-file, Complete rejection paths (missing/expired/oversized).

Modified:
- `internal/config/config.go` — `VerificationCodeRedirectURI` const.
- `internal/auth/credential.go` — `GrantManual GrantKind = "manual"`.
- `internal/auth/oauth.go` — reusable `exchangeCode()` helper.
- `internal/auth/flow_loopback.go` — inline `conf.Exchange` → `exchangeCode()`.
- `internal/cli/login.go` — `--manual/--begin/--complete/--code` gate at top of RunE; `resolveManualTarget` helper; `manualBeginPayload`. Default loopback/device path unchanged.
- `.gitignore` — `manual-login.json` defensive ignore.
- `docs/agent-contract.md` — manual JSON examples, payload shape, "Headless Manual Login" section.
- `README.md` — "Вход без браузера (headless/VDS)" section.

## Verify results

| cmd | exit | tail |
|-----|------|------|
| `go build -o bin/yx360 ./cmd/yx360` | 0 | BUILD_OK |
| `go vet ./...` | 0 | VET_OK |
| `go test ./...` | 0 | all ok (auth incl. new flow_manual_test) |
| `login --manual --begin` smoke (dummy client id) | 0 | see below |

Smoke (`YX360_CLIENT_ID=smoke-test-id ... --json login --manual --begin`):
- `auth_url` carries `redirect_uri=https://oauth.yandex.ru/verification_code` ✓
- `code_challenge_method=S256` + `code_challenge` present (PKCE) ✓
- `state` present ✓
- pending file written `-rw-------` (0600) ✓
- JSON output = `{status, auth_url}` only — **no verifier, no token** ✓
- artifact removed post-smoke.

Not exercised live (needs B-1 + real org + human browser): full `--begin → browser consent → --complete → token`. This is the ANTI-11 live gate that stays open until B-1 lands.

## Out-of-scope (declared, carried from plan)

- Device flow (Yandex device-code→token needs `client_secret`) — dropped from v1.
- Token refresh (D-004) — unchanged.
- Profile-aware logout (OQ-011) — unchanged.
- Generic OOB `urn:…:oob` — not used.

## Open issues raised during implementation

1. **Pending file not profile-keyed** — single `manual-login.json`, not `manual-login.<profile>.json` (plan N-4 suggested profile-key). Two concurrent `--begin` clobber. Acceptable for single-user VDS; note for future multi-profile concurrent begin.
2. **`--code` on argv** — visible briefly in `ps` / shell history (plan N-5 stdin-read not built). Mitigated: code single-use, 10-min TTL, immediate exchange + pending-delete. Low risk; stdin variant is a cheap future add.
3. **Live end-to-end unverified** — blocked on B-1 (register redirect). Feature is build/unit/smoke-verified only; not "done" per ANTI-11 until a live run passes. → OQ-018.
4. **Plan cited ADR "D-012"** — now occupied by harness-sync D-012. Use **D-013** in `/post-feature`.

## Suggested commit message (draft — not committed)

```
feat(auth): headless two-step manual login (verification_code + PKCE)

Add `yx360 login --manual --begin/--complete` for browser-less remote
hosts. --begin prints the auth URL and persists a 0600 pending-state
file (verifier/state/profile, 10m TTL); --complete exchanges the pasted
code (or redirect URL) secretless via PKCE and stores the token in the
resolved profile. Verifier never printed; pending file deleted on
complete and failure. Device flow stays out (Yandex needs client_secret).
```

PR title: `feat(auth): headless manual login (verification_code + PKCE)`

## Next

- `/post-feature remote-headless-manual-login` — append **D-013**, close nothing yet (OQ-018 stays open until live), draft PR.
- Human: register `https://oauth.yandex.ru/verification_code` in mail/calendar-telemost/forms OAuth apps (B-1 / OQ-018), then live smoke.
