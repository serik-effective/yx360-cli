# Vision

> Seeded by `/setup` from the install interview (translated from the owner's spoken Russian). Project owner: expand the TODO sections.

## One-line

A Yandex 360 CLI that authenticates by intercepting a real session token and then drives the private endpoints the official public API does not expose — also shippable as an agent skill and installable via Homebrew.

## The problem

Yandex 360's documented public API is narrower than what the first-party clients can do. The mobile app and the web app reach internal endpoints (calendar, mail, Telemost, disk, and more) that the public API never surfaces. We want that full surface.

## The approach

1. **Reverse-engineer the private surface.** Start with the easy target — the **web app** — and escalate to the **mobile app** only when the web surface is missing something we need.
2. **Sign-in via interception, not API keys.** `yx360 login` opens a webview to Yandex 360, the user logs in normally, and the CLI captures the resulting session token via a local webhook / callback.
3. **Drive the private endpoints through that token** behind a clean CLI-like interface.
4. **Ship as an agent skill** so any AI agent can drive the same CLI (drop-in skill, not a bespoke integration).
5. **Distribute via a Homebrew tap** so `brew install` provisions the CLI.

## Target audience

TODO — owner to fill. (Likely: the owner's own agents + power users who need the full Yandex 360 surface.)

## Definition of done

TODO — owner to fill. Candidate DoD: `yx360 login` captures a token end-to-end; at least one private endpoint (e.g. calendar list) works through the CLI; the agent skill drives it; `brew install` from the tap works.

## What we don't do

TODO — owner to fill. (Candidate non-goals: no credential storage beyond the captured token's lifetime; no scraping of other users' data; respect Yandex 360 ToS boundaries — owner to confirm the legal/authorization stance.)
