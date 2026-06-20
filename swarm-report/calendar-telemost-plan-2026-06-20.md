# Consilium Plan - Calendar + Telemost

Status: calendar-and-telemost-proof-complete
Date: 2026-06-20
Slug: calendar-telemost
Request: "Calendar: read events, create/update/delete meetings, create a Telemost link."

## Conclusion

Calendar CRUD can use documented CalDAV with OAuth after the Yandex OAuth app is granted `calendar:all` and the user re-logins with that scope. A live proof against `https://caldav.yandex.ru` succeeded with `Authorization: OAuth <token>` and failed with `Authorization: Bearer <token>`. Telemost creation also works through the second OAuth app: `POST /v1/telemost-api/conferences` returned `201 Created` with a `join_url`.

The safe plan is:

1. Implement Calendar through CalDAV with `Authorization: OAuth <token>`, not `Bearer`.
2. Add `calendar:all` to the first-class login scope set and preserve existing Mail scopes during re-login.
3. Use a separate OAuth client/profile for Calendar/Telemost because Yandex rejected Mail+Calendar+Telemost as one scope set.
4. Implement Telemost as a separate REST client and attach the returned `join_url` to a calendar VEVENT.

Current worktree note: there is unrelated Mail unsubscribe implementation work in progress. It is not part of this Calendar/Telemost plan.

## Primary Blockers

| ID | Severity | Blocker | Decision Needed |
| --- | --- | --- | --- |
| B1 | RESOLVED | Calendar OAuth bearer support was unverified. | Live proof: `calendar:all` + `Authorization: OAuth <token>` returns `207 Multi-Status`; `Bearer` still fails. |
| B2 | RESOLVED | Exact Calendar OAuth scope was unknown. | Live proof: `calendar:all` works for CalDAV discovery. |
| B3 | RESOLVED | Telemost create scope can be consented and conference creation works. | Live proof returned `201 Created` and `join_url`. |
| B4 | HIGH | Create/update/delete meetings and Telemost link creation are externally visible side effects. | Use preview-by-default and require explicit confirmation or `--yes` for every mutation. |
| B5 | HIGH | Unit tests cannot prove this feature works. | Require a live smoke scenario with create/read/update/delete cleanup and Telemost link verification. |

## Live Proof Results

First run date: 2026-06-20
Command: `YX360_PROOF=calendar-telemost go test ./internal/proof -run TestCalendarTelemostProof -v`

Credential used:

- Account: redacted `<account>`
- Valid until: `2027-06-20T14:31:33+06:00`
- Stored scopes: `login:info`, `mail:imap_full`, `mail:smtp`

Calendar CalDAV results:

| Endpoint | Authorization | Status | Evidence |
| --- | --- | --- | --- |
| `https://caldav.yandex.ru/` | `OAuth <token>` | `401` | `WWW-Authenticate: Basic realm="CalDAV"` |
| `https://caldav.yandex.ru/principals/users/<account>/` | `OAuth <token>` | `401` | `WWW-Authenticate: Basic realm="CalDAV"` |
| `https://caldav.yandex.ru/` | `Bearer <token>` | `401` | `WWW-Authenticate: Basic realm="CalDAV"` |
| `https://caldav.yandex.ru/principals/users/<account>/` | `Bearer <token>` | `401` | `WWW-Authenticate: Basic realm="CalDAV"` |

Conclusion: the previous token did not include Calendar scope, so CalDAV rejected it.

Second run date: 2026-06-20
Command: `YX360_PROOF=calendar go test ./internal/proof -run TestCalendarProof -v`

Credential used:

- Account: redacted `<account>`
- Valid until: `2027-06-20T16:11:16+06:00`
- Stored scopes: `login:info`, `calendar:all`

Calendar CalDAV results after re-login:

| Endpoint | Authorization | Status | Evidence |
| --- | --- | --- | --- |
| `https://caldav.yandex.ru/` | `OAuth <token>` | `207` | `current-user-principal` returned the authenticated account principal |
| `https://caldav.yandex.ru/principals/users/<account>/` | `OAuth <token>` | `207` | principal resource returned |
| `https://caldav.yandex.ru/` | `Bearer <token>` | `401` | `WWW-Authenticate: Basic realm="CalDAV"` |
| `https://caldav.yandex.ru/principals/users/<account>/` | `Bearer <token>` | `401` | `WWW-Authenticate: Basic realm="CalDAV"` |

Conclusion: Calendar CalDAV OAuth works with `calendar:all`, but the auth scheme must be `OAuth`, not `Bearer`.

Telemost result:

- Request: non-mutating read probe against `https://cloud-api.yandex.net/v1/telemost-api/conferences/yx360-proof-nonexistent`
- Authorization: `OAuth <token>`
- Status: `403`
- Body summary: `ForbiddenError`; message says access is denied and the app may lack rights.

Conclusion: current credential cannot access Telemost API. Telemost needs OAuth consent with `telemost-api:conferences.create`, and likely account eligibility verification for Yandex 360 Business/org-domain. Conference creation was skipped because the current credential lacks Telemost scopes and the official API has no verified delete endpoint for cleanup.

Second Telemost run date: 2026-06-20
Command: `YX360_PROOF=telemost go test ./internal/proof -run TestTelemostProof -v`

Credential used:

- Stored scopes: `login:info`, `calendar:all`, `telemost-api:conferences.create`

Telemost read probe after create-scope login:

- Request: non-mutating read probe against `https://cloud-api.yandex.net/v1/telemost-api/conferences/yx360-proof-nonexistent`
- Authorization: `OAuth <token>`
- Status: `403`
- Body summary: `ForbiddenError`; message says access is denied and the app may lack rights.

Conclusion: the new OAuth app can issue a token with Telemost create scope. The read probe is not enough to validate create because only `telemost-api:conferences.create` was granted.

Telemost create proof date: 2026-06-20
Command: `YX360_PROOF=telemost-create go test ./internal/proof -run TestTelemostCreateProof -v`

Request:

- Method: `POST`
- URL: `https://cloud-api.yandex.net/v1/telemost-api/conferences`
- Authorization: `OAuth <token>`
- Body: `{"waiting_room_level":"PUBLIC"}`

Result:

- Status: `201 Created`
- ID: `<redacted-test-conference-id>`
- Join URL: `<redacted-test-join-url>`

Conclusion: Telemost conference creation is live-verified. The test link may remain live because no official delete endpoint has been verified.

## Scope For V1

Included:

- Read events from the authenticated user's own calendar.
- Create a single non-recurring meeting.
- Update title, time range, description, location, and explicit attendee emails.
- Delete an event created or explicitly selected by the user.
- Create a Telemost conference and attach its `join_url` to the event.
- Use confirmation gates for every mutating command.

Out of scope:

- Recurring event edits.
- Shared/delegated calendars.
- Room/resource booking.
- Organization directory lookup or autocomplete.
- Background sync.
- Private Yandex web/mobile endpoints.
- Telemost conference deletion/cancellation, unless official API support is verified.
- App-password Calendar auth unless the human explicitly accepts a separate credential model.

## Proposed Architecture

Add separate modules:

- `internal/calendar`: CalDAV client, iCalendar serialization, ETag-aware event CRUD.
- `internal/telemost`: small REST client for the official Telemost API.
- `internal/cli/calendar.go`: command wiring, previews, confirmations, JSON output.

Keep Telemost outside the Calendar client. Calendar stores events; Telemost creates conference links. Composition belongs in the CLI/use-case layer so partial failures are explicit, for example "Telemost link created, event creation failed."

Suggested commands:

```text
yx360 calendar list --from <time> --to <time> [--json]
yx360 calendar read <event-id> [--json]
yx360 calendar create --title <text> --starts-at <time> --ends-at <time> [--attendee <email>] [--telemost] [--yes] [--json]
yx360 calendar update <event-id> [fields...] [--yes] [--json]
yx360 calendar delete <event-id> [--yes] [--json]
yx360 telemost create [--title <text>] [--yes] [--json]
```

Preview output for mutations must include event UID, title, time range, attendees, whether notifications may be sent, and whether a Telemost link will be created.

## Implementation Slices

### Slice 0 - Auth And Scope Proof

Goal: prove the integration surfaces before feature code.

Tasks:

- Completed: probe CalDAV at `https://caldav.yandex.ru` before Calendar scope; it returned `401`.
- Completed: re-login with `calendar:all`.
- Completed: prove Calendar CalDAV discovery with `Authorization: OAuth <token>`; it returned `207`.
- Completed: prove `Authorization: Bearer <token>` does not work for Calendar CalDAV.
- Completed: re-login with second OAuth app using `calendar:all` and `telemost-api:conferences.create`.
- Completed: create a Telemost test conference with `waiting_room_level=PUBLIC`; API returned `201` and `join_url`.
- Next: implement Calendar with the `OAuth` auth scheme.
- Verify Telemost scopes in live OAuth consent:
  - `telemost-api:conferences.create`
  - `telemost-api:conferences.read`
  - `telemost-api:conferences.update`
- Verify Telemost account eligibility.
- Record the result in the implementation report and, if accepted, decisions.

Gate: Calendar read/list can proceed. Calendar mutation still requires CalDAV collection discovery and live cleanup tests.

### Slice 1 - Calendar Read

Goal: read-only Calendar over CalDAV.

Tasks:

- Add CalDAV discovery/config for the primary calendar.
- Add `calendar list` and `calendar read`.
- Parse iCalendar VEVENTs into stable CLI/JSON output.
- Add unit tests for calendar-query request construction and VEVENT parsing.
- Add live smoke: list a narrow time window and read one event.

### Slice 2 - Calendar Mutations

Goal: create/update/delete non-recurring meetings.

Tasks:

- Add VEVENT generation with stable UID.
- Use CalDAV `PUT` for create/update and WebDAV `DELETE` for delete.
- Preserve and enforce ETags / `If-Match` to avoid overwriting remote changes.
- Add preview-by-default and `--yes` gates.
- Add live smoke: create test event, read back, update, read back, delete, verify cleanup.

### Slice 3 - Telemost Link

Goal: create a Telemost link and attach it to a Calendar event.

Tasks:

- Add a small `internal/telemost` HTTP client for `POST https://cloud-api.yandex.net/v1/telemost-api/conferences`.
- Request only `telemost-api:conferences.create` for link creation unless read/update commands ship.
- Store `join_url` in VEVENT `LOCATION`, `URL`, and `DESCRIPTION`; add `CONFERENCE` only if the chosen iCalendar library supports it safely.
- Add `calendar create --telemost`.
- Add live smoke: create Telemost conference, create event containing `join_url`, read event back, delete event.

## Engineering Risks

| Severity | Risk | Mitigation |
| --- | --- | --- |
| HIGH | Current login flow can save only the scopes requested in the latest login and drop previous Mail scopes. | Centralize scope profiles and request union of existing and new scopes. |
| MEDIUM | Credentials record requested scopes, not provider-returned scopes. | Store returned scopes when available; fail closed when required scopes cannot be proven. |
| MEDIUM | Current token model has no refresh. | Keep v1 synchronous and user-triggered; check token validity before mutation. |
| MEDIUM | Existing IPv4-only decision covers OAuth/account-info/Mail, not Calendar/Telemost. | Live-test Calendar and Telemost endpoints before extending IPv4 policy. |
| MEDIUM | Missing `.memory-bank/product-overview/pipeline-stages.md` means a canonical gate link is broken. | Duplicate required gates in this plan and the implementation report. |
| MEDIUM | Telemost link may be created but Calendar event creation may fail. | Report partial failure clearly; do not hide orphaned Telemost conference risk. |
| MEDIUM | Attendees, notifications, organizer identity, recurrence, and time zones are user-visible behavior. | Keep v1 narrow: explicit attendees, non-recurring events, no directory/room lookup. |

## Verification Gate

Unit tests:

- CalDAV request construction for list/read/create/update/delete.
- iCalendar parse/generate for VEVENT.
- ETag conflict behavior.
- Confirmation gate behavior for create/update/delete/Telemost.
- Telemost request/response parsing.
- Scope/eligibility failure messages.

Live tests:

- Login with required scopes or configured Calendar auth.
- Calendar list in a narrow date range.
- Create a uniquely named test event.
- Read it back by UID.
- Update title/time/description.
- Read it back again and verify changes.
- Delete it.
- Verify cleanup.
- Create a Telemost link.
- Create a Calendar event with that link attached.
- Read event back and verify the link is present.
- Delete the test event.

Live tests must use a dedicated test event prefix and must not delete unrelated calendar events.

## Research Notes

- Official Calendar sync docs document CalDAV access with server `https://caldav.yandex.ru` and a principal path under `/principals/users/<login@domain>/`.
- Those Calendar sync docs require a Yandex ID app password for Calendar, so OAuth bearer support remains unverified.
- Official Yandex 360 REST API docs do not list Calendar event CRUD as a general REST surface.
- Official Telemost docs describe a REST API, OAuth auth, and conference creation returning `join_url`.
- Official Telemost docs restrict API access to Yandex 360 Business users on organization-domain accounts.
- Official Telemost docs list create/read/update scopes, but no verified delete/cancel conference endpoint was found.
- RFC 4791 CalDAV and RFC 5545 iCalendar are the correct standards base for Calendar CRUD.

Sources checked on 2026-06-20:

- https://yandex.ru/support/yandex-360/customers/calendar/web/ru/data-exchange/synchronization/sync-desktop.md
- https://yandex.ru/support/yandex-360/customers/calendar/web/ru/data-exchange/synchronization/sync-mobile.md
- https://yandex.ru/dev/api360/doc/ru/
- https://yandex.ru/dev/api360/doc/ru/access
- https://yandex.ru/dev/telemost/doc/ru/
- https://yandex.ru/dev/telemost/doc/ru/access
- https://yandex.ru/dev/telemost/doc/ru/conference-create
- https://doc-static.yandex.net/dev/telemost/api-specification.yaml
- https://www.rfc-editor.org/rfc/rfc4791.html
- https://www.rfc-editor.org/rfc/rfc5545
- https://pkg.go.dev/github.com/emersion/go-webdav/caldav
- https://pkg.go.dev/github.com/essentialkaos/telemost

## Human Questions

1. For v1, is single-calendar, non-recurring meeting CRUD acceptable?
2. Should `calendar create --telemost` create the Telemost conference before the Calendar event and report possible orphaned links, or should Telemost stay as a separate `telemost create` command first?
3. Should the CLI store separate credentials for `mail` and `calendar-telemost` profiles before implementing commands?

## Agent Findings

### Architect

```yaml
- severity: HIGH
  category: scope
  file: .memory-bank/tech-details/stack.md
  line: 41
  problem: Calendar CRUD is not ready for implementation until CalDAV OAuth bearer auth is live-verified because current project knowledge says sources conflict between OAuth and app-password auth.
  suggested_fix: Start with a live CalDAV spike against `caldav.yandex.ru` using the stored OAuth token and only then commit calendar read/create/update/delete to v1. If OAuth bearer fails, keep Calendar out of scope or require a human decision on app-password support.
  requires_human: true
  confidence: corroborated

- severity: HIGH
  category: scope
  file: .memory-bank/tech-details/stack.md
  line: 42
  problem: Telemost link creation cannot be safely bundled into the Calendar feature until the exact Telemost OAuth scope is verified in the live Yandex OAuth consent screen.
  suggested_fix: Treat Telemost scope discovery as a blocking migration step before implementation. Add the verified constant only after live consent and a minimal link-create smoke pass.
  requires_human: true
  confidence: high

- severity: HIGH
  category: migration
  file: internal/cli/login.go
  line: 50
  problem: The current login flow saves a credential with only the scopes selected for that invocation, so adding `--calendar` or `--telemost` flags can accidentally drop previously granted Mail scopes.
  suggested_fix: Centralize scope profiles and make re-login request the union of existing granted scopes plus newly requested scopes, with explicit output of the final requested scope set. If Yandex consent cannot preserve scope union reliably, document that `login --all` is the safe migration path.
  requires_human: false
  confidence: high

- severity: HIGH
  category: module-boundary
  file: .memory-bank/tech-details/stack.md
  line: 50
  problem: Calendar event persistence and Telemost room creation are separate transports and lifecycles, so a single Calendar service that directly owns Telemost HTTP calls would blur module boundaries and make partial-failure handling brittle.
  suggested_fix: Use `internal/calendar` for CalDAV event CRUD, `internal/telemost` for Telemost HTTP, and compose them from `internal/cli` or a small use-case layer for `calendar create --telemost`. Make the composition handle "Telemost created but event create failed" explicitly in the result.
  requires_human: false
  confidence: high

- severity: HIGH
  category: architecture
  file: internal/cli/mail.go
  line: 162
  problem: Calendar create/update/delete and Telemost link creation are externally visible side effects, but Mail's existing confirmation gate only covers SMTP send.
  suggested_fix: Mirror the Mail send pattern with preview-by-default and explicit `--yes` for meeting create/update/delete and Telemost link creation. Deletion should show the event identity and require confirmation unless `--yes` is present.
  requires_human: false
  confidence: high

- severity: MEDIUM
  category: architecture
  file: .assistant/decisions.md
  line: 69
  problem: The accepted IPv4 decision is explicitly limited to OAuth/account-info and Mail IMAP, so applying `tcp4` blindly to Calendar or Telemost would exceed the evidence.
  suggested_fix: Give Calendar and Telemost their own transport configuration and live-verify connectivity before deciding `tcp4` versus default dialing. If IPv6 fails there too, record a new decision that expands D-006.
  requires_human: false
  confidence: high

- severity: HIGH
  category: testing
  file: .memory-bank/product-overview/anti-stories.md
  line: 51
  problem: Unit tests alone cannot prove this feature works because Calendar and Telemost both depend on live Yandex service behavior and side effects.
  suggested_fix: The verify gate must include live OAuth consent, calendar list, create, update, read-back, delete cleanup, Telemost link creation, and confirmation that the link is present in the created meeting. Use a dedicated test event name and always clean it up.
  requires_human: true
  confidence: high
```

### Skeptic

```yaml
- severity: HIGH
  category: premise-flaw
  file: .memory-bank/tech-details/stack.md
  line: 40
  problem: The feature assumes Calendar CRUD is available through the current OAuth path, but stack memory explicitly says Calendar/CalDAV bearer support must be empirically verified before code depends on it.
  suggested_fix: Split the work into a read-only CalDAV proof first. Do not implement create/update/delete until a live XOAUTH2 Calendar smoke passes against a disposable calendar.
  requires_human: true
  confidence: corroborated
- severity: HIGH
  category: verification
  file: .memory-bank/tech-details/stack.md
  line: 42
  problem: Exact non-Mail OAuth scopes are unresolved, so adding calendar or Telemost flags now risks shipping commands that cannot be consented to or that falsely record requested scopes as granted.
  suggested_fix: Have the owner verify exact scope strings in the Yandex OAuth app UI and then live-login with those scopes before implementation. Add a decision entry only after live consent and API auth succeed.
  requires_human: true
  confidence: corroborated
- severity: HIGH
  category: risk
  file: .assistant/decisions.md
  line: 87
  problem: Creating, updating, deleting meetings, and creating invite links are externally visible actions, but the only established safety precedent is Mail send being human-gated by default.
  suggested_fix: Require preview plus confirmation for every mutating calendar command, with `--yes` explicit and non-default. For delete/update, show UID, title, time range, attendees, and whether notifications will be sent.
  requires_human: false
  confidence: high
- severity: HIGH
  category: hidden-cost
  file: .memory-bank/tech-details/stack.md
  line: 41
  problem: If Yandex Calendar requires app passwords instead of OAuth bearer for CalDAV, implementing it would conflict with the documented-OAuth/no-extra-secret direction used for Mail.
  suggested_fix: Treat app-password Calendar support as out-of-scope unless the human explicitly revises the auth decision. Prefer OAuth-only Calendar or defer mutation.
  requires_human: true
  confidence: corroborated
- severity: MEDIUM
  category: scope-creep
  file: n-a
  line: n-a
  problem: "Meetings" is broader than CRUD events because attendees, recurrence, time zones, reminders, organizer identity, notifications, ETags, and conflict handling are all user-visible behavior.
  suggested_fix: Define v1 as single-calendar non-recurring events with explicit attendees and no recurrence edits, or require a separate design pass. Anything involving recurrence or organizer transfer should be a later slice.
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: premise-flaw
  file: .memory-bank/tech-details/stack.md
  line: 50
  problem: Telemost link creation may be a separate documented Telemost API rather than Calendar conferencing metadata, so bundling it into Calendar CRUD hides a second integration surface.
  suggested_fix: Implement Telemost link creation as a separate proof with its own scope, endpoint, and live smoke. Calendar should only attach a link after the link source is proven.
  requires_human: true
  confidence: medium
- severity: MEDIUM
  category: security
  file: internal/auth/oauth.go
  line: 41
  problem: The credential records requested scopes rather than the scopes returned by the token response, which can make future Calendar or Telemost authorization checks trust scopes the provider did not actually grant.
  suggested_fix: Store provider-returned scope data when available and fail closed when required scopes are absent or unverifiable. Add a regression test before adding more scope-gated services.
  requires_human: false
  confidence: medium
- severity: MEDIUM
  category: risk
  file: .assistant/decisions.md
  line: 45
  problem: Tokens cannot be refreshed without a client secret, so long-running calendar automation may fail near expiry and leave partially completed meeting updates.
  suggested_fix: Keep v1 synchronous and user-initiated, and check token validity before any mutation batch. Avoid background calendar sync or automation until refresh strategy changes.
  requires_human: false
  confidence: corroborated
- severity: MEDIUM
  category: verification
  file: .assistant/decisions.md
  line: 67
  problem: The IPv4-only network decision currently covers OAuth/account-info and Mail endpoints, not Calendar CalDAV or Telemost API endpoints.
  suggested_fix: Test Calendar and Telemost endpoints over the same network before choosing default dialing behavior. Extend the IPv4 policy only with live evidence.
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: risk
  file: .assistant/open-questions.md
  line: 22
  problem: Meeting attendees may cross from personal self-data into org or Directory-adjacent behavior, where authorization and consent posture remains open.
  suggested_fix: State that v1 only acts on the authenticated user's own calendar and explicit attendee emails supplied by the user. Do not add org lookup, room lookup, or directory autocomplete in this feature.
  requires_human: true
  confidence: corroborated
```

### Researcher

```yaml
- severity: HIGH
  category: protocol
  file: n-a
  line: n-a
  problem: "As of 2026-06-20, official Yandex Calendar sync docs say Calendar supports CalDAV and lets users view/edit meetings via external calendars, with server https://caldav.yandex.ru and principal path /principals/users/<login@domain>/; source: https://yandex.ru/support/yandex-360/customers/calendar/web/ru/data-exchange/synchronization/sync-desktop.md"
  suggested_fix: "Implement Calendar CRUD through CalDAV first, not a private Calendar REST API. Use RFC 4791 CalDAV operations plus RFC 5545 iCalendar event payloads."
  requires_human: false
  confidence: high
- severity: HIGH
  category: security
  file: n-a
  line: n-a
  problem: "As of 2026-06-20, official Yandex Calendar desktop/mobile CalDAV docs require a Yandex ID app password created for Calendar, not an OAuth bearer token; sources: https://yandex.ru/support/yandex-360/customers/calendar/web/ru/data-exchange/synchronization/sync-desktop.md and https://yandex.ru/support/yandex-360/customers/calendar/web/ru/data-exchange/synchronization/sync-mobile.md"
  suggested_fix: "Do not silently add Calendar to the existing OAuth-first credential model until bearer support is live-verified. If Calendar ships now, require an explicit user-provided Calendar app password stored in keychain and clearly label it as separate from OAuth."
  requires_human: true
  confidence: corroborated
- severity: MEDIUM
  category: scope
  file: n-a
  line: n-a
  problem: "As of 2026-06-20, I found no official Yandex Calendar OAuth scope or official CalDAV bearer-token support in Yandex Calendar, Yandex 360 API, or Yandex ID docs; source checked: https://yandex.ru/dev/api360/doc/ru/access"
  suggested_fix: "Mark Calendar OAuth/bearer support as unverified and run a live CalDAV probe with Authorization: OAuth before designing token reuse. If it fails, keep Calendar behind app-password auth."
  requires_human: true
  confidence: unverified
- severity: MEDIUM
  category: research
  file: n-a
  line: n-a
  problem: "As of 2026-06-20, official Yandex 360 API docs list organizations, groups, departments, users, contacts, mail/admin/security entities, domains, DNS, audit logs, and service applications, but not Calendar events; source: https://yandex.ru/dev/api360/doc/ru/"
  suggested_fix: "Do not plan Calendar event CRUD against Yandex 360 REST API unless a separate official Calendar API is found. Treat CalDAV as the documented integration surface."
  requires_human: false
  confidence: medium
- severity: HIGH
  category: protocol
  file: n-a
  line: n-a
  problem: "As of 2026-06-20, official Telemost API creates meetings with POST https://cloud-api.yandex.net/v1/telemost-api/conferences and returns join_url; source: https://yandex.ru/dev/telemost/doc/ru/conference-create"
  suggested_fix: "Create the Telemost link before creating/updating the calendar VEVENT, then put join_url into LOCATION, URL, DESCRIPTION, and preferably CONFERENCE if the chosen iCalendar library supports RFC 7986-style conference metadata."
  requires_human: false
  confidence: high
- severity: HIGH
  category: scope
  file: n-a
  line: n-a
  problem: "As of 2026-06-20, official Telemost API scopes are telemost-api:conferences.create, telemost-api:conferences.read, and telemost-api:conferences.update, with Authorization: OAuth <token>; source: https://yandex.ru/dev/telemost/doc/ru/access"
  suggested_fix: "Add a Telemost login scope set separate from Mail and Calendar. Request only create for link creation unless read/update features are implemented."
  requires_human: false
  confidence: high
- severity: HIGH
  category: compatibility
  file: n-a
  line: n-a
  problem: "As of 2026-06-20, official Telemost docs restrict API access to Yandex 360 for Business users with accounts on an organization domain; source: https://yandex.ru/dev/telemost/doc/ru/"
  suggested_fix: "Gate Telemost link creation with a clear eligibility error for personal accounts. Do not promise Telemost API support for ordinary personal Yandex accounts."
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: verification
  file: n-a
  line: n-a
  problem: "As of 2026-06-20, official Telemost docs and OpenAPI spec expose create/read/update and cohost/settings endpoints, but I did not find an official delete/cancel conference endpoint; source: https://doc-static.yandex.net/dev/telemost/api-specification.yaml"
  suggested_fix: "Do not rely on third-party Telemost client Delete unless verified against official spec or live API. Calendar event deletion should delete the CalDAV VEVENT; Telemost meeting cancellation remains unverified."
  requires_human: true
  confidence: medium
- severity: MEDIUM
  category: protocol
  file: n-a
  line: n-a
  problem: "CalDAV is the standards-track protocol for accessing/managing/sharing calendaring data based on iCalendar, and defines PUT/REPORT/free-busy/calendar-query semantics; source: https://www.rfc-editor.org/rfc/rfc4791.html"
  suggested_fix: "Use RFC 4791 calendar-query REPORT for reads, PUT for create/update with stable UID and ETag conflict handling, and WebDAV DELETE for event removal. Use RFC 5545 for VEVENT serialization."
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: compatibility
  file: n-a
  line: n-a
  problem: "As of 2026-06-20, github.com/emersion/go-webdav/caldav provides a Go CalDAV client/server package with QueryCalendar, PutCalendarObject, MultiGetCalendar, and embedded WebDAV operations, but it is pre-v1 and pkg.go.dev marks the viewed version as not latest; source: https://pkg.go.dev/github.com/emersion/go-webdav/caldav"
  suggested_fix: "Prefer emersion/go-webdav/caldav for protocol operations because it aligns with the existing emersion mail stack, but pin and smoke-test against caldav.yandex.ru. Use github.com/emersion/go-ical or github.com/arran4/golang-ical for iCalendar payloads."
  requires_human: false
  confidence: medium
- severity: LOW
  category: compatibility
  file: n-a
  line: n-a
  problem: "As of 2026-06-20, github.com/essentialkaos/telemost v0.2.0 is a Go client for the official Telemost API and exposes Create/Get/Update/Cohost methods, but it is pre-v1 and has zero known importers on pkg.go.dev; source: https://pkg.go.dev/github.com/essentialkaos/telemost"
  suggested_fix: "Either vendor a tiny internal Telemost client over net/http or use the library only after checking its requests against the official OpenAPI spec. The REST surface is small enough that a local client may be lower risk."
  requires_human: false
  confidence: medium
```

### Reviewer

```yaml
- severity: HIGH
  category: scope
  file: swarm-report/mail-unsubscribe-implementation-2026-06-20.md
  line: 1
  problem: Reviewed working tree implements Mail unsubscribe, not the requested Calendar/Telemost feature.
  suggested_fix: Do not accept this as the Calendar feature. Create or review the actual Calendar/Telemost plan and diff.
  requires_human: true
  confidence: corroborated
- severity: HIGH
  category: decision-drift
  file: .memory-bank/tech-details/stack.md
  line: 41
  problem: Calendar/Contacts CalDAV/CardDAV personal-account OAuth is explicitly marked as requiring empirical verification before v1 scope, but no Calendar implementation or verification evidence exists in the reviewed diff.
  suggested_fix: Run a documented CalDAV XOAUTH2 spike against caldav.carddav.yandex.ru before implementing calendar read/write/delete. Record the result in decisions or the feature plan.
  requires_human: false
  confidence: high
- severity: HIGH
  category: testing
  file: n-a
  line: n-a
  problem: No Type 2 live smoke evidence exists for reading events, creating, modifying, deleting meetings, or creating a Telemost link.
  suggested_fix: Add a live smoke scenario covering all externally visible Calendar/Telemost mutations with explicit human approval and recorded cleanup evidence. Unit tests alone are insufficient under ANTI-11.
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: decision-drift
  file: .memory-bank/tech-details/stack.md
  line: 42
  problem: Exact non-Mail OAuth scope strings, including Telemost scopes, are still marked as needing live consent-screen verification.
  suggested_fix: Verify the minimum Calendar and Telemost scopes before hardcoding them or requesting consent. Keep read, write/delete, and Telemost-link scopes separate if Yandex exposes that split.
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: docs
  file: .memory-bank/steerings/project-types.md
  line: 83
  problem: The canonical Pipeline Stages link points to missing .memory-bank/product-overview/pipeline-stages.md, so Type 2 gate references are incomplete for this feature.
  suggested_fix: Restore the missing pipeline-stages.md or duplicate the required gates inside the Calendar feature plan and final report. Do not rely on the broken link during handoff.
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: scope
  file: .memory-bank/product-overview/vision.md
  line: 28
  problem: The product DoD still requires at least one non-Mail Yandex 360 surface through the CLI, but the reviewed changes remain Mail-only.
  suggested_fix: Treat Calendar/Telemost as the next non-Mail surface only after documented protocol/API verification succeeds. Do not mark broader product progress from Mail unsubscribe work.
  requires_human: false
  confidence: high
```
