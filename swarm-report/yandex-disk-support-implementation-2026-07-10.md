# Implementation — Yandex Disk Support

**Slug:** `yandex-disk-support`
**Date:** 2026-07-10
**Plan:** `swarm-report/yandex-disk-support-plan-2026-07-10.md`
**Branch:** `feat/yandex-disk-support` (off `main`)
**Status:** `complete` (build/vet/unit/smoke verified; live end-to-end blocked on B-1 scope confirmation — ANTI-11 caveat)

---

## Layers executed

1. **infra/netutil** — extract shared `ipv4Client()` to `internal/netutil`; update 4 call sites.
2. **config** — add `Disk` struct, `DiskReadScope`, `DiskWriteScope`, `DiskClientID()`, `DefaultDisk()`.
3. **backend** — `internal/disk/types.go` + `internal/disk/service.go` (List/Get/Put/Share/Unshare/Remove/Mkdir).
4. **cli** — `internal/cli/disk.go` (7 subcommands); `internal/cli/login.go` (`--disk` flag + `diskProfile`); `internal/cli/root.go` (`newDiskCmd()`).
5. **verify** — orchestrator.

---

## Files touched

### New
- `internal/netutil/client.go` — `IPv4Client()` extracted from 3 duplicate packages (D-006, C-8)
- `internal/disk/types.go` — `Resource`, `ResourceList`, `Link`, `OperationStatus`, `resourceResponse`
- `internal/disk/service.go` — `Service` with `List`, `Get`, `Put`, `Share`, `Unshare`, `Remove`, `Mkdir`; `ErrReauthRequired`, `ErrConflict`
- `internal/cli/disk.go` — `diskCmd` with 7 subcommands: `list`, `get`, `put`, `share`, `unshare`, `rm`, `mkdir`

### Modified
- `internal/auth/http_client.go` — `httpContext()` now uses `netutil.IPv4Client()` (removed `net`/`net/http`/`time` imports)
- `internal/forms/service.go` — `NewService` uses `netutil.IPv4Client()`; removed local `ipv4Client()` + `net`/`time` imports
- `internal/telemost/service.go` — same as forms
- `internal/calendar/service.go` — same as forms
- `internal/config/config.go` — added `Disk` struct, scope constants, `DiskClientID()`, `DefaultDisk()`
- `internal/cli/login.go` — added `diskProfile` const, `diskScope` flag, `selectedApps` counter branch, profile/clientID/scopes block, `--disk` flag; updated error message
- `internal/cli/root.go` — `root.AddCommand(newDiskCmd())`

---

## Design decisions (from plan + implementation)

- **REST only, not WebDAV** — `cloud-api.yandex.net/v1/disk/`; auth header `Authorization: OAuth <token>` (same as Calendar)
- **`disk:` scheme** — `diskPath()` helper auto-prepends; CLI accepts plain POSIX paths
- **`--json` not re-declared** — inherited persistent root flag (cobra panic prevention, N-6)
- **Upload two-step** — get upload URL → PUT file with `Content-Length` via `io.Copy` (no buffering, upload URL 30-min TTL)
- **Download two-step** — get `href` → follow redirect; path-traversal protection via `filepath.Base` + prefix check
- **`--yes` gates** — `disk share`, `disk put` (on 409 ErrConflict), `disk rm`; without `--yes` prints preview + exits 0 (dry-run style)
- **`disk rm`** — default trash (`permanently=false`); `--permanent` for hard delete; async 202 → poll up to 5×1s
- **`disk get` on directory** — service returns typed error; recursive download is v2
- **Error codes** — 413 (too large) and 507 (quota) map to human-readable messages
- **`disk unshare`** — included in v1 (researcher confirmed `PUT /v1/disk/resources/unpublish`)
- **netutil refactor** — eliminated 3-way `ipv4Client()` duplication across `forms`, `telemost`, `calendar`; `auth/http_client.go` also consolidated

---

## Verify results

| cmd | exit | tail |
|-----|------|------|
| `go build -o bin/yx360 ./cmd/yx360` | 0 | BUILD_OK |
| `go vet ./...` | 0 | VET_OK |
| `go test ./...` | 0 | all ok (auth/calendar/cli/forms/mail/tokenstore) |
| `disk --help` | 0 | 7 subcommands listed ✓ |
| `disk share /test/file.txt` (no `--yes`) | 0 | preview printed, gate held ✓ |
| `disk rm /test/file.txt` (no `--yes`) | 0 | preview printed, gate held ✓ |
| `YX360_DISK_CLIENT_ID=… login --disk --no-browser` | 0 | OAuth device flow started → `invalid_scope` (B-1 live) |

**Live `invalid_scope`** — confirms the OAuth flow reaches Yandex with correct clientID; scope strings `cloud_api:disk.read`/`cloud_api:disk.write` rejected because B-1 (register disk scopes in app UI) is not yet done. Feature is build/vet/unit/gate-smoke verified; not "done" per ANTI-11 until B-1 is resolved and a live `login --disk → disk list` passes.

---

## Out-of-scope (carried from plan)

- Chunked/resumable upload (OQ-019)
- `disk get` on directory path — typed error in v1; recursive `disk pull` is v2 (OQ-020)
- `disk move` / `disk copy`
- Trash management (empty/restore)
- WebDAV transport
- Docs (`docs/agent-contract.md`, `README.md`) — deferred; run `/post-feature` to include

---

## Open issues raised during implementation

1. **B-1 live** — `invalid_scope` confirms scope strings `cloud_api:disk.read`/`cloud_api:disk.write` need to be registered in the Yandex OAuth app UI. Owner: open the app at oauth.yandex.ru, add disk access checkboxes, verify the exact scope strings shown, re-test `login --disk`.
2. **`disk/service_test.go` missing** — `internal/disk` has no test files. Unit tests for `diskPath()`, gate logic (ErrConflict), and http error mapping should be added in a follow-up.
3. **`netutil` has no tests** — `IPv4Client()` is trivial but untested. Low priority; add if test coverage gate is introduced.

---

## Suggested commit message

```
feat(disk): add Yandex Disk support (list/get/put/share/unshare/rm/mkdir)

Add yx360 disk subcommand tree backed by the Yandex Disk REST API
(cloud-api.yandex.net/v1/disk, Authorization: OAuth). Includes:
- disk list [--path /] [--limit N] [--offset N]
- disk get <remote-path> [--out dir]   (path-traversal-safe download)
- disk put <local-file> --to <path>    (two-step upload, --yes for overwrite)
- disk share <path> [--yes]            (public link gate)
- disk unshare <path>                  (revoke public link)
- disk rm <path> [--yes] [--permanent] (trash-by-default, async poll)
- disk mkdir <path>                    (non-recursive)

Also extracts shared ipv4Client() to internal/netutil, eliminating 3-way
duplication across forms/telemost/calendar (D-006). Requires separate
yx360 login --disk (YX360_DISK_CLIENT_ID) per D-009 credential profile
pattern. Live end-to-end blocked on B-1 (register disk scopes in OAuth app).

Plan: swarm-report/yandex-disk-support-plan-2026-07-10.md
```

**PR title:** `feat(disk): Yandex Disk support (list/get/put/share/rm/mkdir + netutil refactor)`

---

## Next

- **Human B-1** — open oauth.yandex.ru → Disk app → add disk access scopes → confirm exact scope strings → re-test `login --disk → disk list`
- `/post-feature yandex-disk-support` — D-NNN в decisions.md, memory bank updates
- Follow-up: `internal/disk/service_test.go` unit tests
