# AGENTS.md

## Cursor Cloud specific instructions

`gitia` é uma CLI em Go (Go 1.22, `github.com/spf13/cobra` + `gopkg.in/yaml.v3`) que gera
conventional commits a partir de um `git diff` via IA (OpenAI / OpenRouter / Gemini) e
integra com o GitHub CLI (`gh`) para push + PR. É um binário único, sem servidor/DB.

### Build / lint / test / run

Comandos padrão (rodar da raiz do repo):

- Build: `go build ./...`
- Lint: `go vet ./...` (não há `golangci-lint` configurado no repo)
- Test: `go test ./...` (atualmente não há arquivos de teste)
- Instalar o binário: `go install ./cmd/gitia` (vai para `$(go env GOPATH)/bin`, i.e. `~/go/bin`)
- Rodar: `gitia --help` (garanta que `~/go/bin` está no `PATH`)

### Caveats não óbvios

- **Toda** ação de `commit`/`push`/`pr` (inclusive com `--dry-run`) carrega a config e
  faz uma chamada HTTP real ao provider de IA para gerar a mensagem. Sem uma `api_key`
  válida o comando falha com erro da API (ex.: `401`). Só `config init` e `config show`
  funcionam sem chave.
- A chave pode vir do arquivo de config OU da env var `GITIA_API_KEY` (que sobrescreve o
  arquivo). Para testes/CI use `GITIA_API_KEY`.
- Caminho da config: `~/.config/gitia/config.yaml`. Pode-se sobrescrever com a env var
  `GITIA_CONFIG` (útil para testes sem poluir o home). Um `.gitia.yaml` na raiz do repo
  tem precedência sobre a config global.
- `config init` é um wizard interativo (lê de stdin). Para automação, faça pipe das
  respostas: `printf 'openrouter\n<key>\ndeepseek/deepseek-chat\npt-BR\nmain\n\n' | gitia config init`.
- Em `--dry-run` o `git add` NÃO é executado (apenas logado). Além disso, `gitia` usa
  `git diff --cached` e faz fallback para `git diff` — arquivos novos não rastreados não
  aparecem em `git diff`, então rode `git add` antes de testar o fluxo de commit com
  arquivos novos (ou use um arquivo já rastreado).
- Para testar `commit`/`push`/`pr` sem afetar este repo, opere dentro de um repositório
  git descartável (ex.: `git init` em `/tmp/...`), pois a CLI age sobre o diretório atual.
- `gitia pr` requer `gh` autenticado (`gh auth login`).
