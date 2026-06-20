# Meetily Native — macOS 26 (Tahoe) Design Research

Audience: senior engineer about to redo the Meetily Native shell so it stops looking like a Lovable demo and starts looking like an Apple app.
Targets: SwiftUI on macOS 26, Xcode 26, Liquid Glass design language.
Last reviewed: 2026-05-29.

---

## 1. Liquid Glass on macOS 26 — the actual rules

### 1.1 What `glassEffect` is for (and isn't)

`View.glassEffect(_:in:)` applies a Liquid Glass material to a custom view. Default shape is a capsule; pass a `Shape` to override. Signature usage:

```swift
Text("Hello")
    .padding()
    .glassEffect()                                  // regular, capsule
    .glassEffect(in: .rect(cornerRadius: 16))       // regular, rounded rect
    .glassEffect(.regular.tint(.orange).interactive())
```

**Apple's rule, paraphrased from WWDC25 session 323 and the HIG:** Liquid Glass is reserved for the **navigation layer that floats above your content**. Toolbars, sidebars, floating controls. Not lists, not cards, not "the hero card on the empty state". The OS already renders the bottom material under your window — when you wrap your entire content area in glass you stack glass on glass, which the system cannot sample correctly and which Apple explicitly tells you not to do ("Always avoid glass on glass. Stacking Liquid Glass elements can quickly make the interface feel cluttered.").

Plain English: **if you ship with Xcode 26, your toolbar already gets glass for free**. You almost never call `.glassEffect()` yourself on macOS unless you're building a custom floating control (HUD, floating action button, capsule overlay).

### 1.2 `glassEffect` vs `GlassEffectContainer`

Two sibling glass views render incorrectly side-by-side because glass cannot sample other glass. Fix: wrap them in a `GlassEffectContainer` so they share one sampling region and can morph between each other.

```swift
@Namespace private var ns

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

Rules: the `spacing:` parameter controls morph distance — glass elements closer than `spacing` blend; further apart, they stay separate. Use `glassEffectID(_:in:)` if you want shape morphing on state change. Use `glassEffectUnion(id:namespace:)` to merge non-adjacent siblings.

For toolbar buttons specifically, **you do not need a container** — `.toolbar { … }` already runs them through the system's shared material when you build with Xcode 26. The container is for your *own* floating clusters (HUD, mini-player, capsule).

### 1.3 Glass styles

| Style | When to use |
|---|---|
| `.regular` | Default. 95% of cases. Adapts to backdrop. |
| `.clear` | Only over busy media (image, video). Requires bold/bright foreground content and tolerates dimming. Without those, text becomes unreadable. |
| `.regular.tint(.color)` | Primary actions only — record button, "Save", destructive confirm. Tint communicates meaning, not decoration. |
| `.regular.interactive()` | Element responds to pointer (highlight on hover, press feedback). Apply only if user *interacts* with the glass surface. |

Modifier order doesn't matter: `.regular.tint(.orange).interactive()` == `.interactive().tint(.orange)`.

### 1.4 `ToolbarSpacer`

New on iOS/iPadOS/macOS 26. Two variants:

```swift
.toolbar {
    ToolbarItem { Button("Filter", systemImage: "line.3.horizontal.decrease") { } }
    ToolbarSpacer(.fixed)                         // small fixed gap, groups items
    ToolbarItem { Button("Sort", systemImage: "arrow.up.arrow.down") { } }
    ToolbarSpacer(.flexible)                      // pushes the next item to the trailing edge
    ToolbarItem(placement: .primaryAction) {
        Button("Record", systemImage: "record.circle") { }
    }
}
```

Why it exists: macOS 26 toolbars are now glass capsules. The spacer is what lets you **break one capsule into two** ("group A" + "group B") instead of one giant blob — same visual grammar as the new Safari / Mail / Notes toolbars. Use `.fixed` to separate related-but-distinct groups; `.flexible` to push trailing actions to the edge.

There is also `.sharedBackgroundVisibility(.hidden)` to opt a single toolbar item *out* of the shared glass capsule (rare — use when an item needs to look fully detached, e.g. an avatar).

### 1.5 The backdrop trap

The Apple-reference look has **real content** under glass: a list of notes, a song queue, a photo. The lavender→pink gradient in the current Meetily shell is the giveaway that it's a "glassmorphism" landing page, not a macOS app. Apple's own demos use:

- Photos → an actual photo.
- Music → an album artwork or the queue.
- Notes → the note list / a yellow paper background.
- Voice Memos → the waveform of the selected memo.

Quote from Conor Luddy's Liquid Glass reference, reflecting Apple's own design talks: *"Real Content vs. Gradients: Prefer real content beneath glass for authentic lensing and refraction effects. Gradients work as fallbacks when actual content is unavailable."*

**Rule for Meetily empty state:** no gradient. The window background should be the system window material; the empty state should be a centered `ContentUnavailableView` over that material, period. Glass shows up only when the user has a real list of meetings to lens through.

---

## 2. Which Apple app to copy

### 2.1 Layout inventory

| App | Sidebar width | Sidebar content | Detail | Floating element | Notes |
|---|---|---|---|---|---|
| **Voice Memos** | ~260–280 pt | Flat list of memos: title, date, duration. Search at top. | Big waveform top, transcript below, transport at bottom. | Big red circular record button anchored bottom-center. | Closest analog. List density: ~64 pt per row. |
| **Notes** | ~240 pt (sections) + ~280 pt (note list) | Three-column: Folders → Notes → Editor. Sort/Filter under the search field. | Editor with thin toolbar (format actions in a single glass capsule). | None. | Reference for hierarchical sidebar + sort controls. |
| **Reminders** | ~240 pt | Smart lists at top (Today, Scheduled, Flagged) as colored tiles + user lists below. | Custom rows with leading metadata (color, icon, check). | Quick-add capsule at bottom of list (glass). | Reference for "leading icon + multi-line cell" pattern. |
| **Music** | ~240 pt with sections (Apple Music / Library / Playlists) | Sectioned outline with `DisclosureGroup` look. | Library grid / Now Playing. | **Bottom transport bar** spanning the detail area (glass, full-width). Mini-player NSPanel as a separate window option. | Reference for the recording HUD. |

### 2.2 Recommendation

**Spiritual reference: Voice Memos for the screen, Music for the recording HUD.**

Why, in three lines:
1. Voice Memos is literally the same shape app — list of recordings on the left, big content area on the right with waveform/transcript and a transport bar at the bottom. You can copy 80% of the chrome 1:1.
2. The recording-active state needs a persistent surface (waveform + live transcript preview). Music's full-width bottom transport bar is the macOS-native pattern for "thing that stays visible while you navigate elsewhere". Drop Voice Memos' centered record button, take Music's bottom bar.
3. Both are first-party Tahoe apps — copying them is the fastest way to inherit Apple's defaults for spacing, density, and glass.

---

## 3. Telegram macOS — what makes it feel right

Telegram for macOS (the `overtake/TelegramSwift` lineage, now also the new desktop client) is the gold-standard third-party macOS chat client. What it does right:

| Aspect | Telegram does | What to copy |
|---|---|---|
| Sidebar width | ~300 pt (resizable, but defaults around 280–320). | Default Meetily sidebar to **300 pt**, allow user resize 240–400. |
| Row density | ~64 pt per chat row (avatar 36 pt + 2 lines of text). | Use the same — 60–68 pt per meeting row. Two lines: title + "yesterday · 14:23 · 8 speakers". |
| Dividers | **No dividers** between rows. Separation is whitespace + the selection highlight (rounded rect, inset 4–6 pt from sidebar edges). | Do this. `List { … }.listStyle(.sidebar)` already gives inset rounded selection on macOS 26. |
| Accent | System blue, but the unread badge and folder-tab tint use the user's chosen accent. | Stick with system `.tint(.accentColor)`. Don't ship a custom accent. |
| Title bar | Unified — toolbar and sidebar header sit on the same plane as the traffic lights. Sidebar has its own thin search field inline, not in the toolbar. | Use `.toolbar` + `.searchable(placement: .sidebar)`. Do not put search in the trailing toolbar. |
| Hover | Subtle background fill (~6% white in dark, ~4% black in light) appears on row hover. No scale, no glow. | Free with `List` on macOS — don't override. |
| Selection | Tinted rounded rect, full row, inset from sidebar edge. Selected text stays the same color (not white). | Default `.listStyle(.sidebar)` does this. |
| Animations | ~150 ms ease-in-out for selection. No spring, no bounce. | Use `.animation(.easeInOut(duration: 0.15), value: selection)` if you animate at all — usually you don't need to. |
| Window chrome | `.windowStyle(.hiddenTitleBar)` with toolbar drawn over the unified header area. Traffic lights inset ~20 pt from the top-leading. | Apply `.windowStyle(.hiddenTitleBar)` + `.windowToolbarStyle(.unified)` on the `WindowGroup`. |

The thing that makes Telegram feel native: **density**, **no decorative chrome**, and **content lives directly on the window material with no extra container around it**. That's the opposite of what the current Meetily shell does.

---

## 4. Component inventory for SwiftUI on macOS 26

### 4.1 `NavigationSplitView`

```swift
NavigationSplitView {
    MeetingsSidebar(selection: $selection)
        .navigationSplitViewColumnWidth(min: 240, ideal: 300, max: 400)
} detail: {
    if let id = selection {
        MeetingDetailView(id: id)
    } else {
        EmptyMeetingState()
    }
}
.navigationSplitViewStyle(.balanced)
```

| Style | When |
|---|---|
| `.balanced` (default) | Resizing the window resizes both columns proportionally. Use this for Meetily — it's what Voice Memos / Notes do. |
| `.prominentDetail` | Detail keeps full size, sidebar slides over it. Use for media-heavy apps (Photos in slideshow mode). **Don't use for Meetily.** |

Anti-patterns: do not put `.background(LinearGradient(...))` on the split view. Do not wrap it in a `ZStack` with a custom background. Let the system window material show through.

### 4.2 Sidebar list

```swift
List(meetings, selection: $selection) { meeting in
    NavigationLink(value: meeting.id) {
        MeetingRow(meeting: meeting)
    }
}
.listStyle(.sidebar)
.searchable(text: $query, placement: .sidebar, prompt: "Search meetings")
```

`.listStyle(.sidebar)` on macOS 26 gives you: rounded inset selection, hover highlight, correct row insets, glass material treatment. Don't override the row background — let it inherit.

`.searchable(placement: .sidebar)` is the macOS 26 way to put the search field at the top of the sidebar (instead of the toolbar). This is the Telegram / Mail / Notes pattern.

### 4.3 Toolbar

```swift
.toolbar(id: "main") {
    ToolbarItem(id: "new", placement: .primaryAction) {
        Button("New Recording", systemImage: "record.circle") {
            startRecording()
        }
    }
    ToolbarSpacer(.flexible)
    ToolbarItem(id: "engine", placement: .automatic) {
        Menu {
            Picker("Engine", selection: $engine) {
                Text("WhisperKit").tag(Engine.whisperKit)
                Text("SpeechAnalyzer").tag(Engine.speechAnalyzer)
            }
        } label: {
            Label(engine.title, systemImage: "waveform.badge.mic")
        }
    }
}
```

| Placement | Use for |
|---|---|
| `.primaryAction` | The one big action (New Recording). Becomes the tinted primary button. |
| `.automatic` | Everything else. The system picks the right spot per platform. |
| `.principal` | Center of the toolbar — for a title or scrubber. Avoid unless the toolbar genuinely needs a centerpiece. |
| `.navigation` | Back/forward, sidebar toggle. macOS adds the sidebar toggle automatically — don't add your own. |

Anti-patterns: don't add `.glassEffect()` to toolbar items. Toolbar already gets glass.

### 4.4 `ContentUnavailableView`

```swift
ContentUnavailableView {
    Label("No Meetings Yet", systemImage: "waveform")
} description: {
    Text("Start your first recording to see it here.")
} actions: {
    Button("New Recording") { startRecording() }
        .buttonStyle(.borderedProminent)
        .controlSize(.large)
}
```

This is Apple's standard empty state. Use it as the detail view when no meeting is selected and the list is empty. Use the search variant (`ContentUnavailableView.search(text:)`) when the user has typed a query with no results.

### 4.5 TipKit

```swift
struct StartRecordingTip: Tip {
    var title: Text { Text("Start your first recording") }
    var message: Text? { Text("Meetily captures both your mic and system audio.") }
    var image: Image? { Image(systemName: "record.circle") }
}

// In the view:
.popoverTip(StartRecordingTip(), arrowEdge: .top)
```

Use `.popoverTip` on macOS (not `TipView` inline) — Apple's pattern is a transient popover anchored to the actual control (the record button), dismissing on first use. Register tips at app start with `Tips.configure()`.

### 4.6 Floating recording HUD

SwiftUI doesn't have a native floating-panel modifier. The Apple-blessed pattern is `NSPanel` hosted via `NSHostingView`. Skeleton:

```swift
final class FloatingHUDPanel<Content: View>: NSPanel {
    init(rootView: Content) {
        super.init(
            contentRect: NSRect(x: 0, y: 0, width: 420, height: 64),
            styleMask: [.nonactivatingPanel, .fullSizeContentView, .borderless],
            backing: .buffered,
            defer: false
        )
        isFloatingPanel = true
        level = .floating
        collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
        titleVisibility = .hidden
        titlebarAppearsTransparent = true
        isMovableByWindowBackground = true
        hasShadow = true
        backgroundColor = .clear
        contentView = NSHostingView(rootView: rootView.ignoresSafeArea())
    }
    override var canBecomeKey: Bool { true }
}
```

Inside the SwiftUI root view, use a `Capsule` shape with `.glassEffect(.regular.interactive(), in: .capsule)` — this is one of the few legitimate uses of `glassEffect` on macOS. Anchor the panel programmatically to `NSScreen.main.visibleFrame` bottom-center on show.

Alternative: a second `WindowGroup` with `.windowStyle(.hiddenTitleBar)` + `.defaultPosition(.bottom)` + `.windowLevel(.floating)` (new on macOS 14+). Simpler if you don't need `nonactivatingPanel` behavior. **Use the WindowGroup approach first** — only drop down to NSPanel if you need it to coexist with a fullscreen Keynote/Zoom.

### 4.7 SF Symbols (SF Symbols 7, macOS 26)

| Use | Symbol |
|---|---|
| Record | `record.circle` (filled when recording: `record.circle.fill`, tint `.red`) |
| Stop | `stop.fill` |
| Pause | `pause.fill` |
| Play | `play.fill` |
| Waveform (idle) | `waveform` |
| Waveform (live) | `waveform.badge.mic` — use the new animation: `.symbolEffect(.variableColor.iterative)` |
| Transcript | `text.bubble` or `quote.bubble` |
| Speaker chip | `person.wave.2.fill` |
| New folder | `folder.badge.plus` |
| Engine picker | `waveform.badge.mic` |
| Settings | `gearshape` |
| Search | (don't add — `.searchable` provides it) |

Use `.symbolRenderingMode(.hierarchical)` by default. Use `.palette` with explicit colors only when a symbol carries state (red record dot on hover, etc.). Use `.symbolEffect(.bounce)` sparingly for transient feedback.

---

## 5. Colors & typography

### 5.1 Type

| Where | Font |
|---|---|
| Body text (transcript, lists) | `.system(.body)` → SF Pro Text. Don't override. |
| Sidebar row title | `.headline` (this is the standard sidebar weight). |
| Sidebar row subtitle | `.subheadline` + `.foregroundStyle(.secondary)`. |
| Big numbers (duration counter, time elapsed) | `.system(.title, design: .rounded, weight: .medium).monospacedDigit()`. **Rounded** is correct here — Apple uses it for timers, scoreboards, transient counters. Voice Memos uses rounded for the duration. |
| Window title in detail | `.system(.title2, weight: .semibold)`. |

Rule: SF Pro Rounded **only for numerical / transient / playful surfaces** (timers, badges). Everywhere else SF Pro Text via `.system(...)`. Never set the font family explicitly.

### 5.2 Color

| Token | Where |
|---|---|
| `.primary` | Headlines, transcript body. |
| `.secondary` | Timestamps, metadata, "5 min ago". |
| `.tertiary` | Disabled rows, placeholder hints. |
| `Color.accentColor` | Selection, primary button. |
| `.red` | Record button, recording-active waveform. |
| Materials: `.regularMaterial`, `.thickMaterial` | Behind floating content if you need it. **Sidebar is already material — don't add more.** |

### 5.3 Accent recommendation

**Use system blue.** Reasoning:
- Audio apps that ship a custom red accent (Voice Memos red, Logic orange) only use it on the *record affordance*, not as the app's accent. The selection highlight, link colors, and form controls stay system.
- Custom accents break user expectations on macOS where users can globally pick their accent from System Settings → Appearance.
- The red belongs to the record button (`.tint(.red)` on that one control) and to the live waveform during recording. That's it.

Do not ship a coral or mint app-wide accent. It's a Lovable tell.

---

## 6. What to remove from the current shell

Blunt list. Each item is a thing that says "AI demo" instead of "native app".

1. **Kill the `LinearGradient`.** No purple, no pink, no gradient backdrop. The macOS window already has a material — use it. `ZStack { LinearGradient(...) ... }` becomes `NavigationSplitView { ... } detail: { ... }` with no background at all.
2. **Kill the centered "hero" glass card.** Apple doesn't put titles in glass cards. The empty state is `ContentUnavailableView`, anchored center, no background.
3. **Kill `.glassEffect(in: .rect(cornerRadius: 28))` on the welcome card.** That's glass on glass (you're inside a window that already has a material backdrop). Remove the modifier entirely.
4. **Stop centering "Meetily Native" as the headline.** The window title bar already shows the app name. If you want a headline in the detail view, it's `ContentUnavailableView`'s `Label`, not a 48 pt centered title.
5. **Add a sidebar.** Even when empty. The `NavigationSplitView` with an empty list + `ContentUnavailableView` already looks like an Apple app on day one.
6. **Add a real toolbar.** At least: trailing `.primaryAction` "New Recording" button. The toolbar gets glass for free; you don't need to decorate the window with it manually.
7. **Use `.windowStyle(.hiddenTitleBar)` + `.windowToolbarStyle(.unified)`.** This is what gives you the Telegram/Mail "traffic lights sit on the same plane as the toolbar" look. Default style looks dated next to Tahoe apps.
8. **Drop any custom font.** No "Inter", no "Geist", no rounded-everywhere. `.system(...)`.
9. **Drop any explicit `.background(Color...)`.** Materials cascade from the window. Anywhere you set an explicit color you're fighting the system.
10. **The 28 pt corner radius is wrong** for the card and irrelevant once the card is gone. macOS 26 uses smaller corner radii for cards (12–16 pt) and concentric-container radii for inner elements (`.containerConcentric`).

---

## 7. Recommendation

### 7.1 Reference target

**Make Meetily look like Voice Memos with a Music-style bottom transport bar during recording, and a Notes-style sidebar list.** That's the one sentence to write on a Post-it.

### 7.2 The new empty-state root, sketched

```swift
import SwiftUI

@main
struct MeetilyApp: App {
    @State private var selection: Meeting.ID?

    var body: some Scene {
        WindowGroup("Meetily") {
            RootView(selection: $selection)
        }
        .windowStyle(.hiddenTitleBar)
        .windowToolbarStyle(.unified(showsTitle: true))
        .defaultSize(width: 1100, height: 720)
    }
}

struct RootView: View {
    @Binding var selection: Meeting.ID?
    @State private var query = ""
    @Environment(MeetingStore.self) private var store

    var body: some View {
        NavigationSplitView {
            List(store.filtered(query), selection: $selection) { meeting in
                NavigationLink(value: meeting.id) {
                    MeetingRow(meeting: meeting)
                }
            }
            .listStyle(.sidebar)
            .navigationSplitViewColumnWidth(min: 240, ideal: 300, max: 400)
            .searchable(text: $query, placement: .sidebar, prompt: "Search meetings")
            .navigationTitle("Meetings")
        } detail: {
            if let id = selection, let meeting = store.meeting(id) {
                MeetingDetailView(meeting: meeting)
            } else if store.isEmpty {
                ContentUnavailableView {
                    Label("No Meetings Yet", systemImage: "waveform")
                } description: {
                    Text("Start your first recording to capture and transcribe a meeting locally.")
                } actions: {
                    Button("New Recording") { store.startRecording() }
                        .buttonStyle(.borderedProminent)
                        .controlSize(.large)
                        .tint(.red)
                }
            } else {
                ContentUnavailableView("Select a Meeting",
                    systemImage: "sidebar.left",
                    description: Text("Pick a meeting from the sidebar to view its transcript."))
            }
        }
        .toolbar(id: "main") {
            ToolbarItem(id: "new", placement: .primaryAction) {
                Button("New Recording", systemImage: "record.circle") {
                    store.startRecording()
                }
            }
            ToolbarSpacer(.flexible)
            ToolbarItem(id: "settings", placement: .automatic) {
                SettingsLink {
                    Image(systemName: "gearshape")
                }
            }
        }
    }
}
```

What this gives you on first launch with no recordings: a real sidebar with a search field, a real toolbar with a tinted New Recording button, an empty detail area with a `ContentUnavailableView` and a prominent red "New Recording" button. Zero gradients, zero custom backgrounds, zero `glassEffect` calls. It will look indistinguishable from a stock Apple app the moment you build it.

### 7.3 Three visual directions to pick between

| Direction | What it is | Tradeoff |
|---|---|---|
| **A. Voice Memos clone** | One-column sidebar (flat meeting list, search at top) + detail with big waveform / transcript split / transport bar at the bottom of the detail area. Record button is a red circle anchored bottom-center of detail. | Fastest to ship. Looks unmistakably Apple. Less flexible if you want folders/projects later. |
| **B. Notes clone** | Three-column: Folders/Projects → Meetings → Detail. Sort & filter under the sidebar search. | More structure for power users. More chrome to design correctly. Sidebar feels heavier — wrong vibe if Meetily is "fire and forget recording". |
| **C. Music sidebar + floating HUD** | Sectioned sidebar (Recent / Pinned / All) + detail. Recording state lives in a **separate floating capsule** (`NSPanel`) that stays on top while the user is in Zoom/Teams. | Best UX during the actual meeting — Meetily stays visible without stealing focus. More plumbing (NSPanel bridge, multi-window state). Highest payoff if Meetily is used *during* the call, not just after. |

My pick: **A for v1**, evolve to **C** once recording is solid. Direction A is one weekend of work and ships a fully-native shell. Direction C is the differentiator — but you don't need it on day one.

---

## Do this Monday

1. Delete the `LinearGradient`, the centered glass card, and the "Meetily Native" hero in `RootView.swift` (or whatever file holds the current shell).
2. Drop in the `NavigationSplitView` + `ContentUnavailableView` skeleton from §7.2. Build. Run.
3. Set `.windowStyle(.hiddenTitleBar)` and `.windowToolbarStyle(.unified(showsTitle: true))` on the `WindowGroup`.
4. Add ten fake meetings to a `MeetingStore` so the sidebar populates. Verify search works.
5. Wire the toolbar `New Recording` button to *something* (a `print`, a sheet — anything). Confirm the red tint shows on the primary button.
6. Screenshot it next to Voice Memos. They should be siblings.

Stop there for Monday. Don't add a HUD, don't pick an accent, don't add TipKit, don't `glassEffect` anything. The hard part is getting the shell to disappear into the OS — once it does, every component you add later inherits the look for free.

---

## Sources

- [WWDC25 — Build a SwiftUI app with the new design (session 323)](https://developer.apple.com/videos/play/wwdc2025/323/)
- [WWDC25 — What's new in SwiftUI (session 256)](https://developer.apple.com/videos/play/wwdc2025/256/)
- [Apple Developer — `glassEffect(_:in:)`](https://developer.apple.com/documentation/swiftui/view/glasseffect(_:in:))
- [Apple Developer — `GlassEffectContainer`](https://developer.apple.com/documentation/swiftui/glasseffectcontainer)
- [Apple Developer — `NavigationSplitView`](https://developer.apple.com/documentation/swiftui/navigationsplitview)
- [Apple Developer — Liquid Glass technology overview](https://developer.apple.com/documentation/TechnologyOverviews/liquid-glass)
- [Apple HIG](https://developer.apple.com/design/human-interface-guidelines/)
- [Apple Newsroom — new software design announcement (June 2025)](https://www.apple.com/newsroom/2025/06/apple-introduces-a-delightful-and-elegant-new-software-design/)
- [Conor Luddy — Liquid Glass SwiftUI reference](https://github.com/conorluddy/LiquidGlassReference)
- [Artem Novichkov — Xcode 26 system prompts: Implementing Liquid Glass](https://github.com/artemnovichkov/xcode-26-system-prompts/blob/main/AdditionalDocumentation/SwiftUI-Implementing-Liquid-Glass-Design.md)
- [Artem Novichkov — Xcode 26 system prompts: New Toolbar Features](https://github.com/artemnovichkov/xcode-26-system-prompts/blob/main/AdditionalDocumentation/SwiftUI-New-Toolbar-Features.md)
- [Liquid Glass in Swift — official best practices (dev.to)](https://dev.to/diskcleankit/liquid-glass-in-swift-official-best-practices-for-ios-26-macos-tahoe-1coo)
- [WWDC notes — session 323](https://wwdcnotes.com/documentation/wwdcnotes/wwdc25-323-build-a-swiftui-app-with-the-new-design/)
- [Swift with Majid — What's new in SwiftUI after WWDC25](https://swiftwithmajid.com/2025/06/10/what-is-new-in-swiftui-after-wwdc25/)
- [TahoeMenuDemo — macOS 26 menu bar reference](https://github.com/sjhooper/TahoeMenuDemo)
- [Cindori — SwiftUI floating panel on macOS](https://cindori.com/developer/floating-panel)
- [TelegramSwift — historical macOS client source](https://github.com/overtake/TelegramSwift)
- [MacStories — macOS Tahoe overview](https://www.macstories.net/news/macos-tahoe-the-macstories-overview/)
- [9to5Mac — SF Symbols 7 beta](https://9to5mac.com/2025/06/11/apple-releases-sf-symbols-7-beta/)
- [Apple Support — Voice Memos User Guide for Mac](https://support.apple.com/guide/voice-memos/welcome/mac)
