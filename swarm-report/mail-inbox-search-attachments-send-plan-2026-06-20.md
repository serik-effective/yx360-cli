# Consilium Plan - yx360 Mail

**Slug:** mail-inbox-search-attachments-send
**Date:** 2026-06-20
**Status:** consilium-complete
**Feature:** `Почта: читать входящие, искать письма, читать вложения, отправлять письма.`
**Reset note:** Previous bad-environment report was overwritten at the owner's request on 2026-06-20.
**Basis:** D-002 documented OAuth, D-004 OAuth host `.ru` + no secretless refresh, D-005 read-side Mail scope `mail:imap_full`, PROJECT_TYPE 2.

Type-2 plan. Owner sign-off required before `/implementor mail-inbox-search-attachments-send`.

## TL;DR

- **Severity:** HIGH 8, MEDIUM 10, LOW 3.
- **Read-only slice is buildable:** transport is IMAP/SMTP, not a private message REST API. Yandex 360 Mail docs confirm IMAP/SMTP, OAuth recommendation, app-password/OAuth-token mailbox toggle, and `.ru`/`.com` host split. Owner found read-side OAuth scope `mail:imap_full` in Yandex OAuth app UI; SMTP/send scope is still unresolved.
- **Architecture:** add `internal/mail` with IMAP/SMTP clients; CLI owns commands/output only. Keep deps one-way: `cli -> mail -> {auth, tokenstore, config}`.
- **PR slicing:** read-only mail first; send later. Do not ship send in the same PR as inbox/search/attachments.
- **Send is externally visible:** require preview + interactive confirmation; `--yes` must be explicit and non-default.

## Blockers

- **B1 - SMTP/send scope is still unresolved.** Read-side Mail scope is `mail:imap_full` (D-005), enough to start inbox/search/read/attachments. Before `mail send`, verify the SMTP/send scope and bearer behavior live, or explicitly revise D-002 to allow app-password auth for Mail only. Source: owner UI check + https://yandex.com/dev/id/doc/en/register-api.md, 2026-06-20.
- **B2 - Existing login requests only `login:info`.** [config.go](/Users/sbeysenov/dev/yx360-cli/internal/config/config.go:29) cannot authorize read-side mail commands. Add `mail:imap_full` to the read-only login/re-consent path; because D-004 says refresh without `client_secret` fails, existing users must run full `yx360 login` again.
- **B3 - Mailbox IMAP/OAuth toggle is a user prerequisite.** Yandex 360 Mail requires enabling mail-client access and `App passwords and OAuth tokens` / `Пароли приложений и OAuth-токены`. Mail commands must detect auth failure and print a precise setup hint. Sources: https://yandex.com/support/yandex-360/customers/mail/en/mail-clients/others.html.md, https://yandex.ru/support/yandex-360/customers/mail/ru/mail-clients/others.html.md, 2026-06-20.
- **B4 - Split send from read-only behavior.** Read/search/attachments are account-local reads; send is an external side effect. Ship read-only first, then send in a separate PR with its own verification.
- **B5 - `mail send` needs a human gate.** Email leaves the account and may notify third parties. Default behavior must show from/to/cc/bcc/subject/attachment list and require confirmation; `--yes` may bypass only when explicitly passed.
- **B6 - Type-2 hash-lock/sign-off.** PROJECT_TYPE 2 and ANTI-4 require owner approval before implementation.
- **B7 - Live verify required.** ANTI-11 requires live account verification: list inbox, search a known message, download a known attachment, send a test message to self, read it back.

## Concerns

- **C1 - Host defaults must follow account geography.** For RU account defaults use `imap.yandex.ru:993` and `smtp.yandex.ru:465`; Yandex's English docs show `.com`, Russian docs show `.ru`, and D-004 already found RU OAuth behavior works on `.ru`. Keep `.com` configurable for non-RU accounts. Sources: Yandex 360 Mail EN/RU docs, 2026-06-20.
- **C2 - No REST Mail API found.** Current public docs found for Yandex 360 Mail describe IMAP/SMTP, not a first-party REST Mail API for message read/send. Treat REST as out-of-scope until a public source appears.
- **C3 - `internal/mail` should own protocols.** Add `internal/mail` for connection/session/message parsing/sending. `internal/cli` should map flags to calls and DTOs only; no IMAP/SMTP code in command files.
- **C4 - Token leak risk in JSON output.** [output.go](/Users/sbeysenov/dev/yx360-cli/internal/cli/output.go:9) serializes arbitrary payloads. Define narrow DTOs: message metadata, body excerpt/full body on explicit read, attachment manifest; never emit OAuth token or raw attachment bytes.
- **C5 - Login output reports requested scopes, not granted scopes.** [login.go](/Users/sbeysenov/dev/yx360-cli/internal/cli/login.go:45) uses `cfg.Scopes`; after optional or partial grants this can lie. Report `cred.Scope` parsed from [credential.go](/Users/sbeysenov/dev/yx360-cli/internal/auth/credential.go:19).
- **C6 - Attachment filenames are untrusted input.** Sanitize filename, reject path traversal, default output directory to cwd or explicit `--out`, write mode 0600, enforce a size limit, never auto-open.
- **C7 - Search must be bounded.** IMAP search over all mailboxes can be slow. v1 flags: `--folder`, `--limit`, `--since`, `--from`, `--subject`, `--text`; default folder `INBOX`, default limit 20.
- **C8 - MIME/header handling must use libraries.** Do not string-concat email headers. Use `go-message` for MIME parse/build; use IMAP/SMTP/SASL libraries for protocol/auth.

## Notes

- **N1 - Candidate packages:** `github.com/emersion/go-imap/v2` latest visible tag `v2.0.0-beta.8`; `github.com/emersion/go-sasl` latest visible pseudo-version `v0.0.0-20241020182733-b788ff22d5a6`; `github.com/emersion/go-message` latest visible tag `v0.18.2`. Source: pkg.go.dev version pages, 2026-06-20. Pin during implementation after `go mod tidy` succeeds.
- **N2 - Config owns endpoints.** Put IMAP/SMTP host/port in `internal/config`, with env/flag override if needed.
- **N3 - Account username:** Yandex docs say username is the mailbox part before `@` for `username@yandex.*`; for custom domains, test live before assuming full address vs local part.

## Proposed Commands

- `yx360 mail list --folder INBOX --limit 20 --json`
- `yx360 mail search --from alice@example.com --subject invoice --since 2026-01-01 --limit 20`
- `yx360 mail read <message-id> [--body full|text|html|none]`
- `yx360 mail attachment <message-id> <attachment-id> --out ./downloads`
- `yx360 mail send --to user@example.com --subject "..." --body-file body.txt [--attach file] [--yes]`

## Proposed PR Slicing

- **PR-mail-1:** add `mail:imap_full`; add config endpoints; add scope precheck; add `internal/mail` IMAP session; implement `mail list`.
- **PR-mail-2:** implement bounded `mail search` and `mail read`.
- **PR-mail-3:** implement safe attachment listing/download.
- **PR-mail-4:** implement `mail send` with preview, confirmation, MIME builder, selected auth mode, and live self-send verification.

## Research Findings

- **Yandex 360 Mail supports IMAP/SMTP clients and recommends OAuth when available.** Source: https://yandex.com/support/yandex-360/customers/mail/en/mail-clients/others.html.md, source date 2026-06-20, confidence medium.
- **RU docs require enabling `imap.yandex.ru` IMAP access plus `Пароли приложений и OAuth-токены`.** Source: https://yandex.ru/support/yandex-360/customers/mail/ru/mail-clients/others.html.md, source date 2026-06-20, confidence medium.
- **RU server settings are `imap.yandex.ru:993` over SSL and `smtp.yandex.ru:465` over SSL, with `587` for unencrypted-start clients.** Source: https://yandex.ru/support/yandex-360/customers/mail/ru/mail-clients/others.html.md, source date 2026-06-20, confidence medium.
- **EN server settings are `imap.yandex.com:993` and `smtp.yandex.com:465`, with `imap.ya.ru` as non-Russia alternative.** Source: https://yandex.com/support/yandex-360/customers/mail/en/mail-clients/others.html.md, source date 2026-06-20, confidence medium.
- **Yandex ID docs say app permissions are selected at registration and permission names live in the target service docs.** Source: https://yandex.com/dev/id/doc/en/register-api.md, source date 2026-06-20, confidence medium.
- **Yandex ID docs confirm tokens carry app ID, account ID, and permission set; apps include tokens in requests to services that support OAuth.** Source: https://yandex.com/dev/id/doc/en/concepts/ya-oauth-intro.md, source date 2026-06-20, confidence medium.
- **No current public Yandex page found in this pass listing exact Mail OAuth scope strings; owner found `mail:imap_full` in the OAuth app UI for read-side Mail.** Source: searched current Yandex ID docs + Yandex 360 Mail docs + owner UI check, source date 2026-06-20, confidence medium.
- **Documented baseline auth in Yandex Mail help is app password; OAuth is recommended when the client supports it, but exact OAuth mail details are not published in fetched docs.** Source: https://yandex.com/support/yandex-360/customers/mail/en/mail-clients/others.html.md, source date unknown, confidence corroborated.
- **Yandex 360 API covers organization/admin mail settings and shared/delegated mailbox administration, not documented individual message read/search/send.** Source: https://yandex.ru/dev/api360/doc/ru/index.md, source date unknown, confidence high.

## Open Questions Raised

- **OQ-007 - Exact Yandex Mail OAuth scopes.** Read-side IMAP scope is `mail:imap_full` (D-005). Still open: SMTP/send scope.
- **OQ-008 - Account host/username matrix.** For RU, non-RU, and custom-domain Yandex 360 accounts, should config use `.ru`, `.com`, `ya.ru`, and local-part or full mailbox login?
- **OQ-009 - Send safety posture.** What max recipient count, attachment size, and automation policy are acceptable for `yx360 mail send`?

## Per-Agent Verbatim Sections

### Architect

```yaml
- severity: HIGH
  category: scope
  file: internal/config/config.go
  line: 29
  problem: The current default OAuth scope is only `login:info`, so the stored credential cannot read inboxes, search mail, fetch attachments, or send messages.
  suggested_fix: Add an explicit scope-upgrade path before Mail work: define Mail read/send scope groups, detect missing scopes from `auth.Credential.Scope`, and force re-login instead of failing inside IMAP/SMTP. Exact Yandex mail scopes need live verification before implementation.
  requires_human: true
  confidence: high
- severity: HIGH
  category: module-boundary
  file: proposal
  line: n-a
  problem: Mail protocol code should not be added directly to Cobra command files because `internal/cli` currently owns only command wiring and presentation.
  suggested_fix: Keep `internal/cli/mail*.go` thin and put mailbox operations behind `internal/mail` or `internal/mailclient` services. The dependency direction should be `cli -> mail -> auth/tokenstore/config`, not `mail -> cli`.
  requires_human: false
  confidence: high
- severity: HIGH
  category: pattern-choice
  file: internal/tokenstore/keyring.go
  line: 14
  problem: The keychain uses one fixed `credential` slot, which makes future scope upgrades and any later multi-account support ambiguous.
  suggested_fix: For this feature, either explicitly stay single-account and replace the stored credential only through `yx360 login --mail`, or introduce a credential metadata/key scheme keyed by account and scope set. Do not silently reuse a login-only token for Mail commands.
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: migration
  file: internal/auth/provider.go
  line: 19
  problem: `auth.Refresher` is intentionally unimplemented, so long-running attachment downloads or sends near expiry can fail mid-operation.
  suggested_fix: Mail commands should call `Credential.Valid()` before network work and return a clear re-auth-required error. Do not implement refresh in the Mail PR unless the app registration is changed and secretless refresh is live-verified.
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: dependency
  file: go.mod
  line: 7
  problem: The repo currently has no IMAP, SMTP, MIME, or XOAUTH2 mail dependency choice, and implementing protocol parsing ad hoc would create fragile infrastructure code.
  suggested_fix: Pick a small, maintained IMAP/SMTP/MIME stack in a dedicated Mail transport package and wrap it behind project-owned interfaces. Keep raw protocol details out of command handlers and auth/token storage.
  requires_human: true
  confidence: medium
- severity: MEDIUM
  category: scope
  file: proposal
  line: n-a
  problem: Reading inboxes, searching mail, reading attachments, and sending mail is too large for one safe PR because it spans OAuth consent, IMAP queries, MIME parsing, filesystem writes, and SMTP.
  suggested_fix: Slice as PR-1 Mail auth/scope migration plus credential loading, PR-2 inbox/search/read message metadata and bodies, PR-3 attachment listing/download, PR-4 SMTP send. Each PR should have mocked protocol tests plus one live verification note.
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: module-boundary
  file: internal/cli/root.go
  line: 24
  problem: Root command registration is flat today, but Mail has multiple verbs and will become noisy if each verb is registered as a top-level command.
  suggested_fix: Add a `mail` parent command with subcommands such as `inbox`, `search`, `read`, `attachment`, and `send`. Keep future non-Mail services separate at root level.
  requires_human: false
  confidence: high
- severity: LOW
  category: pattern-choice
  file: internal/auth/credential.go
  line: 19
  problem: Credential scopes are stored as a single string, which is awkward for reliable required-scope checks in Mail commands.
  suggested_fix: Add helper methods such as `HasScopes(required []string)` or normalize scopes on load without changing the persisted JSON shape. Avoid duplicating scope parsing in each command.
  requires_human: false
  confidence: high
```

### Skeptic

```yaml
- severity: HIGH
  category: scope
  file: proposal
  line: n-a
  problem: The feature bundles read-only mailbox operations with `send`, but sending email is an externally visible side effect with a different safety and verification profile.
  suggested_fix: Split implementation into read/search/attachment PRs first and a separate `mail send` PR only after read-only behavior is live-verified. Treat send as a human-gated operation by default.
  requires_human: true
  confidence: high
- severity: HIGH
  category: invariant
  file: .memory-bank/product-overview/anti-stories.md
  line: 13
  problem: `mail send` can notify third parties, so non-interactive sending without explicit confirmation would violate the harness rule that risky actions stay behind a human gate.
  suggested_fix: Require a preview of from/to/cc/bcc/subject/body source/attachments and an interactive confirmation unless an explicit `--yes` flag is passed. Do not make `--yes` the default in any generated workflow.
  requires_human: true
  confidence: high
- severity: HIGH
  category: risk
  file: internal/config/config.go
  line: 29
  problem: The current OAuth configuration requests only `login:info`, so any mail command built on the existing login token will fail or create pressure to over-broaden consent silently.
  suggested_fix: Resolve exact Yandex Mail OAuth scopes first, then add an explicit re-consent path for mail scopes. Existing users must re-run login because the current token cannot be upgraded in place.
  requires_human: true
  confidence: high
- severity: HIGH
  category: risk
  file: .assistant/decisions.md
  line: 45
  problem: D-004 says secretless refresh is not supported for the registered app, so long-running mail workflows cannot rely on transparent token renewal.
  suggested_fix: Mail commands must fail clearly on expiry and tell the user to run `yx360 login` again. Do not add a shipped `client_secret` or user-supplied secret workaround.
  requires_human: false
  confidence: high
- severity: HIGH
  category: risk
  file: .assistant/open-questions.md
  line: 22
  problem: The ToS posture is only narrowed for personal-account self-data, while mail features can easily drift into org-wide or delegated mailbox behavior.
  suggested_fix: Scope v1 to the authenticated user's own mailbox only. Require a new decision before supporting organization mailboxes, shared mailboxes, or admin/delegated access.
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: hidden-cost
  file: .memory-bank/tech-details/stack.md
  line: 37
  problem: Exact mail scope strings are still listed as needing empirical verification, so implementation may stall at OAuth app configuration rather than Go code.
  suggested_fix: Make scope verification the first implementation checkpoint and block code beyond a tiny spike until the live consent screen proves the strings. Record the resolved scopes in decisions before shipping.
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: risk
  file: internal/cli/output.go
  line: 9
  problem: `emit` serializes arbitrary payloads, which becomes risky once messages, headers, attachment metadata, or raw bodies are added.
  suggested_fix: Define narrow mail DTOs and never pass credentials, raw attachment bytes, or unredacted debug protocol state to `emit`. Add tests equivalent to the token leak test for mail payloads.
  requires_human: false
  confidence: medium
- severity: MEDIUM
  category: risk
  file: proposal
  line: n-a
  problem: Attachment filenames and MIME parts are untrusted mailbox content and can cause path traversal, overwrite, or accidental execution risks if downloaded naively.
  suggested_fix: Sanitize filenames, reject traversal and absolute paths, write with restrictive permissions, require explicit `--out`, enforce size limits, and never auto-open downloaded files.
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: hidden-cost
  file: proposal
  line: n-a
  problem: Searching mail through IMAP can become slow or expensive if v1 defaults to all folders, unbounded text search, or full-body fetches.
  suggested_fix: Default to `INBOX`, require `--limit`, support bounded filters such as `--since`, `--from`, and `--subject`, and fetch full bodies only for explicit reads.
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: risk
  file: internal/cli/login.go
  line: 45
  problem: Login output reports requested scopes from config rather than granted scopes from the credential, which can mislead mail command eligibility checks after optional or partial grants.
  suggested_fix: Store and report the granted scope from `cred.Scope` and make mail commands check for required granted scopes before connecting to IMAP or SMTP.
  requires_human: false
  confidence: medium
- severity: MEDIUM
  category: verification
  file: .memory-bank/product-overview/anti-stories.md
  line: 49
  problem: Unit tests alone cannot prove mail works because ANTI-11 requires live smoke verification for list, search, attachment download, and send.
  suggested_fix: Define a live verification script using a test mailbox: list inbox, find a known message, read a known attachment, send to self, and read the sent message back. Do not mark the feature done without this evidence.
  requires_human: true
  confidence: high
- severity: LOW
  category: scope
  file: internal/config/config.go
  line: 24
  problem: Existing OAuth endpoints are `.ru`, but mail server host choice may differ for RU, non-RU, and custom-domain accounts.
  suggested_fix: Put IMAP and SMTP hosts in config with sane defaults and an override path. Verify RU and custom-domain username formats before hardcoding local-part versus full email.
  requires_human: false
  confidence: medium
```

### Researcher

```yaml
- finding: Official Russian Yandex 360 Mail docs list IMAP as imap.yandex.ru:993 over SSL and SMTP as smtp.yandex.ru:465 over SSL, with SMTP 587 for clients that start unencrypted.
  source: https://yandex.ru/support/yandex-360/customers/mail/ru/mail-clients/others.html.md
  source_date: unknown
  confidence: high
  relevance: Use these hosts for Russian-locale/Yandex.ru mailbox setup, especially if the deployment is Russia-facing.
  contradicts: n-a
- finding: Official English Yandex 360 Mail docs list IMAP as imap.yandex.com:993 over SSL and SMTP as smtp.yandex.com:465 over SSL, with SMTP 587 for clients that start unencrypted.
  source: https://yandex.com/support/yandex-360/customers/mail/en/mail-clients/others.html.md
  source_date: unknown
  confidence: high
  relevance: Use these hosts for English/global Yandex.com setup; this conflicts with the Russian doc only by TLD.
  contradicts: ru-hosts
- finding: Both Russian and English Yandex Mail docs say imap.ya.ru may be used for IMAP when connecting from outside Russia.
  source: https://yandex.ru/support/yandex-360/customers/mail/ru/mail-clients/others.html.md
  source_date: unknown
  confidence: corroborated
  relevance: The implementation should allow configurable IMAP host instead of hardcoding only .ru or only .com.
  contradicts: n-a
- finding: Yandex requires enabling mail-client access via IMAP and the App passwords and OAuth tokens option before IMAP or SMTP client access.
  source: https://yandex.com/support/yandex-360/customers/mail/en/mail-clients/others.html.md
  source_date: unknown
  confidence: corroborated
  relevance: Onboarding must tell the user to enable the mailbox toggle before credentials will work.
  contradicts: n-a
- finding: For password-based mail access, Yandex requires creating an app password of type Mail in Yandex ID and using that app password rather than the account password.
  source: https://yandex.com/support/yandex-360/customers/mail/en/mail-clients/others.html.md
  source_date: unknown
  confidence: corroborated
  relevance: The feature should support app-password authentication for IMAP/SMTP as the documented baseline only if D-002 is revised for Mail; current accepted direction remains OAuth.
  contradicts: D-002
- finding: Yandex Mail docs recommend OAuth authorization when the mail client supports it, but the fetched public docs do not publish exact IMAP/SMTP OAuth scope strings.
  source: https://yandex.ru/support/yandex-360/customers/mail/ru/mail-clients/others.html.md
  source_date: unknown
  confidence: medium
  relevance: The plan should not claim exact Yandex IMAP/SMTP scope names unless they are verified from the OAuth application UI or another official source.
  contradicts: n-a
- finding: Official Yandex OAuth docs describe tokens as carrying an application identifier, account identifier, and access-right set, but the public OAuth intro and registration docs do not enumerate Mail IMAP/SMTP scope strings.
  source: https://yandex.ru/dev/id/doc/ru/concepts/ya-oauth-intro.md
  source_date: unknown
  confidence: medium
  relevance: OAuth support is plausible, but scope selection is a discovery risk for implementation.
  contradicts: n-a
- finding: I did not find official Yandex documentation for XOAUTH2 wire format; go-sasl supports RFC 7628 OAUTHBEARER, not a named XOAUTH2 helper.
  source: https://pkg.go.dev/github.com/emersion/go-sasl
  source_date: 2024-10-20
  confidence: low
  relevance: Prefer app-password first only if D-002 is revised; OAuth mail auth needs a spike against a real Yandex mailbox before committing to XOAUTH2/OAUTHBEARER behavior.
  contradicts: n-a
- finding: Yandex 360 API is a REST API for organization/admin entities including mail user settings, antispam, routing rules, domain policies, and shared/delegated mailboxes, not a documented REST API for reading, searching, or sending individual messages.
  source: https://yandex.ru/dev/api360/doc/ru/index.md
  source_date: unknown
  confidence: high
  relevance: Message read/search/attachments/send should be designed around IMAP plus SMTP, not a Yandex Mail REST API.
  contradicts: n-a
- finding: Current candidate IMAP Go library is github.com/emersion/go-imap/v2 at v2.0.0-beta.8, published 2025-12-16, and pkg.go.dev marks it as latest but not stable.
  source: https://pkg.go.dev/github.com/emersion/go-imap/v2
  source_date: 2025-12-16
  confidence: high
  relevance: Use it cautiously for IMAP read/search/fetch because it is v2 beta rather than v1 stable.
  contradicts: n-a
- finding: Current candidate SASL Go library is github.com/emersion/go-sasl at pseudo-version v0.0.0-20241020182733-b788ff22d5a6, published 2024-10-20, with PLAIN and OAUTHBEARER mechanisms.
  source: https://pkg.go.dev/github.com/emersion/go-sasl
  source_date: 2024-10-20
  confidence: high
  relevance: Use it for IMAP/SMTP SASL auth, especially PLAIN/app-password auth and possible OAuth bearer experiments.
  contradicts: n-a
- finding: Current candidate MIME/message Go library is github.com/emersion/go-message at v0.18.2, published 2024-09-28, supporting Internet Message Format and MIME parsing/writing.
  source: https://pkg.go.dev/github.com/emersion/go-message
  source_date: 2024-09-28
  confidence: high
  relevance: Use it to parse fetched messages, traverse MIME parts, extract attachments, and compose outbound messages.
  contradicts: n-a
```

### Reviewer

```yaml
- severity: HIGH
  category: decision-drift
  file: proposal
  line: n-a
  problem: The feature must not use token interception, app passwords, or an embedded client_secret because D-002 requires documented OAuth and D-004 says refresh with client_secret is forbidden for the distributed CLI.
  suggested_fix: Implement Mail through documented OAuth bearer/XOAUTH2 only, and on token expiry force `yx360 login` re-auth instead of adding refresh-with-secret. If Yandex requires a confidential secret for the needed mail flow, stop and bring the decision back to a human.
  requires_human: true
  confidence: high
- severity: HIGH
  category: testing
  file: proposal
  line: n-a
  problem: This Type 2 feature touches live inbox data and sends email, so it cannot pass `/pre-feature` without human-approved requirements, hash-locked architecture, security review, live smoke, preserved report, and an E2E scenario.
  suggested_fix: Split the plan into read-only mail commands and send-mail commands, then define the Type 2 gates and smoke cases before implementation. Include a real-account verification matrix for inbox read, search, attachment download, draft/send, expired token, and revoked token.
  requires_human: true
  confidence: high
- severity: HIGH
  category: security
  file: proposal
  line: n-a
  problem: `send email` is an externally visible action and the proposal does not define safeguards against accidental or automated sends.
  suggested_fix: Require an explicit recipient/subject/body source, a default dry-run or confirmation gate for interactive use, and a non-interactive `--yes` only when all send fields are explicit. Never send from tests except to a controlled live-test mailbox.
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: decision-drift
  file: internal/config/config.go
  line: 29
  problem: The current OAuth scope set is only `login:info`, while Mail read/search/attachments/send requires additional exact mail scopes that D-002 left as an open empirical risk.
  suggested_fix: Re-verify Yandex's current Mail OAuth scope strings and consent behavior before coding, then add the minimum scopes needed for read versus send. Keep read and send scopes separable if Yandex supports that split.
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: security
  file: internal/tokenstore/file.go
  line: 33
  problem: The accepted `--insecure-file-store` fallback writes the full OAuth credential to plaintext mode 0600, and Mail scopes raise the blast radius from login identity to inbox read and email send.
  suggested_fix: Keep keychain default, but require an explicit high-risk warning when storing Mail-scoped credentials in the file store. Consider refusing send-capable scopes with plaintext storage unless a separate human-approved flag is provided.
  requires_human: true
  confidence: medium
- severity: MEDIUM
  category: anti-story
  file: .assistant/open-questions.md
  line: 20
  problem: OQ-003 leaves the web-versus-mobile reverse-engineering boundary open, and Mail should not escalate to private web/mobile API work before proving documented IMAP/SMTP/OAuth is insufficient.
  suggested_fix: Treat documented IMAP/SMTP with OAuth as the first surface for inbox/search/attachments/send. Escalate to private API research only for a named capability gap with human approval.
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: testing
  file: internal/auth/provider.go
  line: 19
  problem: `auth.Refresher` is intentionally unimplemented after D-004, so Mail commands need explicit tests for expired-token behavior rather than silently assuming long-lived access.
  suggested_fix: Add tests that an expired credential fails closed with a re-login instruction and does not attempt secret-based refresh. Include live smoke coverage for the re-auth path before marking the feature done.
  requires_human: false
  confidence: high
```
