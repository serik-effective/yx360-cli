# Open Questions

> Unresolved design/scope questions. Close an entry by moving it to `.assistant/decisions.md` as a D-NNN when resolved. Seeded by `/setup`.

---

## OQ-001 — Lock the stack and tooling versions
Pin `go.mod` toolchain, pick the CLI framework (`cobra` vs `urfave/cli`), and decide the distribution path (GoReleaser + Homebrew tap vs manual formula). Resolve during the first `/pre-feature`.

## OQ-002 — Token-interception mechanism
How is the Yandex 360 session token captured at `yx360 login`? OS-native webview vs embedded/headless browser, and how the local webhook/callback receives the token. Affects the security posture (token at rest, lifetime).

## OQ-003 — Web vs mobile reverse-engineering boundary
Vision starts with the web surface and escalates to mobile only if needed. Define the trigger: which capabilities justify moving to APK/Frida analysis of the mobile app vs staying on the web surface.

## OQ-INV-1 — Authorization / ToS posture (confirm before shipping)
Reverse-engineering private Yandex 360 endpoints and intercepting session tokens — confirm the legal/authorization stance (own-account use only? ToS boundaries?). This gates anything user-facing. Owner to state the authorization context explicitly.
