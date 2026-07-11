# Implementation: `dry-run`

**Status:** complete
**Date:** 2026-07-11

---

## Layers executed

1. **infra** — `root.go` + `output.go` (dryRun global, persistent flag, isDryRun(), emitDryRun())
2. **backend** — `disk.go` × 5 guards, `mail.go` × 1 guard, `calendar.go` × 4 guards

---

## Files touched

| File | Change |
|------|--------|
| `internal/cli/root.go` | +`dryRun bool` global; +`--dry-run` PersistentFlag |
| `internal/cli/output.go` | +`isDryRun()`, +`emitDryRun()` (human + JSON modes) |
| `internal/cli/disk.go` | +guard in put/share/unshare/rm/mkdir (5 commands) |
| `internal/cli/mail.go` | +guard in mail send |
| `internal/cli/calendar.go` | +guard in calendar create/update/delete + telemost create (4 commands) |

---

## Design decisions applied

- **B-1:** exit 0 (like gogcli), signal via stdout `[dry-run]` + JSON `{"dry_run":"true","would":"..."}`
- **B-2:** `isDryRun()` checked BEFORE any `!yes` check → dry-run always wins silently
- **B-3:** `login`/`logout` — silent no-op (no guard, flag is a persistent no-op there; consistent with gogcli `auth.credentials` approach)
- **calendar create:** guard placed AFTER title validation but BEFORE Telemost API call and CalDAV call
- **calendar delete:** guard placed BEFORE `svc.Read()` (avoids unnecessary network call)

---

## Verify results

| Command | Exit | Result |
|---------|------|--------|
| `go build ./...` | 0 | BUILD_OK |
| `go vet ./...` | 0 | VET_OK |
| `go test ./...` | 0 | all ok (`internal/cli` 0.682s) |
| `disk share /test.txt --dry-run` | 0 | `[dry-run] would make disk:/test.txt publicly accessible` |
| `disk rm /test.txt --dry-run` | 0 | `[dry-run] would move to Trash disk:/test.txt` |
| `disk rm /test.txt --dry-run --yes` | 0 | dry-run wins; `[dry-run] would move to Trash disk:/test.txt` |
| `disk share /test.txt --dry-run --json` | 0 | `{"dry_run":"true","would":"would make disk:/test.txt publicly accessible"}` |

---

## Out-of-scope (declared)

- `forms create/publish/unpublish` — separate PR
- `disk get` — read-only, no server mutation
- `login`/`logout` — undefined semantics, silent no-op
- `disk move/copy` — commands don't exist

---

## Suggested commit

```
feat(dry-run): add --dry-run flag to all mutating commands

Adds a persistent root-level --dry-run flag (parallel to --json) that
prints what would happen without executing. Overrides --yes when both
are supplied.

Scope: disk put/share/unshare/rm/mkdir, mail send,
calendar create/update/delete, telemost create.

Design follows gogcli pattern: exit 0, [dry-run] prefix in human
mode, {"dry_run":"true","would":"..."} in JSON mode.

Refs: swarm-report/dry-run-plan-2026-07-10.md
      swarm-report/dry-run-implementation-2026-07-11.md
```

**PR title:** `feat(dry-run): add --dry-run flag to all mutating commands`

---

## Next

→ `/post-feature dry-run`
