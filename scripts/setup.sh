#!/usr/bin/env bash
# gitia-setup — instala, configura e atualiza o gitia
#
# Uso:
#   ./scripts/setup.sh install   # instala o binário e configura PATH
#   ./scripts/setup.sh config    # wizard de configuração (API key, provider...)
#   ./scripts/setup.sh update    # puxa última versão e reinstala
#   ./scripts/setup.sh help      # ajuda
#
# Fluxo recomendado (primeira vez):
#   git clone https://github.com/laerciocrestani/gitia.git
#   cd gitia
#   ./scripts/setup.sh install
#   ./scripts/setup.sh config

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

info()  { printf '→ %s\n' "$*"; }
ok()    { printf '✓ %s\n' "$*"; }
warn()  { printf '! %s\n' "$*" >&2; }
fail()  { printf '✗ %s\n' "$*" >&2; exit 1; }

go_bin_dir() {
  if command -v go >/dev/null 2>&1; then
    echo "$(go env GOPATH)/bin"
  else
    echo "$HOME/go/bin"
  fi
}

gitia_bin() {
  local bin
  bin="$(go_bin_dir)/gitia"
  if [[ -x "$bin" ]]; then
    echo "$bin"
    return 0
  fi
  if command -v gitia >/dev/null 2>&1; then
    command -v gitia
    return 0
  fi
  return 1
}

require_go() {
  command -v go >/dev/null 2>&1 || fail "Go não encontrado. Instale: https://go.dev/dl/"
  local ver
  ver="$(go env GOVERSION | sed 's/go//')"
  info "Go $ver detectado"
}

check_optional_tools() {
  command -v git >/dev/null 2>&1 || warn "git não encontrado — necessário para usar o gitia"
  if ! command -v gh >/dev/null 2>&1; then
    warn "gh não encontrado — necessário apenas para 'gitia pr' (https://cli.github.com/)"
  elif ! gh auth status >/dev/null 2>&1; then
    warn "gh não autenticado — rode: gh auth login"
  fi
}

ensure_path() {
  local go_bin shell_rc line
  go_bin="$(go_bin_dir)"

  if echo "$PATH" | tr ':' '\n' | grep -qx "$go_bin"; then
    ok "PATH já inclui $go_bin"
    return 0
  fi

  line="export PATH=\"\$PATH:$go_bin\""

  if [[ -n "${ZSH_VERSION:-}" ]] || [[ "${SHELL:-}" == *zsh* ]]; then
    shell_rc="$HOME/.zshrc"
  else
    shell_rc="$HOME/.bashrc"
  fi

  if [[ -f "$shell_rc" ]] && grep -Fq "$go_bin" "$shell_rc" 2>/dev/null; then
    ok "Entrada PATH já existe em $shell_rc"
    return 0
  fi

  info "Adicionando $go_bin ao PATH em $shell_rc"
  {
    echo ""
    echo "# gitia (Go bin)"
    echo "$line"
  } >> "$shell_rc"

  ok "PATH configurado. Rode: source $shell_rc"
  warn "Ou abra um novo terminal antes de usar 'gitia'"
}

cmd_install() {
  info "Instalando gitia..."
  require_go
  check_optional_tools

  cd "$REPO_ROOT"
  go install ./cmd/gitia

  local bin
  bin="$(gitia_bin)" || fail "Instalação falhou — binário não encontrado em $(go_bin_dir)"
  ok "gitia instalado em $bin"

  ensure_path

  echo ""
  ok "Instalação concluída!"
  info "Próximo passo: ./scripts/setup.sh config"
  info "Teste: $bin --help"
}

cmd_config() {
  local bin
  if ! bin="$(gitia_bin)"; then
    warn "gitia não encontrado no PATH"
    info "Instalando primeiro..."
    cmd_install
    bin="$(gitia_bin)"
  fi

  info "Abrindo wizard de configuração..."
  "$bin" config init

  echo ""
  ok "Configuração salva!"
  info "Verifique: $bin config show"
  info "Use: $bin pr"
}

cmd_update() {
  require_go

  if [[ ! -d "$REPO_ROOT/.git" ]]; then
    fail "Diretório git não encontrado. Rode update dentro do clone do repositório."
  fi

  cd "$REPO_ROOT"

  local branch before after
  branch="$(git rev-parse --abbrev-ref HEAD)"
  before="$(git rev-parse --short HEAD)"

  info "Atualizando branch '$branch'..."
  git fetch origin "$branch" 2>/dev/null || git fetch origin
  git pull --ff-only origin "$branch" 2>/dev/null || git pull --ff-only

  after="$(git rev-parse --short HEAD)"

  info "Reinstalando binário..."
  go install ./cmd/gitia

  local bin
  bin="$(gitia_bin)" || fail "Reinstalação falhou"
  ok "gitia atualizado em $bin"

  if [[ "$before" == "$after" ]]; then
    info "Já estava na versão mais recente ($after)"
  else
    ok "Atualizado: $before → $after"
    git log -1 --oneline
  fi

  echo ""
  info "Teste: $bin --help"
}

usage() {
  cat <<'EOF'
gitia-setup — instala, configura e atualiza o gitia

Uso:
  ./scripts/setup.sh install    Instala o binário e configura PATH
  ./scripts/setup.sh config     Wizard de configuração (provider, API key...)
  ./scripts/setup.sh update     git pull + reinstala
  ./scripts/setup.sh help       Exibe esta ajuda

Primeira instalação:
  git clone https://github.com/laerciocrestani/gitia.git
  cd gitia
  ./scripts/setup.sh install
  ./scripts/setup.sh config

Depois:
  gitia pr
EOF
}

main() {
  local cmd="${1:-help}"
  case "$cmd" in
    install) cmd_install ;;
    config)  cmd_config ;;
    update)  cmd_update ;;
    help|-h|--help) usage ;;
    *) fail "Comando desconhecido: $cmd (use: install | config | update | help)" ;;
  esac
}

main "$@"
