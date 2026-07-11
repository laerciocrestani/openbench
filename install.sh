#!/usr/bin/env bash
#
# openbench installer — instala Go (se necessário), compila ob, configura PATH e roda o wizard.
#
# Uso:
#   ./install.sh
#   curl -fsSL https://raw.githubusercontent.com/laerciocrestani/openbench/main/install.sh | bash
#
# Opções:
#   --no-config    Não executa ob config ao final
#   --skip-go      Falha se Go não estiver instalado (não instala automaticamente)
#   -h, --help     Ajuda
#
set -euo pipefail

# ./install.sh reaplica o profile na sessão do instalador.
# Para instalar no shell interativo atual (sem subshell), prefira: source ./install.sh
if [[ "${BASH_SOURCE[0]}" == "${0}" ]] && [[ -z "${OPENBENCH_INSTALL_SOURCED:-}" ]]; then
  export OPENBENCH_INSTALL_SOURCED=1
  # shellcheck disable=SC1090
  source "${BASH_SOURCE[0]}"
  exit $?
fi

readonly OB_REPO_URL="${OB_REPO_URL:-https://github.com/laerciocrestani/openbench.git}"
readonly OB_INSTALL_DIR="${OB_INSTALL_DIR:-${HOME}/.config/openbench/repository}"
readonly GO_VERSION="${GO_VERSION:-1.25.0}"
readonly GO_MIN_VERSION="${GO_MIN_VERSION:-1.22}"
readonly GO_SDK_DIR="${GO_SDK_DIR:-${HOME}/sdk/go}"
readonly PATH_MARKER="# openbench installer"
readonly ALIAS_MARKER="# openbench alias (ob)"

RUN_CONFIG=1
SKIP_GO_INSTALL=0
CREATE_ALIAS=1

log()  { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m✓\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
openbench installer

Instala, nesta ordem:
  1. Verifica dependências (git, curl)
  2. Instala Go em ~/sdk/go se não houver versão compatível
  3. Clona ou usa o repositório openbench
  4. Compila e instala o binário (go run ./cmd/ob install)
  5. Adiciona Go e ~/go/bin ao ~/.zshrc ou ~/.bashrc
  6. Opcional: alias ob → openbench no shell
  7. Executa ob config (wizard interativo)

Uso:
  ./install.sh
  curl -fsSL https://raw.githubusercontent.com/laerciocrestani/openbench/main/install.sh | bash

Variáveis:
  OB_REPO_URL         URL do repositório (default: GitHub oficial)
  OB_INSTALL_DIR      Diretório do clone (default: ~/.config/openbench/repository)
  GO_VERSION          Versão do Go a instalar (default: 1.25.0)
  GO_SDK_DIR          Onde extrair o Go (default: ~/sdk/go)

Opções:
  --no-config         Pula o wizard ob config
  --no-alias          Não cria alias ob no shell
  --skip-go           Não instala Go automaticamente
  -h, --help          Esta ajuda
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --no-config) RUN_CONFIG=0; shift ;;
      --no-alias)  CREATE_ALIAS=0; shift ;;
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
  reload_shell_profile
}

resolve_openbench_bin() {
  export_paths_for_session
  if command -v openbench >/dev/null 2>&1; then
    command -v openbench
    return 0
  fi
  if [[ -x "${HOME}/go/bin/openbench" ]]; then
    echo "${HOME}/go/bin/openbench"
    return 0
  fi
  if command -v go >/dev/null 2>&1; then
    local candidate
    candidate="$(go env GOPATH)/bin/openbench"
    if [[ -x "$candidate" ]]; then
      echo "$candidate"
      return 0
    fi
  fi
  return 1
}

prompt_ob_alias() {
  if [[ "$CREATE_ALIAS" -eq 0 ]]; then
    return 0
  fi
  if [[ ! -t 0 ]]; then
    ok "Terminal não interativo — alias ob será criado automaticamente"
    return 0
  fi

  echo ""
  read -r -p "Criar alias 'ob' no shell para chamar openbench? [S/n] " reply
  case "${reply:-S}" in
    n|N|no|No|NO)
      CREATE_ALIAS=0
      ok "Alias ob não será criado"
      ;;
    *)
      ok "Alias ob será adicionado ao shell"
      ;;
  esac
}

append_ob_alias_block() {
  [[ "$CREATE_ALIAS" -eq 1 ]] || return 0

  local rc bin
  rc="$(shell_rc_file)"
  if [[ -z "$rc" ]]; then
    warn "Não encontrei ~/.zshrc nem ~/.bashrc — adicione manualmente:"
    warn "  alias ob='$(resolve_openbench_bin)'"
    return 0
  fi

  if ! bin="$(resolve_openbench_bin)"; then
    warn "openbench não encontrado — alias ob não criado"
    return 0
  fi

  if grep -qF "$ALIAS_MARKER" "$rc" 2>/dev/null; then
    ok "Alias ob já configurado em ${rc}"
    return 0
  fi

  {
    echo ""
    echo "$ALIAS_MARKER"
    echo "alias ob='${bin}'"
  } >>"$rc"

  ok "Alias ob gravado em ${rc}"
  reload_shell_profile
}

reload_shell_profile() {
  local rc bin old_opts
  rc="$(shell_rc_file)"
  [[ -n "$rc" && -f "$rc" ]] || return 0

  export_paths_for_session

  old_opts="$-"
  set +eu
  # shellcheck disable=SC1090
  source "$rc" 2>/dev/null || true
  [[ "$old_opts" == *e* ]] && set -e
  [[ "$old_opts" == *u* ]] && set -u

  export_paths_for_session

  if bin="$(resolve_openbench_bin 2>/dev/null)"; then
    alias ob="$bin" 2>/dev/null || true
  fi

  hash -r 2>/dev/null || true
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
  log "1/7 Verificando dependências"
  need_cmd git
  need_cmd curl
  need_cmd tar
  ok "git, curl e tar disponíveis"
}

step_install_go() {
  log "2/7 Verificando Go (mínimo ${GO_MIN_VERSION})"

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
  mkdir -p "${HOME}/.config/openbench"
  echo "${GO_SDK_DIR}" >"${HOME}/.config/openbench/.go-sdk-installed"
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

  if [[ -n "${OPENBENCH_ROOT:-}" ]] && [[ -f "${OPENBENCH_ROOT}/go.mod" ]]; then
    echo "$(cd "${OPENBENCH_ROOT}" && pwd)"
    return 0
  fi

  if [[ -d "${OB_INSTALL_DIR}/.git" ]] && [[ -f "${OB_INSTALL_DIR}/go.mod" ]]; then
    log "Atualizando clone existente em ${OB_INSTALL_DIR}"
    git -C "${OB_INSTALL_DIR}" fetch --quiet origin 2>/dev/null || true
    git -C "${OB_INSTALL_DIR}" pull --ff-only --quiet 2>/dev/null || true
    echo "${OB_INSTALL_DIR}"
    return 0
  fi

  log "Clonando repositório em ${OB_INSTALL_DIR}"
  mkdir -p "$(dirname "${OB_INSTALL_DIR}")"
  git clone --depth 1 "$OB_REPO_URL" "${OB_INSTALL_DIR}"
  echo "${OB_INSTALL_DIR}"
}

step_repository() {
  log "3/7 Preparando repositório openbench"
  REPO_ROOT="$(resolve_repo_root)"
  ok "Repositório em ${REPO_ROOT}"
}

step_build_install() {
  log "4/7 Compilando e instalando openbench"
  export_paths_for_session
  cd "$REPO_ROOT"
  go run ./cmd/ob install
  ok "Binário instalado como openbench e ob"
}

step_shell_path() {
  log "5/7 Configurando PATH no shell"
  append_path_block
  export_paths_for_session
}

step_ob_alias() {
  log "6/7 Configurando alias ob"
  prompt_ob_alias
  append_ob_alias_block
  export_paths_for_session
}

step_config() {
  log "7/7 Configuração inicial"
  reload_shell_profile

  local ob_bin
  if command -v ob >/dev/null 2>&1; then
    ob_bin="ob"
  elif command -v openbench >/dev/null 2>&1; then
    ob_bin="openbench"
  elif [[ -x "${HOME}/go/bin/openbench" ]]; then
    ob_bin="${HOME}/go/bin/openbench"
  else
    die "openbench não encontrado após instalação"
  fi

  if [[ "$RUN_CONFIG" -eq 0 ]]; then
    warn "Wizard pulado (--no-config). Rode depois: ob config"
    return 0
  fi

  if [[ ! -t 0 ]]; then
    warn "Terminal não interativo — rode manualmente: ob config"
    return 0
  fi

  log "Iniciando wizard (provider, API key, idioma…)"
  "$ob_bin" config
  ok "Configuração concluída"
}

finish() {
  reload_shell_profile
  echo ""
  ok "Instalação completa!"
  echo ""
  echo "  openbench          Binário principal"
  echo "  ob                 Atalho (binário ou alias)"
  echo "  ob docker up       Subir ambiente Docker Compose"
  echo "  ob commit          Commit com IA"
  echo "  ob pr              Pull Request com IA"
  echo "  ob config show     Ver configuração"
  echo "  ob update          Atualizar binário"
  echo ""
}

main() {
  parse_args "$@"
  echo ""
  echo "  openbench installer"
  echo "  ─────────────────────────────────────"
  echo ""

  step_preflight
  step_install_go
  step_repository
  step_build_install
  step_shell_path
  step_ob_alias
  step_config
  finish
}

main "$@"
