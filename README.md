<p align="center">
  <img src="avatar.png" alt="GitAi" width="160">
</p>

# gitai

Go CLI to generate **Conventional Commits** with affordable AI, automate **push**, and create detailed **Pull Requests** via GitHub CLI.

---

## Table of contents

- [Why gitai?](#why-gitai)
- [Requirements](#requirements)
- [Quick install](#quick-install)
- [TUI dashboard](#command-reference)
- [Manual install](#manual-install)
- [Updating](#updating)
- [Configuration](#configuration)
- [Versioning](#versioning)
- [Command reference](#command-reference)
- [Global and per-command flags](#global-and-per-command-flags)
- [Detailed usage](#detailed-usage)
- [Token usage and cost](#token-usage-and-cost)
- [AI providers](#ai-providers)
- [Commit and PR format](#commit-and-pr-format)
- [Troubleshooting](#troubleshooting)
- [Security](#security)
- [License](#license)

---

## Why gitai?

Editor AI assistants often burn expensive tokens reading diffs, writing commit messages, and running git. **gitai** moves that workflow to a configurable AI (DeepSeek via OpenRouter, GPT-4o-mini, Gemini Flash) for fractions of a cent — works with any editor or agent (Claude Code, Copilot, terminal, etc.).

With gitai you get:

- Messages following **Conventional Commits**
- Structured PRs with **Summary**, **Changes**, **Test plan**, and **Notes**
- **Token and cost** summary (estimate before AI + total after execution)
- **Spending report** (`gitai report`) with CSV history
- Native integration with **`gh pr create`**

---

## Requirements

| Tool | Minimum version | Purpose |
|------|-----------------|---------|
| [git](https://git-scm.com/) | any recent | Local repo, diff, commit, push |
| [Go](https://go.dev/dl/) | 1.22+ | Build gitai (`install.sh` installs automatically if missing) |
| [GitHub CLI (`gh`)](https://cli.github.com/) | authenticated | Create PR (`gitai pr`) — optional until you use PR |

Authenticate GitHub CLI before using `gitai pr`:

```bash
gh auth login
gh auth status
```

---

## Quick install

### One command (recommended)

The `install.sh` script runs **everything in order**:

1. Checks `git`, `curl`, and `tar`
2. Installs Go in `~/sdk/go` if no compatible version is found
3. Clones the repo to `~/.config/gitai/repository` (or uses the current clone)
4. Builds and installs the binary (`go run ./cmd/gitai install`)
5. Writes `PATH` to `~/.zshrc` or `~/.bashrc` (Go + `~/go/bin`)
6. Runs `gitai config` (interactive wizard)

**From a clone:**

```bash
git clone https://github.com/laerciocrestani/gitai.git
cd gitai
./install.sh
```

**Without cloning (curl):**

```bash
curl -fsSL https://raw.githubusercontent.com/laerciocrestani/gitai/main/install.sh | bash
```

Installer options:

| Option | Description |
|--------|-------------|
| `--no-config` | Skip the `gitai config` wizard at the end |
| `--skip-go` | Do not install Go automatically (fails if missing) |
| `--help` | Help |

Useful variables: `GITAI_REPO_URL`, `GITAI_INSTALL_DIR`, `GO_VERSION` (default `1.25.0`).

### Uninstall

Removes the binary, `~/.config/gitai/`, PATH blocks in your shell, and (if installed by `install.sh`) Go in `~/sdk/go`:

```bash
./uninstall.sh
# or
curl -fsSL https://raw.githubusercontent.com/laerciocrestani/gitai/main/uninstall.sh | bash
```

| Option | Description |
|--------|-------------|
| `-y`, `--yes` | Skip confirmation |
| `--remove-go` | Remove `~/sdk/go` even without installer marker |
| `--keep-go` | Keep Go in `~/sdk/go` |

**Does not remove:** `.gitai.yaml` files in projects or manually set `GITAI_*` variables.

The `./scripts/setup.sh uninstall` script delegates to `./uninstall.sh`.

After installing, open a new terminal (or `source ~/.zshrc`) and run:

```bash
gitai              # TUI dashboard inside a git repo
gitai commit
gitai pr
```

### Post-install commands

| Command | What it does |
|---------|--------------|
| `./install.sh` | Full install (Go + binary + PATH + config) |
| `./uninstall.sh` | Remove gitai, data, and installer PATH |
| `gitai config` | Configuration wizard (same as `gitai config init`) |
| `gitai config show` | Show active config (masked API key) |
| `gitai update` | Update and reinstall binary (works from any directory) |
| `gitai version` | Auto version + commit + commit count |
| `gitai report` | AI usage and cost report (last 24h by default) |
| `gitai pricing update` | Fetch official Gemini prices and save locally |
| `gitai status` | Alias for `git status` |

The `./scripts/setup.sh` script is a compatibility wrapper that delegates to `./install.sh` and `./uninstall.sh`.

### Update later

From any directory:

```bash
gitai update
```

gitai uses the saved clone in `~/.config/gitai/repository`, the `GITAI_ROOT` variable, or downloads the latest version from GitHub automatically if no local clone is found.

---

## Manual install

If you prefer not to use `install.sh`:

### 1. Clone the repository

```bash
git clone https://github.com/laerciocrestani/gitai.git
cd gitai
```

### 2. Install Go 1.22+

https://go.dev/dl/ — or let `./install.sh` install to `~/sdk/go`.

### 3. Install the binary

```bash
go run ./cmd/gitai install
```

The binary is installed to `~/go/bin/gitai` and the installer configures PATH.

### 4. Configure

```bash
gitai config
```

### Alternative without changing PATH

Run directly by full path:

```bash
~/go/bin/gitai config init
~/go/bin/gitai pr
```

---

## Updating

```bash
gitai update
```

Optional — point to your local clone:

```bash
export GITAI_ROOT=~/projects/gitai
gitai update
```

Or manually, inside the clone:

```bash
cd gitai
git pull origin main
go install ./cmd/gitai
```

---

## Configuration

### Interactive wizard (recommended)

```bash
gitai config
```

Same as `gitai config init`.

The wizard asks, in this order:

| Field | Options / default | Description |
|-------|-------------------|-------------|
| Provider | `openrouter`, `openai`, `gemini` | Selector with ↑↓ and Enter |
| Model | suggestions + **Other...** | Selector; "Other" lets you type a custom model |
| API key | — | Provider key (Enter keeps the current value) |
| Language | default: `pt-BR` | Language of generated commit/PR messages |
| Base branch | default: `main` | Branch used as PR base |
| Co-author | optional | Trailer appended to the commit |
| Clear terminal | `y` / `n` | Clear the console before each gitai command |

Provider and model use arrow navigation (`↑↓`) or `j`/`k`. Outside a TTY (CI, pipe), it falls back to a numbered list.

If config already exists, **Enter on any field keeps the current value** (e.g. `[gemini]`).

Saved to `~/.config/gitai/config.yaml` with permission `0600`.

### Global config file

Default path: `~/.config/gitai/config.yaml`

```yaml
provider: openrouter        # openai | gemini | openrouter
api_key: "sk-..."
model: "deepseek/deepseek-chat"
language: "pt-BR"
base_branch: "main"
co_author: ""
max_diff_bytes: 120000
clear_screen: false       # true = clear terminal before each command
interactive_ui: true      # true = gitai opens TUI in terminal (default)
ui_color: true            # colors in CLI and TUI (default)
ui_auto_refresh_seconds: 5   # dashboard polling (0 = off)
ui_watch_files: true      # fsnotify on working tree (default)

# optional — overrides default Gemini prices
# input_price_per_1m: 0.14
# output_price_per_1m: 0.28
```

### Per-repository local config

Create `.gitai.yaml` at the project root. **Takes priority** over global config.

Useful for:

- Different model per project
- Base branch `develop` instead of `main`
- `en-US` language on open source projects

### Environment variables

| Variable | Description |
|----------|-------------|
| `GITAI_API_KEY` | Overrides YAML `api_key` (recommended in CI) |
| `GITAI_CONFIG` | Alternate config file path |
| `GITAI_ROOT` | Path to gitai clone (used by `gitai update` and `install.sh`) |
| `GITAI_NO_CLEAR` | Disable terminal clear (`clear_screen` ignored) |
| `GITAI_NO_UI` | Force CLI overview instead of TUI (`interactive_ui` ignored) |
| `NO_COLOR` | Disable ANSI colors (Unix convention; see [no-color.org](https://no-color.org)) |

Example:

```bash
export GITAI_API_KEY="sk-or-v1-..."
export GITAI_CONFIG="$HOME/.config/gitai/work.yaml"
gitai pr
```

### Show current configuration

```bash
gitai config show
```

The API key is **masked** in output (e.g. `sk-o...abcd`).

### Full field reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `provider` | string | yes | `openrouter` | `openai`, `gemini`, or `openrouter` |
| `api_key` | string | yes* | — | API key (* or `GITAI_API_KEY`) |
| `model` | string | yes | depends | Model identifier on the provider |
| `language` | string | no | `pt-BR` | Commit and PR language |
| `base_branch` | string | no | `main` | Default base branch for `gitai pr` |
| `co_author` | string | no | empty | Commit trailer (e.g. `Co-authored-by: Name <email@example.com>`) |
| `max_diff_bytes` | int | no | `120000` | Max diff size sent to AI |
| `clear_screen` | bool | no | `false` | Clear terminal before each command |
| `interactive_ui` | bool | no | `true` | Open TUI when running `gitai` with no subcommand |
| `ui_color` | bool | no | `true` | ANSI colors in CLI and TUI |
| `ui_auto_refresh_seconds` | int | no | `5` | Dashboard polling in seconds (`0` = off) |
| `ui_watch_files` | bool | no | `true` | Watch filesystem changes (fsnotify) |
| `input_price_per_1m` | float | no | — | USD per 1M input tokens (cost estimate) |
| `output_price_per_1m` | float | no | — | USD per 1M output tokens (cost estimate) |

### Default models per provider (wizard)

| Provider | Default model |
|----------|---------------|
| `openrouter` | `deepseek/deepseek-chat` |
| `openai` | `gpt-4o-mini` |
| `gemini` | `gemini-2.5-flash-lite` |

---

## Versioning

Version is **automatic**, derived from the number of commits in the repository (no git tags):

- 1st commit → `v0.1.0`
- each additional commit increments patch → e.g. 14 commits = `v0.1.13`

```bash
gitai version
```

Shows version, commit, total commits, and whether the tree is dirty.

`go install` injects version and commit via `-ldflags` at build time.

---

## Command reference

Running **`gitai` with no subcommand** inside a git repository opens the **fullscreen TUI** (dashboard) with panels:

![gitai TUI dashboard](docs/tui-dashboard.png)

- **Git Graph** — current branch vs base
- **Repository Summary** — changed files and `+N · -M` stats
- **Changed Files** — list with dot leaders and per-file stats
- **Recent Commits** — last 3 commits
- **AI Engine** — provider, model, and status
- **Suggested Action** — recommended next step

### Dashboard shortcuts (TUI)

| Key | Action |
|-----|--------|
| `c` | AI commit (preview → edit → confirm) |
| `p` | Push (preview → confirm) |
| `P` | AI Pull Request (preview → edit → confirm) |
| `d` | View diff |
| `b` | Switch branch (list + context + checkout) |
| `l` | Commit log |
| `y` | Copy HEAD hash |
| `s` | Sync (when behind) |
| `o` | Open PR in browser |
| `u` | AI usage/cost report |
| `r` | Refresh dashboard |
| `?` | Help |
| `q` | Quit |

Commit, push, and PR go through **preview with confirmation** (`Enter` confirms, `esc` cancels). On preview, `e` edits the message (commit/push) or title/body (PR).

With `GITAI_NO_UI=1` or outside a git repo, shows the **CLI overview** (ANSI text).

```
gitai                 TUI dashboard or CLI overview (default)
├── sync              fetch + pull base branch (--prune to clean branches)
├── update            update and reinstall binary
├── version           auto version + commit
├── report            AI usage and cost report
├── status            alias for git status
├── commit            generate AI commit from local diff
├── push              commit (if diff) + push to origin
├── pr                commit (if needed) + push + detailed PR via gh
├── pricing           Gemini prices (update / show / report)
│   ├── update        fetch official prices from the web
│   ├── show          show saved prices
│   └── report        alias for gitai report --all
└── config            configuration wizard (or init/show subcommands)
    ├── init          interactive wizard (alias for gitai config)
    └── show          show active config (masked key)
```

> Install: `./install.sh` or `curl -fsSL …/install.sh | bash`

### Overview

| Command | What it does | Calls AI? | Runs git? | Runs gh? |
|---------|--------------|-----------|-----------|----------|
| `gitai` | Repository overview | no | read-only | no |
| `gitai sync` | Sync base branch with origin | no | `fetch`, `pull` | no |
| `gitai sync --prune` | Sync + remove merged/absorbed branches (remote first, then local) | no | `fetch`, `pull`, `push --delete`, `fetch`, `branch -d/-D` | no |
| `gitai sync --prune-remote` | Sync + remove merged/absorbed branches on GitHub only | no | `fetch`, `pull`, `push --delete`, `fetch` | no |
| `gitai version` | Show version, commit, and commit count | no | read-only | no |
| `gitai report` | AI usage/cost report | no | read-only | no |
| `gitai pricing update` | Update Gemini price table | no | no | no |
| `gitai commit` | Commit with generated message | 1× (commit) | `add`, `commit` | no |
| `gitai push` | Commit (if diff) + push | 0–1× | `add`, `commit`, `push` | no |
| `gitai pr` | Commit + push + PR | 1–2× (commit + PR) | `add`, `commit`, `push` | `pr create` |
| `gitai status` | Show repository status | no | `status` | no |
| `gitai config` | Create/update config.yaml | no | no | no |
| `gitai config init` | Same as `gitai config` | no | no | no |
| `gitai config show` | Show config | no | no | no |
| `gitai update` | Update and reinstall binary | no | no | no |

---

## Global and per-command flags

### Global flags (valid on all commands)

Available on `commit`, `push`, and `pr`:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | `false` | Simulates the flow: calls AI, shows what would run, **does not** run `git commit`, `git push`, or `gh pr create` |
| `--verbose` | bool | `false` | Shows parsed AI JSON (type, scope, title, commit bullets or PR sections) |

Examples:

```bash
gitai commit --dry-run
gitai pr --verbose --dry-run
gitai push --verbose
```

### `gitai commit` flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--no-add` | bool | `false` | Skip `git add .` — use only already staged files (or unstaged as fallback) |

```bash
git add src/auth.go
gitai commit --no-add
```

### `gitai push` flags

Inherits all `commit` flags:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--no-add` | bool | `false` | Skip `git add .` before commit |

After commit, runs:

```bash
git push -u origin HEAD
```

### `gitai pr` flags

Inherits global flags and `--no-add`, plus:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--no-add` | bool | `false` | Skip `git add .` |
| `--draft` | bool | `false` | Create PR as **draft** (`gh pr create --draft`) |
| `--base` | string | config `base_branch` | PR base branch (e.g. `main`, `develop`) |

Examples:

```bash
gitai pr
gitai pr --draft
gitai pr --base develop
gitai pr --no-add --draft --base main --verbose --dry-run
```

### Combining flags

```bash
# Full pr flow preview without changing anything
gitai pr --dry-run --verbose

# Commit only what is already staged, no push
gitai commit --no-add

# Draft PR against develop, no git add
git add .
gitai pr --no-add --draft --base develop
```

### `gitai report` flags

| Flag | Description |
|------|-------------|
| `--hour` | Last hour |
| `--hours N` | Last N hours |
| `--days N` | Last N days |
| `--month` | Current calendar month |
| `--all` | Full history |

Default (no flags): **last 24 hours**.

```bash
gitai report
gitai report --hour
gitai report --days 7
gitai report --all
```

---

## Detailed usage

### Recommended daily workflow

```bash
# 1. Work on your feature branch
git checkout -b feat/my-feature

# 2. Make your code changes

# 3. Commit + push + PR in one command
gitai pr
```

`gitai pr` runs internally:

```
git add .
    ↓
[if there are staged changes]
    → AI generates commit message → git commit
    ↓
git push -u origin HEAD
    ↓
git diff base...HEAD  (+ branch commit log)
    ↓
AI generates detailed PR (title, summary, changes, test plan, notes)
    ↓
gh pr create --title "..." --body "..." --base main
    ↓
Shows token and cost summary
```

### `gitai commit`

**When to use:** commit only, no push or PR.

**Flow:**

1. `git add .` (unless `--no-add`)
2. Get staged diff (or unstaged if nothing staged)
3. Send diff to AI → Conventional Commit
4. `git commit -m "..."`
5. Show token/cost summary

**Diff used:** pending local changes (staged preferred).

```bash
gitai commit
gitai commit --no-add
gitai commit --dry-run --verbose
```

**Common errors:**

- `no changes to commit` — clean working tree
- `current directory is not a git repository` — run inside a git repo

---

### `gitai push`

**When to use:** push branch to origin. If there are pending changes, commits first; otherwise pushes existing commits.

**In the TUI:** preview with confirmation before execution (same as PR).

**Flow:** `git add .` (unless `--no-add`) → AI commit (only if diff) → `git push -u origin HEAD`.

```bash
gitai push
gitai push --no-add
gitai push --dry-run
```

> Token/cost summary is shown after commit (inside the push flow). Push itself does not consume AI.

---

### `gitai pr` (main command)

**When to use:** finish work on the branch — pending commit, push, and detailed PR.

**In the TUI:** editable preview (title, markdown body, draft toggle) with confirmation before creating the PR.

**Smart flow:**

| Situation | Behavior |
|-----------|----------|
| Uncommitted changes | `git add .` → AI commit → commit |
| Only branch commits, nothing pending | Skip commit, use existing commits |
| Branch equals base, no changes | Error: `no changes relative to main` |

**Diff used for PR:** `git diff base...HEAD` — **all** branch changes vs base, not just the last commit.

**Diff used for commit (when staged):** only the current staged diff.

**Base branch resolution:**

1. Try `main` (or `--base` / config value)
2. Try `origin/main`
3. Error if neither exists → run `git fetch`

```bash
gitai pr
gitai pr --draft
gitai pr --base develop
gitai pr --verbose --dry-run
```

**Generated PR body:**

```markdown
## Summary
- overview and impact

## Changes
- technical details by area

## Test plan
- [ ] step 1
- [ ] step 2

## Notes
- risks or follow-ups (if any)
```

**Common errors:**

- `PR already exists: https://...` — branch already has an open PR
- `base branch "main" not found` — run `git fetch origin`
- `config not found` — run `gitai config init`

---

### `gitai config init`

Interactive wizard. Does not change git repositories — only creates/updates global YAML.

```bash
gitai config init
```

### `gitai config show`

Loads effective config (local `.gitai.yaml` or global) and prints with masked key.

```bash
gitai config show
```

---

## Token usage and cost

### Estimate (before AI)

Before the `Thinking` step, gitai shows an estimate:

```
Estimate: ~1750 tokens · $0.000275 USD (Gemini) (input ~1500 + output ~250)
```

### After execution

At the end of **`commit`**, **`push`**, and **`pr`**:

```
AI usage
• commit: 8420 prompt + 186 completion = 8606 tokens | $0.000412 USD (Gemini)
• Total: 8606 prompt + 186 completion = 8792 tokens | total cost: $0.000412 USD
```

Each call is logged to `~/.config/gitai/usage/ledger.csv` for `gitai report`.

### How cost is calculated

| Provider | Tokens | Cost |
|----------|--------|------|
| **OpenRouter** | `usage.*` | Real via `usage.cost` (USD) |
| **OpenAI** | `usage.*` | Estimate (default or config prices) |
| **Gemini** | `usageMetadata.*` | Estimate with defaults or `gitai pricing update` |

### Gemini prices

```bash
gitai pricing update   # fetch official prices and save to ~/.config/gitai/pricing.yaml
gitai pricing show     # show saved table
```

Models with built-in default prices (e.g. `gemini-2.5-flash-lite` → $0.10 / $0.40 per 1M tokens).

### Manual estimate (override)

Add to config to override any provider:

```yaml
input_price_per_1m: 0.15
output_price_per_1m: 0.60
```

### Retries

| Type | Behavior |
|------|----------|
| **API unavailable** (503, 429, etc.) | Up to **3 attempts**, **3s** between each |
| **Invalid AI JSON** | Up to 2 parse retries (consumes extra tokens) |

### `--dry-run`

AI **is called** (you see tokens/cost and ledger entry), but git/gh **do not run**.

---

## AI providers

| Provider | Recommended model | Typical cost | Cost in response |
|----------|-------------------|--------------|------------------|
| `openrouter` | `deepseek/deepseek-chat` | Very cheap | Yes (`usage.cost`) |
| `openai` | `gpt-4o-mini` | Cheap | No (tokens only) |
| `gemini` | `gemini-2.5-flash-lite` | Cheap | No (tokens only) |

### OpenRouter (recommended)

```yaml
provider: openrouter
api_key: "sk-or-v1-..."
model: "deepseek/deepseek-chat"
```

Get a key at: https://openrouter.ai/keys

### OpenAI

```yaml
provider: openai
api_key: "sk-..."
model: "gpt-4o-mini"
input_price_per_1m: 0.15
output_price_per_1m: 0.60
```

### Gemini

```yaml
provider: gemini
api_key: "AIza..."
model: "gemini-2.5-flash-lite"
```

Built-in default prices ($0.10 input / $0.40 output per 1M tokens). Update with `gitai pricing update`.

---

## Commit and PR format

### Conventional Commit

AI returns JSON and gitai formats:

```
fix(leads): do not create clients with invalid broker

- avoids FK violation
- sets broker to null when invalid

Co-authored-by: Name <email@example.com>
```

Accepted types: `fix`, `feat`, `refactor`, `docs`, `test`, `chore`, `perf`, `ci`, `build`, `style`.

### Pull Request

| Section | Content |
|---------|---------|
| **Summary** | 2–4 bullets — why and business impact |
| **Changes** | 3–8 technical bullets by area/file |
| **Test plan** | Actionable checklist for validation |
| **Notes** | Risks, breaking changes, migrations (optional) |

> Commit and PR **language** follows the `language` field in `gitai config` (default `pt-BR`). The TUI itself is always in English.

---

## Troubleshooting

### `gitai: command not found`

Run the installer or add to PATH:

```bash
./install.sh
# or
export PATH="$HOME/sdk/go/bin:$PATH:$HOME/go/bin"
source ~/.zshrc
```

### `config not found. Run: gitai config init`

```bash
gitai config init
```

### `api_key not configured`

Set in YAML or:

```bash
export GITAI_API_KEY="your-key"
```

### `base branch "main" not found`

```bash
git fetch origin
git branch -a   # confirm origin/main
```

Or adjust in config / flag:

```bash
gitai pr --base develop
```

### `PR already exists`

The branch already has a PR. Open the displayed link or close/merge the existing PR.

### `gh: command not found` or auth error

```bash
brew install gh        # macOS
gh auth login
gh auth status
```

### Truncated diff

Increase in config:

```yaml
max_diff_bytes: 200000
```

### Cost not shown

- Use **OpenRouter** for automatic real cost
- Run `gitai pricing update` for Gemini
- Or set `input_price_per_1m` and `output_price_per_1m` in YAML

### Empty `gitai report`

The ledger is only filled after running `commit`, `push`, or `pr` with AI. Check `~/.config/gitai/usage/ledger.csv`.

---

## Security

- **Never** commit `config.yaml` or `.gitai.yaml` with API keys
- Add `.gitai.yaml` to `.gitignore` if it contains local secrets
- Prefer `GITAI_API_KEY` in CI and shared environments
- `gitai config show` masks the key (`sk-o...abcd`)
- Global config is saved with permission `0600` (user read only)

---

## License

MIT
