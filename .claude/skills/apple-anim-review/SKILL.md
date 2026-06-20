---
name: apple-anim-review
description: Frame-by-frame review of an Apple-platform animation or transition. Captures a simulator recording, extracts ~150ms key frames, and grades the motion against Apple HIG and the project's animation playbook. Use when the user says "review this animation", "review this transition", "is this transition janky", or asks to grade motion quality on iOS/iPadOS/macOS/visionOS.
license: MIT
argument-hint: "[screen or transition name]"
metadata:
  version: "1.0"
  scope: "Apple platforms — SwiftUI + UIKit/AppKit. iOS 17+, macOS 14+. Liquid Glass review only on iOS 26 / macOS 26."
---

Review a single animation or transition the way a Human Interface team would: from a real recording on a real simulator, frame by frame, then trace each defect to a file:line fix.

## Hard preconditions

1. Read `@.memory-bank/apple-native/animation.md` BEFORE doing anything else. That file is the project-specific playbook (allowed curves, banned patterns, matched-geometry conventions). Cite it in findings as `.memory-bank/apple-native/animation.md:LINE`.
2. If the file is missing, stop and tell the user. Do not invent rules.
3. Verify the deployment target before grading. Liquid Glass APIs (`glassEffect`, `.glassBackgroundEffect`, GlassEffectContainer) are iOS 26 / macOS 26 only — flag any usage on lower targets as a defect, and require an `if #available(iOS 26, macOS 26, *)` fallback to a pre-26 visual (material, opacity, or shape).
4. If the diff touches no animation, transition, gesture-driven motion, or `withAnimation` / `Animation` / `matchedGeometryEffect` / `PhaseAnimator` / `KeyframeAnimator` / `UIView.animate` / `CATransaction` / `NSAnimationContext`, say so and stop — do not fabricate animation findings.

## Step 1 — Capture

Ask the user to drive the interaction manually so the recording reflects real input timing. Script:

> Boot the simulator to the screen right before the animation. Tell me when you are ready. I will start recording, then you perform the transition once at normal speed. After it settles, tell me to stop.

Capture flow:

1. Confirm a booted simulator with `mcp__xcodebuildmcp__list_sims` (filter `onlyAvailable: true`) and pick the booted one. If none, ask the user to boot via `mcp__xcodebuildmcp__boot_sim`.
2. Start `mcp__xcodebuildmcp__record_sim_video` with the simulator UUID. Tell the user "recording — go".
3. When the user says stop, end the recording. The tool returns the `.mp4` / `.mov` path.
4. Save the path. Do not delete it until the review is published.

If the user cannot drive it (CI, no hands), refuse to fake a recording. Ask for an existing video file path instead.

## Step 2 — Extract frames

Use `ffmpeg` via Bash. Default cadence is 1 frame per 150 ms (≈6.66 fps); for fast snap transitions (<400 ms total) drop to 80 ms (12.5 fps).

```
ffmpeg -hide_banner -loglevel error -i <video> \
  -vf "fps=1000/150,scale=iw/2:-1" \
  -start_number 0 \
  <out_dir>/f_%03d.png
```

Notes:
- `scale=iw/2:-1` keeps frames small enough to inline-review without losing edge detail.
- Verify ffmpeg exists (`command -v ffmpeg`); if absent, instruct `brew install ffmpeg` and stop.
- Write frames into `swarm-report/anim-<slug>-<YYYY-MM-DD>/frames/`. Keep the source video next to them.

Also extract a duration probe to ground the review in real numbers:

```
ffprobe -v error -show_entries format=duration -of csv=p=0 <video>
```

Record the duration in milliseconds and the resulting frame count in the report header.

## Step 3 — Read frames

Read every PNG with the `Read` tool, in order. For each frame, write one line of observation (≤120 chars). Look for:

- Jolts — discontinuity in position, scale, or alpha between adjacent frames that should be continuous.
- Hitches — repeated frames where motion should still be progressing (capture cadence is even, so a frozen frame = the renderer missed a vsync).
- Easing mistakes — linear ramps where Apple HIG calls for `.smooth`, `.snappy`, or `spring(response:dampingFraction:)`; over-bouncy springs on dismissals; ease-in on entrances (entrance wants ease-out).
- Missing `matchedGeometryEffect` — when an element appears to fade out and a sibling fades in at a different position, but they are visually "the same thing". Cross-fade between same-identity elements is the tell.
- Broken interruption — if the user reverses mid-animation (you will need a second recording for this), the motion should retarget from the current velocity, not snap to the start. Look for a hard reset to the initial transform.
- Liquid Glass misuse (iOS 26 / macOS 26 only) — glass layers animating opacity instead of `glassEffect(_:in:)` morph, or two glass surfaces overlapping without a `GlassEffectContainer` (causes z-fighting and double-tint).
- Reduce Motion compliance — if the project supports it, verify the animation has a non-motion fallback path; flag if absent.

## Step 4 — Cross-reference to code

For every defect, open the relevant SwiftUI/UIKit file and locate the exact construct. Acceptable identifiers:

- `withAnimation(.smooth) { ... }`
- `.animation(.spring(response: 0.4, dampingFraction: 0.8), value: state)`
- `.transition(.move(edge: .bottom).combined(with: .opacity))`
- `.matchedGeometryEffect(id:, in:)`
- `UIView.animate(withDuration:delay:options:)`
- `CABasicAnimation` / `CASpringAnimation`
- `withAnimation` inside a gesture's `onChanged` / `onEnded`

Cite as `path/to/File.swift:LINE`. No "somewhere in this view".

## Step 5 — Output

Write the report to `swarm-report/anim-<slug>-<YYYY-MM-DD>.md`. English only. Structure:

```
# Animation review — <screen/transition>

Recording: swarm-report/anim-<slug>-<date>/source.mp4 (duration: <ms> ms, frames: <N>)
Cadence: <ms> ms/frame
Deployment target: iOS <X> / macOS <Y>
Memory bank rules consulted: .memory-bank/apple-native/animation.md

## Verdict
<one of: ship / fix-before-ship / redo>

## Per-frame critique
| Frame | t (ms) | Observation |
|-------|--------|-------------|
| f_000 | 0      | <obs>       |
| f_001 | 150    | <obs>       |
| ...   | ...    | ...         |

## Defects → fixes
1. **Jolt at f_004 → f_005** (≈600 → 750 ms). Scale jumps from ~0.92 to 1.04 with no in-between.
   - Cause: `.animation(.easeInOut(duration: 0.3), value: isExpanded)` on a spring-shaped target.
   - Fix: replace with `.spring(response: 0.42, dampingFraction: 0.82)` at `Sources/Detail/CardView.swift:87`.
   - Rule: `.memory-bank/apple-native/animation.md:LINE` ("springs for size/position, curves for opacity only").
   - Apple ref: https://developer.apple.com/documentation/swiftui/animation (2024).
2. **Missing matchedGeometryEffect (f_002 vs f_003)**. The avatar cross-fades between list and detail.
   - Fix: add `.matchedGeometryEffect(id: user.id, in: heroNS)` on both sites at `Sources/List/Row.swift:42` and `Sources/Detail/Header.swift:31`.
3. ...

## Liquid Glass note
<if iOS 26 / macOS 26: cite glassEffect usage and any GlassEffectContainer issue; else: "n/a, target below 26">

## Backward-compat
<for any 26-only API used, show the `if #available` branch + fallback>
```

## Citation discipline

- Every code claim: `file:line`. Every rule claim: `.memory-bank/...:line`. Every Apple-doc claim: a developer.apple.com URL with year ≥ 2024.
- No claim about a frame without naming the frame file (`f_007`) and its timestamp.
- If you cannot verify a claim, drop it. Do not soften it.

## Refusals

- Refuse to review a still screenshot — animations need motion. Ask for a recording.
- Refuse to grade against vibes. If `animation.md` does not cover a case and Apple HIG is silent, say so explicitly and mark the finding as "subjective".
- Refuse to recommend Liquid Glass APIs on targets below iOS 26 / macOS 26. Offer the pre-26 fallback (`.ultraThinMaterial`, `.regularMaterial`, custom shape + shadow) instead.
