# Calendar Room Booking Implementation

Status: complete
Slug: calendar-room-booking
Date: 2026-06-20

Plan: `swarm-report/calendar-room-booking-plan-2026-06-20.md`

## Layers Executed

1. model/backend
   - Added structured Calendar participants, organizer, rooms, and resources.
   - Added local room registry.
   - Added room discovery from parsed Calendar events.
2. CLI
   - Added `calendar rooms list/add/discover`.
   - Added `calendar create --room` and `calendar update --room`.
   - Updated mutation preview to show rooms, location, and URL.
3. docs
   - Updated README and agent command contract.
4. verification
   - Ran unit tests, build, local room-registry smoke, live room booking smoke, and cleanup check.

## Files Touched

- `README.md`
- `docs/agent-contract.md`
- `internal/calendar/ical.go`
- `internal/calendar/ical_test.go`
- `internal/calendar/rooms.go`
- `internal/calendar/rooms_test.go`
- `internal/calendar/service.go`
- `internal/cli/calendar.go`
- `internal/cli/calendar_test.go`
- `swarm-report/calendar-room-booking-plan-2026-06-20.md`

Unrelated dirty file left untouched:

- `swarm-report/yandex-forms-get-create-publish-plan-2026-06-20.md`

## Implementation Notes

- Existing JSON fields remain compatible: `attendees`, `location`, `url`, `starts_at`, `ends_at`, etc. stay in place.
- Added additive JSON fields: `organizer`, `participants`, `rooms`, and `resources`.
- Room booking uses Yandex-proven iCalendar shape:

```ics
ORGANIZER;CN=<account>:mailto:<account>
ATTENDEE;CUTYPE=ROOM;PARTSTAT=ACCEPTED;CN=<room>;ROLE=REQ-PARTICIPANT:mailto:<room-address>
```

- `--location` remains display-only. When `--room` is passed and `--location` is omitted, the CLI sets `LOCATION` to the room name for web UI readability while still booking through `ATTENDEE;CUTYPE=ROOM`.
- `--telemost` and `--room` can coexist. With a room present, Telemost updates `url` and description, but does not replace the physical room location.
- Room registry is stored under the user config dir as `yx360/calendar-rooms.json` with mode `0600`.
- `YX360_CONFIG_HOME` can override the config root for agents, CI, and sandboxed runs.
- `calendar rooms discover` is opportunistic: it finds rooms already present in scanned Calendar events and cannot discover rooms that never appear in that range.

## Per-Agent Verbatim YAML

```yaml
status: complete_local_verification
files_changed:
  - README.md
  - docs/agent-contract.md
  - internal/calendar/ical.go
  - internal/calendar/rooms.go
  - internal/calendar/service.go
  - internal/calendar/ical_test.go
  - internal/calendar/rooms_test.go
  - internal/cli/calendar.go
  - internal/cli/calendar_test.go
untouched:
  - swarm-report/yandex-forms-get-create-publish-plan-2026-06-20.md
verify:
  - command: gofmt -w internal/calendar/ical.go internal/calendar/rooms.go internal/calendar/service.go internal/calendar/ical_test.go internal/calendar/rooms_test.go internal/cli/calendar.go internal/cli/calendar_test.go
    exit_code: 0
    output: ""
  - command: go test ./...
    exit_code: 0
    output: |
      ?    github.com/effective-dev-os/yx360-cli/cmd/yx360 [no test files]
      ok   github.com/effective-dev-os/yx360-cli/internal/auth (cached)
      ok   github.com/effective-dev-os/yx360-cli/internal/calendar 0.772s
      ok   github.com/effective-dev-os/yx360-cli/internal/cli 1.552s
      ?    github.com/effective-dev-os/yx360-cli/internal/config [no test files]
      ok   github.com/effective-dev-os/yx360-cli/internal/mail (cached)
      ?    github.com/effective-dev-os/yx360-cli/internal/telemost [no test files]
      ok   github.com/effective-dev-os/yx360-cli/internal/tokenstore (cached)
live_verification:
  status: not_run
  reason: not requested in final verification commands; no Yandex Calendar mutation was performed
summary:
  - Added structured participant, room, and resource JSON fields while preserving existing event fields.
  - Added ATTENDEE parameter parsing and room/resource ICS serialization with CUTYPE, CN, ROLE, PARTSTAT, RSVP, and SCHEDULE-STATUS support.
  - Added local user config room registry with 0600 JSON persistence, case-insensitive resolution, manual add/override, and discovery from parsed events.
  - Added calendar rooms list/add/discover plus calendar create/update --room.
  - Kept --location display-only and avoided overwriting physical room/location with Telemost URL when --room and --telemost coexist.
```

## Verify Results

- `gofmt -w internal/cli/calendar.go internal/cli/calendar_test.go internal/calendar/ical.go internal/calendar/ical_test.go internal/calendar/rooms.go internal/calendar/rooms_test.go internal/calendar/service.go`
  - exit 0
- `go test ./...`
  - exit 0
- `go build -o ./bin/yx360 ./cmd/yx360`
  - exit 0
- Local registry smoke:
  - command: `YX360_CONFIG_HOME=<tmp> ./bin/yx360 --json calendar rooms add Sun sun@effective.band`
  - exit 0
  - returned `Sun -> sun@effective.band`
  - command: `YX360_CONFIG_HOME=<tmp> ./bin/yx360 --json calendar rooms list`
  - exit 0
  - returned `Sun -> sun@effective.band`
- Live discovery smoke:
  - command: `./bin/yx360 --json calendar rooms discover --from 2026-06-22 --to 2026-06-24`
  - exit 0
  - discovered `Sun -> sun@effective.band`
- Live room booking smoke:
  - command: `./bin/yx360 --json calendar create --title "yx360 room smoke Sun" --starts-at 2026-07-04T17:00:00+06:00 --ends-at 2026-07-04T17:15:00+06:00 --room Sun --yes`
  - exit 0
  - returned `rooms[0].name = Sun`, `rooms[0].email = sun@effective.band`, `rooms[0].status = ACCEPTED`
- Live cleanup:
  - command: `./bin/yx360 --json calendar delete <smoke-event-href> --yes`
  - exit 0
  - command: `./bin/yx360 --json calendar list --from 2026-07-04 --to 2026-07-05`
  - exit 0
  - returned `[]`

## Out-of-Scope Declared

- No organization Directory lookup.
- No admin/org-wide capabilities.
- No guarantee that auto-discovery returns rooms that never appeared in scanned events.
- No full availability search across all rooms.
- No recurring room booking.
- No automatic conflict resolution beyond Yandex-returned booking status.

## Open Issues Raised

- `calendar rooms discover` currently depends on events returned by the selected primary calendar. If some rooms only appear in shared calendars outside this collection, v1 will not discover them automatically.
- Only `Sun` was live-verified end-to-end. `Callbox` and `Mercury` need discovery from existing events or manual `rooms add` mappings.
- Running `yx360` through `/bin/zsh -lc` in this desktop environment intermittently made the macOS keychain credential unavailable, while direct command execution worked. This appears environment-specific and did not affect direct CLI usage.

## Suggested Commit Message

```text
feat: add calendar room booking
```

## Suggested PR Title

```text
Add Yandex Calendar room booking support
```

## Next

Run `/post-feature calendar-room-booking` to record the room-booking decision and update memory-bank docs, then commit after reviewing the unrelated untracked forms plan.
