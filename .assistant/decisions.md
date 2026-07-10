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

---

## D-010 — Yandex Forms surface added (responses + gated create/publish), live-unverified
**Date:** 2026-06-20
**Status:** accepted
**Decision:** `yx360` gains a Forms surface through the documented Yandex Forms API (`https://api.forms.yandex.net`, `Authorization: OAuth <token>` + `X-Org-Id`, IPv4 per D-006). Commands: `forms responses <survey-id>` (read, ungated, paginated), and `forms create --title` / `forms publish <survey-id>` / `forms unpublish <survey-id>` (externally-visible writes, human-gated with preview + non-default `--yes`, per ANTI-2). Auth uses a **separate** `forms` credential profile and OAuth app (`YX360_FORMS_CLIENT_ID`) with scopes `forms:read` + `forms:write`, plus `YX360_FORMS_ORG_ID` for the `X-Org-Id` header; `login --forms` rejects mixing with mail/calendar scopes (D-009 pattern, now 3-way). Refresh stays unimplemented; re-auth at expiry (D-004). Scope decision: Option B (read + gated write) per owner.
**Why now:** Owner has a Yandex 360 for Business org (the Forms API is Business-org-only; personal accounts are excluded) and provisioned `YX360_FORMS_CLIENT_ID` + `YX360_FORMS_ORG_ID`, unblocking the consilium's B-1 feasibility gate. Built on branch `feat/yandex-forms-get-create-publish` in an isolated git worktree to stay clear of in-progress calendar-room-booking work.
**Verification status:** build/vet/`go test ./...` green, including a `forms` unit test (scope guard, org-id guard, 401→reauth, tolerant decode). **Live API smoke NOT yet run** — endpoint paths and response JSON come from Yandex docs and are flagged live-unverified (C-1); `forms` JSON structs decode tolerantly and may need adjustment after the first real call. Not "done" per ANTI-11 until the live smoke passes.
**Alternatives rejected:**
- Personal-account Forms: impossible — documented Forms API is Yandex 360 for Business-only.
- `forms list` / enumerate-all-surveys: not built — no documented enumeration endpoint (C-1/R3); `survey_id` is user-supplied.
- Read-only v1 only (Option A): owner chose Option B (include gated create/publish) since the `forms:write` scope was already provisioned.
- Private forms.yandex.ru web-surface scraping: unnecessary — documented REST API covers the core path.
- Question authoring in `forms create`: out of scope v1 — `create` sets only the title (empty survey).
**Source:** `swarm-report/yandex-forms-get-create-publish-plan-2026-06-20.md`, `swarm-report/yandex-forms-get-create-publish-implementation-2026-06-20.md`. Consilium agents architect/skeptic/researcher/reviewer; exec agent `a8fbeeac416c59114`. Forms credentials handled only via untracked `.env`.
**Note:** D-010 may collide with a parallel D-010 on the calendar-room-booking branch; resolve by renumbering at merge.
**Closes:** plan OQ-NEW-A (org-account feasibility — confirmed). Partially addresses OQ-INV-1 (first org-scoped surface ships, but gated by the user's own Forms permission + `X-Org-Id`, not full Directory admin).
**Raises:** OQ-014 (Forms list-all endpoint), OQ-015 (Forms live verification), OQ-016 (Forms question authoring). Reinforces OQ-011 (now a third un-cleared credential profile).

---

## D-011 — Forms surface live-verified; API contract corrected; question authoring + link derivation added
**Date:** 2026-06-20
**Status:** accepted
**Amends:** D-010 (its "live-unverified / C-1" caveat is now resolved). D-010 stays as-is per §8.
**Decision:** The Forms surface is **live-verified end to end** against a real Yandex 360 for Business org (org id `7023313`, account `serik.beysenov@effective.band`): create → add 5 rating questions → publish → read submitted answers all succeeded. Live testing forced several documented-contract corrections, now in code:
1. **Org header by id format:** numeric org id → `X-Org-Id`; non-numeric (hex) → `X-Cloud-Org-Id`. The earlier hex value failed with `Требуется организация`; the numeric `7023313` worked.
2. **Create body field is `name`, not `title`** (`{"name": "<title>"}`).
3. **`forms questions add <survey-id> --label --rating N`** added (OQ-016): posts an `enum`/`radio` question with items `1..N` to `POST /v1/surveys/{id}/questions/`. Live-confirmed.
4. **Public links derived by the CLI** (the API returns none): published form `https://forms.yandex.ru/cloud/<survey_id>`, answer stats `https://forms.yandex.ru/cloud/admin/<survey_id>/answers?view=stats`. Printed by `create`/`publish` and in their JSON.
5. **`forms responses` now surfaces the full raw answer payload** via `Answer.MarshalJSON` — real answer fields are `id` (numeric), `created`, and `data[].value`, which did not match the assumed `id`/`respondent_id`/`submitted_at`, so typed-only output dropped the data.
6. HTTP error bodies are surfaced (`httpError`) instead of bare status codes.
**Why now:** Owner ran the live end-to-end flow (build a 5-category event-rating form), which both proved the surface and exposed the contract mismatches above; fixing them in the same session is the ANTI-11 "feature actually works" gate.
**Alternatives rejected:**
- Keeping `title` in the create body: rejected — API requires `name` (HTTP 400 otherwise).
- Sending both org headers always: rejected — pairing a numeric value with `X-Cloud-Org-Id` (or vice-versa) yields `Требуется организация`; select by format.
- Typed-only answer decode: rejected — drops real answer content; raw passthrough preserves fidelity (field names vary).
- Runtime org-id auto-discovery: rejected for now — no documented endpoint for a forms-scoped token; org id is a one-time operator config (`YX360_FORMS_ORG_ID`), with a user prompt as the fallback (future work).
**Source:** live session 2026-06-20 (survey `6a368226969f14081ef9ece7`, 1 submission read: Контент=1, Спикеры=1, Организация=3, Локация=4, Нетворкинг=2); `swarm-report/yandex-forms-get-create-publish-implementation-2026-06-20.md`. Branch `feat/yandex-forms-get-create-publish`. Forms credentials only via untracked `.env`.
**Closes:** OQ-015 (Forms live verification — done), OQ-016 (Forms question authoring — `forms questions add` shipped + live-confirmed).
**Raises:** OQ-017 (Forms org-id auto-discovery / user-prompt fallback). OQ-014 (list-all endpoint) and OQ-011 (profile-aware logout) remain open.

---

## D-012 — Harness synced 698eb86 → 6ea46b4

**Date:** 2026-07-10
**Status:** accepted
**Decision:** Harness updated `698eb86` → `6ea46b4` (5 commits) via `/sync` from a fresh checkout at `~/dev/harness`.
**Counts:** +14 add ~3 overwrite -0 delete ↻0 restore; 0 conflicts.
**Added:** 8 Stop-gate hooks (`cost-cap`, `evidence-gate`, `lint-gate`, `loop-detect`, `orphan-guard`, `read-imperative`, `slop-gate`, `verify-gate`) + `.gitkeep`; `reflect` skill (+ `reflect_extract.py`); 3 embedded agents (`cortex-m-low-level`, `embedded-build`, `embedded-c-reviewer`).
**Overwritten:** `.claude/hooks/inject-state.sh`, `.claude/settings.json` (registers the new hooks), `.claude/skills/implementor/SKILL.md`.
**Template updates pending manual merge:** `.assistant/decisions.md`, `.assistant/open-questions.md`, `.memory-bank/index.md`, `.memory-bank/product-overview/vision.md`, `CLAUDE.md` — project-owned, upstream seed changed; merge by hand if wanted.
**Local drift left untouched:** `AGENTS.md` (target-edited, upstream unchanged).
**Note:** 3 embedded agents are irrelevant to this Go/CLI project but are harness-owned framework files; harmless. New Stop-gate hooks are now active in-session.
**Source:** `git@github.com:effective-dev-os/harness.git@6ea46b4dd7af80cdf774f168022c3df00e1dbb26`

---

## D-013 — `yx360 login --manual`: headless two-step authorization-code + PKCE (`verification_code` redirect)

**Date:** 2026-07-10
**Status:** accepted
**Decision:** Added `yx360 login --manual --begin/--complete` for browser-less remote/VDS hosts. `--begin` resolves the credential profile and scopes from the same flags as the interactive flow, prints the Yandex auth URL (redirect set to `https://oauth.yandex.ru/verification_code`), and persists a 0600 pending-state file (`UserConfigDir/yx360/manual-login.json`, mode 0600, dir 0700, 10-min TTL) containing `{code_verifier, state, profile, scopes, clientID}`. `--complete --code <code-or-redirect-url>` loads the pending state, validates CSRF via `subtle.ConstantTimeCompare`, exchanges the code secretlessly via PKCE (reusing the extracted `exchangeCode()` helper from `oauth.go`), stores the token in the resolved credential profile, and deletes the pending file. Device flow is out of scope: Yandex requires `client_secret` for device-code→token exchange (same gap as D-004). Verifier is never printed; pending file deleted on success and failure.
**Why now:** P0 roadmap item: VDS/remote/headless hosts where no local browser and no loopback port are reachable cannot use the existing loopback → device ladder. Owner waived the three blockers from the consilium: B-1 (register the `verification_code` redirect URI in each Yandex OAuth app — human task, tracked OQ-018), B-2 (explicit `--insecure-file-store` required, no silent plaintext — satisfied in code by `selectStoreFor(profile)`), and B-3 (0600 pending-state file per spec — satisfied in code).
**Alternatives rejected:**
- **Device Authorization Grant** — Yandex device-code→token exchange requires `client_secret` (plan researcher, corroborated); blocked for public CLI same as refresh (D-004).
- **Single-process in-memory state** — rejected: not agent-driveable when `--begin` and `--complete` are separate process invocations (consilium architect).
- **Paste a localhost redirect URL** — rejected: requires a listener on the remote host, broken "can't connect" page UX, SSH port-forward complexity; consilium researcher pivoted to `verification_code` display redirect.
- **Generic OOB `urn:ietf:wg:oauth:2.0:oob`** — deprecated; Yandex `verification_code` redirect is the provider-specific still-supported variant.
**Source:** `swarm-report/remote-headless-manual-login-plan-2026-06-20.md` (consilium: architect + skeptic + researcher + reviewer); `swarm-report/remote-headless-manual-login-implementation-2026-07-10.md`.
**Closes:** nothing yet — OQ-018 (register redirect + live end-to-end verify) stays open until B-1 lands and a full `--begin → browser consent → --complete → token` run passes (ANTI-11).
**Raises:** OQ-018.

---

## D-014 — Yandex Disk surface added (list/get/put/share/unshare/rm/mkdir + netutil refactor)
**Date:** 2026-07-10
**Status:** accepted
**Decision:** `yx360` gains a Disk surface through the documented Yandex Disk REST API (`https://cloud-api.yandex.net/v1/disk/`, `Authorization: OAuth <token>`, IPv4 per D-006). Commands: `disk list [--path /] [--limit N] [--offset N]` (read, ungated, paginated), `disk get <remote-path> [--out dir]` (path-traversal-safe two-step download), `disk put <local-file> --to <path>` (two-step upload via `io.Copy`; `--yes` required to overwrite on 409 Conflict), `disk share <remote-path> --yes` (publish to public URL; human-gated per ANTI-2), `disk unshare <remote-path>` (revoke public URL, non-destructive, ungated), `disk rm <remote-path> --yes [--permanent]` (trash by default; human-gated; async 202 poll up to 5×1s), `disk mkdir <remote-path>` (non-recursive, ungated). Auth uses a separate `disk` credential profile and OAuth app (`YX360_DISK_CLIENT_ID`) with scopes `cloud_api:disk.read` + `cloud_api:disk.write` per D-009 pattern; obtained via `yx360 login --disk`. Refactored `ipv4Client()` — extracted shared `internal/netutil.IPv4Client()` from three-way duplication across `forms`, `telemost`, and `calendar` services, eliminating D-006 drift risk. Service handles 413 (file too large) and 507 (quota full) with human-readable messages; upload URL 30-min TTL covered by streaming `io.Copy`.
**Why now:** Owner provisioned a Yandex OAuth app for Disk (`YX360_DISK_CLIENT_ID` set) immediately after the Forms live-verify (D-011), unblocking B-2. Disk is the core storage surface of Yandex 360 and the natural next surface after Mail/Calendar/Forms. The `ipv4Client()` duplication reached three copies during planning, triggering extraction now per C-8.
**Alternatives rejected:**
- WebDAV transport (`webdav.yandex.ru`): REST preferred for v1; public-link management, native `overwrite` param, and metadata queries are simpler via REST. WebDAV deferred; tracked in OQ-020.
- Interactive `--yes` prompt on destructive/visible actions: rejected — non-interactive is mandatory for agent use (ANTI-2); non-`--yes` prints dry-run preview and exits non-zero.
- Recursive directory download (`disk get` on directory path): out of scope v1; service returns typed error; recursive `disk pull` deferred to OQ-020.
- Chunked/resumable upload: out of scope v1; two-step upload with streaming `io.Copy` covers most cases; OQ-019.
- Single OAuth app for all scopes: rejected per D-009 precedent — Yandex returns `invalid_scope` for mixed scope sets across services.
**Source:** `swarm-report/yandex-disk-support-plan-2026-07-10.md`, `swarm-report/yandex-disk-support-implementation-2026-07-10.md`. Consilium: architect + skeptic + researcher + reviewer (4/4 strict YAML). PR: https://github.com/serik-effective/yx360-cli/pull/4. Branch: `feat/yandex-disk-support`.
**Verification status:** `go build` / `go vet ./...` / `go test ./...` all green; `disk --help` shows 7 subcommands; `--yes` gates smoke-confirmed (share/rm without `--yes` print preview + exit); `login --disk` starts OAuth flow → `invalid_scope` (expected: B-1 scope strings not yet registered in OAuth app UI). Not "done" per ANTI-11 until B-1 resolved and live `login --disk → disk list` passes.
**Closes:** nothing previously open (OQ-019 and OQ-020 are newly raised by this feature).
**Raises:** OQ-019 (chunked/resumable upload for large files), OQ-020 (WebDAV vs REST for future COPY/MOVE/recursive ops). OQ-011 now has a 4th un-cleared profile (`disk`).
