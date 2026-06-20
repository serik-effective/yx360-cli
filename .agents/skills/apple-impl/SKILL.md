---
name: apple-impl
description: Implement features in Swift / SwiftUI / AppKit / UIKit across iOS, iPadOS, macOS, watchOS, and visionOS. Defaults to Liquid Glass on OS 26+ with backward-compat shims for OS 16+. Use when the user asks to build, scaffold, wire up, or extend an Apple-platform feature in a Swift codebase — including new screens, view models, navigation, animations, or system integrations.
license: internal
---

# apple-impl

Implementation skill for Apple platforms. Routes through the project's Apple memory bank, pins the canonical 2024–2026 APIs, and hands off to design and runtime critics after the code lands.

## 0. Read this first (mandatory)

Before writing any Swift, read `@.memory-bank/apple-native/index.md` and follow its links. That index is the source of truth for:

- platform/OS floor of this specific project
- shared design tokens, theming, and component inventory
- navigation shell convention (split view vs. tab vs. stack)
- accessibility baseline
- testing + simulator conventions

If the file does not exist, stop and ask the user to seed it — do not invent project conventions.

## 1. Platform & version routing

Decide three things up-front and state them in the response header before the code:

1. Target platforms (iOS / iPadOS / macOS / watchOS / visionOS).
2. Deployment floor (the lowest OS the project supports).
3. Whether Liquid Glass is the default (iOS 26 + macOS 26 only) and what the fallback path looks like for older OS versions.

Default floor assumption when the memory bank is silent: iOS 16, macOS 13. Liquid Glass requires runtime gating, never blanket adoption.

## 2. Canonical APIs (pin these; do not regress)

Pick from this set unless the memory bank explicitly overrides.

**Navigation**
- `NavigationStack` + `NavigationPath` for push/pop flows (iOS 16+, macOS 13+).
- `NavigationSplitView` for sidebar shells on iPad/macOS/visionOS (iOS 16+, macOS 13+).
- Do not use `NavigationView` in new code. It is deprecated for new development since iOS 16 (2022).

**State**
- `@Observable` macro for view models (iOS 17+, macOS 14+, 2023). Use `@State var vm = MyModel()` to own it, `@Bindable` to write through it.
- Fall back to `ObservableObject` + `@Published` only when the floor is iOS 16 / macOS 13.
- `@Environment` for dependency-style injection. Never singletons for UI state.

**Liquid Glass (iOS 26 / macOS 26, 2025)**
- `.glassEffect(...)` on floating controls and bars.
- `GlassEffectContainer` to merge sibling glass surfaces.
- `.toolbar { ToolbarItem { ... } }` with system-provided glass styling — do not hand-roll a translucent bar.
- Reference: Apple HIG, "Materials" and "Liquid Glass" (2025): https://developer.apple.com/design/human-interface-guidelines/materials

**Backward-compat shim pattern (required when project floor < OS 26):**

```swift
extension View {
    @ViewBuilder
    func appGlass() -> some View {
        if #available(iOS 26, macOS 26, *) {
            self.glassEffect()
        } else {
            self.background(.regularMaterial, in: .rect(cornerRadius: 16))
        }
    }
}
```

Wrap every glass call site in an `#available` gate or an extension like the one above. Never ship a raw `.glassEffect` without a fallback unless the project floor is already OS 26.

**Concurrency**
- `async`/`await` and structured concurrency. `Task { @MainActor in ... }` for UI work spawned from non-isolated code.
- `@MainActor` on view models that publish UI state.
- Swift 6 language mode where the project allows it; otherwise Swift 5 with strict concurrency checking on.

**Lists & layout**
- `List` with `.listStyle(.insetGrouped)` on iOS, `.sidebar` on macOS sidebars.
- `LazyVGrid` / `LazyHGrid` for grids. `ScrollView` + `LazyVStack` only when `List` cannot express the layout.
- `Grid` (iOS 16+, macOS 13+) for static 2D forms.

**Animation**
- `withAnimation(.smooth)` / `.snappy` / `.bouncy` (iOS 17+, macOS 14+).
- `.matchedGeometryEffect` for shared-element transitions.
- `Animatable` + `AnimatableData` for custom interpolated values.

## 3. Forbidden patterns (block before merging)

- `!` force-unwrap on optionals. Use `guard let`, `if let`, `??`, or `try?`. The only acceptable `!` is `@IBOutlet` legacy and `fatalError`-equivalent crash points with an explicit comment explaining the invariant.
- UI work off the main actor. Mutating `@Published` / `@Observable` properties that drive views must happen on `@MainActor`. `DispatchQueue.main.async { ... }` inside an `async` function is a smell — use `await MainActor.run` or annotate the function.
- Ignoring Dynamic Type. Every text style must be a semantic font (`.body`, `.headline`, `.title2`, etc.) or `.system(size:..., relativeTo:)`. Hard-coded `.system(size: 14)` without a `relativeTo:` is forbidden. Test at `XXXL` and `AX1` minimum.
- Prose-only deliverables. The output of this skill is code (file paths + diffs or full files), not an essay.

## 4. Accessibility baseline (non-negotiable)

- Every interactive element has an `accessibilityLabel` if the visible label is an icon.
- Dynamic Type: see §3.
- `.accessibilityElement(children: .combine)` on composite rows so VoiceOver reads them as one item.
- Color is never the only signal — pair with shape, icon, or label.
- Reduce Motion: gate decorative animations behind `@Environment(\.accessibilityReduceMotion)`.
- Reference: Apple HIG, Accessibility (2024–2025): https://developer.apple.com/design/human-interface-guidelines/accessibility

## 5. Project hygiene

- Edit existing files over creating new ones (Working Agreement §Code rules).
- One feature = one folder under the project's existing convention. Don't invent a new architecture layer.
- No comments explaining *what* the code does. Comments only for *why* (invariant, workaround, version gate reason).
- No backwards-compat shims for *internal* code — only for OS-version gates.

## 6. Output format

Reply with, in this order:

1. **Header** — three lines: platforms, deployment floor, Liquid Glass posture.
2. **Plan** — 3–7 bullets, what files change and why.
3. **Code** — full file contents for new files, focused diffs for edits. Use absolute paths.
4. **Verification checklist** — runnable steps: build target, simulator, Dynamic Type setting, VoiceOver toggle.
5. **Handoff** — one line each:
   - "Run `/apple-design-critic` to review visual + HIG conformance."
   - "Run `/apple-simulator-debug` to verify on simulator and capture screenshots."

No marketing prose. No ritual apologies. No emojis.

## 7. References (cite these when justifying an API choice)

- Apple HIG, "Designing for iOS" (2024–2025): https://developer.apple.com/design/human-interface-guidelines/designing-for-ios
- Apple HIG, "Materials" / Liquid Glass (2025): https://developer.apple.com/design/human-interface-guidelines/materials
- "Meet Liquid Glass", WWDC25 (2025): https://developer.apple.com/videos/play/wwdc2025/219/
- "Discover Observation in SwiftUI", WWDC23 (2023): https://developer.apple.com/videos/play/wwdc2023/10149/
- "The SwiftUI cookbook for navigation", WWDC22 (2022): https://developer.apple.com/videos/play/wwdc2022/10054/
- Swift Concurrency, Apple Developer (2024): https://developer.apple.com/documentation/swift/concurrency

If a referenced API is newer than the project floor, cite the URL *and* the `#available` gate you used.
