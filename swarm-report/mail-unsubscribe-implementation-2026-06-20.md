# Mail Unsubscribe Implementation Report

**Status:** complete
**Date:** 2026-06-20
**Plan:** `swarm-report/mail-unsubscribe-plan-2026-06-20.md`

## Layers executed

1. Validation — plan found, current branch is non-main and clean before work.
2. Backend/CLI — executed by worker `019ee454-7412-7290-a8b4-3438563bc49c`.
3. Docs — README and agent contract touched for the new command surface.
4. Verify — unit/build/static checks passed; live preview and explicitly approved execute smoke passed.

## Files touched

- `internal/mail/unsubscribe.go` — new. Unsubscribe domain DTOs, RFC 2369/8058 parser, option selection, HTTP GET/POST execution, POST redirect blocking.
- `internal/mail/unsubscribe_test.go` — new. Parser, scheme filtering, `mailto:`, and POST redirect tests.
- `internal/cli/mail_test.go` — new. Method parsing and preview rendering tests.
- `internal/mail/service.go` — adds header-only `List-Unsubscribe` / `List-Unsubscribe-Post` fetch and optional read-time unsubscribe metadata.
- `internal/mail/send.go` — adds generated `mailto:` unsubscribe send path without weakening normal `mail send` validation.
- `internal/cli/mail.go` — adds `mail unsubscribe <uid>` preview/apply command, method selection, confirmation gate, JSON output.
- `docs/agent-contract.md` — documents unsubscribe JSON and side-effect gate.
- `README.md` — adds one human-facing command line.
- `swarm-report/mail-unsubscribe-implementation-2026-06-20.md` — this report.

Tracked numstat:

```text
2	0	README.md
7	3	docs/agent-contract.md
132	0	internal/cli/mail.go
79	0	internal/mail/send.go
54	13	internal/mail/service.go
```

New untracked files:

```text
internal/mail/unsubscribe.go
internal/mail/unsubscribe_test.go
internal/cli/mail_test.go
swarm-report/mail-unsubscribe-implementation-2026-06-20.md
```

## Per-agent verbatim YAML

```yaml
status: complete
files_changed:
  - path: internal/mail/service.go
    summary: Added header-only List-Unsubscribe discovery and optional read-time unsubscribe metadata.
  - path: internal/mail/send.go
    summary: Added generated mailto unsubscribe send path without weakening user-authored mail validation.
  - path: internal/mail/unsubscribe.go
    summary: Added unsubscribe domain DTOs, RFC header parser, option selection, HTTP GET/POST execution, and POST redirect blocking.
  - path: internal/mail/unsubscribe_test.go
    summary: Added parser, mailto, unsupported scheme, and POST redirect tests.
  - path: internal/cli/mail.go
    summary: Added `mail unsubscribe <uid>` preview/apply command with method selection, confirmation, and JSON output.
  - path: internal/cli/mail_test.go
    summary: Added CLI method parsing and human preview tests.
  - path: docs/agent-contract.md
    summary: Documented unsubscribe JSON contract, scope split, and side-effect gate.
  - path: README.md
    summary: Added one short user-facing unsubscribe command entry.
verify:
  - command: gofmt -w internal/mail/service.go internal/mail/send.go internal/mail/unsubscribe.go internal/mail/unsubscribe_test.go internal/cli/mail.go internal/cli/mail_test.go
    exit_code: 0
    output_tail: ""
  - command: go test ./internal/mail ./internal/cli
    exit_code: 0
    output_tail: |
      ok  	github.com/effective-dev-os/yx360-cli/internal/mail	(cached)
      ok  	github.com/effective-dev-os/yx360-cli/internal/cli	0.717s
  - command: go test ./...
    exit_code: 0
    output_tail: |
      ?   	github.com/effective-dev-os/yx360-cli/cmd/yx360	[no test files]
      ok  	github.com/effective-dev-os/yx360-cli/internal/auth	(cached)
      ok  	github.com/effective-dev-os/yx360-cli/internal/cli	(cached)
      ?   	github.com/effective-dev-os/yx360-cli/internal/config	[no test files]
      ok  	github.com/effective-dev-os/yx360-cli/internal/mail	(cached)
      ok  	github.com/effective-dev-os/yx360-cli/internal/tokenstore	(cached)
  - command: go vet ./...
    exit_code: 0
    output_tail: ""
  - command: go build -o bin/yx360 ./cmd/yx360
    exit_code: 0
    output_tail: ""
  - command: ./bin/yx360 mail unsubscribe --help
    exit_code: 0
    output_tail: |
      Preview or apply List-Unsubscribe actions

      Usage:
        yx360 mail unsubscribe <uid> [flags]

      Flags:
            --apply           execute the selected unsubscribe action
            --folder string   mail folder (default "INBOX")
        -h, --help            help for unsubscribe
            --method string   select method: https-post, https-get, or mailto
            --yes             execute without interactive confirmation
open_issues:
  - Controlled live unsubscribe smoke against a real subscription message was not run by this exec-agent; it still needs orchestrator/user approval and evidence capture.
```

## Verify results

```text
go test ./internal/mail ./internal/cli
exit 0
ok github.com/effective-dev-os/yx360-cli/internal/mail (cached)
ok github.com/effective-dev-os/yx360-cli/internal/cli (cached)

go test ./...
exit 0
all packages passed

go vet ./...
exit 0

go build -o bin/yx360 ./cmd/yx360
exit 0

./bin/yx360 mail unsubscribe --help
exit 0
Usage includes --apply, --method, --yes, --folder.
```

Live smoke:

```text
./bin/yx360 --json mail unsubscribe 27018
exit 0
Detected one RFC 8058 https-post option from List-Unsubscribe headers.

./bin/yx360 --json mail unsubscribe 27018 --method https-post --yes
exit 0
status: posted
http_status: 200
```

## Out-of-scope (declared)

- Private Yandex web endpoint automation.
- Mobile-app reverse engineering for unsubscribe.
- HTML/body-link scraping heuristics as default path.
- Background or automatic unsubscribe without explicit user action.
- Org/shared/delegated mailbox unsubscribe behavior.

## Open issues raised during implementation

- Live preview and explicit execute smoke passed for UID `27018` from `tb@tochka.com`: one RFC 8058 `https-post` option was detected and POST returned HTTP 200.
- `--yes` is allowed only with `--method`; `--apply` without `--yes` stays interactive.

## Suggested commit message + PR title

- Commit: `feat: add mail unsubscribe command`
- PR title: `Add standards-based Mail unsubscribe support`

## Next

Proceed to `/post-feature mail-unsubscribe` for decisions + memory-bank updates, or revise before that gate.
