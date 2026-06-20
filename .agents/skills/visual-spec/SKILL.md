---
name: visual-spec
description: Produce a single-file standalone HTML mockup of a feature, architecture, or workflow so the human can evaluate visually before implementation. Use when the user says "draw the architecture", "make me a mockup", "накинь макап", "show me a visual spec", "give me a preview", or invokes /visual-spec. Two modes — walkthrough (annotated multi-scene) and toy-demo (interactive single-screen). Output goes to swarm-report/, never into the application source tree.
license: internal
---

<!-- @harness-owned: true; harness-version: 0.0.1 -->

# visual-spec

A visual artifact beats a wall of prose for early-stage review. This skill produces a **single self-contained HTML file** the human opens in a browser to grade an architecture, a flow, or a feature design before any production code is written.

Trigger phrases include:
- "draw the architecture / draw me a diagram"
- "make a mockup of <feature>"
- "show me a visual spec / preview / a quick demo"
- "накинь макап", "накидай архитектуру", "сделай превьюху"
- Direct: `/visual-spec <slug>` or `/visual-spec`

## Hard rules (block on violation)

1. **One file. Standalone. No build step.** Tailwind CDN only (`https://cdn.tailwindcss.com`). No npm, no React, no bundler.
2. **Output path: `swarm-report/<slug>-mockup-<YYYY-MM-DD>.html`.** Never `app/`, `src/`, `Apps/`, `Packages/`, `frontend/`, `web/`. The mockup is a disposable review artifact, not shipped code.
3. **Slug required.** If the user did not provide one, derive it from the topic (kebab-case, 3–6 words max).
4. **Date from the user's environment.** Use today's date in `YYYY-MM-DD`.
5. **No real secrets, no real customer data.** Toy data only — fake card numbers, sample names, lorem-grade prose.
6. **English only inside the file.** Chat may be Russian; the artifact is English.
7. **Plain-language captions over technical jargon.** The human reviewer skims this in 60 seconds — annotations should read like a product-spec one-pager, not a system-design RFC.
8. **No external assets.** No `<img src>` to remote URLs (other than Tailwind CDN). Use inline SVG for icons.

## Mode picker

| Mode | When to pick | Visual grammar |
|------|--------------|----------------|
| `walkthrough` (default) | Architecture, workflow, queue mechanics, state machines, multi-step flows | Sticky top-nav linking 4–8 scenes. Each scene = a card with title + ≤3 paragraphs of plain-English caption + an illustrative diagram or mock UI. End with a state-machine block + a "pill catalog" of vocabulary. |
| `toy-demo` | UI redesign, single-screen change, interaction shape | One central UI mock. Top nav exposes ~6 buttons that switch the UI through its states (loading / empty / error / happy / dense / overflow). Vanilla JS handles state switching. |

If the user is ambiguous, default to `walkthrough` (richer for design review).

## Visual style (locked — keeps mockups recognizable)

- Dark theme. Body `#0a0a0a`. Surfaces `#141414` → `#1c1c1c`. Borders `#262626`.
- Typography: system font stack. Headings `text-white`. Body `text-gray-200` / `text-gray-400` for secondary.
- Accents: emerald-400 (good / proceed), amber-400 (warn), rose-400 (block / fail), sky-400 (info), violet-400 (alt-flow).
- Pills: rounded-full, `0.72rem` font, semi-transparent tinted background + matching border.
- Icons: inline SVG, stroke-width 2, 16–20 px.
- Cards: `rounded-2xl border border-[#262626] bg-[#0a0a0a]`.
- Sticky nav: 56–64 px tall, blurred translucent (`bg-[#0a0a0a]/95 backdrop-blur`).

## Walkthrough template

Start with this skeleton and adapt:

```html
<!doctype html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title><SLUG> — visual spec</title>
<script src="https://cdn.tailwindcss.com"></script>
<style>
  body { background:#0a0a0a; }
  .scene { scroll-margin-top: 80px; }
  .pill { display:inline-flex; align-items:center; gap:.35rem; padding:.2rem .55rem; border-radius:9999px; font-size:.72rem; font-weight:500; }
  .pill-amber { background:rgba(245,158,11,.12); color:#fbbf24; border:1px solid rgba(245,158,11,.3); }
  .pill-rose  { background:rgba(244,63,94,.12);  color:#fb7185; border:1px solid rgba(244,63,94,.3); }
  .pill-green { background:rgba(34,197,94,.12);  color:#4ade80; border:1px solid rgba(34,197,94,.3); }
  .pill-sky   { background:rgba(56,189,248,.12); color:#7dd3fc; border:1px solid rgba(56,189,248,.3); }
  .pill-gray  { background:rgba(148,163,184,.10);color:#94a3b8; border:1px solid rgba(148,163,184,.25); }
</style>
</head>
<body class="text-gray-200 font-sans antialiased">

<nav class="sticky top-0 z-50 bg-[#0a0a0a]/95 backdrop-blur border-b border-[#262626] px-5 py-3 text-sm">
  <div class="max-w-6xl mx-auto flex flex-wrap items-center gap-x-5 gap-y-2">
    <span class="font-semibold text-white"><SLUG></span>
    <span class="text-gray-500 text-xs">visual spec · <N> scenes</span>
    <div class="ml-auto flex flex-wrap gap-1 text-xs">
      <a href="#scene-1" class="px-2 py-1 rounded bg-[#1a1a1a] hover:bg-[#262626]">1. <name></a>
      <!-- … -->
      <a href="#machine"  class="px-2 py-1 rounded bg-[#1a1a1a] hover:bg-[#262626]">⚙ state machine</a>
      <a href="#pills"    class="px-2 py-1 rounded bg-[#1a1a1a] hover:bg-[#262626]">🏷 vocabulary</a>
    </div>
  </div>
</nav>

<section class="max-w-3xl mx-auto pt-8 pb-2 px-5">
  <h1 class="text-2xl font-semibold text-white"><SLUG></h1>
  <p class="text-gray-400 mt-2 leading-relaxed text-[15px]">
    <one-paragraph plain-English summary: what the feature does, why we care, what changes.>
  </p>
</section>

<!-- Scenes -->
<section id="scene-1" class="scene max-w-5xl mx-auto my-10 px-5">
  <h2 class="text-lg font-semibold text-white mb-1">1. <Scene name></h2>
  <p class="text-gray-400 text-sm mb-4"><one-line setup of this scene></p>
  <div class="rounded-2xl border border-[#262626] bg-[#0a0a0a] p-5">
    <!-- mock UI or diagram for this scene -->
  </div>
</section>

<!-- State machine block (mandatory in walkthrough mode) -->
<section id="machine" class="scene max-w-5xl mx-auto my-12 px-5">
  <h2 class="text-lg font-semibold text-white mb-2">⚙ State machine</h2>
  <div class="rounded-2xl border border-[#262626] bg-[#0a0a0a] p-5">
    <!-- SVG state-graph OR an HTML grid of named nodes + arrow text labels. No mermaid runtime. -->
  </div>
</section>

<!-- Pill catalog -->
<section id="pills" class="scene max-w-5xl mx-auto my-12 px-5">
  <h2 class="text-lg font-semibold text-white mb-2">🏷 Vocabulary</h2>
  <div class="rounded-2xl border border-[#262626] bg-[#0a0a0a] p-5 flex flex-wrap gap-2">
    <span class="pill pill-green">accepted</span>
    <span class="pill pill-amber">pending</span>
    <span class="pill pill-rose">rejected</span>
    <!-- … -->
  </div>
</section>

<footer class="max-w-5xl mx-auto my-12 px-5 text-xs text-gray-500">
  Mockup. Not production. Generated <YYYY-MM-DD> by /visual-spec.
</footer>

</body>
</html>
```

## Toy-demo template additions

For `toy-demo` mode, replace the multi-scene body with one centered UI mock plus a top-nav state segmenter. Use vanilla JS to swap states:

```html
<div class="sticky top-0 ..." data-group="state">
  <button data-v="empty">empty</button>
  <button data-v="loading">loading</button>
  <button data-v="happy">happy (9 items)</button>
  <button data-v="dense">dense (200 items)</button>
  <button data-v="error">error</button>
</div>

<script>
let state = 'happy';
function render() { /* swap content per state */ }
document.querySelectorAll('[data-group=state] button').forEach(b => {
  b.addEventListener('click', () => { state = b.dataset.v; render(); });
});
render();
</script>
```

## Workflow

1. **Read context.** If a plan file `swarm-report/<slug>-plan-*.md` exists, read it. Otherwise, read what the user just described.
2. **Pick the mode** per the table above.
3. **Pick a slug** if the user did not. Derive 3–6 kebab-case words from the topic.
4. **Choose 4–8 scenes** (walkthrough) or 4–8 states (toy-demo). Scenes/states should cover happy path + edge cases + failure modes.
5. **Write the HTML** to `swarm-report/<slug>-mockup-<YYYY-MM-DD>.html`. Use `mkdir -p swarm-report` first if missing.
6. **Report back** with the file path + a one-line "open this in a browser to review" instruction. Do not paste the full HTML into chat.
7. If the user requests changes, **edit the file in place** — do not create a `-v2`.

## Anti-patterns (block immediately)

- Writing the mockup anywhere outside `swarm-report/`.
- Pulling in React, Vue, Svelte, or a CSS framework other than Tailwind CDN.
- Producing a multi-file artifact.
- Embedding real user data, real card numbers, real emails, real names from the codebase.
- Writing the mockup in any non-English language (the file is English-only).
- Adding a build step, Vite config, package.json, or node_modules to make it run.
- Returning the entire HTML in chat instead of writing to disk and citing the path.
- Going past 1500 lines. If the mockup wants to be bigger, you are over-detailing — cut scenes.
- Generating a mockup when the user asked for production code. Ask first if unsure.

## Companion: `.gitignore` suggestion

If the host project does not already gitignore `swarm-report/`, point this out once. The mockup is disposable; long-term storage of every `-mockup-*.html` will bloat the repo. Suggested entry:

```
# Visual specs / mockups produced by /visual-spec
swarm-report/*-mockup-*.html
```

The user decides; do not edit `.gitignore` without asking.

## Output format (chat reply)

After writing the file, reply in this shape — no more:

```
Mockup written: swarm-report/<slug>-mockup-<YYYY-MM-DD>.html
Mode: <walkthrough|toy-demo> · scenes: <N>
Open in a browser to review. Request edits by scene number.
```
