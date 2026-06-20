# Implementation Report - yx360 Mail Read-Only

**Status:** complete
**Date:** 2026-06-20
**Plan:** `swarm-report/mail-inbox-search-attachments-send-plan-2026-06-20.md`

## Layers Executed

1. Auth/config: added Mail read scope support and IMAP endpoint config.
2. Backend: added `internal/mail` IMAP read-only service.
3. CLI: added `mail` parent command with read-only subcommands.
4. Live smoke: OAuth login, IMAP list/search/read, and attachment download passed.
5. Verify: static gate passed after live fixes.

## Files Touched

- `go.mod`, `go.sum` - added IMAP/SASL/MIME dependencies.
- `internal/config/config.go` - added `MailReadScope`, default IMAP host/port config, `YX360_IMAP_HOST` override.
- `internal/auth/credential.go` - added scope parsing and required-scope checks.
- `internal/auth/credential_test.go` - added scope helper coverage.
- `internal/auth/http_client.go` - added IPv4 OAuth HTTP transport for this environment's broken IPv6 route to Yandex.
- `internal/auth/flow_loopback.go`, `internal/auth/flow_device.go`, `internal/auth/oauth.go` - wired OAuth requests through the IPv4-capable HTTP client.
- `internal/cli/login.go` - added `login --mail`; login output now reports granted credential scopes.
- `internal/cli/root.go` - registered `mail`.
- `internal/cli/mail.go` - added `mail list`, `mail search`, `mail read`, `mail attachment`.
- `internal/mail/service.go` - added read-only IMAP service, bounded search/read, MIME body parsing, attachment manifest/download flow, and IPv4 IMAP TLS dial.
- `internal/mail/xoauth2.go` - added XOAUTH2 SASL client.
- `internal/mail/file.go` - added 0600 attachment file creation.
- `internal/mail/service_test.go` - added attachment filename sanitization coverage.

## Per-Agent YAML

```yaml
- agent: implementor
  layer: auth-config-backend-cli
  status: complete
  changed_files:
    - go.mod
    - go.sum
    - internal/config/config.go
    - internal/auth/credential.go
    - internal/auth/credential_test.go
    - internal/auth/http_client.go
    - internal/auth/flow_loopback.go
    - internal/auth/flow_device.go
    - internal/auth/oauth.go
    - internal/cli/login.go
    - internal/cli/root.go
    - internal/cli/mail.go
    - internal/mail/service.go
    - internal/mail/xoauth2.go
    - internal/mail/file.go
    - internal/mail/service_test.go
  verify:
    - command: go test ./...
      exit: 0
    - command: go vet ./...
      exit: 0
    - command: go build -o bin/yx360 ./cmd/yx360
      exit: 0
    - command: ./bin/yx360 mail --help
      exit: 0
    - command: ./bin/yx360 login --help
      exit: 0
    - command: ./bin/yx360 login --mail
      exit: 0
    - command: ./bin/yx360 mail list --limit 5
      exit: 0
    - command: ./bin/yx360 mail search --from noreply@tm.openai.com --limit 2
      exit: 0
    - command: ./bin/yx360 mail read 27035
      exit: 0
    - command: ./bin/yx360 mail attachment 26943 2 --out /tmp/yx360-mail
      exit: 0
  notes:
    - mail send intentionally not implemented.
    - Yandex endpoints were reachable over IPv4; Go's default IPv6 route failed in this environment.
```

## Verify Results

- `go test ./...` - passed.
- `go vet ./...` - passed.
- `go build -o bin/yx360 ./cmd/yx360` - passed.
- `./bin/yx360 mail --help` - passed; read-only commands visible.
- `./bin/yx360 login --help` - passed; `--mail` visible.
- `./bin/yx360 login --mail` - passed; token stored, account label returned.
- `./bin/yx360 mail list --limit 5` - passed.
- `./bin/yx360 mail search --from noreply@tm.openai.com --limit 2` - passed.
- `./bin/yx360 mail read 27035` - passed.
- `./bin/yx360 mail attachment 26943 2 --out /tmp/yx360-mail` - passed.

## Out-of-Scope (Declared)

- `mail send` / SMTP.
- Organization-wide, delegated, or shared mailbox access.
- Secret-based token refresh.
- Silent attachment opening or overwrite.

## Open Issues

- None for read-only Mail v1.
- Yandex's accepted SASL mechanism is not publicly documented. Implementation tries XOAUTH2 first, then OAUTHBEARER.

## Suggested Commit Message

`Add read-only Mail IMAP commands`

## Suggested PR Title

`Add read-only Yandex 360 Mail support`

## Next

Proceed to `/post-feature mail-inbox-search-attachments-send`. Keep SMTP/send as a separate pre-feature/implementor slice.
