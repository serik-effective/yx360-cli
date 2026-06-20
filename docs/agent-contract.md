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
- `YX360_FORMS_CLIENT_ID` and `YX360_FORMS_ORG_ID` must be set for `yx360 login --forms` and all `forms` commands. The Forms API is available only to Yandex 360 for Business organizations. The org id is sent as `X-Org-Id` when numeric, or `X-Cloud-Org-Id` when non-numeric (Yandex Cloud org). There is no auto-discovery; an invalid org id returns an `organization required` / `user not in organization` API error.
- `forms` commands require a stored `forms` profile credential containing `forms:read` and `forms:write`; obtain it with `yx360 login --forms`.

Credentials are stored in the OS keychain by default. Agents must not read the keychain blob directly; use CLI commands.

Mail, Calendar/Telemost, and Forms use different Yandex OAuth apps. Do not request scopes from more than one of these groups in a single `login`; the CLI rejects the combination before OAuth because Yandex rejects mixed scope sets.

## Account Types And Organizations

Surfaces differ in what kind of Yandex account they need:

- **Personal-account self-data:** Mail (IMAP/SMTP), Calendar (CalDAV), Telemost. Documented OAuth with the user's own consent; no organization required.
- **Yandex 360 for Business (org-bound):** Forms. The Forms API is available only to a Yandex 360 for Business (or Yandex Cloud) organization and requires an org id sent as `X-Org-Id` (numeric) or `X-Cloud-Org-Id` (non-numeric/Cloud). A personal account without such an org cannot use `forms`. There is no auto-discovery of the org id; it is operator-configured via `YX360_FORMS_ORG_ID`.

## Environment Variables

Use environment variables for headless/CI/agent deployment; command-line flags override them where both exist.

| Variable | Required for | Purpose |
|---|---|---|
| `YX360_CLIENT_ID` | `login`, Mail | OAuth client id of the Mail/default app |
| `YX360_CALENDAR_CLIENT_ID` | `login --calendar`/`--telemost` | OAuth client id of the Calendar+Telemost app |
| `YX360_FORMS_CLIENT_ID` | `login --forms`, `forms *` | OAuth client id of the Forms app |
| `YX360_FORMS_ORG_ID` | `forms *` | Yandex 360 org id; sent as `X-Org-Id`/`X-Cloud-Org-Id` |
| `YX360_CONFIG_HOME` | optional | Override config root (token file store, room registry) |
| `YX360_IMAP_HOST` / `YX360_SMTP_HOST` | optional | Override Mail hosts (default `imap.yandex.ru` / `smtp.yandex.ru`) |
| `YX360_CALDAV_URL` | optional | Override CalDAV base (default `https://caldav.yandex.ru`) |
| `YX360_TELEMOST_API_URL` | optional | Override Telemost API base |
| `YX360_FORMS_API_URL` | optional | Override Forms API base (default `https://api.forms.yandex.net`) |

No `client_secret` is used or accepted anywhere; the CLI is a public OAuth client (PKCE). Never put secrets, tokens, or org-internal URLs in committed files.

## JSON Mode

Use the global `--json` flag for machine-readable output:

```bash
yx360 --json login --mail
yx360 --json mail list --limit 20
yx360 --json mail read <uid>
yx360 --json mail send --to user@example.com --subject "Subject" --body "Body" --yes
yx360 --json mail unsubscribe <uid>
yx360 --json calendar list --from 2026-06-20 --to 2026-06-21
yx360 --json calendar rooms discover --from 2026-01-01 --to 2026-12-31
yx360 --json calendar rooms list
yx360 --json calendar rooms add Sun sun@effective.band
yx360 --json calendar create --title "Meeting" --starts-at 2026-06-22T09:00:00+06:00 --ends-at 2026-06-22T09:30:00+06:00 --yes
yx360 --json calendar create --title "Meeting" --starts-at 2026-06-22T09:00:00+06:00 --ends-at 2026-06-22T09:30:00+06:00 --room Sun --yes
yx360 --json calendar create --title "Call" --starts-at 2026-06-22T10:00:00+06:00 --ends-at 2026-06-22T10:30:00+06:00 --telemost --yes
yx360 --json calendar update <event-href> --title "New title" --room Sun --yes
yx360 --json calendar delete <event-href> --yes
yx360 --json telemost create --yes
yx360 --json forms responses <survey-id> --page-size 50
yx360 --json forms create --title "Survey title" --yes
yx360 --json forms questions add <survey-id> --label "Контент" --rating 5 --yes
yx360 --json forms publish <survey-id> --yes
yx360 --json forms unpublish <survey-id> --yes
```

Current JSON output is pretty-printed JSON on stdout. The shape is command-specific and contains non-secret result fields only, for example:

- `login`: `status`, `account`, `scopes`, optional `expiry`.
- `logout`: `status`.
- `mail list` and `mail search`: an array of message summaries.
- `mail read`: one message with metadata, body, attachment manifest, and optional `unsubscribe_options`.
- `mail attachment`: `path`.
- `mail send`: `status`, `from`, `recipients`, `subject`.
- `mail unsubscribe`: preview has `uid`, `folder`, `options`, and optional `selected`; apply result has `status`, `uid`, `folder`, `method`, `uri`, optional `http_status`, and optional `mail`.
- `calendar list`: array of events. Event fields include `id`, `href`, optional `etag`, `uid`, `title`, optional `description`, optional `location`, optional `url`, `starts_at`, `ends_at`, optional `attendees`, optional `participants`, optional `rooms`, and optional `resources`.
- `calendar read/create/update/delete`: one event object with the same fields as above.
- `calendar rooms list/discover/add`: an array of room mappings with `name`, optional `email`, optional `uri`, and optional `kind`.
- `telemost create`: `id`, `join_url`.
- `forms responses`: `answers` (array) and optional `next_page_token`. Each answer is the raw API object — observed fields are `id` (number), `created` (timestamp), and `data` (array of `{value: [...]}` in question order). Field names vary; agents must tolerate additional or differing fields.
- `forms create`: `id`, optional `title`, `public_url`, `answers_url`.
- `forms questions add`: the created question object (`id`, `type`, `label`, `widget`, `items[]` with `id`/`label`/`slug`).
- `forms publish`/`forms unpublish`: `survey_id`, `status` (`published` or `unpublished`); `publish` also returns `public_url` and `answers_url`.

Public links are derived by the CLI (the API does not return them): a published form is at `https://forms.yandex.ru/cloud/<survey_id>`, answer stats at `https://forms.yandex.ru/cloud/admin/<survey_id>/answers?view=stats`.

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
- `mail, calendar/telemost, and forms scopes use different Yandex OAuth apps; run separate login commands`
- `no Calendar/Telemost OAuth client_id: set YX360_CALENDAR_CLIENT_ID`
- `forms: stored credential is missing, expired, or does not include the required forms scope; run yx360 login --forms`
- `forms: no Forms OAuth client_id: set YX360_FORMS_CLIENT_ID`
- `forms: no Forms org id: set YX360_FORMS_ORG_ID`
- `OS keychain unavailable (...): on headless/CI hosts re-run with --insecure-file-store`

Expired tokens are handled by re-running `yx360 login`; refresh is intentionally not implemented.

## Exit Codes And Machine Output

- **Exit codes:** `0` on success, non-zero (currently `1`) on any failure. Branch on the exit code first; only then inspect the error text on stderr. Differentiated codes per failure class are not yet implemented — do not depend on a specific non-zero value.
- **stdout vs stderr:** in `--json` mode the JSON payload is the only thing written to stdout, *provided the command does not stop at an interactive confirmation*. A human-gated write without `--yes` prints a preview and a `[y/N]` prompt to stdout and then reads stdin — so for machine use always pass `--yes` on gated commands (`mail send`, `mail unsubscribe --apply`, `calendar create/update/delete`, `telemost create`, `forms create/questions add/publish/unpublish`). With `--yes`, no preview/prompt is emitted and stdout stays pure JSON.
- **Errors** are written through the normal Cobra path to stderr, never to stdout.
- **Scope transparency (contract guarantee):** when a credential is missing or lacks a scope, the error text names the exact re-auth command to run (e.g. `run yx360 login --forms`). Agents may parse the trailing `run yx360 login ...` to self-heal.

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

Agents may pass `--yes` for a room-bearing calendar create/update only after the user has explicitly approved the exact room alias and resolved room address shown by the CLI or local room registry.

`calendar create --telemost` creates a Telemost conference first, then writes the returned `join_url` into the event. If Calendar creation fails after Telemost creation, the Telemost link may remain live; agents must surface that partial-failure risk.

`calendar create/update --room <name>` books a room by adding a structured room attendee from the local room registry. `--location` is display-only text and does not book a room by itself. When `--telemost` and `--room` are both passed, the Telemost URL is attached as `url` and description text; the CLI does not overwrite a physical room/location with the Telemost URL.

Room registry commands are local except `calendar rooms discover`, which reads Calendar events in the requested range and saves `ATTENDEE` entries with `CUTYPE=ROOM` or `CUTYPE=RESOURCE` into the user config file. `YX360_CONFIG_HOME` can override the config root for agents, tests, and sandboxed runs. Discovery is opportunistic: rooms that never appear in scanned events must be added manually with `calendar rooms add <name> <address>`.

## Human Gate For Forms Create And Publish

`forms create`, `forms questions add`, `forms publish`, and `forms unpublish` are writes. Without `--yes`, each prints a preview and requires interactive confirmation. A published form is publicly reachable by anyone with the link.

Agents may pass `--yes` only after the user has explicitly approved the exact action — the survey title for `create`, or the target `survey-id` for `publish`/`unpublish`. A general task approval is not enough.

`forms responses` is read-only and not gated. `survey-id` is supplied by the user or caller; there is no list-all-surveys command. `forms create` sets only the survey title (empty survey); add questions separately with `forms questions add <survey-id> --label <text> --rating <N>` (a 1..N radio rating question).

## Calendar IDs

Use the `href` returned by `calendar list` or `calendar create` as `<event-href>` for `calendar read`, `calendar update`, and `calendar delete`. Do not invent short IDs.

Calendar uses CalDAV with `Authorization: OAuth <token>`, not `Bearer <token>`. Agents must not assume generic bearer auth when diagnosing Calendar requests.

## Plaintext Token Warning

`--insecure-file-store` stores credentials as plaintext JSON under the user config directory with file mode `0600`. This is only for headless or CI hosts where the OS keychain is unavailable.

Agents must not select `--insecure-file-store` silently. Warn the user before using it, and do not combine it with send-capable scopes unless the user explicitly accepts the plaintext credential risk for that environment.
