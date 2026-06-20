# yx360-cli

CLI that signs into Yandex 360 via an intercepted web/mobile session token and exposes the private endpoints (calendar, mail, Telemost, disk) that the official public API does not — usable both as a human CLI and as an agent skill.

**PROJECT_TYPE: 2** (production, human-gated deploys).

## Entry points

- Working agreement (philosophy, code rules, hard-stops, routing): `AGENTS.md`
- Hard invariants: `.assistant/INVARIANTS.md`
- Project knowledge: `.memory-bank/index.md`
- Working memory (decisions, open questions): `.assistant/`

## Stack

- **Primary:** Go (single-binary CLI, distributed via a Homebrew tap).
- **Secondary surfaces (future):** a login webview / token-capture page (web), release + tap pipeline (infra).

## Domain

Reverse-engineering Yandex 360 private endpoints. The relevant consilium cluster is scraping / anti-bot — `surface-scout`, `scraping-architect`, `scraping-diagnostician`, `anti-bot-evasion` — not the generic backend path alone. Token interception, request signing, and session capture are first-class concerns.

## Agents

### Executing
| Agent     | Scope            |
|-----------|------------------|
| backend   | `**/*.go`        |

### Models
| Role | Model |
|------|-------|
| architect | opus |
| security  | opus |
| *         | sonnet |
