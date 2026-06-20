# Project Types

From the meeting on 2026-06-01: all Effective projects fall into two categories. The type is determined at harness install and recorded in the project's `CLAUDE.md` via `PROJECT_TYPE: 1` or `PROJECT_TYPE: 2`.

## Type 1 — AI-only / pre-sale MVP / experiment

**Description:** prototypes, MVPs for dem-fest, experiments, hypotheses. Code may be thrown away or rewritten from scratch.

**Examples:** marketing landing for The FEST by Alex Gladkov; Arseniy's pet projects (gift card scanner, meeting recorder in prototype stage).

**Distinguishing properties:**
- Speed > code quality
- Hypothesis validated quickly; then either dropped or migrated to Type 2
- Architecture may be "however it works out" — what matters is whether the feature looks right

**Pipeline gates:**

| Stage | Gate type | Action |
|-------|-----------|--------|
| 1. Requirements | auto-pass + log | Specs collected; human review optional |
| 2. Validation | auto-pass | Gaps logged but non-blocking |
| 3. Architecture | auto-pass | Architect works but human approval not required |
| 4. QA test plan | auto-pass | Tests written but may be skipped for one-shot features |
| 5. Implementation | per-PostToolUse hook | Linters / formatters mandatory |
| 6. Review + smoke | auto-pass | Smoke is mandatory (per ANTI-11); review optional |
| 7. Done | — | Report optional (but desired) |

**What is still mandatory in Type 1:**
- Smoke test before `Done` (ANTI-11)
- Linters / formatters
- No pushes to main without a branch (ANTI-3)

## Type 2 — Production maintained by AI + human

**Description:** projects maintained over years. AI writes; humans control.

**Examples:** jukte (government), sms-hub, clockify-effective, any client production project.

**Distinguishing properties:**
- Code must remain **readable by a human** (if the AI leaves, a junior developer must be able to maintain it)
- Architecture controlled by a human
- Quality > speed

**Pipeline gates:**

| Stage | Gate type | Action |
|-------|-----------|--------|
| 1. Requirements | **human review** | Architect + PM review specs and approve |
| 2. Validation | **human review on gaps** | On gaps, mandatory human decision |
| 3. Architecture | **human approval (hash-locked)** | Plan frozen by hash; if changed, re-approval required |
| 4. QA test plan | optional human review | Auto-pass by default; can pull a human in |
| 5. Implementation | per-PostToolUse hook + per-PR reviewer | Linters + independent reviewer (different model) |
| 6. Review + smoke | **human approval (mandatory)** | Reviewer agent + human; smoke on production |
| 7. Done | — | Report **mandatory** |

**Additional requirements in Type 2:**
- Legal audit at stage 1 (especially for government projects)
- Security audit at stage 3
- Coverage cron (test coverage must not drop)
- Arch audit cron (once per sprint — refactor recommendations)
- All artifacts in `./swarm-report/` are preserved

## Type 1 → Type 2 migration

A prototype "hits" → migration:

1. **Audit existing code** — the consilium runs and produces a tech-debt report
2. **Add tests** — the `test` agent writes unit / integration / E2E for existing code (target 80% coverage)
3. **Add security review** — the `security` agent walks every endpoint and data flow
4. **Lock architecture** — the `architect` writes `architecture/<feature>.md` for each existing feature (capture as-is or refactor proposal)
5. **Enable cron jobs** — arch audit, coverage probe, defrag
6. **Switch `PROJECT_TYPE: 1` → `PROJECT_TYPE: 2`** in `CLAUDE.md`

**Migration estimate:** depends on project size; roughly 5–15 virtual hours for projects under 50k LOC.

## Open questions

1. Can a project have a mixed mode — some features Type 1, some Type 2? **Current answer:** no, type-per-project. If you need an experimental zone, use a separate sandbox project.
2. Who decides "the prototype hit, migrate now"? **Current answer:** Tech Lead.

## Related

- [Pipeline Stages](../product-overview/pipeline-stages.md) — gates per stage
- [Vision](../product-overview/vision.md) — overall goal
- [Anti-Stories](../product-overview/anti-stories.md) — boundaries
