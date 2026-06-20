---
name: swiftui-macos-26
description: >
  Native macOS app development on macOS 26 (Tahoe) with SwiftUI + Liquid
  Glass. Carries Apple's HIG rules for the new design language, the
  NavigationSplitView/Toolbar/ContentUnavailableView component inventory,
  reference-app layout numbers (Voice Memos, Notes, Music, Telegram macOS),
  the anti-patterns that make a SwiftUI shell look like a Lovable demo, and
  the xcodegen + swift-tools-version 6.2 build setup. Use when the user
  asks to build, scaffold, redesign, or "make it look native" for a macOS
  app — especially anything mentioning Liquid Glass, glassEffect, macOS 26
  / Tahoe, SwiftUI shell, sidebar layout, Voice Memos, or
  NavigationSplitView. Skip for iOS-only, AppKit-only, or pre-macOS-26
  work.
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, AskUserQuestion
---

# Native macOS 26 (Tahoe) SwiftUI Development

This skill is the working memory for building macOS apps that sit inside
the new design language without fighting the OS. It encodes the rules,
patterns, and reference-app numbers extracted from Apple HIG, WWDC25
session 323 ("Build a SwiftUI app with the new design"), and a hands-on
review of Voice Memos / Notes / Music / Telegram macOS on Tahoe.

The full source doc is in `references/design-research.md` — read it
end-to-end before redesigning a shell. The bullet list below is the
short reference card.

---

## When to invoke this skill

- "Make this macOS app look native" / "looks like a Lovable demo".
- "Build a SwiftUI shell for macOS".
- "How do I use Liquid Glass / `glassEffect` / `GlassEffectContainer`?"
- "What's the right NavigationSplitView / toolbar / empty state for macOS?"
- Anything mentioning Voice Memos, Music sidebar, or Telegram macOS as a
  reference.
- Scaffolding a new project with `xcodegen` for macOS 26.

**Do NOT invoke for:** iOS-only work, AppKit-only work, pre-macOS-26
targets (Sonoma / Sequoia have a different material model), or generic
SwiftUI questions that don't touch the shell.

---

## Top 10 rules (the short version)

1. **Stop wrapping content in `.glassEffect`.** The window already has a
   material backdrop. Toolbars get glass for free with Xcode 26. Glass is
   for floating navigation elements (HUDs, capsules), not cards.
2. **No decorative gradients behind glass.** The lavender→pink "hero
   gradient + centered card" pattern is the #1 Lovable tell. Real content
   under glass (lists, photos, waveforms) is the Apple pattern.
3. **Use `NavigationSplitView` + `.navigationSplitViewStyle(.balanced)`**
   for any productivity app (mail, notes, voice memos, chat). Never
   `.prominentDetail` unless you're Photos.
4. **`.windowStyle(.hiddenTitleBar)` + `.windowToolbarStyle(.unified(showsTitle: true))`**
   on the `WindowGroup` give the Telegram/Mail "traffic lights on the
   same plane as the toolbar" look. Default style looks dated.
5. **`.listStyle(.sidebar)`** gives you inset rounded selection, hover
   highlight, correct row insets, glass treatment — for free. Don't
   override row backgrounds.
6. **`.searchable(text:, placement: .sidebar, prompt:)`** — search lives
   at the top of the sidebar, not in the trailing toolbar. Notes / Mail /
   Telegram pattern.
7. **`ContentUnavailableView`** is Apple's empty-state component. Use it
   when the list is empty or when nothing is selected. Use
   `ContentUnavailableView.search(text:)` for "no results".
8. **`ToolbarSpacer(.flexible)`** splits the toolbar glass capsule into
   groups; `.primaryAction` is the one big tinted button (Record, Send,
   New). Don't add `.glassEffect` to toolbar items.
9. **System blue accent.** Custom app-wide accents (coral, mint) break
   user expectations on macOS. Reserve `.tint(.red)` for the record
   button itself, not the whole app.
10. **SF Pro Rounded only for numerical / transient surfaces** (timers,
    counters, badges). Everywhere else, plain `.system(...)` — never set
    a font family explicitly.

---

## The minimum-viable native shell (copy this verbatim)

```swift
import SwiftUI

@main
struct MyApp: App {
    @State private var store = MyStore()

    var body: some Scene {
        WindowGroup("MyApp") {
            RootView()
                .environment(store)
        }
        .windowStyle(.hiddenTitleBar)
        .windowToolbarStyle(.unified(showsTitle: true))
        .defaultSize(width: 1100, height: 720)

        Settings { SettingsView() }
    }
}

struct RootView: View {
    @Environment(MyStore.self) private var store
    @State private var selection: MyItem.ID?
    @State private var query = ""

    var body: some View {
        NavigationSplitView {
            List(filtered, selection: $selection) { item in
                NavigationLink(value: item.id) {
                    MyRow(item: item)
                }
            }
            .listStyle(.sidebar)
            .navigationSplitViewColumnWidth(min: 260, ideal: 300, max: 400)
            .searchable(text: $query, placement: .sidebar, prompt: "Search items")
            .navigationTitle("Items")
        } detail: {
            if let id = selection, let item = store.item(id) {
                DetailView(item: item)
            } else if store.isEmpty {
                ContentUnavailableView {
                    Label("No Items Yet", systemImage: "tray")
                } description: {
                    Text("Create your first item to get started.")
                } actions: {
                    Button("New Item", systemImage: "plus") { store.create() }
                        .buttonStyle(.borderedProminent)
                        .controlSize(.large)
                }
            } else {
                ContentUnavailableView("Select an Item",
                    systemImage: "sidebar.left",
                    description: Text("Pick one from the sidebar."))
            }
        }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("New", systemImage: "plus") { store.create() }
            }
            ToolbarSpacer(.flexible)
            ToolbarItem {
                SettingsLink { Image(systemName: "gearshape") }
            }
        }
    }

    private var filtered: [MyItem] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return store.items }
        let needle = trimmed.lowercased()
        return store.items.filter { $0.title.lowercased().contains(needle) }
    }
}
```

---

## Reference app numbers (Tahoe defaults)

| App | Sidebar width | Row height | Density notes |
|---|---|---|---|
| Voice Memos | 260–280 pt | ~64 pt | List + search at top; waveform fills detail; transport at bottom |
| Notes | 240 + 280 pt (3-col) | ~56 pt | Sort/Filter under search |
| Reminders | 240 pt | ~48 pt | Leading icon + multi-line cell |
| Music | 240 pt | ~36 pt | Sectioned outline; full-width bottom transport bar |
| Telegram macOS | ~300 pt | ~64 pt | No dividers; inset rounded selection; ~150 ms easeInOut anim |

Use 300 pt as the default sidebar ideal for chat/list productivity apps;
260 pt for compact single-column lists; 240 pt for sectioned sidebars.

---

## SF Symbols 7 quick reference

| Use | Symbol |
|---|---|
| Record | `record.circle` / `record.circle.fill` + `.tint(.red)` |
| Stop | `stop.fill` |
| Pause | `pause.fill` |
| Waveform idle | `waveform` |
| Waveform live | `waveform.badge.mic` + `.symbolEffect(.variableColor.iterative)` |
| Transcript | `text.bubble` or `quote.bubble` |
| Speaker chip | `person.wave.2.fill` |
| Engine picker | `waveform.badge.mic` |
| Settings | `gearshape` |
| Search | (don't add — `.searchable` provides one) |

Default to `.symbolRenderingMode(.hierarchical)`. Use `.palette` only when
a symbol carries state.

---

## Liquid Glass — the API cheat sheet

```swift
// 95% of cases — never write this on macOS unless the view is a custom
// floating control (HUD, capsule, mini-player).
.glassEffect()                                        // regular, capsule
.glassEffect(in: .rect(cornerRadius: 16))             // regular, rounded rect
.glassEffect(.regular.tint(.red).interactive())       // tinted + hover

// Merge sibling glass surfaces into one shared sampling region:
GlassEffectContainer(spacing: 20) {
    HStack(spacing: 12) {
        Button { } label: { Image(systemName: "play.fill") }
            .glassEffect()
            .glassEffectID("play", in: ns)
        Button { } label: { Image(systemName: "stop.fill") }
            .glassEffect()
            .glassEffectID("stop", in: ns)
    }
}
```

`.toolbar { ... }` does NOT need a container — system applies it.
`GlassEffectContainer` is for your own clusters.

---

## Floating recording HUD pattern

Two ways:

1. **`WindowGroup` + `.windowLevel(.floating)`** (macOS 14+, preferred for
   most cases — simpler, all SwiftUI).
2. **`NSPanel` hosted via `NSHostingView`** (drop-down when you need
   `nonactivatingPanel`, e.g. to stay non-keyable during a Zoom call).

Inside the HUD, a `Capsule().glassEffect(.regular.interactive(), in: .capsule)`
is one of the few legitimate uses of `glassEffect` on macOS.

---

## Build setup (xcodegen + swift-tools 6.2)

```yaml
# Project.yml
name: MyApp
options:
  deploymentTarget:
    macOS: "26.0"
settings:
  base:
    SWIFT_VERSION: "6.0"
    SWIFT_STRICT_CONCURRENCY: complete
packages:
  MyCore:
    path: Packages/MyCore
targets:
  MyApp-macOS:
    type: application
    platform: macOS
    deploymentTarget: "26.0"
    sources:
      - path: Apps/MyApp-macOS
        excludes:
          - Resources/Info.plist
          - Resources/MyApp.entitlements
    info:
      path: Apps/MyApp-macOS/Resources/Info.plist
      properties:
        LSMinimumSystemVersion: "26.0"
        NSMicrophoneUsageDescription: ...
        NSAudioCaptureUsageDescription: ...
    entitlements:
      path: Apps/MyApp-macOS/Resources/MyApp.entitlements
      properties:
        com.apple.security.app-sandbox: true
        com.apple.security.device.audio-input: true
        com.apple.security.device.system-audio-capture: true
        com.apple.security.network.client: true
    dependencies:
      - package: MyCore
        product: MyCoreUI
```

Run `xcodegen generate` after **every** source file add/remove —
xcodeproj is regenerated, not hand-edited.

```swift
// Package.swift — must be 6.2 to reference .macOS(.v26)
// swift-tools-version: 6.2
.macOS(.v26)
```

---

## Common gotchas

- **`xcodebuild` complains "requires Xcode but active is CommandLineTools"** →
  `sudo xcode-select --switch /Applications/Xcode.app/Contents/Developer`
  once, or prefix the command with
  `DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer`.
- **SourceKit errors in the editor but `swift build` / `xcodebuild`
  works** → SourceKit's standalone parse can't resolve cross-file refs
  in app targets reliably. Trust the actual build.
- **Added a .swift file → "Cannot find type X in scope"** → `xcodegen
  generate` wasn't re-run. xcodegen scans the source tree; new files
  aren't visible to xcodebuild until the project is regenerated.
- **`.v26 is unavailable`** → swift-tools-version is too old. Bump to
  6.2 in `Package.swift`.
- **xcodegen overwrites your Info.plist / entitlements** → use the
  `info: { properties: ... }` and `entitlements: { properties: ... }`
  blocks in Project.yml; don't hand-write them at the path xcodegen
  manages.

---

## The 5 anti-patterns to grep for during review

1. `LinearGradient` as a window or detail background — almost always wrong.
2. `.glassEffect` on anything that isn't a custom floating control.
3. A centered "hero card" with the app name as the empty state.
4. Custom accent colors applied app-wide (`.tint(.coral)`, `.tint(.mint)`).
5. Explicit font families (`.font(.custom("Inter", size: 16))`,
   `.font(.system(... design: .rounded))` on body text).

If any of these are present in a new shell, redirect the author to
section 6 ("What to remove from the current shell") of
`references/design-research.md` before generating code.

---

## Pick a visual direction (decision aid)

| Direction | Reference app | Use when |
|---|---|---|
| **A. Voice Memos clone** | Voice Memos | Single-flat-list of items + big detail view. Fastest, most native. **Default for any new recording / capture app.** |
| **B. Notes clone** | Notes | Hierarchical sidebar (folders → items → editor). Use when users genuinely need folders / projects. |
| **C. Music sidebar + floating HUD** | Music | Sectioned sidebar; recording state lives in a separate floating capsule that stays on top during meetings. Highest payoff for "stays useful during the call". |

Default to A for v1; evolve to C once recording is solid.

---

## The "do this Monday" checklist for a new shell

1. Delete any decorative gradient, hero card, and existing
   `.glassEffect()` calls.
2. Drop in the minimum-viable `NavigationSplitView` from above.
3. Set `.windowStyle(.hiddenTitleBar)` + `.windowToolbarStyle(.unified(showsTitle: true))`
   on `WindowGroup`.
4. Seed an in-memory store with 8–10 fake items so the sidebar density
   is reviewable.
5. Wire the toolbar `.primaryAction` button to a `print` — confirm tint
   renders.
6. Screenshot next to Voice Memos. They should be siblings.

Stop there. Don't add a HUD, accent, TipKit, or any glassEffect calls.
The hard part is getting the shell to disappear into the OS; once it
does, every component you add later inherits the look for free.

---

## See also

- `references/design-research.md` — full 2400-word source doc with
  detailed component patterns, Telegram macOS numbers, and citations.
- Apple HIG: https://developer.apple.com/design/human-interface-guidelines/macos
- WWDC25 session 323: https://developer.apple.com/videos/play/wwdc2025/323/
