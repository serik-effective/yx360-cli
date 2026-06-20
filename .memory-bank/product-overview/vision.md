# Vision

> Seeded by `/setup` from the install interview (translated from the owner's spoken Russian). Project owner: expand the TODO sections.

## One-line

A Yandex 360 CLI that authenticates through documented Yandex OAuth and exposes useful Yandex 360 surfaces through a safe command-line interface — also shippable as an agent skill and installable via Homebrew.

## The problem

Yandex 360's documented public API is narrower than what the first-party clients can do. Some products have documented protocols or APIs, while others may still require web/mobile surface research. The CLI should prefer documented OAuth and documented protocols first, then escalate only for named capability gaps.

## The approach

1. **Use documented OAuth first.** `yx360 login` uses Yandex OAuth authorization-code + PKCE as a public client, with loopback and device-flow paths.
2. **Prefer documented service protocols.** Mail v1 uses IMAP/SMTP with OAuth scopes, not a private Mail REST API.
3. **Escalate only for named gaps.** Use web/mobile surface research only when documented OAuth APIs or protocols cannot cover a required capability.
4. **Drive features through a clean CLI-like interface** with safe defaults and JSON output for agents.
5. **Ship as an agent skill** so any AI agent can drive the same CLI (drop-in skill, not a bespoke integration).
6. **Distribute via a Homebrew tap** so `brew install` provisions the CLI.

## Target audience

TODO — owner to fill. (Likely: the owner's own agents + power users who need the full Yandex 360 surface.)

## Definition of done

Current baseline: `yx360 login` works end-to-end through documented OAuth; Mail read/search/read-attachment/send works through IMAP/SMTP; Calendar CRUD via CalDAV and Telemost link creation are implemented and live-smoked. Forms read/create/add-questions/publish is live-verified through the documented Forms API against a real Yandex 360 for Business org (Forms API is Business-org-only). Broader DoD remains: the agent skill drives the CLI, and `brew install` from the tap works.

## What we don't do

TODO — owner to fill. (Candidate non-goals: no credential storage beyond the captured token's lifetime; no scraping of other users' data; respect Yandex 360 ToS boundaries — owner to confirm the legal/authorization stance.)
