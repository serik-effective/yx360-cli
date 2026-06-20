---
name: apple-design-critic
description: Ruthless line-by-line review of a SwiftUI screen, view, or layout against Apple HIG and the project's apple-native rule book. Use when the user says "review this screen for native feel", "review this view", "review this layout for native feel", "is this UI native", "audit this SwiftUI for HIG violations", or invokes /apple-design-critic. Targets iOS 17+ / macOS 14+ projects; Liquid Glass-specific findings only on iOS 26 / macOS 26 codebases.
---

# Apple Design Critic

Goal: produce a brutal, specific, line-cited critique of a SwiftUI surface — not generic vibes. Every finding maps to a rule in `.memory-bank/apple-native/design-critic-rules.md` and to `file:line` in the codebase under review.

## Inputs the agent must collect

1. **Target view(s).** Ask the user for: file path(s), or a screen name, or the running route. If unclear, list candidate `*.swift` files containing `View` and ask which.
2. **Deployment target.** Read `Package.swift` / `project.yml` / `*.xcodeproj/project.pbxproj` to determine `iOSDeploymentTarget` and `macOSDeploymentTarget`. Liquid Glass findings (rules tagged `LG-*`) apply only when target ≥ iOS 26 or macOS 26. Otherwise mark them N/A and skip.
3. **Optional screenshot.** If the user asks for a visual pass, or the violation hinges on rendered output (spacing, glass blur, contrast), capture one:
   - iOS simulator: `mcp__xcodebuildmcp__screenshot` (after `build_run_sim` if needed).
   - Live UI tree: `mcp__xcodebuildmcp__snapshot_ui` for hit-test rects and accessibility labels.
   Skip if the violation is obvious from source (e.g. hardcoded padding, missing `accessibilityLabel`).

## Mandatory pre-read

Before producing any finding, the agent MUST read in this order:

1. `.memory-bank/apple-native/design-critic-rules.md` — the rule book. Each rule has an ID (e.g. `HIG-12`, `LG-04`, `A11Y-07`), a one-line statement, and a canonical Apple doc URL.
2. `.memory-bank/apple-native/hig-canonical.md` — HIG anchors used by the rules.
3. `.memory-bank/apple-native/liquid-glass.md` — only if deployment target ≥ iOS 26 / macOS 26.
4. `.memory-bank/apple-native/design-critic-rules.md` again — scan for rule IDs to use as a checklist.

If `design-critic-rules.md` does not exist, stop and tell the user: "rule book missing at `.memory-bank/apple-native/design-critic-rules.md` — cannot run critic without it."

## Review procedure

1. Read every target file fully (no `head`/`tail`).
2. Walk the rule book top to bottom. For each rule, search the target view for violations. Apply at least **40 rules** before writing the report. If the rule book has fewer than 40 applicable rules for the given OS target, apply every applicable rule and note the count.
3. For each violation, record: rule ID, `file:line`, the offending snippet (≤ 2 lines), and a concrete fix as a SwiftUI diff.
4. Skip rules that don't apply (e.g. iPad-only rule on a macOS-only view). Don't pad the report.
5. No finding without `file:line`. No "feels off", "could be more polished", "consider improving" — banned. Either cite a rule and a fix, or drop the finding.
6. If the same anti-pattern repeats, file it **once** with all locations listed, not N copies.

## Output format

Two sections, in this order, no preamble.

### 1. Findings table

| # | Rule ID | Severity | Location | Violation | Fix summary |
|---|---------|----------|----------|-----------|-------------|
| 1 | HIG-12 | blocker | `Views/Home.swift:42` | `.padding(17)` magic number | use `.padding(.horizontal)` (system metric) |
| 2 | A11Y-03 | major | `Views/Home.swift:88` | `Image("trash")` no label | add `.accessibilityLabel("Delete")` |
| 3 | LG-04 | major | `Views/Toolbar.swift:21` | `.background(.regularMaterial)` on iOS 26 toolbar | use `.glassEffect()` (iOS 26+), gate with `#available` |

Severity scale: `blocker` (ships broken UX or fails a11y), `major` (visibly non-native), `minor` (polish). No `nit` tier — if it's nit, drop it.

### 2. Diff suggestions

One fenced diff per finding that needs code, grouped by file. Use unified diff format with `file:line` anchor in the heading.

```
--- Views/Home.swift:42
+++ Views/Home.swift:42
-        .padding(17)
+        .padding(.horizontal)
```

For Liquid Glass fixes targeting iOS 26 / macOS 26, gate with `#available(iOS 26, macOS 26, *)` and provide the iOS 17 / macOS 14 fallback in the same diff:

```
--- Views/Toolbar.swift:21
+++ Views/Toolbar.swift:21
-        .background(.regularMaterial)
+        .modifier(ToolbarBackground())
+
+private struct ToolbarBackground: ViewModifier {
+    func body(content: Content) -> some View {
+        if #available(iOS 26, macOS 26, *) {
+            content.glassEffect()
+        } else {
+            content.background(.regularMaterial)
+        }
+    }
+}
```

## Hard constraints

- Every finding cites `file:line` (per H-9). No exceptions.
- Every Apple-doc reference includes a URL and a year ≥ 2024. If a rule's URL is older or missing, note "stale ref" next to it and flag for the rule book maintainer.
- Liquid Glass / `glassEffect()` / `Glass` materials are iOS 26 + macOS 26 only. If the deployment target is older, do NOT recommend them — recommend `.background(.regularMaterial)` or `.ultraThinMaterial` with the relevant HIG link.
- English only in the report. No emojis. No "I noticed", "great work", "consider".
- If the agent applied fewer than 40 rules (because the rule book has fewer applicable rules), state the exact count and why at the end of the report — one line.
- If the agent cannot find ≥ 5 real violations, say so. Don't invent findings to fill the table.

## When to refuse

- Rule book missing → stop, ask user to create or restore it.
- No target file specified and codebase has > 20 SwiftUI views → ask which view.
- User asks to review a UIKit/AppKit view → say "this skill covers SwiftUI; for UIKit/AppKit run a manual HIG pass" and stop.
