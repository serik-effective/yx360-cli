# Agent Command Contract

This document defines the stable, agent-facing usage contract for `yx360`. Agents that install or wrap `yx360` as a skill/tool should use this file as the contract. Do not use this repository's root `AGENTS.md` as the install contract; that file is the working agreement for developing `yx360-cli` itself.

## Prerequisites

- `yx360` must be built or installed and available on `PATH`, or called by an explicit local path such as `./bin/yx360`.
- `YX360_CLIENT_ID` must be set for `yx360 login`.
- `YX360_CALENDAR_CLIENT_ID` must be set for `yx360 login --calendar` or `yx360 login --telemost`.
- The user must complete OAuth consent in the browser or device flow.
- Mail read commands require a stored credential containing `mail:imap_full`; obtain it with `yx360 login --mail`.
- `mail send` and `mail unsubscribe --method mailto --apply/--yes` require a stored credential containing `mail:smtp`; obtain it with `yx360 login --mail --mail-send`.
- Calendar commands require a stored `calendar-telemost` profile credential containing `calendar:all`; obtain it with `yx360 login --calendar`.
- Telemost commands require the same `calendar-telemost` profile containing `telemost-api:conferences.create`; obtain it with `yx360 login --telemost`.

Credentials are stored in the OS keychain by default. Agents must not read the keychain blob directly; use CLI commands.

Mail and Calendar/Telemost use different Yandex OAuth apps. Do not request Mail scopes together with Calendar/Telemost scopes; the CLI rejects that combination before OAuth because Yandex rejects the mixed scope set.

## JSON Mode

Use the global `--json` flag for machine-readable output:

```bash
yx360 --json login --mail
yx360 --json mail list --limit 20
yx360 --json mail read <uid>
yx360 --json mail send --to user@example.com --subject "Subject" --body "Body" --yes
yx360 --json mail unsubscribe <uid>
yx360 --json calendar list --from 2026-06-20 --to 2026-06-21
yx360 --json calendar create --title "Meeting" --starts-at 2026-06-22T09:00:00+06:00 --ends-at 2026-06-22T09:30:00+06:00 --yes
yx360 --json calendar create --title "Call" --starts-at 2026-06-22T10:00:00+06:00 --ends-at 2026-06-22T10:30:00+06:00 --telemost --yes
yx360 --json calendar update <event-href> --title "New title" --yes
yx360 --json calendar delete <event-href> --yes
yx360 --json telemost create --yes
```

Current JSON output is pretty-printed JSON on stdout. The shape is command-specific and contains non-secret result fields only, for example:

- `login`: `status`, `account`, `scopes`, optional `expiry`.
- `logout`: `status`.
- `mail list` and `mail search`: an array of message summaries.
- `mail read`: one message with metadata, body, attachment manifest, and optional `unsubscribe_options`.
- `mail attachment`: `path`.
- `mail send`: `status`, `from`, `recipients`, `subject`.
- `mail unsubscribe`: preview has `uid`, `folder`, `options`, and optional `selected`; apply result has `status`, `uid`, `folder`, `method`, `uri`, optional `http_status`, and optional `mail`.
- `calendar list`: array of events. Event fields include `id`, `href`, optional `etag`, `uid`, `title`, optional `description`, optional `location`, optional `url`, `starts_at`, `ends_at`, and optional `attendees`.
- `calendar read/create/update/delete`: one event object with the same fields as above.
- `telemost create`: `id`, `join_url`.

Agents may rely on field names listed above for the current major CLI surface. They must tolerate additional fields.

## Token Redaction

JSON output must never include OAuth access tokens, refresh tokens, cookies, client secrets, or raw authorization headers. Agents must also redact these values from logs, issue reports, transcripts, and generated files.

Do not print:

- OAuth `access_token` or `refresh_token`.
- `Authorization` headers.
- Browser cookies or session captures.
- `YX360_CLIENT_ID` only if the owner treats it as private in the current environment.

If a command fails with a wrapped upstream error, quote only the non-secret message needed for diagnosis.

## Errors

Commands return non-zero on failure and write the error through the normal Cobra error path. Agents should branch on exit code first, then inspect the error text.

Known actionable errors:

- `mail: stored credential is missing, expired, or does not include mail:imap_full; run yx360 login --mail`
- `mail: stored credential is missing, expired, or does not include mail:smtp; run yx360 login --mail --mail-send`
- `mail: IMAP OAuth authentication failed; enable mail-client access and app passwords/OAuth tokens in Yandex 360 Mail settings, then run yx360 login --mail`
- `calendar: stored credential is missing, expired, or does not include calendar:all; run yx360 login --calendar`
- `telemost: stored credential is missing, expired, or does not include telemost-api:conferences.create; run yx360 login --telemost`
- `mail and calendar/telemost scopes use different Yandex OAuth apps; run separate login commands`
- `no Calendar/Telemost OAuth client_id: set YX360_CALENDAR_CLIENT_ID`
- `OS keychain unavailable (...): on headless/CI hosts re-run with --insecure-file-store`

Expired tokens are handled by re-running `yx360 login`; refresh is intentionally not implemented.

## Human Gate For Mail Send And Unsubscribe

`yx360 mail send` is externally visible and is human-gated by default. Without `--yes`, the command prints a preview and requires interactive confirmation.

Agents may pass `--yes` only after the user has explicitly approved the exact send action, including recipients, subject, body source, and attachments. A general task approval is not enough.

Bcc recipients are sent through SMTP but are not included in MIME headers.

`yx360 mail unsubscribe <uid>` previews RFC 2369 / RFC 8058 header-derived options only. It does not execute unless `--apply` is present with interactive confirmation, or `--method <kind> --yes` is passed after explicit user approval of method and URI.

## Human Gate For Calendar And Telemost

Calendar mutations and Telemost link creation are externally visible. Without `--yes`, these commands print a preview and require interactive confirmation:

- `yx360 calendar create`
- `yx360 calendar update`
- `yx360 calendar delete`
- `yx360 telemost create`

Agents may pass `--yes` only after the user has explicitly approved the exact event title, time range, attendees, deletion target, or Telemost link creation.

`calendar create --telemost` creates a Telemost conference first, then writes the returned `join_url` into the event. If Calendar creation fails after Telemost creation, the Telemost link may remain live; agents must surface that partial-failure risk.

## Calendar IDs

Use the `href` returned by `calendar list` or `calendar create` as `<event-href>` for `calendar read`, `calendar update`, and `calendar delete`. Do not invent short IDs.

Calendar uses CalDAV with `Authorization: OAuth <token>`, not `Bearer <token>`. Agents must not assume generic bearer auth when diagnosing Calendar requests.

## Plaintext Token Warning

`--insecure-file-store` stores credentials as plaintext JSON under the user config directory with file mode `0600`. This is only for headless or CI hosts where the OS keychain is unavailable.

Agents must not select `--insecure-file-store` silently. Warn the user before using it, and do not combine it with send-capable scopes unless the user explicitly accepts the plaintext credential risk for that environment.
