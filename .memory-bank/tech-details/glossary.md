# Glossary

> Seeded by `/setup`. Owner: refine definitions (some are best-effort inferences, marked TODO).

| Term | Definition |
|------|------------|
| Yandex 360 | Yandex's workspace suite (mail, calendar, disk, Telemost, etc.). The product whose private API this CLI targets. |
| Public API | Yandex 360's documented/official API. Narrower than first-party clients — the gap this project closes. |
| Private endpoint | An internal endpoint used by the Yandex 360 web/mobile apps but not exposed in the public API. The reverse-engineering target. |
| Telemost | Yandex's video-conferencing product inside Yandex 360. `yx360 telemost create` uses the official conference-create API and returns a `join_url`; conference deletion/cancellation is still out of scope because no official cleanup endpoint has been verified. See [calendar-telemost implementation](../../swarm-report/calendar-telemost-implementation-2026-06-20.md). |
| OAuth token | The credential returned by documented Yandex OAuth after the user consents to scopes. Stored only in the configured TokenStore. |
| Token interception | Superseded initial framing. Current `yx360 login` uses documented Yandex OAuth, not webview/session-token interception. |
| Webhook / callback | Historical term from the initial framing. Current login uses a loopback OAuth callback that receives an authorization code, not a captured session token. |
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
| IMAP | Mail read/search/fetch protocol used by `yx360 mail list`, `search`, `read`, and `attachment`; Yandex RU default is `imap.yandex.ru:993` over TLS. |
| SMTP | Mail send protocol used by `yx360 mail send`; Yandex RU default is `smtp.yandex.ru:465` over TLS. |
| XOAUTH2 | SASL OAuth mechanism tried first for Yandex Mail IMAP/SMTP authentication. |
| OAUTHBEARER | SASL OAuth fallback tried after XOAUTH2 for Yandex Mail IMAP/SMTP authentication. |
| `mail:imap_full` | Yandex OAuth scope used for Mail read-side IMAP operations. |
| `mail:smtp` | Yandex OAuth scope used for Mail SMTP send operations. |
| Mail send preview | Human gate for `yx360 mail send`: the CLI prints from/to/cc/bcc/subject/body size/attachments and sends only after confirmation, unless explicit `--yes` is passed. |
| CalDAV | WebDAV-based calendar protocol used by `yx360 calendar` for list/read/create/update/delete against `https://caldav.yandex.ru`; live proof requires `Authorization: OAuth <token>`, not `Bearer`. See [calendar-telemost plan](../../swarm-report/calendar-telemost-plan-2026-06-20.md). |
| iCalendar / VEVENT | Calendar interchange format and event component used for Calendar payloads. `yx360` parses and generates a narrow non-recurring VEVENT subset for title, time range, description, location, attendees, URL, UID, and ETag-aware updates. |
| `calendar:all` | Yandex OAuth scope required for Calendar CalDAV access. It is requested through the Calendar/Telemost credential profile and was live-verified for CalDAV discovery/list/read/write flows. |
| `telemost-api:conferences.create` | Yandex OAuth scope required to create Telemost conferences through the official API. V1 requests only this Telemost scope for link creation; read/update/delete Telemost scopes are not part of the shipped feature. |
| Credential profile | Named token-storage namespace that prevents Mail and Calendar/Telemost OAuth grants from overwriting each other. V1 uses the `mail` profile for Mail commands and the `calendar-telemost` profile for Calendar/Telemost commands. |
| Event href | CalDAV resource URL/path returned by the server for an event. `yx360 calendar read/update/delete` accepts this href as the event identifier because it is stable and maps directly to WebDAV operations, though it is not a polished human alias. |
| Yandex Forms API | Documented REST API at `https://api.forms.yandex.net`, authenticated with `Authorization: OAuth <token>` plus `X-Org-Id`; available only to Yandex 360 for Business organizations. |
| `survey_id` | Yandex Forms identifier for a single form/survey; supplied by the user since there is no documented list-all endpoint. |
| `X-Org-Id` | Yandex 360 organization id header required by the Forms API; provided via `YX360_FORMS_ORG_ID`. |
| Forms profile | The CLI credential profile (separate OAuth app) holding `forms:read` + `forms:write`, obtained with `yx360 login --forms`. |
| X-Cloud-Org-Id | Alternative org header for Yandex Cloud organizations (non-numeric/hex org id); the CLI sends `X-Org-Id` for numeric org ids and `X-Cloud-Org-Id` for non-numeric ones. |
| Forms public URL | A published survey is reachable at `https://forms.yandex.ru/cloud/<survey_id>`; answer stats at `https://forms.yandex.ru/cloud/admin/<survey_id>/answers?view=stats`. The API does not return these; the CLI derives them. |
