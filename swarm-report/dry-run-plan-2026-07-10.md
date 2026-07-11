# Plan: `dry-run` — print what would be done without executing

**Slug:** `dry-run`
**Date:** 2026-07-10
**Status:** consilium-complete

---

## TL;DR

**3 Blockers · 8 Concerns · 3 Notes**

| # | Top must-fix |
|---|-------------|
| 1 | **Blocker B-1:** Уточнить семантику: `--dry-run` полезен ТОЛЬКО как "не выполнять даже при `--yes`". Без `--yes` CLI уже печатает preview по умолчанию (D-014, D-008). |
| 2 | **Blocker B-2:** Явное правило приоритета: `--dry-run` побеждает `--yes`; если оба флага — выполнение не происходит. |
| 3 | **Blocker B-3:** Область действия: исключить `login`/`logout` (нет осмысленного dry-run для OAuth); точный список команд. |

---

## Blockers

### B-1 — Семантика vs существующий `--yes` gate (requires_human: true)
**Источники:** skeptic HIGH × 2; architect-2 HIGH × 2; reviewer MEDIUM

Текущее поведение (из D-014, D-008):
- Без `--yes` → CLI печатает preview, выходит не-ноль
- С `--yes` → выполняет без подтверждения

`--dry-run` БЕЗ `--yes` = дублирует уже существующее поведение. Нулевая ценность.

`--dry-run` ценен только в одном случае: **блокирует выполнение даже если `--yes` явно передан** — например, в CI-пайплайне, который всегда добавляет `--yes`, но хочет проверить план без сайд-эффектов.

**Нужно решение:** принять это определение или отклонить фичу как redundant.

### B-2 — Приоритет `--dry-run --yes` (requires_human: true)
**Источники:** architect HIGH; architect-2 HIGH

Текущий код проверяет `!yes` для preview-пути; при обоих флагах `--yes` победит и выполнение произойдёт — это нарушает семантику dry-run.

Правило: `isDryRun()` проверяется ДО любой проверки `yes` в каждом `RunE`. `--dry-run` всегда побеждает. Документируется в `--help`.

### B-3 — Область действия на `login`/`logout` (requires_human: true)
**Источники:** skeptic MEDIUM; architect MEDIUM; reviewer LOW

`login --dry-run` не имеет смысла: OAuth flow нельзя проверить без его запуска. `logout --dry-run` = "удалил бы токен" — полезность нулевая.

`--dry-run` как persistent root flag автоматически доступен на `login`/`logout`. Нужно: либо явная ошибка, либо PersistentPreRunE на service-командах.

---

## Concerns (MEDIUM)

1. **disk unshare / disk mkdir** не имеют `--yes` gate и preview-строки → нужны новые preview-блоки при `isDryRun()` (architect MEDIUM)
2. **calendar create --telemost** вызывает Telemost API до CalDAV write → `isDryRun()` должен быть в самом начале `RunE` до обеих сервисных вызовов (architect MEDIUM)
3. **`emit()` не может предотвратить сервисный вызов** — dry-run guard должен быть ДО инициализации сервиса в каждом `RunE` (architect-2 MEDIUM)
4. **Глобальный persistent flag — silent no-op** на read-only командах (`disk list`, `mail list`, `mail read` и т.д.). Документировать в `--help`; не добавлять per-command guards (architect-2 HIGH → MEDIUM post-clarification)
5. **`confirmSend`/`confirmUnsubscribe`** смешивают preview-вывод и stdin-read — нужно разделить на `previewSend()` + `confirmSend()` (architect MEDIUM)
6. **Область скоупа расплывчата** (skeptic MEDIUM): 23+ субкоманд. Ограничить v1: только мутирующие удалённые операции (disk put/share/unshare/rm/mkdir, mail send, calendar create/update/delete, telemost create). Read-only и local-write команды не в скоупе.
7. **`--dry-run` противоречит D-014 (reviewer MEDIUM)** — если принять B-1, это разрешается; если отклонить B-1, фича не нужна.
8. **Многошаговые команды** (disk put: get upload URL → PUT) не могут дать честный dry-run без API call шага 1 (skeptic HIGH). В v1: dry-run для disk put показывает preview локальной информации (путь, размер файла), не вызывает API.

---

## Notes (LOW)

1. **`resolveManualTarget` дублирует** scope-resolution из `login RunE` (architect LOW) — рефакторинг `resolveLoginTarget()` рекомендован как prerequisite, не обязателен.
2. **`mail unsubscribe` без `--apply`** уже ведёт себя как dry-run (architect LOW) — `--dry-run` на нём = no-op, документировать.
3. **disk move/copy не существуют** в текущей реализации (reviewer MEDIUM) — не включать в scope.

---

## Chosen design (при принятии B-1)

```
var dryRun bool
// root.go — PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print what would happen; overrides --yes")
```

```go
// output.go
func isDryRun() bool { return dryRun }
func emitDryRun(cmd *cobra.Command, msg string) error {
    cmd.Printf("[dry-run] %s\n", msg)
    return nil
}
```

**Pattern в каждом мутирующем RunE:**
```go
if isDryRun() {
    return emitDryRun(cmd, "would <action> <target>")
}
// ... проверки --yes gate ...
// ... service call ...
```

**Команды в скоупе v1:**
| Команда | dry-run message |
|---------|----------------|
| `disk put <file> --to <path>` | `would upload <file> (<size>) to disk:<path>` |
| `disk share <path>` | `would make disk:<path> publicly accessible` |
| `disk unshare <path>` | `would revoke public access for disk:<path>` |
| `disk rm <path>` | `would move disk:<path> to Trash` |
| `disk mkdir <path>` | `would create directory disk:<path>` |
| `mail send` | `would send to <to>: "<subject>"` |
| `calendar create` | `would create event "<title>" at <time>` |
| `calendar update` | `would update event <href>` |
| `calendar delete` | `would delete event <href>` |
| `telemost create` | `would create Telemost conference` |

**Исключены из скоупа v1:** `login`, `logout`, `disk list`, `disk get`, `disk unshare` (уже в списке выше нет — ждём, нет мутации удалённых данных в disk get), `mail list`, `mail search`, `mail read`, `mail attachment`, `calendar list`, `calendar read`, `forms *` (read-only большинство).

---

## Out-of-scope (declared)

- `forms create/publish/unpublish` — отдельный PR после v1
- `disk get` (скачивание = read) — нет мутации на сервере
- Recursive download (v2)
- `--dry-run` для `login`/`logout` — undefined semantics, excluded
- disk move/copy — команд не существует

---

## Open questions raised

- **OQ-021** (new): Нужен ли `--dry-run` для `forms create/publish`? (MEDIUM, ждёт B-1 принятия)

---

## Implementation scope (если B-1 принят)

**Файлы:**

| Файл | Изменение |
|------|-----------|
| `internal/cli/root.go` | + `dryRun bool` global + `--dry-run` persistent flag |
| `internal/cli/output.go` | + `isDryRun()` + `emitDryRun()` |
| `internal/cli/disk.go` | + `isDryRun()` guard в 5 командах (put/share/unshare/rm/mkdir) |
| `internal/cli/mail.go` | + `isDryRun()` guard в `mail send`; extract `previewSend()` |
| `internal/cli/calendar.go` | + `isDryRun()` guard в create/update/delete (до Telemost call) |
| `internal/cli/telemost.go` | + `isDryRun()` guard в create |

**~6 файлов, ~60 строк изменений.**

---

## Per-agent verbatim

### Skeptic
```yaml
- severity: HIGH
  problem: D-014 already implements preview+non-zero-exit without --yes; --dry-run without --yes is redundant.
- severity: HIGH
  problem: Multi-step commands (disk put) can't produce honest dry-run without API step 1.
- severity: MEDIUM
  problem: Global flag meaningless on read-only commands (silent no-op).
- severity: MEDIUM
  problem: Undefined for login/logout.
- severity: MEDIUM
  problem: Scope = 23+ subcommands without explicit list.
```

### Architect (batch 1)
```yaml
- severity: HIGH
  problem: Must be persistent root flag parallel to jsonOutput; check dryRun before any service call in each RunE.
- severity: HIGH
  problem: Service layer must NOT receive dryRun parameter — CLI concern only.
- severity: HIGH
  problem: --dry-run overlaps with !yes; needs explicit precedence (dryRun wins).
- severity: MEDIUM
  problem: disk unshare + disk mkdir have no --yes gate and no preview path.
- severity: MEDIUM
  problem: calendar create --telemost: guard must be at very top of RunE before both Telemost and CalDAV calls.
```

### Architect (batch 2 — additional findings)
```yaml
- severity: HIGH
  problem: --dry-run without --yes = null synonym for existing non-yes default; only valuable as --yes override.
- severity: HIGH
  problem: --dry-run --yes precedence undefined in current code.
- severity: MEDIUM
  problem: confirmSend/confirmUnsubscribe mix preview + stdin read; needs split.
- severity: MEDIUM
  problem: emit() called after service execution; dry-run must branch before service init.
```

### Reviewer
```yaml
- severity: MEDIUM
  problem: disk move/copy not implemented; remove from scope.
- severity: MEDIUM
  problem: Redundant with D-014 unless --dry-run overrides --yes (B-1 resolution).
- severity: MEDIUM
  problem: --dry-run + --yes precedence undefined (cites D-008).
- severity: LOW
  problem: --dry-run login = undefined behavior.
```

---

### Researcher 1 — patterns & prior art

Key findings (confidence medium unless noted):

| Finding | Relevance |
|---------|-----------|
| Without `--yes` the preview+exit path already IS dry-run (HIGH confidence, live code) | --dry-run's only novel value = exit-code disambiguation for agents |
| No major CLI surveyed combines `--yes` gate AND `--dry-run` simultaneously | Adding both is unprecedented; simplifies to: `--dry-run` overrides `--yes` |
| `gh pr create` has no `--dry-run`; `gh` uses interactive confirmation instead | Our `--yes` gate is the settled peer-canonical pattern |
| Existing non-yes path exits 0 (return nil) — agents can't distinguish "previewed" from "succeeded" | **This is the concrete gap `--dry-run` closes**: non-zero exit when nothing mutated |
| HN "In praise of --dry-run" (309 upvotes): race-condition risk preview→execute | Acceptable for human-operated CLI; not a blocker |

### Researcher 2 — Go/Cobra implementation

| Finding | Relevance |
|---------|-----------|
| `PersistentFlags().BoolVar(&dryRun, ...)` in root.go = zero-friction (HIGH confidence) | Identical to existing `jsonOutput`/`insecureFileStore` pattern |
| Package-level var read directly in RunE — no context injection needed | No service-layer changes required |
| `emit()` already routes JSON/human; `--dry-run --json` works without extra renderer | `--json` + `--dry-run` compatibility free |
| `--dry-run` and `--yes` are complementary, not replacements (HIGH confidence) | `--dry-run` wins when both set; document in `--help` |
| kubectl evolved from boolean `--dry-run` to string enum; yx360-cli: boolean is sufficient | No server-side validation path in our CLI |
| helm: per-command flag; tfctl: root persistent flag — both valid | Root-persistent matches our `--json` pattern |

---

*Сгенерировано: 2026-07-10 | Консилиум: architect × 2 + skeptic + reviewer + researcher × 2*
