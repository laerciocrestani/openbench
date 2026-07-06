<p align="center">
  <img src="avatar.png" alt="Gitia" width="160">
</p>

# gitia

CLI em Go para gerar **Conventional Commits** com IA barata, automatizar **push** e criar **Pull Requests detalhados** via GitHub CLI.

---

## Sumário

- [Por quê usar o gitia?](#por-quê-usar-o-gitia)
- [Requisitos](#requisitos)
- [Instalação rápida](#instalação-rápida)
- [Instalação manual](#instalação-manual)
- [Atualização](#atualização)
- [Configuração](#configuração)
- [Versionamento](#versionamento)
- [Referência de comandos](#referência-de-comandos)
- [Flags globais e por comando](#flags-globais-e-por-comando)
- [Uso detalhado](#uso-detalhado)
- [Uso de tokens e custo](#uso-de-tokens-e-custo)
- [Providers de IA](#providers-de-ia)
- [Formato do commit e do PR](#formato-do-commit-e-do-pr)
- [Troubleshooting](#troubleshooting)
- [Segurança](#segurança)
- [Licença](#licença)

---

## Por quê usar o gitia?

Assistentes de IA no editor costumam gastar tokens caros para ler diff, gerar mensagem de commit e executar git. O **gitia** externaliza esse fluxo para uma IA configurável (DeepSeek via OpenRouter, GPT-4o-mini, Gemini Flash) por frações de centavo — funciona com qualquer editor ou agente (Claude Code, Copilot, terminal, etc.).

Com o gitia você obtém:

- Mensagens no padrão **Conventional Commits**
- PRs estruturados com **Summary**, **Changes**, **Test plan** e **Notes**
- Resumo de **tokens e custo** (prévia antes da IA + total após execução)
- **Relatório de gastos** (`gitia report`) com histórico em CSV
- Integração nativa com **`gh pr create`**

---

## Requisitos

| Ferramenta | Versão mínima | Para quê |
|------------|---------------|----------|
| [Go](https://go.dev/dl/) | 1.22+ | Compilar e instalar o gitia |
| [git](https://git-scm.com/) | qualquer recente | Repositório local, diff, commit, push |
| [GitHub CLI (`gh`)](https://cli.github.com/) | autenticado | Criar PR (`gitia pr`) |

Autentique o GitHub CLI antes de usar `gitia pr`:

```bash
gh auth login
gh auth status
```

---

## Instalação rápida

O jeito mais simples — três comandos após clonar:

```bash
git clone https://github.com/laerciocrestani/gitia.git
cd gitia
go run ./cmd/gitia install    # única forma de instalar (primeira vez)
gitia config                  # wizard (provider, API key, idioma...)
```

Pronto. Use:

```bash
gitia status
gitia pr
```

### Comandos de setup

| Comando | O que faz |
|---------|-----------|
| `go run ./cmd/gitia install` | `go install`, verifica dependências, adiciona `~/go/bin` ao PATH |
| `gitia config` | Wizard de configuração (equivale a `gitia config init`) |
| `gitia config show` | Exibe config ativa (API key mascarada) |
| `gitia update` | Atualiza e reinstala o binário (funciona de qualquer diretório) |
| `gitia version` | Versão automática + commit + número de commits |
| `gitia report` | Relatório de uso e custos de IA (últimas 24h por padrão) |
| `gitia pricing update` | Busca preços oficiais do Gemini e salva localmente |
| `gitia status` | Alias para `git status` |

### Atualizar depois

De qualquer diretório:

```bash
gitia update
```

O gitia usa o clone salvo em `~/.config/gitia/source`, a variável `GITIA_ROOT` ou, se não encontrar clone local, baixa a última versão do GitHub automaticamente.

O script `./scripts/setup.sh` ainda funciona como wrapper de compatibilidade.

---

## Instalação manual

### 1. Clonar o repositório

```bash
git clone https://github.com/laerciocrestani/gitia.git
cd gitia
```

### 2. Instalar o binário

```bash
go install ./cmd/gitia
```

O binário é instalado em `~/go/bin/gitia`.

### 3. Adicionar ao PATH (permanente)

Adicione ao `~/.zshrc` (ou `~/.bashrc`):

```bash
export PATH="$PATH:$HOME/go/bin"
```

Recarregue o shell:

```bash
source ~/.zshrc
```

### 4. Verificar instalação

```bash
which gitia
gitia --help
```

Saída esperada de `which gitia`:

```
/Users/seu-usuario/go/bin/gitia
```

### 5. Configurar pela primeira vez

```bash
gitia config init
```

### Alternativa sem alterar o PATH

Execute diretamente pelo caminho completo:

```bash
~/go/bin/gitia config init
~/go/bin/gitia pr
```

---

## Atualização

```bash
gitia update
```

Opcional — apontar para o seu clone local:

```bash
export GITIA_ROOT=~/projects/gitia
gitia update
```

Ou manualmente, dentro do clone:

```bash
cd gitia
git pull origin main
go install ./cmd/gitia
```

---

## Configuração

### Wizard interativo (recomendado)

```bash
gitia config
```

Equivalente a `gitia config init`.

O wizard pergunta, nesta ordem:

| Campo | Opções / default | Descrição |
|-------|------------------|-----------|
| Provider | `openrouter`, `openai`, `gemini` | Serviço de IA |
| API Key | — | Chave do provider (Enter mantém a atual) |
| Model | depende do provider | Modelo de linguagem (Enter mantém) |
| Idioma | default: `pt-BR` | Idioma das mensagens geradas |
| Branch base | default: `main` | Branch usada como base do PR |
| Co-author | opcional | Trailer adicionado ao commit |
| Limpar terminal | `s` / `n` | Limpa o console antes de cada comando gitia |

Se já existir configuração, **Enter em qualquer campo mantém o valor atual** (ex.: `[gemini]`).

Salva em `~/.config/gitia/config.yaml` com permissão `0600`.

### Arquivo de configuração global

Caminho padrão: `~/.config/gitia/config.yaml`

```yaml
provider: openrouter        # openai | gemini | openrouter
api_key: "sk-..."
model: "deepseek/deepseek-chat"
language: "pt-BR"
base_branch: "main"
co_author: ""
max_diff_bytes: 120000
clear_screen: false       # true = limpa o terminal antes de cada comando

# opcional — sobrescreve preços padrão do Gemini
# input_price_per_1m: 0.14
# output_price_per_1m: 0.28
```

### Config local por repositório

Crie `.gitia.yaml` na raiz do projeto. **Tem prioridade** sobre o config global.

Útil para:

- Modelo diferente por projeto
- Branch base `develop` em vez de `main`
- Idioma `en-US` em projetos open source

### Variáveis de ambiente

| Variável | Descrição |
|----------|-----------|
| `GITIA_API_KEY` | Sobrescreve `api_key` do YAML (recomendado em CI) |
| `GITIA_CONFIG` | Caminho alternativo para o arquivo de config |
| `GITIA_ROOT` | Caminho do clone gitia (usado por `gitia update`) |
| `GITIA_NO_CLEAR` | Desativa limpeza do terminal (`clear_screen` ignorado) |
| `GITIA_NO_UI` | Desativa cores, spinner e limpeza do terminal |
| `NO_COLOR` | Desativa cores ANSI |

Exemplo:

```bash
export GITIA_API_KEY="sk-or-v1-..."
export GITIA_CONFIG="$HOME/.config/gitia/work.yaml"
gitia pr
```

### Exibir configuração atual

```bash
gitia config show
```

A API key é **mascarada** na saída (ex.: `sk-o...abcd`).

### Referência completa dos campos

| Campo | Tipo | Obrigatório | Default | Descrição |
|-------|------|-------------|---------|-----------|
| `provider` | string | sim | `openrouter` | `openai`, `gemini` ou `openrouter` |
| `api_key` | string | sim* | — | Chave da API (* ou `GITIA_API_KEY`) |
| `model` | string | sim | depende | Identificador do modelo no provider |
| `language` | string | não | `pt-BR` | Idioma do commit e do PR |
| `base_branch` | string | não | `main` | Branch base padrão para `gitia pr` |
| `co_author` | string | não | vazio | Trailer no commit (ex.: `Co-authored-by: Nome <email@exemplo.com>`) |
| `max_diff_bytes` | int | não | `120000` | Tamanho máximo do diff enviado à IA |
| `clear_screen` | bool | não | `false` | Limpa o terminal antes de cada comando |
| `input_price_per_1m` | float | não | — | USD por 1M tokens de input (estimativa de custo) |
| `output_price_per_1m` | float | não | — | USD por 1M tokens de output (estimativa de custo) |

### Models padrão por provider (wizard)

| Provider | Model default |
|----------|---------------|
| `openrouter` | `deepseek/deepseek-chat` |
| `openai` | `gpt-4o-mini` |
| `gemini` | `gemini-2.5-flash-lite` |

---

## Versionamento

A versão é **automática**, calculada pelo número de commits no repositório (sem tags git):

- 1º commit → `v0.1.0`
- cada commit adicional incrementa o patch → ex.: 14 commits = `v0.1.13`

```bash
gitia version
```

Exibe versão, commit, total de commits e se a árvore está dirty.

O `go install` injeta versão e commit via `-ldflags` no build.

---

## Referência de comandos

Rodar **`gitia` sem subcomando** exibe um **overview** do repositório (comando padrão), nesta ordem:

1. **Gitia config** — provider, model, API key
2. **Recent commits** — últimos 5 commits
3. **Branches** — branches locais com tracking
4. **Changed files** — status e `+/-` linhas
5. **Repository / Branch / Status / Sync** — meta do repo
6. **Next steps** — sugestões (`gitia commit`, `gitia pr`, etc.)

Também exibe PR aberto (via `gh`), stash e delta em relação à base quando aplicável.

```
gitia                 Overview do repositório (default)
├── sync              fetch + pull da branch base (--prune para limpar branches)
├── update            atualiza e reinstala o binário
├── version           versão automática + commit
├── report            relatório de uso e custos de IA
├── status            alias para git status
├── commit            gera commit com IA a partir do diff local
├── push              commit (se houver diff) + push para origin
├── pr                commit (se necessário) + push + PR detalhado via gh
├── pricing           preços Gemini (update / show / report)
│   ├── update        busca preços oficiais na web
│   ├── show          exibe preços salvos
│   └── report        alias de gitia report --all
└── config            wizard de configuração (ou subcomandos init/show)
    ├── init          wizard interativo (alias de gitia config)
    └── show          exibe config ativa (key mascarada)
```

> Instalação (uma vez, a partir do clone): `go run ./cmd/gitia install`

### Visão geral

| Comando | O que faz | Chama IA? | Executa git? | Executa gh? |
|---------|-----------|-----------|--------------|-------------|
| `gitia` | Overview do repositório | não | leitura | não |
| `gitia sync` | Sincroniza branch base com origin | não | `fetch`, `pull` | não |
| `gitia sync --prune` | Sync + remove branches mergeadas (local e GitHub) | não | `fetch`, `pull`, `branch -d`, `push --delete` | não |
| `gitia sync --prune-remote` | Sync + remove branches mergeadas só no GitHub | não | `fetch`, `pull`, `push --delete` | não |
| `gitia version` | Exibe versão, commit e commits | não | leitura | não |
| `gitia report` | Relatório de uso/custo de IA | não | leitura | não |
| `gitia pricing update` | Atualiza tabela de preços Gemini | não | não | não |
| `gitia commit` | Commit com mensagem gerada | 1× (commit) | `add`, `commit` | não |
| `gitia push` | Commit (se houver diff) + push | 0–1× | `add`, `commit`, `push` | não |
| `gitia pr` | Commit + push + PR | 1–2× (commit + PR) | `add`, `commit`, `push` | `pr create` |
| `gitia status` | Exibe status do repositório | não | `status` | não |
| `gitia config` | Cria/atualiza config.yaml | não | não | não |
| `gitia config init` | Igual a `gitia config` | não | não | não |
| `gitia config show` | Mostra config | não | não | não |
| `gitia update` | Atualiza e reinstala binário | não | não | não |

---

## Flags globais e por comando

### Flags globais (válidas em todos os comandos)

Disponíveis em `commit`, `push` e `pr`:

| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--dry-run` | bool | `false` | Simula o fluxo: chama a IA, exibe o que seria executado, **não** roda `git commit`, `git push` nem `gh pr create` |
| `--verbose` | bool | `false` | Exibe JSON parseado da IA (type, scope, title, bullets do commit ou seções do PR) |

Exemplos:

```bash
gitia commit --dry-run
gitia pr --verbose --dry-run
gitia push --verbose
```

### Flags do `gitia commit`

| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--no-add` | bool | `false` | Não executa `git add .` — usa apenas arquivos já staged (ou unstaged como fallback) |

```bash
git add src/auth.go
gitia commit --no-add
```

### Flags do `gitia push`

Herda todas as flags de `commit`:

| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--no-add` | bool | `false` | Não executa `git add .` antes do commit |

Após o commit, executa:

```bash
git push -u origin HEAD
```

### Flags do `gitia pr`

Herda flags globais e `--no-add`, mais:

| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--no-add` | bool | `false` | Não executa `git add .` |
| `--draft` | bool | `false` | Cria o PR como **draft** (`gh pr create --draft`) |
| `--base` | string | `base_branch` do config | Branch base do PR (ex.: `main`, `develop`) |

Exemplos:

```bash
gitia pr
gitia pr --draft
gitia pr --base develop
gitia pr --no-add --draft --base main --verbose --dry-run
```

### Combinando flags

```bash
# Preview completo do fluxo pr sem alterar nada
gitia pr --dry-run --verbose

# Commit só do que já está staged, sem push
gitia commit --no-add

# PR draft contra develop, sem git add
git add .
gitia pr --no-add --draft --base develop
```

### Flags do `gitia report`

| Flag | Descrição |
|------|-----------|
| `--hour` | Última hora |
| `--hours N` | Últimas N horas |
| `--days N` | Últimos N dias |
| `--month` | Mês atual (calendário) |
| `--all` | Todo o histórico |

Padrão (sem flags): **últimas 24 horas**.

```bash
gitia report
gitia report --hour
gitia report --days 7
gitia report --all
```

---

## Uso detalhado

### Fluxo recomendado (dia a dia)

```bash
# 1. Trabalhe na sua feature branch
git checkout -b feat/minha-feature

# 2. Faça suas alterações no código

# 3. Commit + push + PR em um comando
gitia pr
```

O `gitia pr` executa internamente:

```
git add .
    ↓
[se houver alterações staged]
    → IA gera mensagem de commit → git commit
    ↓
git push -u origin HEAD
    ↓
git diff base...HEAD  (+ log de commits da branch)
    ↓
IA gera PR detalhado (title, summary, changes, test plan, notes)
    ↓
gh pr create --title "..." --body "..." --base main
    ↓
Exibe resumo de tokens e custo
```

### `gitia commit`

**Quando usar:** só quer commitar, sem push nem PR.

**Fluxo:**

1. `git add .` (salvo com `--no-add`)
2. Obtém diff staged (ou unstaged se nada staged)
3. Envia diff à IA → Conventional Commit
4. `git commit -m "..."`
5. Exibe resumo de tokens/custo

**Diff usado:** alterações locais pendentes (staged prioritário).

```bash
gitia commit
gitia commit --no-add
gitia commit --dry-run --verbose
```

**Erros comuns:**

- `nenhuma alteração para commitar` — working tree limpa
- `diretório atual não é um repositório git` — rode dentro de um repo git

---

### `gitia push`

**Quando usar:** enviar branch para origin. Se houver alterações pendentes, commita antes; senão faz push dos commits existentes.

**Fluxo:** `git add .` (se não `--no-add`) → commit com IA (só se houver diff) → `git push -u origin HEAD`.

```bash
gitia push
gitia push --no-add
gitia push --dry-run
```

> O resumo de tokens/custo é exibido após o commit (dentro do fluxo push). O push em si não consome IA.

---

### `gitia pr` (comando principal)

**Quando usar:** finalizar trabalho na branch — commit pendente, push e PR detalhado.

**Fluxo inteligente:**

| Situação | Comportamento |
|----------|---------------|
| Alterações não commitadas | `git add .` → IA gera commit → commit |
| Só commits na branch, nada pendente | Pula commit, usa commits existentes |
| Branch igual à base, sem mudanças | Erro: `nenhuma alteração em relação à main` |

**Diff usado para o PR:** `git diff base...HEAD` — **todas** as alterações da branch em relação à base, não só o último commit.

**Diff usado para o commit (quando há staged):** apenas o diff staged atual.

**Resolução da branch base:**

1. Tenta `main` (ou valor de `--base` / config)
2. Tenta `origin/main`
3. Erro se nenhuma existir → rode `git fetch`

```bash
gitia pr
gitia pr --draft
gitia pr --base develop
gitia pr --verbose --dry-run
```

**Body do PR gerado:**

```markdown
## Summary
- visão geral e impacto

## Changes
- detalhes técnicos por área

## Test plan
- [ ] passo 1
- [ ] passo 2

## Notes
- riscos ou follow-ups (se houver)
```

**Erros comuns:**

- `PR já existe: https://...` — branch já tem PR aberto
- `branch base "main" não encontrada` — rode `git fetch origin`
- `config não encontrada` — rode `gitia config init`

---

### `gitia config init`

Wizard interativo. Não altera repositórios git — só cria/atualiza o YAML global.

```bash
gitia config init
```

### `gitia config show`

Carrega a config efetiva (local `.gitia.yaml` ou global) e imprime com key mascarada.

```bash
gitia config show
```

---

## Uso de tokens e custo

### Prévia (antes da IA)

Antes do passo `Thinking`, o gitia exibe uma estimativa:

```
Estimativa: ~1750 tokens · $0.000275 USD (Gemini) (input ~1500 + output ~250)
```

### Após execução

Ao final de **`commit`**, **`push`** e **`pr`**:

```
Uso de IA
• commit: 8420 prompt + 186 completion = 8606 tokens | $0.000412 USD (Gemini)
• Total: 8606 prompt + 186 completion = 8792 tokens | custo total: $0.000412 USD
```

Cada chamada é registrada em `~/.config/gitia/usage/ledger.csv` para o `gitia report`.

### Como o custo é calculado

| Provider | Tokens | Custo |
|----------|--------|-------|
| **OpenRouter** | `usage.*` | Real via `usage.cost` (USD) |
| **OpenAI** | `usage.*` | Estimativa (preços padrão ou config) |
| **Gemini** | `usageMetadata.*` | Estimativa com preços padrão ou `gitia pricing update` |

### Preços do Gemini

```bash
gitia pricing update   # busca preços oficiais e salva em ~/.config/gitia/pricing.yaml
gitia pricing show     # exibe tabela salva
```

Modelos com preços padrão embutidos (ex.: `gemini-2.5-flash-lite` → $0.10 / $0.40 por 1M tokens).

### Estimativa manual (override)

Adicione ao config para sobrescrever qualquer provider:

```yaml
input_price_per_1m: 0.15
output_price_per_1m: 0.60
```

### Retries

| Tipo | Comportamento |
|------|---------------|
| **API indisponível** (503, 429, etc.) | Até **3 tentativas**, **3s** entre cada |
| **JSON inválido da IA** | Até 2 tentativas de parse (consome tokens extras) |

### `--dry-run`

A IA **é chamada** (você vê tokens/custo e registro no ledger), mas git/gh **não executam**.

---

## Providers de IA

| Provider | Model recomendado | Custo típico | Custo na resposta |
|----------|-------------------|--------------|-------------------|
| `openrouter` | `deepseek/deepseek-chat` | Muito barato | Sim (`usage.cost`) |
| `openai` | `gpt-4o-mini` | Barato | Não (só tokens) |
| `gemini` | `gemini-2.5-flash-lite` | Barato | Não (só tokens) |

### OpenRouter (recomendado)

```yaml
provider: openrouter
api_key: "sk-or-v1-..."
model: "deepseek/deepseek-chat"
```

Obtenha a key em: https://openrouter.ai/keys

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

Preços padrão embutidos ($0.10 input / $0.40 output por 1M tokens). Atualize com `gitia pricing update`.

---

## Formato do commit e do PR

### Conventional Commit

A IA retorna JSON e o gitia formata:

```
fix(leads): não cria clientes com corretor inválido

- evita violação da FK
- define corretor como null quando inválido

Co-authored-by: Nome <email@exemplo.com>
```

Tipos aceitos: `fix`, `feat`, `refactor`, `docs`, `test`, `chore`, `perf`, `ci`, `build`, `style`.

### Pull Request

| Seção | Conteúdo |
|-------|----------|
| **Summary** | 2–4 bullets — porquê e impacto de negócio |
| **Changes** | 3–8 bullets técnicos por área/arquivo |
| **Test plan** | Checklist acionável para validação |
| **Notes** | Riscos, breaking changes, migrations (opcional) |

---

## Troubleshooting

### `gitia: command not found`

```bash
export PATH="$PATH:$HOME/go/bin"
# ou
~/go/bin/gitia --help
```

### `config não encontrada. Execute: gitia config init`

```bash
gitia config init
```

### `api_key não configurada`

Defina no YAML ou:

```bash
export GITIA_API_KEY="sua-chave"
```

### `branch base "main" não encontrada`

```bash
git fetch origin
git branch -a   # confirme origin/main
```

Ou ajuste no config / flag:

```bash
gitia pr --base develop
```

### `PR já existe`

A branch já tem PR. Abra o link exibido ou feche/merge o PR existente.

### `gh: command not found` ou erro de autenticação

```bash
brew install gh        # macOS
gh auth login
gh auth status
```

### Diff truncado

Aumente no config:

```yaml
max_diff_bytes: 200000
```

### Custo não aparece

- Use **OpenRouter** para custo real automático
- Rode `gitia pricing update` para Gemini
- Ou configure `input_price_per_1m` e `output_price_per_1m` no YAML

### `gitia report` vazio

O ledger só é preenchido após rodar `commit`, `push` ou `pr` com IA. Verifique `~/.config/gitia/usage/ledger.csv`.

---

## Segurança

- **Nunca** commite `config.yaml` ou `.gitia.yaml` com API keys
- Adicione `.gitia.yaml` ao `.gitignore` se contiver secrets locais
- Prefira `GITIA_API_KEY` em CI e ambientes compartilhados
- `gitia config show` mascara a key (`sk-o...abcd`)
- O config global é salvo com permissão `0600` (só o usuário lê)

---

## Licença

MIT
