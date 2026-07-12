# Agent Quickstart

The shortest path for an agent (or operator setting one up) to go from zero to a working `yx360` call. For the full contract see [agent-contract.md](agent-contract.md).

## 1. Build or install

```bash
go build -o bin/yx360 ./cmd/yx360
```

## 2. Register OAuth apps and set client ids

`yx360` is a public OAuth client (PKCE, no `client_secret`). Each service group uses a **separate** Yandex OAuth app, because Yandex rejects mixed scope sets in one app:

| Group | Scopes | Env var |
|---|---|---|
| Mail | `login:info`, `mail:imap_full`, `mail:smtp` | `YX360_CLIENT_ID` |
| Calendar + Telemost | `login:info`, `calendar:all`, `telemost-api:conferences.create` | `YX360_CALENDAR_CLIENT_ID` |
| Forms (Yandex 360 for Business only) | `login:info`, `forms:read`, `forms:write` | `YX360_FORMS_CLIENT_ID` + `YX360_FORMS_ORG_ID` |
| Disk | `login:info`, `cloud_api:disk.read`, `cloud_api:disk.write` | `YX360_DISK_CLIENT_ID` |

```bash
export YX360_CLIENT_ID=<mail-app-client-id>
export YX360_CALENDAR_CLIENT_ID=<calendar-telemost-app-client-id>
export YX360_FORMS_CLIENT_ID=<forms-app-client-id>
export YX360_FORMS_ORG_ID=<numeric-360-org-id>
export YX360_DISK_CLIENT_ID=<disk-app-client-id>
```

## 3. Log in (interactive, once)

Login opens a browser and stores the token in the OS keychain. Each group logs in separately:

```bash
yx360 login --mail --mail-send       # mail read + send
yx360 login --calendar --telemost    # calendar + telemost
yx360 login --forms                  # forms (business org)
yx360 login --disk                   # disk read + write
```

**Headless/remote login** (VDS or agent host with no local browser) — use the two-step manual flow:

```bash
yx360 login --manual --begin --disk      # prints auth URL; open it in your browser
# paste code from https://oauth.yandex.ru/verification_code
yx360 login --manual --complete --code <code> --insecure-file-store
```

Scope flags (`--mail`, `--calendar`, `--telemost`, `--forms`, `--disk`) on `--begin` pick the profile; omit them on `--complete`.

## 4. First machine-readable calls

Pass `--json` for parseable stdout and `--yes` on any write so it does not stop at a confirmation prompt:

```bash
yx360 --json mail list --limit 20
yx360 --json calendar list --from 2026-06-20 --to 2026-06-21
yx360 --json forms responses <survey-id>
yx360 --json mail send --to user@example.com --subject "Hi" --body "Text" --yes
yx360 --json disk list --path /
yx360 --json disk put report.pdf --to /Documents/report.pdf --yes
yx360 --json disk share /Documents/report.pdf --yes
yx360 --json disk rm /Documents/draft.txt --yes
```

Preview any mutating command without executing with `--dry-run` (overrides `--yes`):

```bash
yx360 --json disk rm /Documents/draft.txt --dry-run
# → {"dry_run":"true","would":"move disk:/Documents/draft.txt to Trash"}
```

## 5. MCP stdio server (optional)

To expose all surfaces to an MCP-aware host (Claude Desktop, OpenCode, custom agent):

```bash
# interactive host
yx360 mcp serve

# headless VDS — plaintext token store, no keychain required
YX360_INSECURE_FILE_STORE=1 yx360 mcp serve
```

14 tools are registered: `disk_list/get/put/share/unshare/rm/mkdir`, `mail_list/search/read`, `calendar_list/get/create`, `telemost_create`. Mutating tools require `confirmed=true`; with `confirmed=false` they return a dry-run preview.

Branch on the exit code (`0` ok, non-zero failure), then read stderr. On a missing scope the error names the exact `yx360 login ...` to re-run.
