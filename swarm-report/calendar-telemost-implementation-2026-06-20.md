# Calendar + Telemost Implementation

Status: complete
Date: 2026-06-20
Plan: `swarm-report/calendar-telemost-plan-2026-06-20.md`

## Layers Executed

1. Auth/profile layer: added separate credential profiles for Mail and Calendar/Telemost.
2. Backend layer: added CalDAV Calendar service and Telemost REST service.
3. CLI layer: added `calendar` and `telemost` commands with confirmation gates.
4. Verification layer: ran unit checks, build/vet, and live Calendar/Telemost smoke.

## Files Touched

Calendar/Telemost implementation:

- `internal/tokenstore/keyring.go` - profile-aware keychain keys.
- `internal/tokenstore/file.go` - profile-aware file-store paths.
- `internal/tokenstore/file_test.go` - profile path regression test.
- `internal/config/config.go` - Calendar/Telemost config and scopes.
- `internal/cli/login.go` - `--calendar`, `--telemost`, and profile selection.
- `internal/cli/root.go` - registers `calendar` and `telemost` commands.
- `internal/cli/mail.go` - Mail now loads the `mail` credential profile.
- `internal/cli/calendar.go` - Calendar and Telemost CLI commands.
- `internal/cli/mail_test.go` - CLI helper tests.
- `internal/calendar/ical.go` - minimal VEVENT parse/generate.
- `internal/calendar/service.go` - CalDAV discovery/list/read/create/update/delete.
- `internal/calendar/ical_test.go` - iCalendar unit tests.
- `internal/telemost/service.go` - official Telemost create endpoint client.
- `README.md` - human-facing Calendar/Telemost setup and examples.
- `swarm-report/calendar-telemost-plan-2026-06-20.md` - proof status updated and live identifiers redacted.

Pre-existing dirty files from the Mail unsubscribe work were present before this implementation and are not part of this feature report: `docs/agent-contract.md`, `internal/mail/send.go`, `internal/mail/service.go`, `internal/mail/unsubscribe.go`, `internal/mail/unsubscribe_test.go`, `internal/cli/mail_test.go`, and `swarm-report/mail-unsubscribe-implementation-2026-06-20.md`. Note that `internal/cli/mail.go` is shared: this implementation only changed its credential profile loading.

## Per-Agent YAML

```yaml
- agent: orchestrator
  layer: auth-profile
  status: complete
  files:
    - internal/tokenstore/keyring.go
    - internal/tokenstore/file.go
    - internal/config/config.go
    - internal/cli/login.go
    - internal/cli/mail.go
  output: Added separate token profiles so Mail and Calendar/Telemost no longer overwrite each other in keychain/file-store.
  verify:
    - command: go test ./...
      exit_code: 0
    - command: go vet ./...
      exit_code: 0

- agent: orchestrator
  layer: backend
  status: complete
  files:
    - internal/calendar/ical.go
    - internal/calendar/service.go
    - internal/calendar/ical_test.go
    - internal/telemost/service.go
  output: Added CalDAV Calendar CRUD with OAuth auth scheme and Telemost conference creation client.
  verify:
    - command: go test ./...
      exit_code: 0
    - command: live Calendar list/create/read/update/delete smoke
      exit_code: 0

- agent: orchestrator
  layer: cli
  status: complete
  files:
    - internal/cli/calendar.go
    - internal/cli/root.go
    - README.md
  output: Added calendar list/read/create/update/delete, telemost create, and calendar create --telemost commands with preview/confirmation gates.
  verify:
    - command: go build -o bin/yx360 ./cmd/yx360
      exit_code: 0
```

## Verify Results

Static/unit:

- `gofmt -w ...` - exit 0.
- `go test ./...` - exit 0.
- `go vet ./...` - exit 0.
- `go build -o bin/yx360 ./cmd/yx360` - exit 0.

Live smoke:

- `YX360_CALENDAR_CLIENT_ID=<redacted> ./bin/yx360 login --calendar --telemost` - exit 0; stored token in `calendar-telemost` profile.
- `./bin/yx360 --json calendar list --from 2026-06-20 --to 2026-06-21` - exit 0; returned live Calendar event data.
- `./bin/yx360 --json calendar create ... --telemost --yes` - exit 0; created test event and attached Telemost link.
- `./bin/yx360 --json calendar read <test-event-href>` - exit 0; read back created event.
- `./bin/yx360 --json calendar update <test-event-href> ... --yes` - exit 0; updated title/description and changed ETag.
- `./bin/yx360 --json calendar delete <test-event-href> --yes` - exit 0; deleted test event.
- `./bin/yx360 --json calendar read <test-event-href>` after delete - exit 1 with `calendar: read failed: HTTP 404`; cleanup verified.

The Telemost link created during smoke may remain live because the official API delete/cancel endpoint is still unverified.

## Out Of Scope

- Recurring event edits.
- Shared/delegated calendars.
- Room/resource booking.
- Organization directory lookup or autocomplete.
- Background sync.
- Private Yandex web/mobile endpoints.
- Telemost conference deletion/cancellation.
- App-password Calendar auth.

## Open Issues

- Token storage now supports profiles, but `logout` still clears only the default profile. A follow-up should add profile-aware logout.
- Calendar update cannot intentionally clear a string field to empty yet; empty flag values are treated as "not provided".
- Calendar commands accept event hrefs as IDs. This is reliable but not pretty; a later UX pass can add short aliases.
- Mail and Calendar/Telemost require separate OAuth apps. README documents this; `/post-feature` should turn it into a decision entry.

## Suggested Commit

Commit message:

```text
feat: add calendar and telemost support
```

PR title:

```text
Add Calendar CalDAV CRUD and Telemost link creation
```

## Next

Run `/post-feature calendar-telemost` to record decisions and update memory-bank docs, then commit after reviewing the mixed worktree with the Mail unsubscribe changes.
