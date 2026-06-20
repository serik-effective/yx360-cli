---
name: quickstart
description: Install local dev-environment dependencies needed to work on the current project. Detects required tooling from `.memory-bank/tech-details/stack.md` + the repo `Makefile`, installs missing tools via brew/apt, verifies versions, and copies secret-file templates from `.example` siblings. Use when a developer clones an Effective project for the first time and types `/quickstart`. Cloud / language / framework-agnostic — the skill reads the project's own stack file to learn what to install. Equivalent to `make quickstart` in vaultwarden / vpn, but interactive and explanatory.
---

# Skill: /quickstart

Bootstrap a developer's local environment for the project the skill is invoked in. Reads the project's stack lock + Makefile, installs missing tools via the local package manager, verifies versions, copies secret-file templates, and prints a one-screen "what next" summary.

**Generic.** Same skill works in `vaultwarden`, `vpn`, `litellm`, future infra repos. No project name appears in the skill body — the project tells the skill what it needs by populating `.memory-bank/tech-details/stack.md`.

## When to invoke

- A developer just cloned the repo and needs every CLI tool that `make plan` / `make deploy` will call.
- A repo gained a new dependency (e.g. `sops`) and existing developers need the new tool.
- CI image has drifted from local dev — re-run to re-pin.

**Do NOT use** when the developer already has every tool and just wants to deploy. They run `make deploy`.

## Invocation

```
/quickstart
```

Optional:

```
/quickstart --dry-run    # list what would be installed, install nothing
/quickstart --force      # re-run even if tools look present (e.g. version drift suspected)
```

## Step 1 — Detect platform + package manager

- `darwin` → require Homebrew. Abort with install instructions if missing.
- `linux` → prefer `apt-get` (Debian / Ubuntu — the production targets), fallback `dnf`, `pacman` with a warning.
- Other → emit unsupported message + tool list, exit 1.

## Step 2 — Read project stack

Required source file: `.memory-bank/tech-details/stack.md`.

If missing → abort: "This project does not declare its stack. Run `/quickstart` only after harness is installed (`.memory-bank/` exists)."

Parse `stack.md` for tool names. Look for a `## Languages` or `## Infrastructure` table, plus any explicit `## Pending` items. The skill is forgiving — it pattern-matches known tool keywords:

| Pattern in stack.md | Tool installed |
|---------------------|----------------|
| `OpenTofu`, `tofu`, `terraform` | `opentofu` (preferred) or `terraform` |
| `Ansible` | `ansible`, `ansible-lint` |
| `docker-compose`, `Docker Compose` | `docker` (Docker Desktop / colima / orbstack — leave choice to the user) + `docker compose` plugin |
| `SOPS` (D-009 pattern) | `sops`, `age` |
| `Caddy` | `caddy` (for local Caddyfile lint only; production runs in container) |
| `LiteLLM` | `python3.11+` + `litellm[proxy]` (only if developer wants to smoke locally) |
| `Cloudflare` | `cloudflared` (optional) |
| `Postgres` | `postgresql-client` (for `psql`) |
| `Mattermost webhook` | nothing — webhook is URL only |
| `Vagrant` | `vagrant` |
| `gh` mentioned | `gh` (GitHub CLI) |

Also read `Makefile` to discover any tool referenced via `command -v <tool>` or `which <tool>` patterns. Add those to the install list.

## Step 3 — Show plan + confirm

Print the deduped install list:

```
This project needs these tools (parsed from .memory-bank/tech-details/stack.md + Makefile):

  Already installed:
    opentofu  1.7.3
    ansible   2.20.1

  Will install:
    sops      (via brew install sops)
    age       (via brew install age)
    docker    (please install Docker Desktop / OrbStack manually)

  Will create from template (you fill in real values after):
    secrets/litellm.env             ← copy of secrets/litellm.env.example
    terraform/terraform.tfvars      ← copy of terraform/terraform.tfvars.example

Proceed? (y / n / details)
```

Wait for explicit `y`. `details` re-prints each tool with the line of `stack.md` that triggered it.

## Step 4 — Install missing tools

Use the platform package manager. Never `sudo` without warning the developer first. Never `curl | sh` random scripts — every install goes through `brew` / `apt-get` / similar.

For Docker on macOS — do NOT auto-install. Detect missing, instruct: "Install Docker Desktop, OrbStack, or colima manually, then re-run `/quickstart`."

For age — verify version ≥ 1.1. SOPS — verify ≥ 3.10.

## Step 5 — Verify versions

After install, run `<tool> --version` for each and pin a snapshot to `.assistant/quickstart-snapshot.json`:

```json
{
  "ran_at": "2026-06-09T11:42Z",
  "platform": "darwin-arm64",
  "tools": {
    "opentofu": "1.7.3",
    "ansible":  "2.20.1",
    "sops":     "3.10.0",
    "age":      "1.2.1"
  }
}
```

`.assistant/quickstart-snapshot.json` is gitignored (developer-local) — informational only.

## Step 6 — Copy secret-file templates

For every `<path>.example` in the repo, if `<path>` does not exist, copy it. Print the new path so the developer knows to fill it in.

Specifically check:
- `secrets/*.example` → `secrets/<name>`
- `terraform/terraform.tfvars.example` → `terraform/terraform.tfvars`
- `ansible/group_vars/all/secrets.yml.example` → `ansible/group_vars/all/secrets.yml`
- `.env.example` → `.env`

After copy: print the developer's TODO list with the relative paths.

## Step 7 — Print "what next"

Final summary:

```
✓ Quickstart complete.

Next:
  1. Fill in the secret files listed above. Owners: <names from .sops.yaml / runbook>.
  2. Read .memory-bank/index.md for project knowledge.
  3. Read .memory-bank/product-overview/handoff.md (if present) for the current task brief.
  4. `make plan` to preview infra.
  5. `make deploy` only after the owners have approved the secrets you filled in.

If anything failed, file an issue at: <github URL from .harness-lock or git remote>.
```

## Loop guards

- Already-installed everything → print "all tools present, snapshot updated" and exit 0.
- `--dry-run` → print plan, write no snapshot, copy no files.
- `apt-get` missing AND not Debian/Ubuntu → instruct manual install.
- Network error during `brew install` → retry once, then surface the error verbatim. Never silently skip a tool.

## What this skill does NOT do

- Does not provision cloud resources (`tofu apply`) — that's `make deploy`.
- Does not log into GitHub / cloud providers — secrets stay developer-local.
- Does not run the project (`make plan` / `make deploy`) — separate human gate.
- Does not modify `.memory-bank/` or `.assistant/decisions.md` — read-only on those.
- Does not assume a specific cloud — provider-agnostic, reads stack.md for cloud name.
- Does not install editor / IDE tooling — that's the developer's choice.

## Example

```
$ cd ~/Projects/effective/litellm
$ Codex  # or codex
> /quickstart

Detected: macOS arm64, brew available.
Reading .memory-bank/tech-details/stack.md ... 5 tools needed.

  Already installed:
    docker         28.0.1
    ansible        2.20.0
  Will install:
    opentofu       (brew install opentofu)
    sops           (brew install sops)
    age            (brew install age)
  Will create from template:
    secrets/litellm.env.example          → secrets/litellm.env

Proceed? (y / n / details) y

→ brew install opentofu sops age
[install output...]

✓ Quickstart complete.

Next:
  1. Fill in secrets/litellm.env — ask Arseniy for the master key + Cloudflare token.
  2. Read .memory-bank/index.md for project knowledge.
  3. Read .memory-bank/product-overview/handoff.md for the current task brief.
  4. `make plan` to preview infra.
  5. `make deploy` after the owners have approved your secrets.
```
