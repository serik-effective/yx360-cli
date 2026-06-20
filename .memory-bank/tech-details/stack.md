# Stack

> Stub seeded by `/setup`. Owner: lock exact versions before first real feature (OQ-001).

## Language

- **Go** — single static binary, Homebrew-friendly. Version: TODO (pin `go.mod` toolchain).

## Likely components (from vision — confirm during `/pre-feature`)

| Concern | Candidate approach | Status |
|---------|--------------------|--------|
| CLI framework | `cobra` / `urfave/cli` | TODO — pick one |
| Login webview | OS webview or headless browser to render Yandex 360 login | TODO |
| Token interception | local HTTP callback / webhook capturing the session token | TODO |
| HTTP client + signing | replicate first-party request signing / headers | TODO — reverse-engineer |
| Token storage | OS keychain vs file — owner has non-goals on storage (see vision) | TODO |
| Distribution | Homebrew tap + GoReleaser (or manual formula) | TODO |
| Agent skill | `.claude/skills/`-style drop-in wrapping the CLI | TODO |

## Detected at install

- Empty repository at install time — no `go.mod`, no source yet. Greenfield.
- Not a git repo at install. Run `git init` before opening the first PR (ANTI-3: branch workflow).

## Reverse-engineering posture

This project lives in the scraping / anti-bot domain. Expect to use `mitmproxy` / browser devtools to map the web surface, and `frida` / `jadx` / APK inspection only if the web surface is insufficient and the mobile app must be analyzed (see routing table in the harness for the relevant agents).
