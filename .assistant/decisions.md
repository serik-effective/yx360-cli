# Decisions Log

> Append-only chronological record. When a decision is overturned, add a new entry with date + reason. Never edit or delete prior entries.

---

## D-001 — Harness installed
**Date:** 2026-06-20
**Status:** accepted
**Decision:** Effective Harness installed at commit `698eb86` via the `/setup` skill. `PROJECT_TYPE: 2`. Primary stack: Backend/CLI, Go.
**Source:** `git@github.com:effective-dev-os/harness.git@698eb86489901cb0cd49d3c4a91643730dc5c1ea`
**Touch policy chosen at install:** N/A — empty target directory, nothing to overwrite.
**Notes:** Target was an empty, non-git directory at install. Owner to `git init` before the first PR (ANTI-3). Domain is scraping / anti-bot (Yandex 360 private-API reverse-engineering), so the `surface-scout` / `scraping-architect` / `scraping-diagnostician` / `anti-bot-evasion` agents are in scope alongside `backend`.
