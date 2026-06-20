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

**Update 2026-06-20:** D-009 live-verified Calendar personal-account CRUD through documented CalDAV (`calendar:all`) and Telemost conference creation through the official Telemost API (`telemost-api:conferences.create`). The open part remains org-wide Directory/admin capabilities and any future shared/delegated-calendar behavior.

## OQ-006 — Headless / CI token storage — CLOSED by D-003 (2026-06-20)
`zalando/go-keyring` errors when no OS secret service is present (headless Linux / CI / Docker, no D-Bus). Resolved: **flag-gated plaintext file store** (`--insecure-file-store` → `~/.config/yx360/credential.json`, mode 0600); keychain remains the default; a headless keychain error points the user at the flag. Never silent plaintext. Implemented in PR-1.

## OQ-007 — Exact Yandex Mail OAuth scopes — CLOSED by D-005 + D-007 (2026-06-20)
Resolved: read-side Mail uses `mail:imap_full`; SMTP/send uses `mail:smtp`. Both came from Yandex OAuth app UI. Live SMTP self-send passed during implementation.

## OQ-010 — IMAP combined search backend instability
**Priority:** low
**Question:** Does Yandex IMAP reliably support combined `FROM` + `SUBJECT` searches on the target mailbox, or do we need a client-side fallback for combined filters?
**Why it matters:** During Mail send verification, one combined search returned `NO [UNAVAILABLE] UID SEARCH Backend error`, while list/read confirmed the sent message. If this repeats, `yx360 mail search` may need retry or staged filtering.
**Linked:** D-008; `swarm-report/mail-send-implementation-2026-06-20.md`
**Status:** open; run `/diagnose mail search` if the failure repeats.

## OQ-011 — Profile-aware logout
**Priority:** medium
**Question:** Should `yx360 logout` clear all credential profiles by default, or expose profile-specific flags such as `--mail`, `--calendar-telemost`, and `--all`?
**Why it matters:** D-009 added separate `mail` and `calendar-telemost` credential profiles, while the existing logout command still clears only the legacy default credential slot.
**Linked:** D-009; `swarm-report/calendar-telemost-implementation-2026-06-20.md`
**Status:** open; decide before relying on logout for operational cleanup.

## OQ-012 — Calendar event identifiers and field clearing
**Priority:** low
**Question:** Should Calendar commands keep using raw CalDAV `href` values as event identifiers, and how should users intentionally clear fields like description, location, URL, or attendees?
**Why it matters:** Event hrefs are stable but not ergonomic, and the current update command treats empty string flags as "not provided" rather than "clear this field".
**Linked:** D-009; `swarm-report/calendar-telemost-implementation-2026-06-20.md`
**Status:** open; revisit after basic Calendar usage settles.

## OQ-013 — Telemost conference cleanup
**Priority:** low
**Question:** Is there an official Telemost endpoint or supported workflow to cancel/delete a created conference link?
**Why it matters:** D-009 can create Telemost links, including links attached to Calendar events, but smoke-test and partial-failure links may remain live because no official cleanup endpoint was verified.
**Linked:** D-009; `swarm-report/calendar-telemost-plan-2026-06-20.md`; `swarm-report/calendar-telemost-implementation-2026-06-20.md`
**Status:** open; research only when cleanup becomes necessary.
