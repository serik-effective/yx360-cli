# Implementation Report - yx360 Mail Send

**Status:** complete
**Date:** 2026-06-20
**Requested:** `implementor send mail`
**Related plan:** `swarm-report/mail-inbox-search-attachments-send-plan-2026-06-20.md`

## Blocker Resolution

- Initial `/implementor send mail` blocked because SMTP/send scope was unresolved.
- Owner found `mail:smtp` in Yandex OAuth app UI.
- D-007 recorded `mail:smtp` as the send-side scope and closed OQ-007.

## Layers Executed

1. Auth/config: added Mail SMTP host/port config and send scope.
2. CLI: added `login --mail-send` and `mail send`.
3. Backend: added MIME message building and SMTP TLS send via OAuth.
4. Verify: unit/static checks plus live self-send smoke.

## Files Touched

- `internal/config/config.go` - added SMTP host/port and `MailSendScope`.
- `internal/cli/login.go` - added `--mail-send`.
- `internal/cli/mail.go` - added `mail send`, preview, confirmation, `--yes`, body/body-file, recipients, and attachments.
- `internal/mail/send.go` - added MIME builder, SMTP TLS `tcp4` client, XOAUTH2/OAUTHBEARER auth, attachment handling, and send result DTO.
- `internal/mail/send_test.go` - covered Bcc omission from MIME headers and Bcc inclusion in SMTP recipients.

## Per-Agent YAML

```yaml
- agent: implementor
  layer: auth-config-backend-cli
  status: complete
  changed_files:
    - internal/config/config.go
    - internal/cli/login.go
    - internal/cli/mail.go
    - internal/mail/send.go
    - internal/mail/send_test.go
  verify:
    - command: go test ./...
      exit: 0
    - command: go vet ./...
      exit: 0
    - command: go build -o bin/yx360 ./cmd/yx360
      exit: 0
    - command: ./bin/yx360 mail send --help
      exit: 0
    - command: ./bin/yx360 login --mail --mail-send
      exit: 0
    - command: ./bin/yx360 mail send --to serik.beysenov@effective.band --subject "yx360 mail send smoke 2026-06-20" --body "yx360 SMTP smoke test" --yes
      exit: 0
    - command: ./bin/yx360 mail list --limit 10
      exit: 0
    - command: ./bin/yx360 mail read 27040
      exit: 0
  notes:
    - Send defaults to preview plus confirmation.
    - --yes is explicit and was used only for the self-send smoke.
    - Bcc is sent as SMTP recipient but omitted from MIME headers.
```

## Verify Results

- `go test ./...` - passed.
- `go vet ./...` - passed.
- `go build -o bin/yx360 ./cmd/yx360` - passed.
- `./bin/yx360 mail send --help` - passed.
- `./bin/yx360 login --mail --mail-send` - passed.
- `./bin/yx360 mail send --to serik.beysenov@effective.band --subject "yx360 mail send smoke 2026-06-20" --body "yx360 SMTP smoke test" --yes` - passed.
- `./bin/yx360 mail list --limit 10` - passed; self-send appeared as UID 27040.
- `./bin/yx360 mail read 27040` - passed.

## Out-of-Scope

- Bulk mail / automation.
- HTML authoring.
- Reply/thread-aware send.
- App-password auth.
- Organization-wide, shared, or delegated mailboxes.

## Open Issues

- One IMAP search attempt with combined `--from` + `--subject` hit a transient Yandex backend `NO [UNAVAILABLE] UID SEARCH Backend error`; `mail list` and `mail read` verified the sent message. If search instability repeats, run a separate `/diagnose mail search`.

## Suggested Commit Message

`Add SMTP mail send support`

## Suggested PR Title

`Add Yandex 360 Mail send command`

## Next

Proceed to `/post-feature mail-inbox-search-attachments-send` for memory-bank updates and closeout.
