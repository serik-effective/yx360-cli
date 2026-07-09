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

**Update 2026-06-20 (D-010):** First org-scoped surface shipped — the Yandex Forms API is Yandex 360 for Business-only, accessed with an `X-Org-Id` header (`YX360_FORMS_ORG_ID`). Access is gated by the authenticated user's own Forms permission rather than a separate Directory/admin grant, so it stays inside the personal-consent posture. Full org-wide Directory/admin capability remains open.

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
**Linked:** D-009, D-010; `swarm-report/calendar-telemost-implementation-2026-06-20.md`
**Status:** open; decide before relying on logout for operational cleanup. **Update 2026-06-20 (D-010):** now a *third* un-cleared profile (`forms`) exists alongside `mail` and `calendar-telemost`; the gap widened.

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

## OQ-014 — Forms list-all endpoint
**Priority:** medium
**Question:** Does the Yandex Forms API expose a documented endpoint to enumerate the forms a user owns/can access, or must `survey_id` always be supplied out-of-band?
**Why it matters:** D-010 shipped without a `forms list` command because no enumeration endpoint was found in the documented examples; agents and users must obtain `survey_id` from the web UI. If an endpoint exists, `forms list` should be added.
**Linked:** D-010; `swarm-report/yandex-forms-get-create-publish-plan-2026-06-20.md`
**Status:** open; re-verify against the full `ru/api-ref` reference before adding `forms list`.

## OQ-015 — Forms live API verification
**Priority:** high
**Question:** Do the Forms endpoint paths, the `X-Org-Id` requirement, the exact `forms:read`/`forms:write` scope strings, and the response JSON shapes match the real Yandex Forms API?
**Why it matters:** D-010 is build/unit-verified only; all Forms request/response shapes come from Yandex docs (live-unverified, C-1). The feature is not "done" per ANTI-11 until a live smoke (`login --forms` → `forms responses <real_survey_id>` → `forms create`/`publish`) passes; structs may need adjustment.
**Linked:** D-010, D-011; `swarm-report/yandex-forms-get-create-publish-implementation-2026-06-20.md`
**Status:** **closed by D-011 on 2026-06-20.** Live end-to-end run (org `7023313`) passed: create→add-questions→publish→read-answers. Contract corrections folded in (org header by id format, create body `name`, raw answer passthrough). Scopes `forms:read`/`forms:write` confirmed live.

## OQ-016 — Forms question authoring
**Priority:** low
**Question:** Should `forms create` support adding questions (e.g. rating/choice fields), or stay title-only with question authoring left to the web UI?
**Why it matters:** D-010's `forms create` sets only the survey title and produces an empty form; building a usable survey (e.g. multi-category rating) currently requires the web editor. The documented API has a `POST /surveys/{id}/questions/` endpoint that could back a `forms questions add` command.
**Linked:** D-010, D-011; `swarm-report/yandex-forms-get-create-publish-plan-2026-06-20.md`
**Status:** **closed by D-011 on 2026-06-20.** `forms questions add <survey-id> --label --rating N` shipped and live-confirmed (enum/radio items 1..N via `POST /v1/surveys/{id}/questions/`). `forms create` stays title-only by design; questions are a separate command.

## OQ-017 — Forms org-id auto-discovery / user-prompt fallback
**Priority:** medium
**Question:** Should the CLI auto-discover the Forms org id, or interactively prompt for it when `YX360_FORMS_ORG_ID` is unset/invalid, instead of failing?
**Why it matters:** The Forms API requires an org id, but no documented endpoint returns it for a `forms:*`-scoped token; today it is operator-configured env (`YX360_FORMS_ORG_ID`). Live testing burned several tries on wrong values (a uid, a hex id) before the numeric `7023313` worked, with only raw API errors as feedback. A prompt-on-missing fallback (owner-requested) would improve out-of-box UX; true runtime auto-discovery likely needs a different API/scope and a surface consilium.
**Linked:** D-011; OQ-014
**Status:** open; decide prompt-fallback (cheap) vs auto-discovery (needs research) before broader rollout.

## OQ-018 — Register `verification_code` redirect + live end-to-end verify for `--manual`
**Priority:** high
**Question:** Has `https://oauth.yandex.ru/verification_code` been registered as a redirect URI in each Yandex OAuth app (mail, calendar-telemost, forms), and has the full `login --manual --begin → browser consent → --complete → token` flow been verified live against a real org account?
**Why it matters:** Yandex matches `redirect_uri` exactly and returns an error if the registered value does not match; no documented port-flexible loopback exists for this redirect. The `--manual` flow is build/unit/smoke-verified (PKCE, state, pending file 0600, secretless exchange via `exchangeCode()`) but is not "done" per ANTI-11 until a live round-trip confirms the `verification_code` display + code exchange succeed.
**Linked:** D-013; `swarm-report/remote-headless-manual-login-implementation-2026-07-10.md`; `swarm-report/remote-headless-manual-login-plan-2026-06-20.md`
**Status:** open; blocked on human B-task (register redirect in Yandex OAuth app UI for each app, then run live smoke).
