# Anti-Stories

What the harness does **not** do. These are invariants. If a design decision violates an anti-story — revise the design, not the anti-story.

## ANTI-1. The harness is not a CLI wrapper over features

There is no `harness do "build payment"` / `harness implement <feature>` / `harness fix <bug>` command. Harness = files; entry point = the developer's normal AI-CLI (Claude Code, OpenCode, Codex), which reads the project's `CLAUDE.md`.

**Why:** if we ship our own CLI, we depend on its maintenance, we break runtime portability between Claude / OpenCode / Codex, and we fall into the "harness as wrapper product, not standardization" trap.

(Per D-006 §1.2, install/update utilities are outside this rule — they provision files, not features.)

## ANTI-2. The harness does not execute destructive actions without confirmation

Deploys, `git push --force`, `DROP TABLE`, `rm -rf`, branch deletion, posting to pastebin / gist / external renderers — always behind a human gate, even in Type 1.

## ANTI-3. The harness does not push to main

Feature branches → PR → review → merge. Auto-merge to main is forbidden.

## ANTI-4. The harness does not make architectural decisions without human review in Type 2

Type 1 (MVP) — permitted. Type 2 (production) — stage 3 (architecture) requires mandatory human approval (hash-locked plan).

## ANTI-5. The harness does not work without internet for research-class agents

Agents without internet hallucinate facts. Every agent doing research / architecture / legal audit must have MCP access to web search (`mcp-omnisearch`, Tavily, web_fetch).

## ANTI-6. The harness does not impose a fixed tech stack

Each project declares its own stack in `STACK.md` / `CLAUDE.md`. The harness doesn't mandate React / Mobx / FastAPI / etc. Standardization is at the **process** level (stages, agents, gates), not technologies.

## ANTI-7. The harness does not try to replace the developer

The goal is multiplication. If a solution requires the developer to "step aside and not interfere," that's an anti-goal. The developer must be able to review, stop, or roll back any stage.

## ANTI-8. The harness does not ignore legacy

We frequently enter existing projects. Requirements collection from code (memory-bank-init from an existing project) is a mandatory capability, not optional.

## ANTI-9. The harness does not write secrets to public systems

Prompt logs (LiteLLM), memory bank — inside company infrastructure. No automatic exports to external services without explicit opt-in.

## ANTI-10. The harness does not stay quiet about side effects

If a change in one agent might affect other projects, the harness must surface it in release notes before the update rolls out.

## ANTI-11. The harness does not mask errors

"Tests passed" ≠ "feature works." Smoke tests and manual verification via the `verify` skill are mandatory. If an agent reports "done" without a smoke test, that's a bug.

## ANTI-12. The harness does not block manual development

A developer must be able to write code by hand without an AI-CLI, and the harness files must not get in the way (no pre-commit hooks requiring an agent in the pipeline).

## Related

- [Vision](vision.md) — positive formulation of the goal
- [Project Rules](../steerings/project-rules.md) — common principles
- [Project Types](../steerings/project-types.md) — Type 1 vs Type 2 distinctions
