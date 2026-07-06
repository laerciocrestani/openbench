# AGENTS.md

## Cursor Cloud specific instructions

`gitia` é uma CLI em Go (Go 1.22, `github.com/spf13/cobra` + `gopkg.in/yaml.v3`) que gera
conventional commits a partir de um `git diff` via IA (OpenAI / OpenRouter / Gemini) e
integra com o GitHub CLI (`gh`) para push + PR. É um binário único, sem servidor/DB.

### Build / lint / test / run

Comandos padrão (rodar da raiz do repo):

- Build: `go build ./...`
- Lint: `go vet ./...`
- Test: `go test ./...`
- Instalar o binário: `go install ./cmd/gitia` (vai para `$(go env GOPATH)/bin`)
- Rodar: `gitia --help` (garanta que `~/go/bin` está no `PATH`)

### Versionamento

A versão é **automática**, derivada do número de commits no repositório (sem tags git).
O primeiro commit equivale a `v0.1.0`; cada commit adicional incrementa o patch.
Ex.: 13 commits → `v0.1.12`. O `go install` injeta versão e commit via `-ldflags`.

### Caveats não óbvios

- **Toda** ação de `commit`/`push`/`pr` (inclusive com `--dry-run`) carrega a config e
  faz uma chamada HTTP real ao provider de IA. Sem `api_key` válida o comando falha.
- A chave pode vir do arquivo de config OU da env var `GITIA_API_KEY`.
- Config: `~/.config/gitia/config.yaml` (ou `.gitia.yaml` local, ou `GITIA_CONFIG`).
- `gitia config` preserva valores existentes — Enter mantém cada campo.
- `gitia update` funciona de qualquer diretório (usa clone salvo ou GitHub).
- `gitia pr` requer `gh` autenticado (`gh auth login`).
