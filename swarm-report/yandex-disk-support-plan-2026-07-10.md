# Pre-Feature Consilium — Yandex Disk Support

**Slug:** `yandex-disk-support`
**Date:** 2026-07-10
**Status:** consilium-complete
**Proposal (verbatim):** "yandex disk support — yx360 disk list [--path /] [--json], disk get <path> --out <dir>, disk put <file> --to <path>, disk share <path>, disk rm <path> [--yes], disk mkdir <path>"
**Consilium:** architect + skeptic + researcher + reviewer (4/4 strict YAML).

---

## TL;DR

- **Counts:** 4 Blockers (HIGH, requires_human) · 10 Concerns (MEDIUM) · 6 Notes (LOW).
- **Buildable, with blockers resolved.** The feature fits the project pattern (new profile + credential + package + CLI subcommand tree — identical to Mail, Calendar/Telemost, Forms). REST API is documented; researcher confirmed all six endpoint shapes.
- **Pivot (researcher, decisive):** Yandex Disk `DELETE` is **async** (202 Accepted + operation poll); `disk rm` must poll for completion. `overwrite` is a native API query param — use it to gate `disk put` without a separate pre-check call. Default rm to **trash** (reversible), add `--permanent` for hard delete.
- **WebDAV vs REST:** both exist; consilium recommends **REST only** for v1 (simpler for public links; WebDAV would duplicate CalDAV plumbing without clear benefit for the six target operations).

**Top 3 must-fix before code:**
1. **B-1** — owner opens Yandex OAuth app UI and confirms exact Disk scope strings (expected `cloud_api:disk.read` / `cloud_api:disk.write`); researcher found them in docs but live UI confirmation is mandatory per D-005/D-007 precedent.
2. **B-2** — register new Disk OAuth app and add `http://localhost:8899` + `https://oauth.yandex.ru/verification_code` as redirects; obtain `YX360_DISK_CLIENT_ID`; add `yx360 login --disk`.
3. **B-3** — `disk share` and `disk put` (overwrite) must have `--yes` gates; without `--yes` these must abort (not prompt), same as `mail send`.

---

## Blockers (HIGH, requires_human: true)

### B-1 — Verify Disk OAuth scope strings in Yandex app UI
- **Anchor:** architect + skeptic + reviewer (§6)
- **Problem:** `cloud_api:disk.read` / `cloud_api:disk.write` come from Yandex docs (researcher, medium confidence), not from a live OAuth consent screen check. D-005/D-007 established that Mail scopes were confirmed in the UI, not just docs. Wrong scope strings → silent 403 at runtime.
- **Fix:** Owner opens Yandex OAuth app UI, confirms exact scope strings, records as next D-NNN. Do not start implementation without this confirmation.

### B-2 — Register Disk OAuth app; add `login --disk`
- **Anchor:** skeptic (hidden-cost), architect (module-boundary)
- **Problem:** Per D-009 pattern, Disk needs a separate OAuth app and credential profile (`disk`) because Yandex rejects mixed scope sets. Owner must register the app, add both redirect URIs, and obtain `YX360_DISK_CLIENT_ID`.
- **Fix:** Human B-task: register Disk OAuth app → `http://localhost:8899` + `https://oauth.yandex.ru/verification_code` as redirects. Add `disk` profile constant in `internal/config/config.go`; add `--disk` flag to `login.go` following the `resolveClientID` pattern. Set `YX360_DISK_CLIENT_ID` env var before live verify.

### B-3 — `disk share` and `disk put` (overwrite) require `--yes` gate
- **Anchor:** reviewer (ANTI-2), skeptic (hidden-cost), architect (pattern-choice)
- **Problem:** Creating a public URL (`disk share`) is externally visible and hard to reverse — same class as `mail send`. Overwriting an existing remote file (`disk put`) is destructive. Neither has a `--yes` gate in the original proposal.
- **Fix:**
  - `disk share <path> [--yes]` — without `--yes` print path + "this will create a public URL" + exit non-zero. With `--yes`, execute and print the public link.
  - `disk put <file> --to <path> [--yes]` — use the API's `overwrite=false` default; on 409 Conflict print "target exists, re-run with --yes to overwrite" + exit non-zero. With `--yes`, re-request with `overwrite=true`.
  - Never fall into interactive prompt (breaks agent mode).

### B-4 — `disk rm` semantics: trash-by-default, no interactive prompt
- **Anchor:** skeptic (hidden-cost), researcher (DELETE is async)
- **Problem:** Without `--yes` the proposal was ambiguous (abort or prompt?). Additionally, the Disk REST API `DELETE` is async — returns 202 Accepted + operation URL; naive callers will think it succeeded while the server is still deleting.
- **Fix:**
  - `disk rm <path> [--yes] [--permanent]` — without `--yes` print what would be trashed/deleted, exit non-zero. With `--yes`, execute.
  - Default: move to Trash (`permanently=false`). `--permanent` for hard delete.
  - Poll the operation URL after 202 until `status` is `success` or `failed` (2-3 polls with 1s sleep; document that rm is async).
  - Human B-task: none (code-only concern, but blocking on design clarity before implementation).

---

## Concerns (MEDIUM)

- **C-1 — IPv4 enforcement.** Disk HTTP client must use the project-wide IPv4 dialer (`tcp4`) per D-006. Create `internal/disk/client.go` with `http.Client` using the same `net.Dialer{}.DialContext = tcp4` pattern as OAuth/IMAP/CalDAV clients. (`internal/auth/oauth.go` is the reference.)
- **C-2 — Separate `internal/disk/` package.** Do not put Disk logic in `internal/cli/disk.go`. Create `internal/disk/` with `client.go` (IPv4 HTTP client, OAuth header), `disk.go` (List/Get/Put/Share/Remove/Mkdir functions), `types.go` (Resource, PublicLink structs). `internal/cli/disk.go` imports `internal/disk` only — consistent with `cli → domain → infra` layering (D-003).
- **C-3 — `disk list` pagination.** API paginates via `limit`/`offset`. Add `--limit N` (default 20) flag matching `mail list` precedent. Document that full enumeration is not done by default.
- **C-4 — `disk get` path traversal.** Remote filenames may contain `..` or `/`. Apply `filepath.Base(remotePath)` before joining with `--out <dir>`, reject names matching `..` or starting with `/`. Mirror `mail attachment` download pattern.
- **C-5 — `disk share` paired `disk unshare`.** Researcher confirmed `PUT /v1/disk/resources/unpublish`. Add `disk unshare <path>` as a paired command in v1 (no gate needed — it removes the public URL, not destructive). If scope grows too large, defer to follow-up PR and note in OQ.
- **C-6 — `disk rm` async polling.** After 202 Accepted, poll `GET <operation_href>` until `status == success|failed`, with up to 5 polls at 1s intervals. Surface errors from `status == failed`. Document that operation may still be in progress when CLI returns if polling times out.
- **C-7 — `disk mkdir` is needed (not scope-creep).** Researcher confirmed the API does NOT create parent directories on PUT. `disk mkdir` is therefore a necessary companion to `disk put` for non-existent paths. Keep it in v1. No `--yes` gate (creates a directory, not destructive, not externally visible).
- **C-8 — `ipv4Client()` duplication → extract `internal/netutil` first.** Second architect pass found that `ipv4Client()` is defined privately in both `internal/forms/service.go` and `internal/auth/http_client.go`. Adding a third copy in `internal/disk/` would create three-way drift. Extract to `internal/netutil` (single exported function) and replace both existing call sites before adding the disk service. No circular dependency risk.
- **C-9 — `disk:` scheme prefix auto-prepended by service layer.** The Yandex Disk REST API requires the `disk:` scheme on all path params (e.g. `disk:/Documents/file.txt`). CLI accepts plain POSIX paths (`/Documents/file.txt`); service layer prepends `disk:` when absent. Do not expose the scheme in CLI help text or flag descriptions.
- **C-10 — `disk get` restricted to files only in v1.** Behavior when `<path>` resolves to a directory is undefined in the proposal — recursive download is a materially larger scope. v1 returns a typed error for directory paths. Recursive `disk pull` is a v2 OQ.
- **C-11 — Upload URL expires in 30 minutes.** The two-step upload returns a temporary PUT URL valid for 30 min. For large files this may expire before the PUT completes. Stream via `io.Copy` + `Content-Length` from `os.Stat` (no buffering) to minimize window. Surface 4xx on expired URL as "upload URL expired — retry".
- **C-12 — HTTP error code handling: 413 and 507.** Researcher confirmed: 413 Payload Too Large (file exceeds 1 GB for standard / 50 GB for Yandex 360 accounts), 507 Insufficient Storage (quota full). Map these to human-readable CLI errors, not raw HTTP status codes.

---

## Notes (LOW)

- **N-1 — Large file streaming.** Use `io.Copy` for download and upload (never load entire file into memory). Add a warning if file size > 100 MB. Note in plan that chunked resumable upload is out of scope v1 (OQ to raise).
- **N-2 — Token expiry on long transfers.** D-004: refresh unimplemented; token ~12 months. Add a pre-flight token-age check: if `created_at + 11 months < now`, print a warning. Low priority but document.
- **N-3 — `disk rm` trash vs permanent UX.** Default to trash (`--permanent` for permanent delete). Trash is reversible; matches user expectation on accidental rm.
- **N-4 — WebDAV alternative.** Yandex Disk exposes WebDAV at `webdav.yandex.ru` with `Authorization: OAuth <token>` (identical to CalDAV pattern). REST is preferred for v1 (public links, metadata, native `overwrite` param). Noted for future evaluation (COPY/MOVE operations may be simpler via WebDAV).
- **N-5 — `disk list` verbose mode.** Consider `--long` / `-l` flag for size, modified date, type alongside names. Not blocking v1; `--json` covers agent use.
- **N-6 — Don't re-declare `--json` on disk commands.** `--json` is already a persistent root flag (`internal/cli/root.go:21`); redeclaring it per disk command causes a cobra flag-redefinition panic at startup. Use the inherited persistent flag; `emit()` in `internal/cli/output.go` reads the package-level `jsonOutput` variable directly.

---

## Research findings (confidence + sources, 2026-07-10)

| # | Finding | Confidence | Source |
|---|---------|-----------|--------|
| R1 | REST API at `https://cloud-api.yandex.net/v1/disk/`; paths via `?path=disk:/foo/bar.txt` URL-encoded. | medium | [Disk API ref](https://yandex.com/dev/disk/api/reference/) |
| R2 | OAuth scopes: `cloud_api:disk.read` (read) + `cloud_api:disk.write` (write). Corroborated by Yandex Cloud security docs + independent Python library docs. **Still requires live UI confirm (B-1).** | corroborated | [Quickstart](https://yandex.com/dev/disk/api/concepts/quickstart.html), [Yandex Cloud security](https://yandex.cloud/en/docs/security/standard-360/integrations) |
| R2a | Auth header is `Authorization: OAuth <token>` (not `Bearer`) — same as Calendar/Telemost. Confirmed via third-party library source and code examples. | high | [gist example](https://gist.github.com/mdukat/59e1b1eafc2d5c047b9d0c443d215bd8) |
| R2b | File size limits: 1 GB standard accounts, 50 GB Yandex 360 accounts. API returns 413 on oversize, 507 on quota full. | high | [Upload ref](https://yandex.com/dev/disk/api/reference/upload.html) |
| R2c | Upload URL (from two-step upload flow) is valid for 30 minutes; the PUT step requires no OAuth token. | high | [Upload ref](https://yandex.com/dev/disk/api/reference/upload.html) |
| R3 | List: `GET /v1/disk/resources?path=&limit=&offset=` → `embedded.items[]`. Pagination: `limit`/`offset`. | medium | [Meta ref](https://yandex.com/dev/disk/api/reference/meta.html) |
| R4 | Download: `GET /v1/disk/resources/download?path=` → `{href}` redirect URL; follow redirect for content. Two-step. | medium | [Content ref](https://yandex.com/dev/disk/api/reference/content.html) |
| R5 | Upload: `GET /v1/disk/resources/upload?path=&overwrite=<true\|false>` → `{href}`; then `PUT` content. `overwrite=false` returns 409 on conflict — natural gate for `--yes`. | medium | [Upload ref](https://yandex.com/dev/disk/api/reference/upload.html) |
| R6 | Share: `PUT /v1/disk/resources/publish?path=` → public link in response `href`. Unpublish: `PUT /v1/disk/resources/unpublish?path=`. | medium | [Publish ref](https://yandex.com/dev/disk/api/reference/publish.html) |
| R7 | Delete: `DELETE /v1/disk/resources?path=&permanently=<true\|false>`. May return 202 Accepted + `Location` operation URL; poll `GET <op>` until `status == success\|failed`. | medium | [Delete ref](https://yandex.com/dev/disk/api/reference/delete.html) + [Operations](https://yandex.com/dev/disk/api/reference/operations.html) |
| R8 | Mkdir: `PUT /v1/disk/resources?path=` (no body) → 201 Created; 409 Conflict if exists. **No recursive parent creation** — each level must be created individually. | medium | [Create folder ref](https://yandex.com/dev/disk/api/reference/create-folder.html) |
| R9 | WebDAV interface at `webdav.yandex.ru`, `Authorization: OAuth <token>`. PROPFIND/GET/PUT/MKCOL/DELETE supported. Simpler for recursive ops; REST preferred for links + metadata. | medium | [WebDAV docs](https://yandex.com/support/disk/desktop/ru/webdav.html) |

---

## Out-of-scope (declared)
- **Chunked/resumable upload** — out of scope v1; raise as OQ-019.
- **`disk get` on a directory path** — v1 returns a typed error; recursive download is a separate `disk pull` in v2 (OQ-020).
- **`disk move` / `disk copy`** — REST supports these; defer until needed. WebDAV may be better fit.
- **Shared/public disk enumeration** — reading other users' public disks; personal-account only v1.
- **Trash management** (`disk rm` soft-deletes to trash by default; emptying/restoring trash is a separate future command).
- **WebDAV transport** — documented alternative; REST for v1 per simplicity.
- **`disk unshare`** — researcher confirmed endpoint (`PUT /v1/disk/resources/unpublish`); include in v1 if scope allows, else defer. Tracked via C-5.

## Open questions raised
- **OQ-019** — Chunked/resumable upload for large files: is the Disk REST API's two-step upload sufficient for >1GB files, or does it require the resumable upload protocol?
- **OQ-020** — Transport choice: for future COPY/MOVE/recursive-listing operations should yx360 add WebDAV or extend the REST client?

---

## Suggested package / file layout

```
internal/disk/
  client.go      — http.Client with IPv4 dialer + OAuth header injection
  disk.go        — List / Get / Put / Share / Unshare / Remove / Mkdir
  types.go       — Resource, ResourceList, PublicLink, OperationStatus structs

internal/cli/
  disk.go        — diskCmd + subcommands (list, get, put, share, unshare, rm, mkdir)

internal/config/
  config.go      — add ProfileDisk const, DiskClientIDEnv const, DiskBaseURL const

internal/cli/
  login.go       — add --disk flag, resolveClientID for disk profile
```

---

## Per-agent verbatim (audit trail)

### architect
HIGH: scope strings unverified. HIGH: new `disk` credential profile + `login --disk` needed (D-009 pattern). HIGH: `internal/disk/` package with IPv4 client required. MEDIUM: cobra subcommand tree in `internal/cli/disk.go`. MEDIUM: `internal/disk/` package isolation (cli→domain→infra). MEDIUM: `--yes` on rm/share/put-overwrite. MEDIUM: path traversal on get. LOW: large file streaming. LOW: pagination `--limit`.

### skeptic
HIGH: scope strings premise-flaw — must come from UI not just docs. HIGH: §6 violation — no source URL / date on API facts. HIGH: `disk share` missing `--yes` (ANTI-2). HIGH: `disk rm` without `--yes` semantics ambiguous — must abort not prompt. MEDIUM: `disk put` overwrite gate. MEDIUM: Disk OAuth app registration is human B-task. MEDIUM: `disk mkdir` — may be scope-creep if API creates parents (researcher resolved: API does NOT create parents → keep). LOW: D-004 token expiry on long transfers. LOW: WebDAV alternative.

### researcher
R1-R9: REST API confirmed, all six endpoint shapes documented (medium confidence). Key: download and upload are two-step. DELETE is async (202+poll). mkdir does NOT create parents recursively. WebDAV available as alternative. `overwrite` param native on upload.

### reviewer
HIGH: `disk share` ANTI-2 (no `--yes`). HIGH: `disk put` overwrite ANTI-2. MEDIUM: D-006 IPv4 not mentioned in proposal. MEDIUM: D-009 separate profile not mentioned. MEDIUM: §6 external facts without date. LOW: D-004 token expiry mid-transfer.

---

_Plan by `/pre-feature` orchestrator. Human gate mandatory before `/implementor` (ANTI-4, Type 2). Do not auto-spawn implementor._
