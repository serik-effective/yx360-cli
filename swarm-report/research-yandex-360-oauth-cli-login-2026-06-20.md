# Research — Sign in with Yandex 360 via documented OAuth (Go CLI)

**Slug:** yandex-360-oauth-cli-login
**Date:** 2026-06-20
**Question:** How to implement "Sign in with Yandex 360" login via documented Yandex OAuth in a brew-distributed Go CLI (grant choice, registration, scopes, token lifetime/refresh, Go prior art, DAV/IMAP auth).
**Decision context:** Owner chose documented OAuth over token interception (reframes the prior `/pre-feature` consilium).
**Agents:** researcher → skeptic + reviewer (two-phase).

---

## TL;DR (top 3)

1. **The OAuth path is viable and §12-clean.** Yandex documents authorization-code + PKCE/S256 as a **public client with NO client_secret** — quote: *"If the PKCE extension is used … you don't need to pass the secret key."* A brew binary ships only a `client_id`. (high)
2. **Use `golang.org/x/oauth2` directly — native PKCE.** `GenerateVerifier()` + `S256ChallengeOption()` + `VerifierOption()`, plus `DeviceAuth`/`DeviceAccessToken`. Hand-configure `Endpoint{AuthURL: oauth.yandex.com/authorize, TokenURL: oauth.yandex.com/token}`. No Yandex-specific PKCE lib needed. (high)
3. **Loopback works but on a FIXED pre-registered port (8899), not arbitrary.** Yandex exact-matches the redirect URI. On a machine where 8899 is taken, login hard-fails → **device flow must be the mandatory fallback.** (loopback medium; fixed-port-constraint high; skeptic flagged inverted confidence)

## Recommended flow ladder (for `yx360 login`)

1. **Default:** authorization-code + PKCE/S256, loopback `http://localhost:8899` — open system browser, local listener captures `?code=`, exchange with `code_verifier`, no secret.
2. **Headless / port-busy fallback:** device-authorization flow (`oauth.yandex.com/device/code` → user enters code at `ya.ru/device` → poll `/token`).
3. **Last resort:** manual-paste `verification_code` redirect behind `--paste` / `--no-browser`.

Token at rest → **OS keychain (go-keyring)** only. Never repo/logs (§12).

## Safe drift applied to memory bank

- `tech-details/stack.md` — replaced the interception/webview rows with the OAuth approach; added an **Auth** section with the high-confidence facts (endpoints, PKCE-no-secret, x/oauth2, XOAUTH2, device flow, 12-month personal-account token) + source URLs; bumped `Last updated`. Medium/low items written as explicit "verify empirically" TODOs, not asserted.

## Risky drift — NEEDS HUMAN REVIEW (not auto-applied)

| Item | Why risky | Action |
|---|---|---|
| **Secretless REFRESH** | CONTRADICTION: code-exchange doc says secret optional with PKCE; refresh doc lists `client_secret` as required and never mentions PKCE. If refresh needs a secret, a secretless CLI can get a token but not silently refresh. | **Empirically test** `grant_type=refresh_token` with no secret against a real public app before committing the storage/refresh design. §12-clean fallback = re-run loopback at expiry (cheap given 12-mo TTL). |
| **localhost:8899 loopback** | Confirmed at only medium confidence; fixed-port = occupied-port failure mode on user machines. | Register 8899, but **ship device-flow fallback** for EADDRINUSE/headless. Re-source from official app-registration docs. |
| **CalDAV/CardDAV (Calendar/Contacts) via OAuth bearer for PERSONAL accounts** | Two contradictory community sources (low). 360-business docs say bearer works; personal-account help pushes app-specific passwords. | **Do NOT scope Calendar/Contacts into v1 on this evidence.** Integration-test a personal token against caldav/carddav.yandex.ru via XOAUTH2 first, or mark out-of-scope. |
| **360 scope strings** | Fragmented across docs (medium); wrong scopes = silent authz failure / over-broad consent. | Verify each against the live consent screen round-trip before hardcoding. |
| **Org / Directory scopes** | Require Yandex 360 org + admin-enabled service app + written user consent (medium). | Personal account = Mail/Disk/Telemost self-scope works; Directory/org features gated on admin. Don't promise Directory for personal accounts. |
| **12-month token TTL** | Marked high but "~" and account-type-dependent; org service-app tokens are **1 hour**, not 12 months. | Pin TTL per account type; don't let "1-year tokens make refresh-failure cheap" drive design for org use. |

## Conflict with prior decision

- **D-001** framed the project as scraping / anti-bot (private-endpoint interception). The OAuth choice **supersedes** that framing. Reviewer recommends appending **D-002** ("documented OAuth, not interception") — append-only, does not edit D-001 (§8). **Pending owner approval** (this skill does not write decisions without it).
- Resolves **OQ-004** (interception vs OAuth → OAuth). Moots **OQ-002 / OQ-005** (no interception ⇒ no credential-type question). **Softens but does NOT close OQ-INV-1** (personal-account user-consented scopes = defensible; org/Directory still admin-gated). **OQ-001** (cobra + go.mod pin) untouched — close in the scaffolding `/pre-feature`.

## Full researcher findings (confidence-flagged)

| # | Fact | Confidence | Source |
|---|---|---|---|
| 1 | auth-code + PKCE/S256 documented (`/authorize`, `/token`) | high | yandex.com/dev/id/doc/en/codes/code-url |
| 2 | client_secret omittable when PKCE `code_verifier` sent (public client) | high | …/codes/code-url |
| 3 | console redirect URIs `http://localhost:8899` + `verification_code` | medium | …/doc/en/register-client |
| 4 | redirect_uri must EXACTLY match registered (fixed port) | high | …/codes/code-url |
| 5 | device-authorization flow (`/device/code`, `ya.ru/device`) | high | yandex.com/dev/oauth/doc/dg/concepts/device-token |
| 6 | manual-paste `verification_code` flow | medium | …/register-client |
| 7 | `golang.org/x/oauth2` native PKCE (GenerateVerifier/S256ChallengeOption/VerifierOption) + DeviceAuth | high | pkg.go.dev/golang.org/x/oauth2 |
| 8 | No ready-made Yandex PKCE-loopback Go lib; `dmfed/yauth` = device+secret | medium | github.com/dmfed/yauth |
| 9 | scope strings cloud_api:disk.*, mail:imap_*, telemost-api:*, directory:* | medium | yandex.cloud/en/docs/security/standard-360/all |
| 10 | Calendar/Contacts via CalDAV/CardDAV, not REST scopes | medium | yandex.ru/support/yandex-360/business/admin/en/security-service-applications |
| 11 | 360-business needs admin-enabled service app + written consent; personal accounts don't | medium | (same as 10) |
| 12 | access token ~12-month TTL; refresh returns new refresh_token | high | yandex.com/dev/id/doc/en/tokens/refresh-client |
| 13 | **CONTRADICTION**: refresh doc requires client_secret; secretless refresh unresolved | medium/critical | …/tokens/refresh-client |
| 14 | IMAP/SMTP accept OAuth via XOAUTH2, no app password | high | tech.yandex.com/oauth/doc/imap |
| 15 | CalDAV personal-account may need app password (contradicts 10) | low | community (cloudbik blog) |
| 16 | org service-app tokens = 1-hour (distinct from 12-mo user token) | medium | (same as 10) |

## Skeptic critique (key challenges)

- Secretless-refresh contradiction may be a doc-version artifact → **must test empirically**, don't design on contradictory docs. (HIGH)
- Fixed-port loopback collides with brew distribution (8899 occupied → hard fail); device-flow fallback mandatory. (HIGH)
- Confidence inverted: high "fixed-port" conclusion built on medium "8899" source. (MEDIUM)
- Two community CalDAV findings contradict each other → "OAuth unlocks Calendar/Contacts" not safe to rely on. (MEDIUM)
- Scope strings from fragmented docs → verify via live consent round-trip. (MEDIUM)
- x/oauth2 supporting a flow ≠ Yandex honoring it → integration-test device+S256 against Yandex. (LOW)

## Reviewer cross-check (invariants / OQs)

- Append **D-002** (OAuth, not interception); supersede D-001 framing without editing it (§8). (MEDIUM)
- Close **OQ-002** (mooted), record **OQ-004** resolved / **OQ-005** mooted. (HIGH/MED)
- Keep **OQ-INV-1** open, narrowed to org/Directory admin-consent. (HIGH)
- §12: public PKCE = nothing secret to embed (clean); if refresh needs a secret, keychain/env only, never repo — re-auth is the §12-clean fallback over embedding a secret. (HIGH)
- **OQ-001** still open — close in scaffolding `/pre-feature`; note x/oauth2 as confirmed dep. (MEDIUM)
- §6/§9: all external claims dated 2026-06-20, re-verify before code relies on them. (LOW)

## Agent IDs (re-query via SendMessage)
researcher=a56e662105e0b424f · skeptic=a9ae0d9e390856401 · reviewer=a0c4529fe3ce1c4b6
