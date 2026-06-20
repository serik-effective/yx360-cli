# Decisions Log

> Append-only chronological record. When a decision is overturned, add a new entry with date + reason. Never edit or delete prior entries.

---

## D-001 — Harness installed
**Date:** 2026-06-20
**Status:** accepted
**Decision:** Effective Harness installed at commit `698eb86` via the `/setup` skill. `PROJECT_TYPE: 2`. Primary stack: Backend/CLI, Go.
**Source:** `git@github.com:effective-dev-os/harness.git@698eb86489901cb0cd49d3c4a91643730dc5c1ea`
**Touch policy chosen at install:** N/A — empty target directory, nothing to overwrite.
**Notes:** Target was an empty, non-git directory at install. Owner to `git init` before the first PR (ANTI-3). Domain is scraping / anti-bot (Yandex 360 private-API reverse-engineering), so the `surface-scout` / `scraping-architect` / `scraping-diagnostician` / `anti-bot-evasion` agents are in scope alongside `backend`. **Superseded in part by D-002** — the auth approach is now documented OAuth, not interception.

---

## D-002 — Login via documented Yandex OAuth, not token interception
**Date:** 2026-06-20
**Status:** accepted
**Decision:** `yx360 login` uses **documented Yandex OAuth** — authorization-code + PKCE/S256 as a public client (no `client_secret`). Flow ladder: loopback `http://localhost:8899` (default) → device-authorization flow (headless / port-busy fallback) → manual-paste `verification_code` (last resort). Token stored in the OS keychain only (§12). OAuth lib: `golang.org/x/oauth2` (native PKCE). This **supersedes** the private-endpoint / token-interception framing in D-001.
**Why:** A prior `/pre-feature` consilium + `/research` pass found (a) Yandex ships documented OAuth that needs no embedded secret, (b) the proposed loopback-webhook could not capture an httpOnly session cookie anyway, and (c) the target services (Mail/Disk/Telemost, and Calendar/Contacts via CalDAV/CardDAV) are reachable with a documented OAuth token. OAuth is also far more ToS-defensible than interception.
**Source:** `swarm-report/research-yandex-360-oauth-cli-login-2026-06-20.md` (+ `…/yx360-login-token-interception-plan-2026-06-20.md`). External facts dated 2026-06-20 — re-verify per §6 before code relies on them.
**Open risks (verify empirically):** secretless token *refresh* (docs contradict); CalDAV/CardDAV bearer-auth for personal accounts; exact scope strings; org/Directory needs admin consent.

---

## D-003 — Stack locked + PR-1 login scaffold built (closes OQ-001)
**Date:** 2026-06-20
**Status:** accepted
**Decision:** Go CLI stack locked and the first slice (PR-1, branch `feat/login-oauth`) built: `go 1.26` + `toolchain go1.26.4`; CLI framework **cobra**; OAuth via `golang.org/x/oauth2` (native PKCE); token storage `github.com/zalando/go-keyring` with a flag-gated plaintext fallback (`--insecure-file-store`, OQ-006). Package layout `cmd/yx360` + `internal/{cli,auth,tokenstore,config}`, deps one-way `cli → auth → {tokenstore, config}`. `yx360 login` runs a loopback(8899)→device flow ladder (PKCE/S256, public client, no `client_secret`, scope `login:info`); `yx360 logout` clears the store. Distribution (GoReleaser + Homebrew tap) decided but DEFERRED to a post-login PR. `auth.Refresher` declared but intentionally unimplemented pending B2.
**Why now:** First real feature per D-002; OQ-001 required pinning the toolchain + framework + distribution before code. Verify gate green (gofmt/vet/build/test, re-run by orchestrator).
**Alternatives rejected:** `urfave/cli` (cobra is the de-facto Go CLI standard + GoReleaser path); embedded webview / headless browser (system browser + loopback is RFC 8252 best practice, avoids the embedded-webview anti-pattern); silent plaintext token file on headless (chose explicit `--insecure-file-store` flag, OQ-006).
**Source:** `swarm-report/yx360-oauth-login-scaffold-plan-2026-06-20.md` + `…-implementation-2026-06-20.md`.
**Resolved dep versions:** cobra v1.10.2, x/oauth2 v0.36.0, go-keyring v0.2.8 (latest-compatible as of 2026-06-20; pinning revisitable).
**Closes:** OQ-001, OQ-006.
**Raises:** none. Pending human tasks (tracked in the plan, not OQs): B1 (register Yandex OAuth app → `YX360_CLIENT_ID`), B2 (empirical secretless-refresh test). Live-login verify (ANTI-11) waits on B1.

---

## D-004 — Live login verified; OAuth domain is .ru; secretless refresh not supported
**Date:** 2026-06-20
**Status:** accepted
**Decision:** PR-1 `yx360 login` passed the live ANTI-11 verify against a real Yandex 360 account (`arseniy.savin@effective.band`). Two findings folded in:
1. **OAuth host must be `oauth.yandex.ru`, not `oauth.yandex.com`.** On `.com` the Passport session does not see RU Yandex 360 accounts ("my accounts aren't shown"). Fixed `AuthURL`/`TokenURL`/`DeviceAuthURL` to `.ru` in `internal/config/config.go`. (`login.yandex.ru/info` was already `.ru`.)
2. **Secretless token refresh is NOT supported for the registered (confidential) app.** Empirically: `grant_type=refresh_token` without `client_secret` → `invalid_client: Wrong client secret`; with the secret → succeeds (new access + rotated refresh). The PKCE *code exchange* works without a secret (login succeeds), but *refresh* demands it. Since §12 forbids shipping a `client_secret` in a brew-distributed binary, `auth.Refresher` stays UNIMPLEMENTED; the strategy is **re-auth at expiry** — `yx360 login` again, ~once a year given the ~12-month token. §12-clean (no secret anywhere in the CLI).
**Why now:** B1 (app registered, `YX360_CLIENT_ID` set) unblocked the live verify; ran B2 in the same pass.
**Alternatives rejected:** refresh-with-secret (would require embedding `client_secret` in the distributed binary — §12 violation); making the user supply a secret at runtime (defeats the public-client UX and still risks leakage). Re-auth-at-expiry chosen.
**Note (unverified lever):** re-registering the Yandex app as a *native/installed* (public) client *might* permit secretless refresh — not tested. Yearly re-auth is acceptable; revisit only if it proves annoying.
**Source:** live verify this session; `swarm-report/yx360-oauth-login-scaffold-implementation-2026-06-20.md`. Secret handled only via untracked `.env` (gitignored in both repos).
**Closes:** the B2 open risk (secretless refresh).

---

## D-005 — Mail read scope identified
**Date:** 2026-06-20
**Status:** accepted
**Decision:** Yandex OAuth app UI exposes `mail:imap_full` for Mail IMAP access. Use this as the read-side scope for inbox listing, search, message read, and attachment fetch. SMTP/send scope is still unresolved; `mail send` remains a separate later slice.
**Why now:** Owner checked the Yandex OAuth app UI after the fresh `/pre-feature` report raised OQ-007.
**Source:** Owner report in this session on 2026-06-20. UI-derived, not found in public docs; live login + IMAP auth still must verify it before implementation is marked done.
**Closes:** read-side part of OQ-007.

---

## D-006 — Yandex network calls use IPv4, not IPv6
**Date:** 2026-06-20
**Status:** accepted
**Decision:** `yx360` uses IPv4 (`tcp4`) for Yandex OAuth/account-info HTTP calls and Yandex Mail IMAP TLS calls. Do not rely on Go's default dual-stack dialing for these endpoints until the deployment network proves IPv6 is reliable.
**Why now:** Live `login --mail` and `mail list` both failed when Go selected IPv6 Yandex addresses (`2a02:...`) with `socket is not connected`; the same endpoints were reachable over IPv4, and the full read-only Mail smoke passed after forcing IPv4.
**Scope:** Applies to Yandex OAuth (`oauth.yandex.ru`, `login.yandex.ru`) and Mail IMAP (`imap.yandex.ru`) in this CLI. This is not a general ban on IPv6 for unrelated future integrations.
**Source:** Live verification on 2026-06-20 during `mail-inbox-search-attachments-send` implementation; see `swarm-report/mail-inbox-search-attachments-send-implementation-2026-06-20.md`.

---

## D-007 — Mail send scope identified
**Date:** 2026-06-20
**Status:** accepted
**Decision:** Yandex OAuth app UI exposes `mail:smtp` for Mail SMTP/send access. Use this as the send-side scope for `yx360 mail send`, separate from read-side `mail:imap_full`.
**Why now:** Owner checked the Yandex OAuth app UI after `/implementor send mail` blocked on unresolved SMTP/send scope.
**Source:** Owner report in this session on 2026-06-20. UI-derived, not found in public docs; live login + SMTP auth/send still must verify it before implementation is marked complete.
**Closes:** remaining SMTP/send part of OQ-007.

---

## D-008 — Mail IMAP/SMTP feature completed
**Date:** 2026-06-20
**Status:** accepted
**Decision:** `yx360` now supports Yandex 360 Mail read/search/read-attachment/send through documented IMAP/SMTP with OAuth scopes. `mail:imap_full` gates read-side commands (`mail list`, `mail search`, `mail read`, `mail attachment`); `mail:smtp` gates `mail send`. Send is human-gated by default with a preview and confirmation; `--yes` is explicit and non-default.
**Why now:** Owner wanted the Mail feature completed end-to-end after read-only Mail passed live verification and the SMTP scope was found in the Yandex OAuth app UI.
**Alternatives rejected:**
- Private Mail REST API: no public individual-message REST API was found; documented IMAP/SMTP is sufficient for v1.
- App-password auth: rejected for v1 because the accepted project direction is documented OAuth, no extra user-managed mail secret.
- Non-interactive send by default: rejected because sending email is externally visible and must stay behind a human gate.
**Source:** `swarm-report/mail-inbox-search-attachments-send-plan-2026-06-20.md`, `swarm-report/mail-inbox-search-attachments-send-implementation-2026-06-20.md`, `swarm-report/mail-send-implementation-2026-06-20.md`.
**Closes:** OQ-007.
**Raises:** OQ-010.

---

## D-009 — Calendar CalDAV and Telemost create are live-verified
**Date:** 2026-06-20
**Status:** accepted
**Decision:** `yx360` now supports Calendar list/read/create/update/delete through documented CalDAV and Telemost conference creation through the official Telemost API. Calendar uses `calendar:all` and `Authorization: OAuth <token>`; `Bearer` auth was live-tested and rejected by CalDAV. Calendar/Telemost use a separate OAuth app and `calendar-telemost` credential profile because Yandex rejects mixing Mail, Calendar, and Telemost scopes in one OAuth application.
**Why now:** Owner wanted the CLI to read events, create/update/delete meetings, and create Telemost links after proving the required Yandex OAuth scopes and live endpoints. The feature also satisfies the non-Mail Yandex 360 surface milestone in the product definition of done.
**Alternatives rejected:**
- One OAuth app for Mail + Calendar + Telemost: rejected because the Yandex OAuth UI/API returned `invalid_scope` / service-count errors for the mixed scope set.
- `Authorization: Bearer <token>` for CalDAV: rejected because live CalDAV proof returned `401 Basic realm="CalDAV"`; `Authorization: OAuth <token>` returned `207 Multi-Status`.
- Private Calendar web/mobile endpoints: rejected because documented CalDAV covered v1 CRUD.
- Telemost conference deletion/cancellation in v1: rejected because no official delete/cancel endpoint was verified.
- Recurring events, shared/delegated calendars, rooms/resources, and org directory lookup: rejected as out-of-scope for the narrow personal-account v1.
**Source:** `swarm-report/calendar-telemost-plan-2026-06-20.md` and `swarm-report/calendar-telemost-implementation-2026-06-20.md`.
**Closes:** none.
**Raises:** OQ-011, OQ-012, OQ-013.
