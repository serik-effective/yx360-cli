---
name: apple-simulator-debug
description: Reproduce, observe, and fix Apple-platform bugs on simulators (iOS, macOS, watchOS, visionOS) and, secondarily, on physical devices. Trigger when the user says "debug this on the simulator", "debug on the device", "repro on simulator", "why is it crashing in the simulator", "check the simulator logs", or "stream the device log". Drives a structured loop — hypothesis, repro, log evidence, fix diff — instead of ad-hoc poking.
---

# apple-simulator-debug

Structured debugging loop for Apple-platform issues on simulators (primary) and devices (secondary). Pairs with `swiftui-pro` for code review and `apple-impl` for the actual fix.

## Step 0 — Load context (mandatory)

Read `@.memory-bank/apple-native/debugging.md` first. It carries project-specific log filters, known-flaky simulators, and the canonical breakpoint set. If absent, note it as missing and continue with the defaults below — do not fabricate project conventions.

Also re-read the failing area of code before forming a hypothesis. Cite `file:line` for every claim you make about the bug.

## Step 1 — Confirm environment

Always start with `session_show_defaults` (XcodeBuildMCP). Verify:
- Project / workspace path
- Scheme
- Simulator UDID + OS version
- Configuration (Debug expected; Release rarely correct for debugging)

If any field is missing or wrong, fix with `session_set_defaults` before booting. Do not run `discover_projs` unless `session_show_defaults` shows a gap.

## Step 2 — Boot + build + run

1. `boot_sim` — boot target simulator. For watchOS / visionOS see Step 6.
2. `open_sim` — bring the Simulator window up so the user can see repro.
3. `build_run_sim` — build and launch. On failure, stop and surface the compiler diagnostic verbatim; do not guess.

Minimum OS targets to keep in mind (2024–2026 Apple docs):
- iOS / iPadOS 17+ (https://developer.apple.com/documentation/ios-ipados-release-notes, 2024)
- macOS 14+ (https://developer.apple.com/documentation/macos-release-notes, 2024)
- watchOS 10+ (https://developer.apple.com/documentation/watchos-release-notes, 2024)
- visionOS 1.2+ (https://developer.apple.com/documentation/visionos-release-notes, 2024)
- Liquid Glass APIs (`glassEffect`, `GlassEffectContainer`) — **iOS 26 + macOS 26 ONLY** (https://developer.apple.com/documentation/SwiftUI/Liquid-Glass, 2025). For older OS, gate with `if #available(iOS 26.0, macOS 26.0, *)` and fall back to `.regularMaterial` / `.thinMaterial`.

## Step 3 — Stream logs

Start log streaming **before** reproducing. Two channels:

- `xcrun simctl spawn <UDID> log stream --level=debug --predicate 'subsystem == "<bundle-id>"'` — focused, low noise.
- Fall back to `--predicate 'processImagePath CONTAINS "<AppName>"'` if the app does not use `Logger(subsystem:)`.

Reference: `os.Logger` + `OSLog` (https://developer.apple.com/documentation/os/logger, 2024). Prefer it over `print` for new code.

Keep the stream running in a background bash task; do not poll.

## Step 4 — Reproduce

Drive the app via `snapshot_ui` + tap/type tools. Each repro attempt must produce:
1. The exact sequence of UI actions (button refs, typed text).
2. The console output captured during those actions.
3. A `snapshot_ui` of the failing state.

If repro is flaky, run it three times and report the hit rate. Do not declare "fixed" off a single green run.

## Step 5 — Diagnose

Form a single hypothesis with citations:
- Suspected file + line.
- Apple API contract that is being violated, with doc URL + year.
- Predicted log signature.

Verify the hypothesis by matching predicted log to actual log. If they diverge, the hypothesis is wrong — discard it, do not patch it.

For crashes: pull the `.ips` from `~/Library/Logs/DiagnosticReports/` and cite the crashing frame. For SwiftUI view-update loops: use `Self._printChanges()` (https://developer.apple.com/documentation/swiftui/view/_printchanges(), 2024) inside the suspect `body`.

## Step 6 — watchOS + visionOS specifics

**watchOS sim:**
- Paired iPhone sim must be booted first; `list_sims` then boot the pair.
- Logs flow through the iPhone host — predicate on the watch bundle ID still works via `simctl spawn` on the watch UDID.
- UI automation is limited; rely on `snapshot_ui` rather than gestures where possible.

**visionOS sim:**
- Heavy — boot one at a time, expect 30–60 s warm-up.
- Use `xrOS` SDK identifier in build settings if you touch them.
- `snapshot_ui` returns spatial coordinates; do not assume 2D layout.
- Reference: https://developer.apple.com/documentation/visionos (2024).

## Step 7 — Device flow (secondary)

Use only when the bug does not reproduce on simulator (camera, ARKit, real sensors, background modes).

- iOS 17+: `xcrun devicectl device install app --device <UDID> <path.app>` then `devicectl device process launch` (https://developer.apple.com/documentation/xcode-release-notes/xcode-15-release-notes, 2024).
- Logs: `devicectl device console --device <UDID>` or stream via Console.app filtered by device.
- Older flows via `idb` (Meta, https://fbidb.io, last verified 2024-09) are acceptable when devicectl misbehaves, but devicectl is the supported path.
- Code signing must be sorted before this step — if signing fails, stop and surface the error; do not auto-resign.

## Step 8 — Fix + re-run

Hand off the diagnosis to `apple-impl` for the patch, or apply it inline if it is < 10 lines and the cause is unambiguous. Then:
1. Re-build with `build_run_sim`.
2. Re-run the exact repro sequence from Step 4.
3. Compare logs — the predicted bad signature must be gone, no new error signature introduced.

If anything is off, loop back to Step 5 with the new evidence. Do not mark done on a green build alone — green build ≠ feature works.

## Output schema (mandatory)

Return the result in this exact shape — no prose-only summary.

```
## Hypothesis
<one sentence, with file:line + Apple doc URL>

## Repro steps
1. <action> → <observed>
2. ...
Hit rate: <N/3>

## Log evidence
<verbatim log excerpts, trimmed; include subsystem + timestamp>
Crashing frame (if any): <symbol> at <file:line>

## Fix diff
```diff
<unified diff applied or proposed>
```

## Verification
- Build: <pass/fail>
- Repro after fix: <N/3 reproduced>
- New signatures introduced: <none | list>
```

## Hard stops

- No "should be fixed" without a post-fix repro run.
- No silent OS-version assumptions — state minimum iOS/macOS/watchOS/visionOS used by every cited API.
- No fabricated log lines. If logs are empty, say so and widen the predicate.
- No emojis in the report.
