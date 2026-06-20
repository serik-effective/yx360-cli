# Consilium Plan - Mail Unsubscribe

**Slug:** mail-unsubscribe
**Date:** 2026-06-20
**Status:** consilium-complete
**Feature:** `unsubscribe email feature (available in web version)`
**Basis:** D-002 documented OAuth/public-protocol direction, D-006 Yandex IPv4 decision, existing Mail IMAP/SMTP implementation, PROJECT_TYPE 2.
**Context drift:** `/pre-feature` references `.memory-bank/product-overview/pipeline-stages.md`, but that file is missing in this repo. This plan therefore spells out its own verify gate instead of relying on that page.

Type-2 plan. Owner sign-off required before `/implementor mail-unsubscribe`.

## TL;DR

- **Severity:** HIGH 6, MEDIUM 9, LOW 1.
- **Top must-fix items:** do not define this as Yandex-web parity; scope v1 to standards-based message headers (`List-Unsubscribe`, `List-Unsubscribe-Post`). Add header discovery to `internal/mail` before planning execution. Keep inspect/apply and read/send scopes split by transport.
- **Recommended direction:** parse unsubscribe metadata from message headers over IMAP, preview ordered options, then execute only on explicit user action. Support three paths: HTTPS POST one-click, HTTPS/HTTP website/manual link, and `mailto:`.
- **Not approved:** any plan that quietly reintroduces private Yandex web heuristics, HTML/body scraping, or ungated side effects.

## Blockers

- **B1 - Source of truth must be protocol, not web parity.** `available in web version` is not a stable contract and conflicts with D-002 if interpreted as private Yandex web automation. Lock v1 to RFC 2369 / RFC 8058 message headers; treat web-only behavior as separate research.
- **B2 - Current mail model does not expose unsubscribe metadata.** [internal/mail/service.go](/Users/sbeysenov/dev/yx360-cli/internal/mail/service.go:48) and [internal/mail/service.go](/Users/sbeysenov/dev/yx360-cli/internal/mail/service.go:214) do not fetch or model `List-Unsubscribe` / `List-Unsubscribe-Post`.
- **B3 - Side-effect gate required.** Unsubscribe can hit arbitrary third-party endpoints or send `mailto:` messages. Default behavior must preview exact method and target, then require confirmation. No auto-execution during `list`, `search`, or `read`.
- **B4 - Transport/scope split required.** Header inspection uses `mail:imap_full`; HTTPS GET/POST unsubscribe needs no extra Yandex scope; `mailto:` execution requires `mail:smtp`. Do not over-request send scope for inspect-only flows.
- **B5 - Yandex-specific web behavior is unverified.** This repo has no dated verified source for how Yandex web performs unsubscribe today. Type-2 plan cannot depend on that premise without new research evidence.
- **B6 - Live smoke is mandatory.** Verification must use at least one real subscription message with unsubscribe metadata and record inspect, preview, execute, and post-action evidence.

## Concerns

- **C1 - Do not scrape HTML as v1.** [internal/mail/service.go](/Users/sbeysenov/dev/yx360-cli/internal/mail/service.go:305) only keeps first text/plain and text/html parts; body-link heuristics would be brittle and unsafe.
- **C2 - Preserve RFC order and method kind.** `List-Unsubscribe` options are ordered by preference. Domain model should preserve left-to-right order and distinguish `https-post`, `https-get`, and `mailto`.
- **C3 - Header parsing needs custom logic.** `List-Unsubscribe` is URL-list syntax, not address syntax. Do not feed it through `AddressList`; parse angle-bracketed URLs conservatively per RFC 2369.
- **C4 - One-click POST must stay explicit.** RFC 8058 requires user consent and specific header pairing. Missing trust context or malformed headers should downgrade to preview/manual handling, not silent POST.
- **C5 - Arbitrary outbound HTTP is new surface.** [internal/auth/http_client.go](/Users/sbeysenov/dev/yx360-cli/internal/auth/http_client.go:12) is Yandex-oriented and forces IPv4. Unsubscribe HTTP needs a separate client: no cookie jar, no auth headers, normal dual-stack dialing, short timeouts, redirect blocking for POST.
- **C6 - `mailto:` should not weaken `mail send`.** [internal/mail/send.go](/Users/sbeysenov/dev/yx360-cli/internal/mail/send.go:54) assumes user-authored mail with subject/body requirements. Generated unsubscribe mail needs a dedicated path.
- **C7 - Command shape should match inspect/apply flow.** [internal/cli/mail.go](/Users/sbeysenov/dev/yx360-cli/internal/cli/mail.go:19) currently fits simple verbs; unsubscribe needs preview-first semantics, optional method selection, and explicit apply.
- **C8 - SMTP state in repo history is noisy.** Existing reports/open questions around send verification should be reconciled before any `mailto:` unsubscribe implementation leans on current SMTP assumptions.
- **C9 - Missing `pipeline-stages.md` leaves gate source incomplete.** Keep verify steps in this hash-locked plan and in the final feature report.

## Notes

- **N1 - Standards fit current stack.** `go-imap/v2` can fetch header-only data, so unsubscribe discovery does not need full-body downloads.
- **N2 - Public Yandex docs found in this pass do not document unsubscribe mechanics.** Treat Yandex as mailbox source, not unsubscribe protocol owner.
- **N3 - Modern practice matches the standards.** Gmail guidance still distinguishes in-client unsubscribe from website fallback, so multi-path UX is expected, not edge-case behavior.

## Proposed Commands

- `yx360 mail unsubscribe <uid>` - inspect unsubscribe options for one message and print preview
- `yx360 mail unsubscribe <uid> --apply` - execute preferred safe option after confirmation
- `yx360 mail unsubscribe <uid> --method https-post|https-get|mailto --yes` - explicit override for automation or debugging

## Proposed PR Slicing

- **PR-unsub-1:** header fetch + parser + `UnsubscribeOption` domain model + JSON/human preview in `mail read` and/or dedicated `mail unsubscribe`.
- **PR-unsub-2:** HTTPS GET and RFC 8058 one-click POST execution with preview, confirmation, dedicated HTTP client, and result DTOs.
- **PR-unsub-3:** `mailto:` execution path using SMTP, consent + scope check, generated unsubscribe mail builder, and live verify.

## Verify Gate

- Prepare controlled sample messages covering:
  - `List-Unsubscribe` with `mailto:`
  - `List-Unsubscribe` with website URL only
  - `List-Unsubscribe` + `List-Unsubscribe-Post: List-Unsubscribe=One-Click`
- For each supported method:
  - `mail unsubscribe <uid>` shows parsed options in RFC order
  - preview shows exact method, destination, and whether SMTP scope is needed
  - execution requires explicit approval unless `--yes`
  - CLI records success/failure without leaking tokens or cookies
  - post-action evidence confirms sender/site accepted the unsubscribe
- Negative tests:
  - malformed header
  - unsupported scheme
  - missing send scope for `mailto:`
  - POST redirect attempt
  - no unsubscribe headers present

## Research Findings

- **RFC 2369 defines `List-Unsubscribe` as one header containing ordered angle-bracketed URLs; clients should use one supported URL and fall back only if the preferred one fails.** Source: [RFC 2369](https://datatracker.ietf.org/doc/html/rfc2369), confidence medium.
- **RFC 2369 expects `mailto:` to remain a valid unsubscribe path and warns clients not to support dangerous schemes like `file://`.** Source: [RFC 2369](https://datatracker.ietf.org/doc/html/rfc2369), confidence medium.
- **RFC 8058 one-click requires an HTTPS URI in `List-Unsubscribe` plus `List-Unsubscribe-Post: List-Unsubscribe=One-Click`; execution is HTTPS POST and requires user consent.** Source: [RFC 8058](https://datatracker.ietf.org/doc/html/rfc8058), confidence medium.
- **Modern client practice still splits between one-click unsubscribe and website fallback.** Sources: [RFC 8058](https://datatracker.ietf.org/doc/html/rfc8058), [Gmail user help](https://support.google.com/mail/answer/15433283?hl=en), confidence corroborated.
- **High-volume sender guidance in Gmail explicitly requires RFC 8058 one-click for subscribed mail above 5,000 messages/day.** Source: [Gmail sender requirements](https://support.google.com/mail/answer/81126?hl=en), confidence medium.
- **No public Yandex-specific unsubscribe transport docs were found in this pass.** Sources: [Yandex Mail EN help](https://yandex.com/support/yandex-360/customers/mail/en/), [Yandex Mail RU help](https://yandex.ru/support/yandex-360/customers/mail/ru/), confidence low.
- **Current Go stack can fetch header-only data with `go-imap/v2`, and `net/mail` can retrieve raw header values before custom parsing.** Sources: [go-imap/v2](https://pkg.go.dev/github.com/emersion/go-imap/v2), [net/mail](https://pkg.go.dev/net/mail), confidence corroborated.

## Out-of-Scope (declared)

- Private Yandex web endpoint automation
- Mobile-app reverse engineering for unsubscribe
- HTML/body-link scraping heuristics as default path
- Background or automatic unsubscribe without explicit user action
- Org/shared/delegated mailbox unsubscribe behavior

## Open Questions Raised

- **OQ-010 - Trust threshold for one-click POST.** Is RFC header pairing alone enough in this CLI, or must v1 also inspect DKIM/authentication evidence before allowing `https-post` execution?
- **OQ-011 - UX placement.** Should unsubscribe preview live only under `mail unsubscribe`, or should `mail read` also surface parsed unsubscribe options passively?
- **OQ-012 - Automation policy.** Is `--yes` acceptable for HTTPS POST and `mailto:` unsubscribe in scripts, or should one or both stay interactive-only in Type 2?

## Per-Agent Verbatim Sections

### Architect

```yaml
- severity: HIGH
  category: scope
  file: proposal
  line: n-a
  problem: "Defining this as parity with whatever Yandex web shows is not a stable protocol contract and would pull the CLI back toward private web heuristics even though D-002 moved the project to documented OAuth and protocol integrations."
  suggested_fix: "Scope v1 to standards-based unsubscribe actions exposed in message headers: RFC 2369 `List-Unsubscribe` plus RFC 8058 `List-Unsubscribe-Post`. If a web-only case appears without these headers, capture a sample message and treat it as a separate research track; record this source-of-truth choice in a draft ADR under `.memory-bank/tech-details/architecture-decisions/`."
  requires_human: true
  confidence: high

- severity: HIGH
  category: module-boundary
  file: /Users/sbeysenov/dev/yx360-cli/internal/mail/service.go
  line: 48
  problem: "The current `mail.Message` model has no place for ordered unsubscribe options or execution metadata, so the CLI would be forced to parse and decide on raw headers itself."
  suggested_fix: "Add a mail-domain type such as `UnsubscribeOption` and expose it from `internal/mail`; preserve RFC 2369 left-to-right preference order and the method kind (`https-post`, `https-get`, `mailto`) in the domain layer, not in `internal/cli`."
  requires_human: false
  confidence: high

- severity: HIGH
  category: pattern-choice
  file: /Users/sbeysenov/dev/yx360-cli/internal/mail/service.go
  line: 214
  problem: "`fetchMessages` currently asks IMAP for envelope, size, body structure, and optionally body bytes, but not the unsubscribe headers needed for this feature."
  suggested_fix: "Fetch only the needed headers via IMAP `BODY.PEEK[HEADER.FIELDS (...)]`, at minimum `List-Unsubscribe`, `List-Unsubscribe-Post`, and optionally `Authentication-Results`, before any body parsing. This keeps discovery cheap and avoids brittle HTML or body scraping."
  requires_human: false
  confidence: high

- severity: HIGH
  category: migration
  file: /Users/sbeysenov/dev/yx360-cli/internal/mail/send.go
  line: 43
  problem: "The only current side-effect transport is SMTP, but many unsubscribe actions are HTTPS and do not require `mail:smtp`; treating unsubscribe as just another send flow would over-request scope and blur the migration path."
  suggested_fix: "Split execution by transport: discovery always works with `mail:imap_full`, HTTPS GET and one-click POST need no extra Yandex scope, and `mailto:` execution alone should require `mail:smtp` plus re-login when missing. Keep `--mail` and `--mail-send` consent separate in the UX."
  requires_human: true
  confidence: high

- severity: MEDIUM
  category: pattern-choice
  file: /Users/sbeysenov/dev/yx360-cli/internal/mail/send.go
  line: 54
  problem: "`Send` validates messages as if they were human-authored mail, requiring a subject and body or attachment, but RFC 2369 `mailto:` unsubscribe commands can be prefilled or minimal."
  suggested_fix: "Add a dedicated generated-mail path for unsubscribe commands instead of weakening `mail send` semantics. The parser should honor `subject` and `body` from the `mailto:` URI and default the sender to the logged-in mailbox."
  requires_human: false
  confidence: high

- severity: MEDIUM
  category: dependency
  file: /Users/sbeysenov/dev/yx360-cli/internal/auth/http_client.go
  line: 12
  problem: "The only in-repo HTTP client builder is Yandex-specific and forces IPv4 per D-006, but unsubscribe targets are arbitrary third-party domains and RFC 8058 forbids sending cookies or other prior-web context."
  suggested_fix: "Create a dedicated unsubscribe HTTP client under `internal/mail` with no cookie jar, no auth headers, normal dual-stack dialing, short timeouts, and redirect blocking for POST. Do not reuse the Yandex auth client outside the Yandex surface."
  requires_human: false
  confidence: high

- severity: MEDIUM
  category: scope
  file: proposal
  line: n-a
  problem: "RFC 8058 says receivers should only offer one-click unsubscribe when the required header pairing is present and the message has the required trust signal, but the current CLI has no receiver-trust model."
  suggested_fix: "In v1, never background or auto-execute unsubscribe; show a preview, require explicit user confirmation, and only treat HTTPS POST as one-click when both headers are present. Surface missing trust context as a warning rather than promising perfect parity with Yandex web."
  requires_human: true
  confidence: medium

- severity: MEDIUM
  category: module-boundary
  file: /Users/sbeysenov/dev/yx360-cli/internal/cli/mail.go
  line: 19
  problem: "The existing mail verbs are simple read or send operations, while unsubscribe needs an inspect-then-apply flow and optional method selection."
  suggested_fix: "Add a dedicated `mail unsubscribe` command that defaults to previewing parsed actions and requires an explicit apply flag such as `--apply` or `--yes` to execute. Let `internal/mail` return structured options and execution results; keep CLI logic to prompting and rendering."
  requires_human: false
  confidence: high

- severity: MEDIUM
  category: scope
  file: /Users/sbeysenov/dev/yx360-cli/swarm-report/mail-inbox-search-attachments-send-plan-2026-06-20.md
  line: 55
  problem: "The existing mail PR slicing stops at inbox, search, attachments, and send, so unsubscribe work will otherwise piggyback on unrelated slices and blur verification ownership."
  suggested_fix: "Slice unsubscribe separately: PR-unsub-1 header discovery plus JSON or preview, PR-unsub-2 HTTPS GET and one-click POST execution, PR-unsub-3 `mailto:` execution via SMTP plus consent and scope migration. Each PR should carry one live verification note with controlled test messages for the supported method types."
  requires_human: false
  confidence: high

- severity: LOW
  category: migration
  file: /Users/sbeysenov/dev/yx360-cli/.memory-bank/steerings/project-types.md
  line: 83
  problem: "`project-types.md` points to `product-overview/pipeline-stages.md`, but that file is missing, so the canonical stage and gate reference for this Type-2 feature is incomplete."
  suggested_fix: "Until the memory-bank link is repaired, bake the unsubscribe verify gate directly into the hash-locked feature plan and final swarm report: inspect action, preview, execute in a controlled mailbox or list, and record post-action evidence. Do not rely on the missing page during implementation handoff."
  requires_human: false
  confidence: high
```

### Skeptic

```yaml
- severity: HIGH
  category: premise-flaw
  file: proposal
  line: n-a
  problem: "`available in web version` is a weak premise because this repo is a Go CLI on documented OAuth plus IMAP/SMTP, and I did not find repo evidence or an official source in this pass that the web client exposes a reusable unsubscribe capability that maps onto that surface."
  suggested_fix: "Restate the request as a concrete CLI capability with a named mechanism instead of UI parity. If it depends on undocumented Yandex web behavior, split it into a separate research-and-approval track."
  requires_human: true
  confidence: medium

- severity: HIGH
  category: hidden-cost
  file: internal/mail/service.go
  line: 48
  problem: "The current mail model stores subject, addresses, body, and attachments but no raw headers, so it drops `List-Unsubscribe` and `List-Unsubscribe-Post`, which are the standard protocol signals for safe unsubscribe."
  suggested_fix: "Prove header extraction first and keep v1 strictly header-driven. Do not promise a one-command unsubscribe until the repo can surface those headers explicitly."
  requires_human: false
  confidence: high

- severity: HIGH
  category: better-alternative-exists
  file: internal/mail/service.go
  line: 305
  problem: "Body parsing only keeps the first `text/plain` and `text/html` payloads, so an `unsubscribe` feature built from message bodies would devolve into scraping arbitrary marketer HTML rather than using protocol metadata."
  suggested_fix: "Narrow the first slice to exposing unsubscribe metadata instead of actioning body links. Reject HTML-link automation unless a separate human-approved heuristic path is explicitly scoped."
  requires_human: true
  confidence: high

- severity: HIGH
  category: hidden-cost
  file: internal/cli/mail.go
  line: 161
  problem: "The current human gate exists only for SMTP send, so a one-click unsubscribe over HTTP would introduce a new externally visible side-effect path without an equivalent preview-and-confirm contract."
  suggested_fix: "Any actioning unsubscribe command must show the exact mechanism and destination (`mailto`, GET, or POST) before execution and require explicit confirmation. Do not classify one-click HTTP as a read-only mail feature."
  requires_human: true
  confidence: high

- severity: HIGH
  category: scope-creep
  file: swarm-report/mail-inbox-search-attachments-send-plan-2026-06-20.md
  line: 376
  problem: "The accepted mail direction says not to escalate into private web/mobile behavior before documented IMAP/SMTP is proven insufficient, but `available in web version` implicitly pulls the feature toward that unresolved surface."
  suggested_fix: "Lock v1 to the documented mail surface for the authenticated user's own mailbox only. If unsubscribe needs private web flows or Yandex-specific heuristics, stop and run a separate `/research` pass."
  requires_human: true
  confidence: high

- severity: MEDIUM
  category: hidden-cost
  file: .assistant/decisions.md
  line: 67
  problem: "D-006 only verifies network behavior for Yandex OAuth/account-info/IMAP, while one-click unsubscribe can hit arbitrary third-party domains and therefore adds a brand-new outbound HTTP surface with unknown transport and tracking behavior."
  suggested_fix: "Treat one-click unsubscribe as a separate network-surface feature with its own live verification and failure policy. Do not assume the current Yandex-only transport constraints cover third-party unsubscribe endpoints."
  requires_human: true
  confidence: medium

- severity: MEDIUM
  category: better-alternative-exists
  file: internal/cli/login.go
  line: 67
  problem: "The CLI already separates read (`--mail`) from send (`--mail-send`) consent, so an undifferentiated `unsubscribe` feature would blur inspect-only behavior with actioning behavior and likely over-request permission."
  suggested_fix: "Keep scope boundaries strict: inspecting unsubscribe metadata must stay on read scope, and any `mailto` execution must be a separate action that requires send scope. Do not request send capability just to show whether a message is unsubscribable."
  requires_human: false
  confidence: high
```

### Researcher

```yaml
- finding: "RFC 2369 defines List-Unsubscribe as a single mailing-list header whose value is one or more angle-bracketed URLs, ordered left-to-right by preference; the client should use one supported URL and fall back only if the first attempt fails."
  source: "https://datatracker.ietf.org/doc/html/rfc2369"
  source_date: unknown
  confidence: medium
  relevance: "Your CLI should preserve URL order and present one chosen unsubscribe action instead of firing every candidate."
  contradicts: n-a

- finding: "RFC 2369 says list generators should usually include a mailto-based command in addition to any other protocol, so mailto is still a standards-compliant unsubscribe path rather than a legacy-only fallback."
  source: "https://datatracker.ietf.org/doc/html/rfc2369"
  source_date: unknown
  confidence: medium
  relevance: "The feature should support mailto targets, not just HTTP(S), even if most modern senders prefer web endpoints."
  contradicts: n-a

- finding: "RFC 2369 parsing rules require angle-bracketed URLs, say to ignore stray whitespace inserted inside brackets by broken MTAs, and say that if a comma-separated subitem is malformed the remainder of the field should be ignored."
  source: "https://datatracker.ietf.org/doc/html/rfc2369"
  source_date: unknown
  confidence: medium
  relevance: "A custom parser must be conservative and RFC-shaped; naive split-on-comma parsing will mis-handle real mail."
  contradicts: n-a

- finding: "RFC 2369 security guidance says mail clients should not support list-header URLs that could compromise the user's system, explicitly including file:// URLs."
  source: "https://datatracker.ietf.org/doc/html/rfc2369"
  source_date: unknown
  confidence: medium
  relevance: "The CLI should allowlist safe schemes such as https/http/mailto and reject local-execution schemes."
  contradicts: n-a

- finding: "RFC 8058 one-click unsubscribe is signaled by a List-Unsubscribe header that contains an HTTPS URI plus a List-Unsubscribe-Post header whose value is exactly List-Unsubscribe=One-Click; MAILTO may still appear as an additional URI."
  source: "https://datatracker.ietf.org/doc/html/rfc8058"
  source_date: unknown
  confidence: medium
  relevance: "Only messages with both headers qualify for standards-based automatic one-click; everything else is manual-style unsubscribe."
  contradicts: n-a

- finding: "RFC 8058 says a receiver performs one-click with an HTTPS POST to the HTTPS URI, the POST body is the fixed key/value pair from List-Unsubscribe-Post, and the sender should handle manual GET and one-click POST on the same target without redirecting the POST."
  source: "https://datatracker.ietf.org/doc/html/rfc8058"
  source_date: unknown
  confidence: medium
  relevance: "For automation, POST is the one-click path; GET is the manual/web path and should not be conflated with one-click."
  contradicts: n-a

- finding: "RFC 8058 says the receiver must not POST the unsubscribe URI without user consent."
  source: "https://datatracker.ietf.org/doc/html/rfc8058"
  source_date: unknown
  confidence: medium
  relevance: "The CLI must never auto-unsubscribe during list/read/search; unsubscribe should be an explicit user-triggered command with confirmation."
  contradicts: n-a

- finding: "RFC 8058 requires a valid DKIM signature that covers both List-Unsubscribe and List-Unsubscribe-Post, and receivers should not offer one-click when that signature is missing."
  source: "https://datatracker.ietf.org/doc/html/rfc8058"
  source_date: unknown
  confidence: medium
  relevance: "A CLI reading raw IMAP mail should treat missing or unverifiable authentication as a reason to downgrade from auto-POST to manual review."
  contradicts: n-a

- finding: "Current Gmail sender guidance says marketing and subscribed mail above 5,000 messages per day must support one-click unsubscribe using the RFC 8058 header pair, and the example request is an HTTP POST with body List-Unsubscribe=One-Click."
  source: "https://support.google.com/mail/answer/81126?hl=en"
  source_date: unknown
  confidence: medium
  relevance: "In modern practice, standards-based one-click is not theoretical; high-volume senders are expected to implement it."
  contradicts: n-a

- finding: "Current Gmail end-user help shows modern clients may either offer in-client Unsubscribe or fall back to Go to website, and a single unsubscribe action may apply only to one mailing list from a sender."
  source: "https://support.google.com/mail/answer/15433283?hl=en"
  source_date: unknown
  confidence: medium
  relevance: "The CLI should expose which target is being used and should not imply that one action always unsubscribes the user from every list from that sender."
  contradicts: n-a

- finding: "Modern unsubscribe UX is bifurcated: RFC 8058 defines consented HTTPS POST for one-click, while Gmail user help still documents senders that require a website flow instead of in-client unsubscribe."
  source: "https://datatracker.ietf.org/doc/html/rfc8058, https://support.google.com/mail/answer/15433283?hl=en"
  source_date: unknown
  confidence: corroborated
  relevance: "The CLI should model at least three execution paths: one-click HTTPS POST, open website/manual GET, and mailto."
  contradicts: n-a

- finding: "Yandex Mail's public help index fetched in this pass exposes broad sections for working with email and security against mailing lists, but it does not surface public documentation of List-Unsubscribe, one-click POST, or a documented unsubscribe web flow."
  source: "https://yandex.com/support/yandex-360/customers/mail/en/, https://yandex.ru/support/yandex-360/customers/mail/ru/"
  source_date: unknown
  confidence: low
  relevance: "I did not find public Yandex-specific docs strong enough to justify hardcoding Yandex-web unsubscribe mechanics."
  contradicts: n-a

- finding: "Yandex Mail web unsubscribe behavior; status: no-public-Yandex-specific-doc-found-in-this-pass for whether the web client issues HTTP POST, HTTP GET, or mailto when a user unsubscribes."
  source: "https://yandex.com/support/yandex-360/customers/mail/en/, https://yandex.ru/support/yandex-360/customers/mail/ru/"
  source_date: unknown
  confidence: low
  relevance: "Implementation should treat Yandex as a generic mailbox source: parse message headers and ask the user which unsubscribe path to execute, rather than assuming Yandex-specific transport behavior."
  contradicts: n-a

- finding: "Go's net/mail package ReadMessage parses a raw message into headers and body, and Header.Get retrieves the first value for a header case-insensitively; RFC 2369 in turn says there must be no more than one List-Unsubscribe field per message."
  source: "https://pkg.go.dev/net/mail, https://datatracker.ietf.org/doc/html/rfc2369"
  source_date: unknown
  confidence: corroborated
  relevance: "For raw MIME pulled over IMAP, Header.Get(\"List-Unsubscribe\") is a reasonable first retrieval path before custom URL parsing."
  contradicts: n-a

- finding: "Go's AddressList helpers are for address headers, while RFC 2369 defines List-Unsubscribe as URL-list syntax, so List-Unsubscribe needs a custom parser over the raw header value rather than AddressList."
  source: "https://pkg.go.dev/net/mail, https://pkg.go.dev/github.com/emersion/go-message/mail, https://datatracker.ietf.org/doc/html/rfc2369"
  source_date: unknown
  confidence: high
  relevance: "Do not feed List-Unsubscribe into AddressList; parse angle-bracketed URLs, comments, and comma-separated alternatives per RFC 2369."
  contradicts: n-a

- finding: "The repo's existing IMAP stack choice is suitable for header-only unsubscribe discovery because go-imap/v2 exposes FetchItemBodySection with PartSpecifierHeader, which fetches only message headers instead of the whole BODY[]."
  source: "https://pkg.go.dev/github.com/emersion/go-imap/v2"
  source_date: 2025-12-16
  confidence: medium
  relevance: "You can implement unsubscribe inspection as a cheap header fetch during list/read without downloading full message bodies."
  contradicts: n-a

- finding: "go-message/mail CreateReader returns a usable Reader even when charset or transfer encoding is unknown, but each Part.Body must be fully read before NextPart."
  source: "https://pkg.go.dev/github.com/emersion/go-message/mail"
  source_date: 2024-09-28
  confidence: medium
  relevance: "If you later add a fallback that scans message bodies or HTML parts for visible unsubscribe links, MIME walking must tolerate charset errors and consume parts sequentially."
  contradicts: n-a

- finding: "go-message/mail HeaderFromMap is only an interoperability helper and loses field ordering, key capitalization, and original whitespace."
  source: "https://pkg.go.dev/github.com/emersion/go-message/mail"
  source_date: 2024-09-28
  confidence: medium
  relevance: "Prefer direct header parsing over map round-trips when debugging malformed List-Unsubscribe values or preserving raw evidence for a confirmation screen."
  contradicts: n-a
```

### Reviewer

```yaml
- severity: HIGH
  category: missing-context
  file: internal/mail/service.go
  line: 214
  problem: "The current mail fetch path only collects envelope/body structure/body data and exposes no raw header or unsubscribe metadata, so a plan that assumes the CLI can already derive an unsubscribe action from messages is not grounded in the existing code."
  cites: ANTI-8
  suggested_fix: "Make the first slice define exactly how unsubscribe metadata is fetched and represented from the message source of truth. If the feature depends on web-only metadata instead of message data, document that source explicitly before hash-locking the plan."
  requires_human: false
  confidence: high

- severity: HIGH
  category: contradicts-prior-decision
  file: proposal
  line: n-a
  problem: "The proposal is underspecified about whether unsubscribe is driven from message metadata or by automating/calling Yandex web behavior, and the web-endpoint route would contradict the accepted move to documented OAuth/public protocols."
  cites: D-002
  suggested_fix: "Constrain the plan to message-derived unsubscribe data, or run `/research` and append a new decision if private Yandex web behavior is truly required. Do not merge a plan that quietly reintroduces reverse-engineering through the web UI."
  requires_human: true
  confidence: high

- severity: HIGH
  category: factual-error
  file: proposal
  line: n-a
  problem: "The premise that unsubscribe is 'available in web version' is not backed by a dated verified source in this repo, so the plan would be basing a Type 2 architecture decision on an unverified current external behavior."
  cites: INVARIANT-§6
  suggested_fix: "Re-verify the current Yandex Mail web unsubscribe behavior with a dated source before merging the plan. Record whether the action comes from message metadata, a sender-owned target, or a Yandex-private web path."
  requires_human: true
  confidence: high

- severity: MEDIUM
  category: contradicts-prior-decision
  file: .assistant/open-questions.md
  line: 28
  problem: "Repo state disagrees on whether SMTP send is already live-verified, so a mailto-style unsubscribe path cannot tell whether reusing `mail send` is an accepted prerequisite or still a blocked assumption."
  cites: D-007
  suggested_fix: "Reconcile D-007 with the current open-questions/report state before hash-locking any unsubscribe plan that may send mail. If SMTP is now accepted, append a new decision revision; if not, keep mailto unsubscribe out of scope."
  requires_human: true
  confidence: high

- severity: MEDIUM
  category: anti-story-violation
  file: proposal
  line: n-a
  problem: "Any unsubscribe plan that reaches Done without a live smoke against a real subscription message would violate the repo's no-fake-completion rule for mail features with external side effects."
  cites: ANTI-11
  suggested_fix: "Define a live verify scenario around a real message that exposes unsubscribe data, execute the action once with explicit human approval, and record the result in the E2E scenario file plus the feature report. Do not treat unit tests or mocked IMAP fixtures as sufficient."
  requires_human: true
  confidence: high
```
