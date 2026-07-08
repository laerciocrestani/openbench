#!/usr/bin/env bash
#
# gitai installer — instala Go (se necessário), compila gitai, configura PATH e roda o wizard.
#
# Uso:
#   ./install.sh
#   curl -fsSL https://raw.githubusercontent.com/laerciocrestani/gitai/main/install.sh | bash
#
# Opções:
#   --no-config    Não executa gitai config ao final
#   --skip-go      Falha se Go não estiver instalado (não instala automaticamente)
#   -h, --help     Ajuda
#
set -euo pipefail

readonly GITAI_REPO_URL="${GITAI_REPO_URL:-https://github.com/laerciocrestani/gitai.git}"
readonly GITAI_INSTALL_DIR="${GITAI_INSTALL_DIR:-${HOME}/.config/gitai/repository}"
readonly GO_VERSION="${GO_VERSION:-1.25.0}"
readonly GO_MIN_VERSION="${GO_MIN_VERSION:-1.22}"
readonly GO_SDK_DIR="${GO_SDK_DIR:-${HOME}/sdk/go}"
readonly PATH_MARKER="# gitai installer"

RUN_CONFIG=1
SKIP_GO_INSTALL=0

log()  { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m✓\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
gitai installer

Instala, nesta ordem:
  1. Verifica dependências (git, curl)
  2. Instala Go em ~/sdk/go se não houver versão compatível
  3. Clona ou usa o repositório gitai
  4. Compila e instala o binário (go run ./cmd/gitai install)
  5. Adiciona Go e ~/go/bin ao ~/.zshrc ou ~/.bashrc
  6. Executa gitai config (wizard interativo)

Uso:
  ./install.sh
  curl -fsSL https://raw.githubusercontent.com/laerciocrestani/gitai/main/install.sh | bash

Variáveis:
  GITAI_REPO_URL      URL do repositório (default: GitHub oficial)
  GITAI_INSTALL_DIR   Diretório do clone (default: ~/.config/gitai/repository)
  GO_VERSION          Versão do Go a instalar (default: 1.25.0)
  GO_SDK_DIR          Onde extrair o Go (default: ~/sdk/go)

Opções:
  --no-config         Pula o wizard gitai config
  --skip-go           Não instala Go automaticamente
  -h, --help          Esta ajuda
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --no-config) RUN_CONFIG=0; shift ;;
      --skip-go)   SKIP_GO_INSTALL=1; shift ;;
      -h|--help)   usage; exit 0 ;;
      *) die "Opção desconhecida: $1 (use --help)" ;;
    esac
  done
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Comando obrigatório não encontrado: $1"
}

version_ge() {
  # Retorna 0 se $1 >= $2 (semver simples)
  local current="${1#go}"
  local required="${2#go}"
  printf '%s\n%s\n' "$required" "$current" | sort -V -C
}

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    linux)  GOOS=linux ;;
    darwin) GOOS=darwin ;;
    *) die "SO não suportado: $os (use Linux ou macOS)" ;;
  esac
  case "$arch" in
    x86_64|amd64)  GOARCH=amd64 ;;
    arm64|aarch64) GOARCH=arm64 ;;
    *) die "Arquitetura não suportada: $arch" ;;
  esac
}

shell_rc_file() {
  local home shell
  home="${HOME}"
  shell="${SHELL:-}"
  if [[ "$shell" == *zsh* ]] && [[ -f "${home}/.zshrc" ]]; then
    echo "${home}/.zshrc"
    return
  fi
  if [[ -f "${home}/.zshrc" ]]; then
    echo "${home}/.zshrc"
    return
  fi
  if [[ -f "${home}/.bashrc" ]]; then
    echo "${home}/.bashrc"
    return
  fi
  echo ""
}

append_path_block() {
  local rc go_bin gopath_bin
  rc="$(shell_rc_file)"
  go_bin="${GO_SDK_DIR}/bin"
  gopath_bin="${HOME}/go/bin"

  if [[ -z "$rc" ]]; then
    warn "Não encontrei ~/.zshrc nem ~/.bashrc — adicione manualmente ao PATH:"
    warn "  export PATH=\"${go_bin}:\$PATH:${gopath_bin}\""
    return 0
  fi

  if grep -qF "$PATH_MARKER" "$rc" 2>/dev/null; then
    ok "PATH já configurado em ${rc}"
    return 0
  fi

  {
    echo ""
    echo "$PATH_MARKER"
    echo "export PATH=\"${go_bin}:\$PATH\""
    echo "export PATH=\"\$PATH:${gopath_bin}\""
  } >>"$rc"

  ok "PATH gravado em ${rc}"
  warn "Abra um novo terminal ou rode: source ${rc}"
}

export_paths_for_session() {
  export PATH="${GO_SDK_DIR}/bin:${PATH}"
  if command -v go >/dev/null 2>&1; then
    export PATH="${PATH}:$(go env GOPATH)/bin"
  else
    export PATH="${PATH}:${HOME}/go/bin"
  fi
}

step_preflight() {
  log "1/6 Verificando dependências"
  need_cmd git
  need_cmd curl
  need_cmd tar
  ok "git, curl e tar disponíveis"
}

step_install_go() {
  log "2/6 Verificando Go (mínimo ${GO_MIN_VERSION})"

  if command -v go >/dev/null 2>&1; then
    local ver
    ver="$(go env GOVERSION 2>/dev/null || go version | awk '{print $3}')"
    if version_ge "$ver" "$GO_MIN_VERSION"; then
      ok "Go já instalado (${ver})"
      return 0
    fi
    warn "Go ${ver} é antigo — será instalado Go ${GO_VERSION}"
  elif [[ "$SKIP_GO_INSTALL" -eq 1 ]]; then
    die "Go não encontrado. Instale em https://go.dev/dl/ ou rode sem --skip-go"
  else
    warn "Go não encontrado — instalando Go ${GO_VERSION}"
  fi

  detect_platform
  local archive="go${GO_VERSION}.${GOOS}-${GOARCH}.tar.gz"
  local url="https://go.dev/dl/${archive}"
  local tmp
  tmp="$(mktemp -d)"

  log "Baixando ${url}"
  curl -fsSL "$url" -o "${tmp}/${archive}"

  log "Extraindo em ${GO_SDK_DIR}"
  rm -rf "${GO_SDK_DIR}"
  mkdir -p "$(dirname "${GO_SDK_DIR}")"
  tar -C "$(dirname "${GO_SDK_DIR}")" -xzf "${tmp}/${archive}"
  mv "$(dirname "${GO_SDK_DIR}")/go" "${GO_SDK_DIR}"
  rm -rf "$tmp"

  export PATH="${GO_SDK_DIR}/bin:${PATH}"
  mkdir -p "${HOME}/.config/gitai"
  echo "${GO_SDK_DIR}" >"${HOME}/.config/gitai/.go-sdk-installed"
  ok "Go $(go version | awk '{print $3}') instalado em ${GO_SDK_DIR}"
}

resolve_repo_root() {
  local script_dir=""
  if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  fi

  if [[ -n "$script_dir" ]] && [[ -f "${script_dir}/go.mod" ]] && [[ -d "${script_dir}/.git" ]]; then
    echo "$script_dir"
    return 0
  fi

  if [[ -n "${GITAI_ROOT:-}" ]] && [[ -f "${GITAI_ROOT}/go.mod" ]]; then
    echo "$(cd "${GITAI_ROOT}" && pwd)"
    return 0
  fi

  if [[ -d "${GITAI_INSTALL_DIR}/.git" ]] && [[ -f "${GITAI_INSTALL_DIR}/go.mod" ]]; then
    log "Atualizando clone existente em ${GITAI_INSTALL_DIR}"
    git -C "${GITAI_INSTALL_DIR}" fetch --quiet origin 2>/dev/null || true
    git -C "${GITAI_INSTALL_DIR}" pull --ff-only --quiet 2>/dev/null || true
    echo "${GITAI_INSTALL_DIR}"
    return 0
  fi

  log "Clonando repositório em ${GITAI_INSTALL_DIR}"
  mkdir -p "$(dirname "${GITAI_INSTALL_DIR}")"
  git clone --depth 1 "$GITAI_REPO_URL" "${GITAI_INSTALL_DIR}"
  echo "${GITAI_INSTALL_DIR}"
}

step_repository() {
  log "3/6 Preparando repositório gitai"
  REPO_ROOT="$(resolve_repo_root)"
  ok "Repositório em ${REPO_ROOT}"
}

step_build_install() {
  log "4/6 Compilando e instalando gitai"
  export_paths_for_session
  (
    cd "$REPO_ROOT"
    go run ./cmd/gitai install
  )
  ok "Binário instalado"
}

step_shell_path() {
  log "5/6 Configurando PATH no shell"
  append_path_block
  export_paths_for_session
}

step_config() {
  log "6/6 Configuração inicial"
  export_paths_for_session

  local gitai_bin
  if command -v gitai >/dev/null 2>&1; then
    gitai_bin="gitai"
  elif [[ -x "${HOME}/go/bin/gitai" ]]; then
    gitai_bin="${HOME}/go/bin/gitai"
  else
    die "gitai não encontrado após instalação"
  fi

  if [[ "$RUN_CONFIG" -eq 0 ]]; then
    warn "Wizard pulado (--no-config). Rode depois: gitai config"
    return 0
  fi

  if [[ ! -t 0 ]]; then
    warn "Terminal não interativo — rode manualmente: gitai config"
    return 0
  fi

  log "Iniciando wizard (provider, API key, idioma…)"
  "$gitai_bin" config
  ok "Configuração concluída"
}

finish() {
  export_paths_for_session
  echo ""
  ok "Instalação completa!"
  echo ""
  echo "  gitai              Dashboard TUI (dentro de um repo git)"
  echo "  gitai commit       Commit com IA"
  echo "  gitai pr           Pull Request com IA"
  echo "  gitai config show  Ver configuração"
  echo "  gitai update       Atualizar binário"
  echo ""
  if ! command -v gitai >/dev/null 2>&1; then
    warn "O comando gitai pode não estar no PATH desta sessão."
    warn "Rode: source $(shell_rc_file)  (ou abra um novo terminal)"
  fi
}

main() {
  parse_args "$@"
  echo ""
  echo "  gitai installer"
  echo "  ─────────────────────────────────────"
  echo ""

  step_preflight
  step_install_go
  step_repository
  step_build_install
  step_shell_path
  step_config
  finish
}

main "$@"
