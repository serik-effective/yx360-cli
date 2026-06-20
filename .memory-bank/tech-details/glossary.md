# Glossary

> Seeded by `/setup`. Owner: refine definitions (some are best-effort inferences, marked TODO).

| Term | Definition |
|------|------------|
| Yandex 360 | Yandex's workspace suite (mail, calendar, disk, Telemost, etc.). The product whose private API this CLI targets. |
| Public API | Yandex 360's documented/official API. Narrower than first-party clients — the gap this project closes. |
| Private endpoint | An internal endpoint used by the Yandex 360 web/mobile apps but not exposed in the public API. The reverse-engineering target. |
| Telemost | Yandex's video-conferencing product inside Yandex 360. One of the surfaces to reach. |
| Session token | The auth token captured from a real logged-in Yandex 360 session, used in place of public API keys. |
| Token interception | `yx360 login` flow: render the Yandex 360 login in a webview, let the user authenticate, capture the resulting token via a local webhook/callback. |
| Webhook / callback | Local HTTP listener that receives the intercepted token after web login. |
| Agent skill | A drop-in skill wrapping the CLI so any AI agent can drive Yandex 360 through it. |
| Homebrew tap | The owner's custom `brew` tap that distributes `yx360-cli` (`brew install`). |
| Web surface | The Yandex 360 web app's endpoint set — the first/easier reverse-engineering target. |
| Mobile surface | The Yandex 360 mobile app's endpoint set — escalation target if the web surface lacks something. |
| PKCE (S256) | Proof Key for Code Exchange; lets the CLI do OAuth as a public client with no `client_secret`. Used on both the authorize request and the token exchange. |
| Public client | An OAuth client that ships no secret (a CLI binary can't keep one); `yx360-cli` relies on PKCE instead. |
| Flow ladder | `yx360 login` tries auth methods in order: loopback (`localhost:8899`) → device flow → (later) manual paste; advances only when a rung is unavailable, aborts on a real auth rejection. |
| Loopback flow | System browser + local listener on `127.0.0.1:8899` captures the OAuth `code`; the redirect must byte-match the registered `http://localhost:8899`. |
| Device flow | Headless fallback: user enters a code at `ya.ru/device` while the CLI polls the token endpoint. No local listener. |
| TokenStore | The storage seam; default OS keychain (`zalando/go-keyring`), or a flag-gated plaintext file (`--insecure-file-store`) for headless/CI. |
| Refresher | The (declared, not-yet-implemented) seam for refreshing the OAuth token; gated on the B2 secretless-refresh test. |
