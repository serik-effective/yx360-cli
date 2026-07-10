# Agent-Mode Roadmap (proposed)

Backlog of "doc + small feature" items that make `yx360` a stronger foundation for an agent skill, especially for an agent running on a **remote/headless server**. Reference: [`openclaw/gogcli`](https://github.com/openclaw/gogcli) (a mature Google Workspace CLI built for agents/scripts/CI).

Status legend: `proposed` (not started). Nothing here is implemented yet — each item is a candidate `/pre-feature`.

---

## P0 — remote/headless login (the blocker for agent-on-another-server)

**Problem.** `yx360 login` uses a loopback PKCE flow (`http://localhost:8899`): browser and CLI must be on the same host. When the agent runs on a remote server with no browser, and the operator can only open a browser on their own machine, loopback cannot return the redirect.

**Chosen approach — manual-paste authorization-code + PKCE (two-step).** Mirrors the codex/openclaw pattern. Reuses the existing, *secretless-verified* code exchange (`internal/auth/oauth.go`; loopback login already exchanges code without `client_secret`, D-004) — lower risk than the device-grant path, whose secretless exchange is unverified.

```
yx360 login --manual --begin [--mail|--calendar|--telemost|--forms]
  -> generates PKCE code_verifier + state, prints the auth URL,
     persists {verifier, state, profile} to a short-lived 0600 temp file on the agent host
yx360 login --manual --complete --redirect-url "<pasted localhost URL or code>"
  -> reads the temp file, validates state, parses code, exchanges code+verifier
     at oauth.yandex.ru/token over TLS, stores the token, deletes the temp file
```

**Key facts / guardrails (for the consilium):**
- The pasted redirect URL contains the one-time authorization **`code`, not a token.** The agent fetches the token itself; the token never transits the chat (§12 redaction).
- `code` is single-use, short-lived, and bound to the `code_verifier` held only by the agent — a leaked code from the chat is useless.
- `state` must be validated (CSRF).
- Reuse the already-registered `localhost:8899` redirect; the operator's browser will fail to load the page but the address bar holds `code`. No OAuth-app re-registration needed.
- Temp file holds the PKCE verifier (sensitive for the flow): mode `0600`, short-lived, deleted after exchange.
- On a headless server the OS keychain is usually absent → combine with `--insecure-file-store` (already exists, plaintext 0600). Warn explicitly.

**Open question:** interactive single-process (block on stdin, codex-style) vs two-step begin/complete. Two-step is agent-friendly (stateless between chat turns); the consilium decides.

Device Authorization Grant (RFC 8628) is an optional secondary rung **only if** Yandex's device-code exchange is confirmed to work secretless.

Suggested branch: `feat/manual-paste-remote-login`.

---

## P1 — agent-safe surface (bundle with P0; one umbrella consilium)

| Item | gogcli ref | Our gap | Proposed shape |
|---|---|---|---|
| **`--no-input`** | `--no-input` | a gated write without a tty hangs on the `[y/N]` prompt | global `--no-input`: a write missing `--yes` fails fast instead of prompting |
| **prompts/preview → stderr** | "human hints → stderr, stdout parseable" | `confirmPrompt`/previews write to **stdout** (`cmd.Print`), so JSON is clean only with `--yes` | route previews + prompts to stderr; keep stdout pure JSON unconditionally |
| **`--dry-run`** | `--dry-run` | preview exists only as part of the confirm flow | `--dry-run` on writes (`mail send`, `calendar create/update/delete`, `telemost create`, `forms create/questions add/publish/unpublish`) prints intended action as JSON and exits 0 without effect |
| **`--wrap-untrusted`** | `--wrap-untrusted` | `mail read` and `forms responses` return external free-text → prompt-injection risk for the ingesting LLM | wrap fetched untrusted text in explicit markers; document the injection guidance in the contract |
| **differentiated exit codes** | exit-code map in `gog schema` | only `0`/`1` today | distinct non-zero codes per class (usage / auth-needed / API error) |
| **command allowlist / `--mail-no-send`** | `--enable-commands-exact`, `--gmail-no-send` | no runtime sandbox for a delegated agent | allowlist of permitted commands + send-disable switch |

Suggested branch: fold into `feat/headless-agent-mode` alongside P0, since all of these are "agent on a remote server, no tty, handling someone else's data."

---

## P2 — independent, high agent value

| Item | gogcli ref | Proposed shape |
|---|---|---|
| **`yx360 schema --json`** | `gog schema --json` | emit the full command tree, flags, which commands are human-gated, required scopes, and the exit-code map — so an agent self-discovers the surface instead of parsing this doc. Single biggest lever for "foundation for an agent skill." |
| **`yx360 mcp` (MCP server)** | `gog mcp` | typed tool interface (read-only by default, opt-in writes); larger effort, later. |

Suggested branches: `feat/schema-json`, `feat/mcp-server` (separate).

---

## Not adopting (out of scope for this CLI)

- **Service accounts / domain-wide delegation** — `yx360` is a public-client OAuth tool acting as the user, not an admin service application.
- **Generated command docs from code** (`make docs-commands`) — nice anti-drift, but with four surfaces the hand-written [agent-contract.md](agent-contract.md) is sufficient for now.

---

## Already strong (keep)

- **Scope transparency** — errors already name the exact `yx360 login ...` re-auth command (gogcli pattern 5).
- **Separate credential profiles per OAuth app** — matches the multi-account model; gap is profile-aware logout (tracked as OQ-011).
