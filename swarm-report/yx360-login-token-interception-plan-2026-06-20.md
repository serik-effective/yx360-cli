# Consilium Plan — yx360 login token interception

**Slug:** yx360-login-token-interception
**Date:** 2026-06-20
**Status:** consilium-complete
**Feature (verbatim):** "yx360 login token interception"
**Project:** yx360-cli (Type 2, Go, greenfield)
**Agents:** architect (opus), skeptic (opus), researcher, reviewer

---

## TL;DR

- **Severity counts:** HIGH 12 · MEDIUM 9 · LOW 3 (after dedup: HIGH 7 · MEDIUM 7 · LOW 3).
- **The headline:** the consilium challenges the feature's core premise. Independent research found that **token interception is probably unnecessary** — Yandex ships documented OAuth (authorization-code + PKCE, device flow, loopback redirect, manual-paste) and the stated target services (calendar, mail, contacts, Telemost, disk) are all reachable through **documented** APIs / standard protocols (CalDAV, CardDAV, IMAP, Directory API, Telemost API, Disk API). No public reverse-engineering of Yandex 360 *private* endpoints was found.
- **Top 3 must-fix before any code:**
  1. **Validate the premise first.** Run a recon spike (surface-scout + mitmproxy/devtools) to confirm (a) what the credential actually is — almost certainly an httpOnly `Session_id`/`sessionid2` cookie, NOT an OAuth code, which means the proposed loopback "webhook intercepts the token" **cannot work as stated** — and (b) whether documented OAuth + documented APIs already cover the use cases. Decide interception vs official-OAuth on evidence.
  2. **OQ-INV-1 (authz/ToS) is a hard blocker** for anything user-facing (brew tap, agent skill). Owner must state the authorization stance (own-account-only?) in writing → D-NNN.
  3. **Token is a live full-account secret (§12).** Must be OS-keychain only, never repo/logs/swarm-report. Plus narrow this feature to *just* `login → usable credential`; defer endpoint RE, mobile escalation, skill packaging, brew tap to separate `/pre-feature` runs.

---

## Blockers (HIGH, requires_human)

- **B1 — Premise may be wrong (interception likely unnecessary).** Yandex OAuth supports authorization-code+PKCE (`oauth.yandex.com/authorize` + `/token`), device flow, and a manual-paste console flow; major Yandex 360 services have documented APIs. *Fix:* before committing, a recon spike decides interception vs documented OAuth. Sources: yandex.com/dev/id/doc, RFC 8252, github.com/bizyumov/yandex-office, github.com/essentialkaos/telemost. Confidence: high (corroborated).
- **B2 — Proposed capture mechanism may be architecturally impossible.** A system-browser + loopback callback cannot read an httpOnly cookie the IdP sets for its own domain. If the credential is that cookie, only an in-process CDP browser (chromedp/rod) or mitmproxy-style TLS interception can capture it. *Fix:* confirm credential type empirically before designing `internal/auth`. Confidence: high.
- **B3 — OQ-INV-1 authz/ToS unresolved.** Intercepting a Passport session + driving undocumented endpoints almost certainly violates Yandex 360 ToS; gates all user-facing work. *Fix:* owner states authz context → D-NNN; v1 own-account-only with consent banner. Confidence: high.
- **B4 — §12 secret-at-rest.** Captured token grants full account access (mail/disk/contacts), not a scoped key. *Fix:* OS keychain (go-keyring) only; never plaintext/repo/logs; add `yx360 logout` to clear. Confidence: high.
- **B5 — Token lifetime / no refresh.** Passport session is cookie-based, short server-side lifetime, silent invalidation (IP/device change), no documented refresh token → dies mid-session. *Fix:* detect 401/redirect-to-login, fail closed, re-trigger interactive login. Confidence: high.
- **B6 — Anti-bot on private endpoints.** Plain Go `net/http` will be fingerprinted (JA3/TLS, CSRF `sk`/`csrf` params, UA/Referer). *Fix:* if private endpoints are truly needed, treat the client as an anti-bot problem (uTLS, replicate browser headers); spike one endpoint end-to-end first. Note: researcher found anti-bot/SmartCaptcha targets the *login* surface, not authenticated calls to *documented* endpoints — another point for the OAuth path. Confidence: medium.
- **B7 — Scope creep.** "login token interception" is conflated with full-surface RE + web→mobile escalation + agent skill + brew tap (4 multi-week tracks). *Fix:* narrow to `yx360 login` → usable credential; defer the rest. Confidence: high.

## Concerns (MEDIUM)

- **C1 — Use the system browser, not an embedded webview.** IETF/Google policy + RFC 8252: embedded webviews are discouraged (security, breaks SSO/password managers, trips SmartCaptcha when scripted). Prefer launching the user's default browser. (architect + researcher, corroborated)
- **C2 — Webview vs headless-browser breaks the single-static-binary promise.** Embedded webview = CGo/WebKit; headless = bundled/downloaded Chromium. If an in-process browser is forced (httpOnly case), prefer chromedp driving the user's installed Chrome (CGo-free, static binary, real human passes anti-bot). Record as ADR.
- **C3 — Close OQ-001 now.** Pick **cobra** (Go standard, GoReleaser/Homebrew integration); pin `go.mod` toolchain before scaffolding.
- **C4 — Token storage abstraction vs vision non-goal.** Define `TokenStore` (Save/Load/Clear), default OS keychain, behind the `auth.Provider` seam. Resolve session-only vs persist-until-expiry now.
- **C5 — OS-native webview is not uniform** (WKWebView / WebView2 / WebKitGTK differ). Constrain v1 to macOS until cross-OS cookie story proven, or use system-browser to sidestep.
- **C6 — Migration path / vertical slices.** (1) git init + go mod + cobra scaffold + no-op `login` (closes OQ-001); (2) recon spike (closes credential-type + premise); (3) `auth.Provider` capturing confirmed credential to memory; (4) `TokenStore`; (5) first authenticated read (calendar list). One PR per slice; auth code starts only after slice 2.
- **C7 — Type-2 human gate + smoke test.** ANTI-4: architecture decision needs hash-locked plan sign-off. ANTI-11: exit gate = real login round-trip via `verify` skill, not "tests pass".

## Notes (LOW)

- **N1 — Define `--json`/structured-output convention at scaffold time** so the agent-skill wrapper is a thin consumer, not a rewrite. (architect)
- **N2 — DoD too weak.** "one private endpoint works" = one-shot demo; redefine around repeatable success across a token re-auth cycle. (skeptic)
- **N3 — git init / branch+PR.** D-001 noted non-git install; repo is now `git init`'d on `main`. Future feature work lands on a branch → PR (§10/ANTI-3), not direct-to-main (owner waived PR for the install commit only).

## Research findings (confidence-flagged)

| Finding | Confidence | Source |
|---|---|---|
| Yandex OAuth authorization-code + PKCE (S256) documented | high | yandex.com/dev/id/doc/en/codes/code-url |
| Manual-paste console flow (`verification_code` redirect) | medium | yandex.com/dev/id/doc/en/codes/screen-code |
| Possible `http://localhost:8899` loopback redirect for console apps | **low — verify in live OAuth app-reg UI** | (uncorroborated snippet) |
| Device-authorization flow supported | medium | yandex.com/dev/id/doc/en/codes/screen-code |
| RFC 8252 loopback pattern = industry CLI standard (gcloud/aws/wrangler/claude) | high | datatracker.ietf.org/doc/html/rfc8252 |
| Embedded webviews discouraged for OAuth (security/SSO) | high | Google OAuth policy + oauth.com |
| Yandex SmartCaptcha guards the login surface | medium | yandex.cloud/en/services/smartcaptcha |
| Documented APIs cover calendar/mail/contacts/Telemost/disk (CalDAV/CardDAV/IMAP/Directory/Telemost/Disk) | medium | github.com/bizyumov/yandex-office |
| Telemost has an **official** public API + OAuth scopes | medium | yandex.ru/dev/telemost/doc/ru/access |
| Go prior art: bearer-token Telemost client | medium | github.com/essentialkaos/telemost |
| No public RE of Yandex 360 *private* endpoints found | low (absence of evidence) | github.com/topics/yandex360 |

## Open questions raised (append to `.assistant/open-questions.md` after human review)

- **OQ-004 — Interception vs documented OAuth.** Does the recon spike justify interception at all, or do documented OAuth + documented APIs cover the real use cases? (supersedes much of the feature framing)
- **OQ-005 — Credential type.** Is the Yandex 360 credential an httpOnly session cookie, an OAuth bearer, or an OAuth code? Determines whether loopback capture is even possible.

## Per-agent verbatim (audit trail)

> Raw YAML from architect / skeptic / researcher / reviewer retained in the orchestrator transcript for this run. Key dedup'd findings folded above. Re-run agents via SendMessage if a verbatim re-read is needed:
> architect=a8649ef9778142753 · skeptic=ae6d16bb8b10d2207 · researcher=afeabded57857e233 · reviewer=a5499ca7d1361dcb7
