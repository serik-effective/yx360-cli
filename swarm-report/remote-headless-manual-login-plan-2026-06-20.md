# Pre-Feature Consilium — Remote/Headless Login (manual-paste authorization-code + PKCE)

**Slug:** `remote-headless-manual-login`
**Date:** 2026-06-20
**Status:** consilium-complete
**Proposal (verbatim):** "remote/headless login: agent prints OAuth URL, user authenticates in their own local browser, pastes the localhost redirect URL (or code) back, agent exchanges code+PKCE-verifier for the token (manual-paste authorization-code+PKCE, two-step begin/complete); reuse the secretless code exchange; device flow optional secondary."
**Consilium:** architect + skeptic + researcher + reviewer (4/4 strict YAML).

---

## TL;DR

- **Counts:** 3 Blockers (HIGH, requires_human) · 8 Concerns (MEDIUM) · 6 Notes (LOW).
- **Buildable, with one design pivot.** The feature fits the project (public-client PKCE, secretless code-exchange — D-002/D-004) and resolves roadmap P0.
- **Pivot (researcher, decisive):** don't paste a broken-`localhost` redirect URL. Use Yandex's **documented `https://oauth.yandex.ru/verification_code` display redirect** — it shows the auth code in the operator's own browser; the operator pastes just the **code** (no URL parsing, no localhost listener, no SSH tunnel). Secretless with PKCE per Yandex docs. Source: <https://yandex.com/dev/id/doc/en/codes/screen-code>.
- **Device flow is OUT (corroborated):** Yandex requires `client_secret` for the device-code→token exchange — same gap that blocks refresh (D-004). Drop the "optional secondary" rung. Source: <https://yandex.com/dev/id/doc/en/codes/screen-code-oauth>.

**Top 3 must-fix before code:**
1. **B-1** — register `https://oauth.yandex.ru/verification_code` as a redirect_uri in **each** Yandex OAuth app (mail, calendar-telemost, forms). Human B-task; Yandex matches redirect_uri exactly and does not document port-flexible loopback, so this is a prerequisite.
2. **B-3** — spec the PKCE-verifier-at-rest store for the two-step begin/complete (0600 file under config dir, expiry ~10 min, profile-keyed, deleted on complete **and** on failure) — §12.
3. **B-2** — headless storage must require an explicit `--insecure-file-store`; never silently fall back to plaintext (OQ-006).

---

## Blockers (HIGH, requires_human: true)

### B-1 — Register the `verification_code` redirect in each OAuth app
- **Anchor:** researcher (Yandex redirect rules), D-002
- **Problem:** Yandex ignores any redirect_uri not pre-registered exactly, and does not document RFC-8252 port-flexible loopback. The `localhost`-paste variant is fragile; the `verification_code` display redirect must be registered to work.
- **Fix:** Operator adds `https://oauth.yandex.ru/verification_code` to the Web-services redirect list of the mail / calendar-telemost / forms apps. Tracked as a human B-task (like B1 client-id registration in D-003). Source: <https://yandex.com/dev/id/doc/en/codes/code-url>.

### B-2 — Headless token storage stays explicit plaintext, never silent
- **Anchor:** ANTI-2, OQ-006, D-003 (skeptic + reviewer HIGH)
- **Problem:** The named use case (server, no keychain) forces token storage onto `--insecure-file-store` plaintext. If `--manual` auto-falls-back to plaintext when keychain is absent, it bypasses the non-silent-plaintext guarantee OQ-006 established.
- **Fix:** Reuse the existing explicit `--insecure-file-store` path + its loud warning; `--manual` must error (not silently store plaintext) when keychain is unavailable and the flag is absent.

### B-3 — Spec the PKCE-verifier/state store between begin and complete (§12)
- **Anchor:** §12/H-6 (skeptic + architect + reviewer HIGH)
- **Problem:** Two-step begin/complete must persist `{code_verifier, state, profile, scopes, clientID}` across two processes. The verifier is sensitive (binds the code). Unspecified storage = secret-at-rest hole.
- **Fix:** Persist a pending-state file at `os.UserConfigDir()/yx360/manual-login.json`, mode `0600` (dir `0700`), **keyed by profile**, with a ~10-min expiry; refuse stale; delete on success **and** on failure. Not the keychain (verifier is ephemeral, pre-token), not `os.TempDir` (predictable/world-readable). Add path to `.gitignore`. Never echo verifier to stdout/stderr. (Alternative: single long-lived interactive process holding verifier in memory only — rejected as not agent-driveable.)

---

## Concerns (MEDIUM)

- **C-1 — Separate provider, not a ladder rung.** Manual-paste is two-step with human interaction between calls; it does not fit `Provider.Authenticate(ctx,opts)`. Add a `ManualProvider` with `Begin(ctx,opts) (authURL,err)` + `Complete(ctx, codeOrURL) (*Credential,err)`, invoked from `login.go` behind `--manual`; leave `Ladder`/`Provider` (loopback/device) untouched. (`internal/auth/ladder.go:12`)
- **C-2 — Extract reusable secretless exchange.** The exchange is inlined at `internal/auth/flow_loopback.go:119` (`conf.Exchange(..., oauth2.VerifierOption(verifier))`). Extract `exchangeCode(ctx, conf, code, verifier, via)` in `oauth.go`; both loopback and manual call it. Headless-safe, no secret (D-004).
- **C-3 — CSRF state validation on complete.** Loopback validates `state` in-handler (`flow_loopback.go:81`, constant-time). `Complete` must load persisted `state` and `subtle.ConstantTimeCompare` it; fail closed on mismatch/missing.
- **C-4 — Code expiry UX.** Auth codes are single-use, ~10-min; the begin→browser→paste round-trip can exceed it. Map `conf.Exchange` `invalid_grant` to an explicit "code expired or already used — re-run `login --manual --begin`" message, not a raw oauth2 error.
- **C-5 — Pasted-input handling.** With `verification_code` the operator pastes the **displayed code** (simple). Still accept a full redirect URL too (parse `code`+`state` from query via `url.Parse`); validate scheme/host/path against the expected redirect, bound input length, treat as untrusted, surface `error` param.
- **C-6 — `--manual` composes with profile flags.** Reuse the profile/clientID/scope resolution in `login.go:34-80` at `--begin`; persist resolved profile in the pending-state; `--complete` reads profile from the file (not re-passed flags) so `selectStoreFor(profile)` routes correctly. Reject `--manual` + `--device`.
- **C-7 — Document `--manual` in the agent contract.** Add a section to `docs/agent-contract.md`: begin emits the auth URL only; complete takes the one-time code (or redirect URL); no token/verifier ever printed; success emits the existing token-free login object (status/account/scopes/expiry).
- **C-8 — Drop device flow from v1.** Yandex device-code→token exchange requires `client_secret` (researcher, corroborated) — blocked for our public CLI, same as refresh. Not "optional secondary"; out of scope.

## Notes (LOW)
- **N-1** — Add `GrantManual GrantKind = "manual"` alongside `GrantLoopback`/`GrantDevice` (`internal/auth/credential.go:11`); pass through `credentialFromToken`.
- **N-2** — Gate the entire manual branch behind `--manual` at the top of `login.go` RunE and return early, so default/loopback/device login (`login.go:82-94`) runs byte-for-byte unchanged.
- **N-3** — Emit the same token-free result object on `complete`; rely on existing redaction.
- **N-4** — Concurrency/crash: profile-keyed pending-state avoids two `--begin` clobbering one file; clean stale on next `--begin`.
- **N-5** — Optional: `--complete` may read the code from stdin when `--redirect-url` omitted (interactive feel for humans); the agent path always passes it explicitly.
- **N-6** — Record an ADR (D-012) noting manual-paste `verification_code` as the headless primary, superseding the loopback→device→verification_code ladder envisioned in D-002.

---

## Research findings (confidence + sources, 2026-06-20)

| # | Finding | Confidence | Source |
|---|---------|-----------|--------|
| R1 | Yandex `…/verification_code` redirect **displays the code** in the browser for copy-paste; valid 10 min. | medium | [screen-code](https://yandex.com/dev/id/doc/en/codes/screen-code) |
| R2 | That flow is **secretless with PKCE** — "if `code_verifier` is passed you don't need the secret key." | medium | [screen-code](https://yandex.com/dev/id/doc/en/codes/screen-code) |
| R3 | Secretless PKCE code-exchange is documented Yandex behavior (corroborates empirical loopback). | corroborated | [screen-code](https://yandex.ru/dev/id/doc/en/codes/screen-code) |
| R4 | redirect_uri must match a **pre-registered** value exactly; loopback port-flexibility not documented. | medium | [code-url](https://yandex.com/dev/id/doc/en/codes/code-url) |
| R5 | Yandex device flow exists (RFC-8628 style) BUT **device-code→token exchange requires `client_secret`** — blocks it for our public CLI. | medium | [screen-code-oauth](https://yandex.com/dev/id/doc/en/codes/screen-code-oauth) |
| R6 | Prior art: `gh` uses device flow (provider doesn't need secret); `gcloud --no-browser` uses print-URL→paste-code two-leg — matches our begin/complete. | medium/low | [GitHub](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps), [WorkOS](https://workos.com/blog/oauth-2-1-whats-new) |
| R7 | Generic OOB (`urn:ietf:wg:oauth:2.0:oob`) is deprecated; Yandex `verification_code` is provider-specific and still supported. PKCE + state + single-use code mitigate. | medium | [Google native-app](https://developers.google.com/identity/protocols/oauth2/native-app) |
| R8 | OAuth 2.1 / BCP (RFC 9700) mandate PKCE + state for every auth-code flow; loopback/device preferred, paste acceptable when no browser/loopback reachable. | medium | [OAuth 2.1 draft](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-10), [RFC 8252](https://www.rfc-editor.org/rfc/rfc8252.html), [RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636) |

---

## Out-of-scope (declared)
- **Device Authorization Grant** — blocked (Yandex needs `client_secret`); already shipped as `flow_device.go` for terminals but not the remote-paste solution. Not extended here.
- Token **refresh** — unchanged (unimplemented, D-004); re-auth at expiry.
- **Profile-aware logout** (OQ-011) — adjacent, not required for v1.
- Generic OOB `urn:…:oob` redirect — deprecated, not used.

## Open questions raised
- **OQ-018** — track the human B-task: register `https://oauth.yandex.ru/verification_code` in all three OAuth apps; verify the `verification_code` display flow + secretless PKCE exchange live before marking done (§6/ANTI-11).
- Decision point for the implementor: pending-state store shape (0600 config-dir file vs in-memory single-process) — consilium recommends the 0600 file (two-step, agent-driveable).

---

## Per-agent verbatim (audit trail)

### architect (a3178d02d5007e385)
MEDIUM: extract reusable `exchangeCode` from `flow_loopback.go:119`. HIGH: not a ladder rung → separate `ManualProvider` Begin/Complete, leave Ladder untouched. HIGH: persist `{verifier,state,profile,scopes,clientID}` to `UserConfigDir/yx360/manual-login.json` 0600, expiry, not keychain/TempDir. MEDIUM: reuse `state` constant-time compare (`flow_loopback.go:81`). MEDIUM: reuse registered `localhost:8899` redirect (avoid deprecated OOB). MEDIUM: parse full URL vs bare code (query, `error` param). MEDIUM: `--manual` composes with profiles; read profile from file at complete. LOW: delete pending file on success+failure; add `GrantManual`; keep device as secondary; gate manual branch so default login unchanged; ADR D-012.

### skeptic (a36b858137a5c0341)
HIGH: pasted code is credential-material until exchanged → never log/print/echo to transcript. HIGH: unspecified verifier-at-rest store = §12 hole → spec 0600 file + delete. MEDIUM: state must be reloaded+compared at complete or CSRF silently dropped. MEDIUM: localhost redirect → browser "can't connect" page UX trap (researcher pivots this to verification_code). MEDIUM: code expiry window → map to "re-run --begin". MEDIUM: device flow = scope creep, drop. MEDIUM: forces silent `--insecure-file-store` plaintext (OQ-006) → keep explicit. LOW: validate/limit hostile pasted URL. LOW: concurrency/crash leaves verifier on disk → profile-key + clean stale.

### researcher (a23d90135a3afffae)
Yandex `verification_code` display redirect = documented, secretless-with-PKCE, exact fit (R1-R3). redirect_uri must be pre-registered exactly, no documented port-flex (R4). **Device flow requires client_secret → blocked** (R5). Prior art `gh`(device)/`gcloud --no-browser`(paste) (R6). Generic OOB deprecated, Yandex variant ok (R7). OAuth 2.1/BCP mandate PKCE+state, single-use code (R8). Headline: verification_code-display is the primary headless rung; device is out.

### reviewer (a59e9e009bf9cdd15)
HIGH §12: verifier temp file is per-flow secret → 0600/keychain, delete, gitignore, never echo. HIGH ANTI-2/OQ-006: don't silently select plaintext; reuse explicit `--insecure-file-store` warning. MEDIUM: document `--manual` in `docs/agent-contract.md` (URL-only begin, code-in complete, no token printed). MEDIUM (D-004): device flow not proven secretless → keep out of v1. LOW: begin uses OOB/non-served redirect, no listener. LOW: complete emits token-free result (status/account/scopes/expiry). Resolves roadmap P0; adjacent OQ-006 (closed)/OQ-011 (open), neither blocks.

---

_Plan by `/pre-feature` orchestrator. Human gate mandatory before `/implementor` (ANTI-4, Type 2). Do not auto-spawn implementor._
