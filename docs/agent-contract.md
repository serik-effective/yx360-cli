# Agent Command Contract

This document defines the stable, agent-facing usage contract for `yx360`. Agents should use it instead of scraping human-oriented command output.

## Prerequisites

- `yx360` must be built or installed and available on `PATH`, or called by an explicit local path such as `./bin/yx360`.
- `YX360_CLIENT_ID` must be set for `yx360 login`.
- The user must complete OAuth consent in the browser or device flow.
- Mail read commands require a stored credential containing `mail:imap_full`; obtain it with `yx360 login --mail`.
- `mail send` and `mail unsubscribe --method mailto --apply/--yes` require a stored credential containing `mail:smtp`; obtain it with `yx360 login --mail --mail-send`.

Credentials are stored in the OS keychain by default. Agents must not read the keychain blob directly; use CLI commands.

## JSON Mode

Use the global `--json` flag for machine-readable output:

```bash
yx360 --json login --mail
yx360 --json mail list --limit 20
yx360 --json mail read <uid>
yx360 --json mail send --to user@example.com --subject "Subject" --body "Body" --yes
yx360 --json mail unsubscribe <uid>
```

Current JSON output is pretty-printed JSON on stdout. The shape is command-specific and contains non-secret result fields only, for example:

- `login`: `status`, `account`, `scopes`, optional `expiry`.
- `logout`: `status`.
- `mail list` and `mail search`: an array of message summaries.
- `mail read`: one message with metadata, body, attachment manifest, and optional `unsubscribe_options`.
- `mail attachment`: `path`.
- `mail send`: `status`, `from`, `recipients`, `subject`.
- `mail unsubscribe`: preview has `uid`, `folder`, `options`, and optional `selected`; apply result has `status`, `uid`, `folder`, `method`, `uri`, optional `http_status`, and optional `mail`.

Agents may rely on field names listed above for the current major CLI surface. They must tolerate additional fields.

## Token Redaction

JSON output must never include OAuth access tokens, refresh tokens, cookies, client secrets, or raw authorization headers. Agents must also redact these values from logs, issue reports, transcripts, and generated files.

Do not print:

- OAuth `access_token` or `refresh_token`.
- `Authorization` headers.
- Browser cookies or session captures.
- `YX360_CLIENT_ID` only if the owner treats it as private in the current environment.

If a command fails with a wrapped upstream error, quote only the non-secret message needed for diagnosis.

## Errors

Commands return non-zero on failure and write the error through the normal Cobra error path. Agents should branch on exit code first, then inspect the error text.

Known actionable errors:

- `mail: stored credential is missing, expired, or does not include mail:imap_full; run yx360 login --mail`
- `mail: stored credential is missing, expired, or does not include mail:smtp; run yx360 login --mail --mail-send`
- `mail: IMAP OAuth authentication failed; enable mail-client access and app passwords/OAuth tokens in Yandex 360 Mail settings, then run yx360 login --mail`
- `OS keychain unavailable (...): on headless/CI hosts re-run with --insecure-file-store`

Expired tokens are handled by re-running `yx360 login`; refresh is intentionally not implemented.

## Human Gate For Mail Send And Unsubscribe

`yx360 mail send` is externally visible and is human-gated by default. Without `--yes`, the command prints a preview and requires interactive confirmation.

Agents may pass `--yes` only after the user has explicitly approved the exact send action, including recipients, subject, body source, and attachments. A general task approval is not enough.

Bcc recipients are sent through SMTP but are not included in MIME headers.

`yx360 mail unsubscribe <uid>` previews RFC 2369 / RFC 8058 header-derived options only. It does not execute unless `--apply` is present with interactive confirmation, or `--method <kind> --yes` is passed after explicit user approval of method and URI.

## Plaintext Token Warning

`--insecure-file-store` stores credentials as plaintext JSON under the user config directory with file mode `0600`. This is only for headless or CI hosts where the OS keychain is unavailable.

Agents must not select `--insecure-file-store` silently. Warn the user before using it, and do not combine it with send-capable scopes unless the user explicitly accepts the plaintext credential risk for that environment.
