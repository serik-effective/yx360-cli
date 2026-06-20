# Calendar Room Booking Plan

Status: consilium-complete
Slug: calendar-room-booking
Date: 2026-06-20

Feature description:

> Support meeting-room names that exist in the Yandex Calendar web UI. The known rooms are Callbox, Mercury, and Sun. The CLI should let users specify these rooms, so agents can interpret requests such as "make tomorrow's meeting in Sun" and create the meeting with that room.

Input validation:

- `.assistant/INVARIANTS.md` read.
- `.memory-bank/product-overview/anti-stories.md` read.
- `.memory-bank/product-overview/pipeline-stages.md` is absent in this repository.
- `.assistant/decisions.md` read.
- `.assistant/open-questions.md` read.
- No hard-stop violation: this is a `yx360` product feature, not a harness dev-workflow wrapper.
- Scope conflict found: D-009 explicitly left rooms/resources and org directory lookup out of the narrow Calendar v1.

## TL;DR

Severity counts: HIGH 4, MEDIUM 7, LOW 2.

Top 3 must-fix items:

1. Decide scope before implementation: `--location Sun` is display text, not room booking. Real booking needs resource identity and verified Yandex CalDAV behavior.
2. Do not hardcode Callbox/Mercury/Sun directly into core calendar code. Use a resolver boundary: CLI collects `--room`, resolver maps aliases to resource identities, calendar package serializes resolved resources.
3. Add a live proof before marking done: create a disposable meeting with the resolved room, read it back, verify Yandex shows the room as booked/conflicted, then delete the event.

Recommended implementation path:

1. Add a narrow proof command/test path, not user-facing behavior yet.
2. Get human-provided mapping for room names to resource calendar addresses/URIs, or discover them through a verified Yandex API if available.
3. Add typed participants/resources in `internal/calendar`, preserving backward-compatible `attendees` JSON.
4. Add `calendar create --room <name>` and `calendar update --room <name>` only after the live proof passes.
5. Update `docs/agent-contract.md` so agents must confirm the exact resolved room before passing `--yes`.

## Blockers

- severity: HIGH
  category: decision-consistency
  file: .assistant/decisions.md
  line: 109
  problem: D-009 explicitly rejected rooms/resources and org directory lookup as out-of-scope for the narrow personal-account Calendar v1.
  suggested_fix: Treat this as a new feature decision. Choose either display-only location support or real resource booking, then record the decision after live proof.
  requires_human: true
  confidence: high

- severity: HIGH
  category: resource-calendar-ambiguity
  file: .assistant/open-questions.md
  line: 23
  problem: Room lookup and booking may require organization/resource capabilities, while org/Directory authorization is still unresolved.
  suggested_fix: For v1, avoid org Directory discovery unless verified. Prefer a human-provided room mapping for Callbox, Mercury, and Sun, then live-test CalDAV booking behavior.
  requires_human: true
  confidence: high

- severity: HIGH
  category: protocol-model
  file: internal/calendar/ical.go
  line: 207
  problem: Current ICS serialization writes attendees only as bare `ATTENDEE:mailto:<value>`, which cannot represent room/resource metadata.
  suggested_fix: Add a structured participant/resource model and serialize rooms only after proving the exact Yandex-accepted resource attendee shape.
  requires_human: true
  confidence: corroborated

- severity: HIGH
  category: mutation-safety
  file: docs/agent-contract.md
  line: 102
  problem: The agent approval contract allows `--yes` after approving title, time range, attendees, deletion target, or Telemost creation, but says nothing about room/resource selection.
  suggested_fix: Extend the contract so `--yes` for room-bearing calendar mutations requires explicit approval of the exact resolved room/resource.
  requires_human: false
  confidence: high

## Concerns

- severity: MEDIUM
  category: cli-contract
  file: internal/cli/calendar.go
  line: 247
  problem: The CLI has `--location` and `--attendee`, but no stable way to express a bookable room.
  suggested_fix: Add repeatable `--room <alias>` and keep `--location` as display-only text.
  requires_human: false
  confidence: high

- severity: MEDIUM
  category: module-boundary
  file: internal/cli/calendar.go
  line: 241
  problem: Calendar flags bind directly into `calendar.Event`, which would tempt room alias parsing to leak into CLI or ICS formatting.
  suggested_fix: Add a resolver boundary that maps aliases to resolved resource identities before calling `internal/calendar`.
  requires_human: false
  confidence: high

- severity: MEDIUM
  category: parser
  file: internal/calendar/ical.go
  line: 51
  problem: Read/list parsing drops ATTENDEE parameters, so room metadata returned by Yandex would be flattened or lost.
  suggested_fix: Parse property parameters and expose rooms/resources separately from human attendees in JSON.
  requires_human: false
  confidence: corroborated

- severity: MEDIUM
  category: caldav-scheduling
  file: internal/calendar/service.go
  line: 93
  problem: Create uses raw CalDAV PUT without explicit ORGANIZER, Schedule-Tag handling, or scheduling result inspection.
  suggested_fix: Live-test whether Yandex accepts room resource attendees through PUT; if not, add the scheduling metadata Yandex requires.
  requires_human: true
  confidence: medium

- severity: MEDIUM
  category: calendar-selection
  file: internal/calendar/service.go
  line: 179
  problem: Calendar operations currently select the first calendar collection; room booking may involve specific shared/resource calendars or resource identities.
  suggested_fix: Fail closed if room booking requires a non-primary calendar or ambiguous resource collection.
  requires_human: false
  confidence: high

- severity: MEDIUM
  category: overfitting
  file: internal/cli/calendar.go
  line: 241
  problem: Hardcoding Callbox, Mercury, and Sun in the binary would make one organization's room names part of generic CLI behavior.
  suggested_fix: Keep code generic and put room aliases in configuration, environment, or a discovered resource table.
  requires_human: false
  confidence: high

- severity: MEDIUM
  category: tests
  file: internal/calendar/ical_test.go
  line: 22
  problem: Tests cover plain event ICS but not resource attendees, room validation, or outgoing CalDAV body semantics.
  suggested_fix: Add unit tests for room serialization/parsing, CLI validation, and an `httptest` CalDAV create path that asserts the outgoing PUT body.
  requires_human: false
  confidence: high

## Notes

- severity: LOW
  category: agent-contract
  file: docs/agent-contract.md
  line: 47
  problem: JSON output lists flat attendees and location only, so agents have no stable field for selected room, room identifier, or booking status.
  suggested_fix: Add additive fields such as `rooms` or `resources` while preserving existing `attendees`.
  requires_human: false
  confidence: high

- severity: LOW
  category: rollout
  file: README.md
  line: 121
  problem: Room booking adds shared-resource state, conflict handling, and alias management on top of an already young Calendar surface.
  suggested_fix: Keep the first implementation slice narrow and live-verified.
  requires_human: true
  confidence: medium

## Research Findings

- Yandex Calendar public help documents event creation with participants, Telemost links, time/date, calendar selection, and a free-form place/location field, but the page does not document a public CLI/API representation for bookable rooms. Source: Yandex Calendar help, "Create event", 2026-06-20, https://yandex.ru/support/yandex-360/customers/calendar/web/ru/plan-events/events/event-create. Confidence: high.
- Yandex Calendar public help documents automatic Telemost creation as a Calendar setting, and says the link appears in the event description when participants are added. Source: Yandex Calendar help, "Create Telemost call automatically", 2026-06-20, https://yandex.ru/support/yandex-360/customers/calendar/web/ru/plan-events/events/auto-call-create. Confidence: high.
- RFC 5545 defines `RESOURCES` as text describing equipment/resources for an activity, but that alone is descriptive metadata, not necessarily booking. Source: RFC 5545 section 3.8.1.10, https://www.rfc-editor.org/rfc/rfc5545.html. Confidence: high.
- RFC 5545 also defines attendee calendar-user typing through `CUTYPE`; RFC 6638 describes server-side scheduling behavior around `ATTENDEE` and `SCHEDULE-AGENT`. This supports modeling rooms as structured calendar users/resources rather than plain strings, but exact Yandex behavior must be live-tested. Sources: RFC 5545 and RFC 6638, https://www.rfc-editor.org/rfc/rfc5545.html and https://www.rfc-editor.org/rfc/rfc6638.html. Confidence: corroborated.

## Out-of-Scope Declared

- Automatic organization Directory lookup unless a separate live proof verifies Yandex scopes/API and consent posture.
- Full availability search across all rooms.
- Recurring room booking.
- Conflict auto-resolution.
- Admin-wide room/resource management.
- Deleting or modifying existing user-created room bookings outside the target event.

## Proposed Slice

## Live Proof Result

Status: proof-complete
Date: 2026-06-20

Room tested: `Sun`

Read-only proof from an existing web-created event showed that Yandex represents `Sun` as a room attendee:

```ics
LOCATION:Sun
ORGANIZER;CN=Elina Murzoeva:mailto:elina.murzoeva@effective.band
ATTENDEE;CUTYPE=ROOM;PARTSTAT=ACCEPTED;CN=Sun;ROLE=REQ-PARTICIPANT:mailto:sun@effective.band
```

Create proof created a disposable event through CalDAV with the same room attendee shape:

```ics
LOCATION:Sun
ORGANIZER;CN=Serik Beysenov:mailto:serik.beysenov@effective.band
ATTENDEE;CUTYPE=ROOM;PARTSTAT=ACCEPTED;CN=Sun;ROLE=REQ-PARTICIPANT:mailto:sun@effective.band
```

Read-back returned `PARTSTAT=ACCEPTED`, and cleanup deleted the proof event with HTTP 204.

Conclusion:

- `Sun` can be booked through CalDAV by writing a room attendee with `CUTYPE=ROOM`.
- The minimal proven v1 mapping for `Sun` is `mailto:sun@effective.band`.
- Implementor still needs mappings/proofs for `Callbox` and `Mercury`, or a configurable room map that starts with only proven rooms.
- The product should parse and preserve attendee parameters; plain `[]string` attendees are insufficient.

## Accepted V1 Scope

Human decision: proceed with a hybrid room registry.

V1 behavior:

- Discover room aliases from existing calendar events by scanning returned ICS for `ATTENDEE` entries with `CUTYPE=ROOM` or `CUTYPE=RESOURCE`.
- Persist discovered rooms in a local user-owned registry.
- Let the user manually add or override room mappings when a room has not appeared in scanned events.
- Support `calendar create --room <name>` and `calendar update --room <name>` using the registry.
- Keep `--location` as display-only text. A room booking must be represented as a room attendee, not only as `LOCATION`.

V1 non-goals:

- No org Directory room discovery.
- No admin/org-wide capabilities.
- No guarantee that auto-discovery returns rooms that never appeared in the scanned calendar range.
- No automatic conflict resolution beyond the booking status Yandex returns.

Command shape:

```bash
yx360 calendar rooms discover --from 2026-01-01 --to 2026-12-31
yx360 calendar rooms list
yx360 calendar rooms add Sun sun@effective.band
yx360 calendar create --title "Meeting" --starts-at ... --ends-at ... --room Sun
```

### Slice 1: Live Proof, No Public CLI Contract

Goal: identify the exact representation Yandex uses for Callbox, Mercury, and Sun.

Tasks:

1. Create or obtain a temporary disposable event in the web UI with one known room, then read raw CalDAV ICS for that event.
2. Capture only non-secret, non-personal structural fields: `ATTENDEE` parameters, `RESOURCES`, `LOCATION`, `ORGANIZER`, schedule/status fields, and resource URI shape.
3. Try a disposable CalDAV-created event using the same room representation.
4. Read back and verify web UI shows the room as selected/booked.
5. Delete the disposable event.

Exit criteria:

- Exact room representation known.
- Exact mapping for Callbox/Mercury/Sun known or explicitly provided by user.
- Result recorded in a follow-up decision before public CLI behavior ships.

### Slice 2: Internal Model

Files:

- `internal/calendar/ical.go`
- `internal/calendar/ical_test.go`
- `internal/calendar/service.go` if schedule metadata handling is needed

Plan:

- Add a typed model, for example:
  - `Participant{Email, URI, Name, Kind, Role, Partstat, RSVP, ScheduleStatus}`
  - `Room{Alias, Name, URI, Email, Status}`
- Preserve existing `Attendees []string` JSON for compatibility.
- Add additive JSON fields such as `rooms` or `resources`.
- Serialize room resources using the live-verified Yandex shape.
- Parse returned room/resource participants and status.

### Slice 3: Resolver And CLI

Files:

- `internal/cli/calendar.go`
- new internal resolver package only if needed
- `docs/agent-contract.md`
- `README.md`

Plan:

- Add `--room <name>` as repeatable.
- Support aliases case-insensitively: `callbox`, `mercury`, `sun`.
- Resolve aliases to configured/proven resource IDs before building the calendar event.
- Reject unknown room names with a clear error.
- Show resolved room in create/update confirmation preview.
- Ensure `--telemost` and `--room` can coexist: Telemost link stays URL/location only if no physical room is specified; if both are present, do not overwrite the physical room location silently.

### Slice 4: Verification

Required checks:

- `go test ./...`
- Unit tests for ICS room parse/build.
- CLI tests for `--room Sun`, unknown room, confirmation preview, JSON shape.
- Live proof:
  - create event in `Sun`
  - read it back
  - verify room field/status in CLI JSON
  - verify web UI shows room
  - delete event

## Open Questions Raised

1. Are Callbox, Mercury, and Sun represented by email-like resource attendees, CalDAV resource calendars, private Yandex IDs, or only a web UI field?
2. Should v1 use a user-provided room mapping, or should the CLI discover rooms automatically from Yandex 360?
3. Does Yandex room booking through CalDAV PUT require ORGANIZER/SCHEDULE-AGENT/Schedule-Tag behavior beyond the current event creation flow?
4. Should `--room` set `LOCATION`, or should `LOCATION` stay free-form while rooms live only in resource attendee metadata?
5. What should happen when both `--room Sun` and `--telemost` are passed?

## Per-Agent Verbatim Sections

### Architect

```yaml
- severity: HIGH
  category: ical-resource-model
  file: internal/calendar/ical.go
  line: 207
  problem: Calendar serialization only emits attendee emails as bare `ATTENDEE:mailto:<value>`, so a meeting room cannot be represented separately from a human attendee or with room/resource metadata.
  suggested_fix: Replace `Event.Attendees []string` as the only participant model with a typed participant/resource model that can preserve email/URI, display name, and resource kind; serialize rooms as iCalendar resource attendees after a live Yandex CalDAV proof confirms the exact accepted shape.
  requires_human: false
  confidence: high
- severity: HIGH
  category: cli-agent-contract
  file: docs/agent-contract.md
  line: 31
  problem: The agent-facing create contract has no room flag, so an agent hearing "create tomorrow's meeting in Sun" has no stable CLI/API surface to express the room.
  suggested_fix: Add and document a stable `calendar create --room <name>` surface, preferably repeatable, with matching JSON output field such as `rooms`; keep `--location` as free-form text, not booking semantics.
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: module-boundary
  file: internal/cli/calendar.go
  line: 241
  problem: Calendar flags currently bind directly into `calendar.Event`, which would tempt room-name parsing and Callbox/Mercury/Sun mapping to leak into CLI or iCalendar formatting.
  suggested_fix: Keep CLI responsible only for collecting `--room` values, introduce a resolver boundary that maps user-facing room aliases to CalDAV resource identifiers, and pass resolved room resources into `internal/calendar` for serialization.
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: migration-path
  file: .assistant/decisions.md
  line: 109
  problem: The accepted Calendar slice explicitly rejected rooms/resources and org directory lookup as out of scope, so this feature reopens a previously deferred architectural area.
  suggested_fix: Treat room support as a new decision: first ship explicit user/org-provided mapping for Callbox/Mercury/Sun, then only add automatic org directory lookup after confirming scopes, consent, and Yandex room identifiers.
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: authorization-scope
  file: .assistant/open-questions.md
  line: 23
  problem: Org/Directory authorization is still open, and automatic discovery of web-version rooms may require org-wide capabilities beyond the live-verified personal Calendar scope.
  suggested_fix: Do not depend on Directory lookup for v1; require a human-provided room mapping or run a separate research/proof slice for room discovery scopes before promising autocomplete/discovery.
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: tests
  file: internal/calendar/ical_test.go
  line: 22
  problem: The iCalendar tests only cover plain human attendees and do not protect the room/resource representation needed for this feature.
  suggested_fix: Add unit tests for parsing and building resource attendees, CLI tests for `--room Sun` preview/JSON behavior, and an `httptest` CalDAV create/update test that asserts the outgoing PUT body contains the resolved room resource.
  requires_human: false
  confidence: high
```

### Skeptic

```yaml
- severity: HIGH
  category: resource-calendar-ambiguity
  file: .assistant/decisions.md
  line: 109
  problem: "Rooms/resources and org directory lookup were explicitly rejected as out-of-scope for the current personal-account Calendar v1, so treating 'Sun' as a bookable meeting room would overturn an accepted boundary without a new decision."
  suggested_fix: "Block implementation until a human decides whether the requirement means a display-only LOCATION string or an actual resource booking with room availability and invitation semantics."
  requires_human: true
  confidence: high
- severity: HIGH
  category: org-scope-risk
  file: .assistant/open-questions.md
  line: 23
  problem: "Org/Directory capabilities still require confirmation of Yandex 360 organization, admin-enabled service application, and written user consent; room lookup is likely an org/resource-calendar capability rather than a personal CalDAV field."
  suggested_fix: "Require a research/verification slice for Yandex room/resource APIs and scopes before exposing CLI room names to agents."
  requires_human: true
  confidence: medium
- severity: HIGH
  category: mutation-safety
  file: internal/cli/calendar.go
  line: 352
  problem: "The calendar mutation confirmation preview omits Location/URL, so implementing rooms as LOCATION would let a human approve a create/update prompt without seeing the requested room."
  suggested_fix: "Print the resolved room/location/resource identity in create/update previews and update the agent contract before allowing --yes for room-bearing mutations."
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: agent-ux-risk
  file: docs/agent-contract.md
  line: 102
  problem: "The agent approval contract lists title, time range, attendees, deletion target, and Telemost creation, but not room/resource selection; agents could interpret 'make tomorrow meeting in Sun' as approval to pass --yes without confirming the resolved room."
  suggested_fix: "Extend the contract so --yes for calendar create/update requires explicit approval of the exact resolved room/resource, including ambiguity handling for aliases."
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: room-booking-semantics
  file: internal/calendar/ical.go
  line: 207
  problem: "ICS generation writes attendees only as bare ATTENDEE:mailto values and has no resource-specific parameters, so adding room names as attendees may not create a real resource booking."
  suggested_fix: "Do not encode rooms through the existing attendee path until live CalDAV proof shows the exact Yandex resource attendee format needed for reservation and conflict behavior."
  requires_human: false
  confidence: medium
- severity: MEDIUM
  category: overfitting
  file: internal/cli/calendar.go
  line: 241
  problem: "The current CLI has generic event flags only; adding hardcoded Callbox/Mercury/Sun choices here would bake one organization's room names into the binary and make aliases/localization brittle."
  suggested_fix: "Use configurable aliases or discovered room resources, and keep built-in code generic over room IDs/names rather than hardcoding the three current web labels."
  requires_human: false
  confidence: high
- severity: MEDIUM
  category: calendar-selection
  file: internal/calendar/service.go
  line: 179
  problem: "Calendar operations select the first CalDAV calendar collection, which is unsafe if room booking requires writing to a specific shared/resource calendar rather than the user's primary calendar."
  suggested_fix: "Add explicit calendar/resource selection once room semantics are verified; fail closed when multiple matching calendars/resources exist."
  requires_human: false
  confidence: high
- severity: LOW
  category: scope-creep
  file: README.md
  line: 121
  problem: "The documented limitations already include unresolved calendar cleanup/logout/field-clearing issues; room booking adds shared-resource state, conflict handling, and alias management on top of an immature Calendar surface."
  suggested_fix: "Keep the first slice narrow: parse and display --room only after resolving it to a verified resource, with no auto-booking until availability and mutation previews are proven."
  requires_human: true
  confidence: medium
```

### Researcher

```yaml
- severity: HIGH
  category: protocol-prior-art
  file: internal/calendar/ical.go
  line: 207
  problem: "Room booking should not be modeled as a plain attendee string; RFC 5545 defines ATTENDEE parameters including CUTYPE=RESOURCE/ROOM, CN, PARTSTAT, ROLE, and RSVP, while this serializer emits only ATTENDEE:mailto:<value>."
  suggested_fix: "Replace []string attendees with a structured participant/resource model and serialize rooms as ATTENDEE;CN=<name>;CUTYPE=ROOM or CUTYPE=RESOURCE;PARTSTAT=NEEDS-ACTION;ROLE=REQ-PARTICIPANT;RSVP=TRUE:mailto:<room-address>. Verify exact Yandex room calendar addresses for Callbox, Mercury, and Sun before marking done."
  requires_human: true
  confidence: corroborated
- severity: HIGH
  category: product-scope
  file: .assistant/decisions.md
  line: 109
  problem: "The requested feature depends on rooms/resources and org directory lookup, which D-009 explicitly rejected as out of scope for the current personal-account Calendar v1."
  suggested_fix: "Reopen scope with a new decision before implementation: either hardcode a user-provided room-name-to-calendar-address map for v1, or add org directory/resource discovery after live Yandex 360 verification."
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: cli-contract
  file: internal/cli/calendar.go
  line: 247
  problem: "The CLI exposes only --attendee email and --location text; agents cannot express 'meeting in Sun' as a bookable room without abusing location, which prior art treats as different from scheduling a room resource."
  suggested_fix: "Add an explicit repeatable --room flag, keep --location for display-only venue text, and resolve room names through configured aliases or discovered resource identities."
  requires_human: true
  confidence: corroborated
- severity: MEDIUM
  category: parser
  file: internal/calendar/ical.go
  line: 51
  problem: "Read/list parsing drops ATTENDEE parameters, so even if Yandex returns a booked room with CUTYPE=ROOM/RESOURCE, CN, PARTSTAT, or SCHEDULE-STATUS, the CLI will flatten it to an email and lose booking semantics."
  suggested_fix: "Parse iCalendar property parameters in splitICSLine or a structured parser, preserve participant type/name/status, and expose rooms separately from human attendees in Event JSON."
  requires_human: false
  confidence: corroborated
- severity: MEDIUM
  category: caldav-scheduling
  file: internal/calendar/service.go
  line: 93
  problem: "Create uses a raw CalDAV PUT of a VEVENT without ORGANIZER, attendee scheduling metadata, or Schedule-Tag handling; RFC 6638 scheduling examples rely on organizer/attendee semantics for invite delivery and status updates."
  suggested_fix: "Live-test Yandex behavior for room attendees created by PUT; if it requires scheduling semantics, add ORGANIZER, resource ATTENDEE parameters, and read back Schedule-Status/Schedule-Tag after create/update."
  requires_human: true
  confidence: medium
- severity: LOW
  category: agent-contract
  file: docs/agent-contract.md
  line: 47
  problem: "The agent-facing JSON contract lists only flat attendees and location, so downstream agents have no stable field for selected room, room booking status, or room identifier."
  suggested_fix: "After the room model is chosen, extend the contract with additive fields such as rooms/resources while preserving existing attendees for compatibility."
  requires_human: false
  confidence: high
```

### Reviewer

```yaml
- severity: HIGH
  category: decision-consistency
  file: .assistant/decisions.md
  line: 109
  problem: "The requested Callbox/Mercury/Sun room booking feature reopens rooms/resources and org directory lookup, which D-009 explicitly rejected as out of scope for the narrow personal-account v1."
  suggested_fix: "Do not implement room booking as a small calendar flag change until a new human-approved decision supersedes this part of D-009. Run a focused research/live-proof step for Yandex room resources, then append a new D-010-style decision before code ships."
  requires_human: true
  confidence: high
- severity: HIGH
  category: authorization-scope
  file: .assistant/open-questions.md
  line: 23
  problem: "Booking named meeting rooms is likely an org/resource capability, but OQ-INV-1 says org/Directory scopes require a Yandex 360 organization, admin-enabled service application, and written user consent before shipping."
  suggested_fix: "Gate the feature on owner confirmation of the target Yandex 360 org, allowed resource-booking scopes/API surface, and consent posture. If it can be done with personal CalDAV only, record the live proof and close or narrow OQ-INV-1."
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: cli-contract
  file: internal/cli/calendar.go
  line: 246
  problem: "The current CLI only exposes free-text --location for event place, so an agent can write 'Sun' into LOCATION but cannot reliably book the Sun room as a calendar resource."
  suggested_fix: "Add an explicit agent-facing room surface such as --room Callbox|Mercury|Sun after the resource identifiers and booking semantics are verified. Keep --location as display text, not as the resource-booking mechanism."
  requires_human: true
  confidence: high
- severity: MEDIUM
  category: calendar-semantics
  file: internal/calendar/ical.go
  line: 207
  problem: "The iCalendar writer emits attendees only as plain ATTENDEE:mailto values and has no representation for room/resource attendees, so even a new CLI flag would not currently encode a room booking."
  suggested_fix: "After live verification, add the exact CalDAV/iCalendar representation Yandex expects for rooms/resources, including any required ATTENDEE parameters or resource IDs. Cover the emitted ICS with unit tests."
  requires_human: false
  confidence: medium
- severity: MEDIUM
  category: agent-command-contract
  file: docs/agent-contract.md
  line: 31
  problem: "The agent contract documents calendar creation without any room parameter, so agents have no stable instruction for turning 'в Sun' into a safe CLI invocation."
  suggested_fix: "Update the contract with the new room flag, allowed values, JSON output fields, and the rule that --yes requires explicit approval of the exact room as well as title, time range, and attendees."
  requires_human: false
  confidence: high
- severity: LOW
  category: tests
  file: internal/calendar/ical_test.go
  line: 14
  problem: "Calendar tests currently cover basic event ICS with a Telemost URL location and attendee, but not room/resource serialization, validation of supported room names, or agent-contract examples."
  suggested_fix: "Add focused tests for Callbox/Mercury/Sun validation, generated ICS/resource semantics, and any CLI-level rejection of unknown room names before marking the feature complete."
  requires_human: false
  confidence: high
```
