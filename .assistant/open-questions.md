# Open Questions

> Unresolved design/scope questions. Close an entry by moving it to `.assistant/decisions.md` as a D-NNN when resolved. Seeded by `/setup`.

---

## OQ-001 — Lock the stack and tooling versions — CLOSED by D-003 (2026-06-20)
Resolved: `go 1.26` + `toolchain go1.26.4`; cobra; `golang.org/x/oauth2`; `zalando/go-keyring`; GoReleaser + Homebrew tap (deferred). See D-003.

## OQ-002 — Token-interception mechanism — CLOSED (mooted by D-002, 2026-06-20)
Mooted: the project no longer intercepts a token. Login uses documented OAuth (loopback PKCE → device flow → manual paste). The credential-type / webview-vs-headless / webhook questions no longer apply. See D-002.

## OQ-004 — Interception vs documented OAuth — RESOLVED (D-002, 2026-06-20)
Resolved in favour of documented OAuth. See D-002.

## OQ-005 — Captured-credential type — MOOTED (D-002, 2026-06-20)
No interception ⇒ no captured session credential to classify. The OAuth token is the credential (12-month personal-account access token + refresh token).

## OQ-003 — Web vs mobile reverse-engineering boundary
Vision starts with the web surface and escalates to mobile only if needed. Define the trigger: which capabilities justify moving to APK/Frida analysis of the mobile app vs staying on the web surface.

## OQ-INV-1 — Authorization / ToS posture (NARROWED by D-002, 2026-06-20)
Softened by the OAuth decision: documented OAuth with explicit user consent is ToS-defensible for **personal-account** scopes (Mail/Disk/Telemost self-data). Remaining open: **org / Directory** scopes require a Yandex 360 organization + admin-enabled service application + written user consent — confirm before shipping any org-wide capability.

## OQ-006 — Headless / CI token storage — CLOSED by D-003 (2026-06-20)
`zalando/go-keyring` errors when no OS secret service is present (headless Linux / CI / Docker, no D-Bus). Resolved: **flag-gated plaintext file store** (`--insecure-file-store` → `~/.config/yx360/credential.json`, mode 0600); keychain remains the default; a headless keychain error points the user at the flag. Never silent plaintext. Implemented in PR-1.
