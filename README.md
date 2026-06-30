# gitia

CLI em Go para gerar **conventional commits** com IA barata e automatizar **push + PR** via GitHub CLI — sem gastar tokens do Cursor Agent.

## Por quê?

Quando o Cursor Agent faz commit/push, ele lê o diff, gera a mensagem e executa git — tudo com o modelo do agente. O `gitia` externaliza isso para uma IA configurável (DeepSeek via OpenRouter, GPT-4o-mini, Gemini Flash) por frações de centavo.

## Instalação

```bash
git clone https://github.com/laerciocrestani/gitia.git
cd gitia
go install ./cmd/gitia
```

Requisitos:

- `git`
- `gh` ([GitHub CLI](https://cli.github.com/)) autenticado (`gh auth login`)

## Configuração

```bash
gitia config init
```

Cria `~/.config/gitia/config.yaml`:

```yaml
provider: openrouter        # openai | gemini | openrouter
api_key: "sk-..."
model: "deepseek/deepseek-chat"
language: "pt-BR"
base_branch: "main"
co_author: ""
max_diff_bytes: 120000
```

Alternativas:

- API key via env: `export GITIA_API_KEY=sk-...`
- Config local por repo: `.gitia.yaml` na raiz do projeto

```bash
gitia config show   # exibe config (key mascarada)
```

## Uso

```bash
# Commit com mensagem gerada por IA
gitia commit

# Commit + push
gitia push

# Commit + push + PR
gitia pr

# Simular sem executar
gitia pr --dry-run

# Usar só arquivos já staged
gitia commit --no-add

# PR draft
gitia pr --draft --base main
```

Fluxo típico:

```bash
git add .
gitia pr
```

## Providers recomendados

| Provider | Model | Custo |
|----------|-------|-------|
| `openrouter` | `deepseek/deepseek-chat` | Muito barato |
| `openai` | `gpt-4o-mini` | Barato |
| `gemini` | `gemini-2.0-flash` | Barato |

## Cursor Hook (bloquear commit do agente)

Copie os arquivos de exemplo para seu user hooks:

```bash
mkdir -p ~/.cursor/hooks
cp examples/cursor-hooks/block-agent-git.sh ~/.cursor/hooks/
cp examples/cursor-hooks/hooks.json ~/.cursor/
chmod +x ~/.cursor/hooks/block-agent-git.sh
```

O hook bloqueia `git commit` e `git push` do Cursor Agent e sugere `gitia pr`.

Adicione também uma regra no Cursor:

> Nunca faça git commit ou git push. Ao final, sugira `gitia pr`.

Reinicie o Cursor após instalar o hook. Verifique em **Settings → Hooks**.

## Formato do commit

A IA retorna JSON estruturado e o gitia formata como Conventional Commit:

```
fix(leads): não cria clientes com corretor inválido

- evita violação da FK
- define corretor como null quando inválido
```

## Segurança

- Nunca commite `config.yaml` com API keys
- Use `GITIA_API_KEY` em CI ou ambientes compartilhados
- `gitia config show` mascara a API key

## Licença

MIT
