#!/usr/bin/env bash
#
# gitai uninstall — remove binário, dados em ~/.config/gitai e entradas de PATH.
#
# Uso:
#   ./uninstall.sh
#   curl -fsSL https://raw.githubusercontent.com/laerciocrestani/gitai/main/uninstall.sh | bash
#
# Opções:
#   -y, --yes       Não pede confirmação
#   --remove-go     Remove também o Go instalado em ~/sdk/go pelo install.sh
#   --keep-go       Mantém o Go em ~/sdk/go mesmo se foi instalado pelo install.sh
#   -h, --help      Ajuda
#
set -euo pipefail

readonly GITAI_CONFIG_DIR="${GITAI_CONFIG_DIR:-${HOME}/.config/gitai}"
readonly GO_SDK_DIR="${GO_SDK_DIR:-${HOME}/sdk/go}"
readonly PATH_MARKER_INSTALLER="# gitai installer"
readonly PATH_MARKER_GOBIN="# gitai (Go bin)"
readonly GO_SDK_MARKER="${GITAI_CONFIG_DIR}/.go-sdk-installed"

ASSUME_YES=0
REMOVE_GO=""
# REMOVE_GO: "" = auto (remove if marker), "1" = force remove, "0" = keep

log()  { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m✓\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
gitai uninstall

Remove do sistema, nesta ordem:
  1. Confirmação (a menos que use -y)
  2. Binário gitai em $(go env GOPATH)/bin
  3. Go em ~/sdk/go (somente se instalado pelo install.sh, ou com --remove-go)
  4. Diretório ~/.config/gitai (config, clone, usage, pricing, etc.)
  5. Blocos de PATH no ~/.zshrc ou ~/.bashrc

Não remove:
  - Arquivos .gitai.yaml em projetos
  - Variáveis GITAI_* exportadas manualmente em outros arquivos
  - Go instalado por outros meios (exceto com --remove-go)

Uso:
  ./uninstall.sh
  curl -fsSL https://raw.githubusercontent.com/laerciocrestani/gitai/main/uninstall.sh | bash

Opções:
  -y, --yes       Confirma sem perguntar
  --remove-go     Remove ~/sdk/go explicitamente
  --keep-go       Não remove ~/sdk/go
  -h, --help      Esta ajuda
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -y|--yes)    ASSUME_YES=1; shift ;;
      --remove-go) REMOVE_GO=1; shift ;;
      --keep-go)   REMOVE_GO=0; shift ;;
      -h|--help)   usage; exit 0 ;;
      *) die "Opção desconhecida: $1 (use --help)" ;;
    esac
  done
}

shell_rc_files() {
  local home="${HOME}"
  for f in "${home}/.zshrc" "${home}/.bashrc" "${home}/.bash_profile"; do
    [[ -f "$f" ]] && echo "$f"
  done
}

gopath_bin_dir() {
  if command -v go >/dev/null 2>&1; then
    echo "$(go env GOPATH)/bin"
    return
  fi
  echo "${HOME}/go/bin"
}

confirm_uninstall() {
  if [[ "$ASSUME_YES" -eq 1 ]]; then
    return 0
  fi
  echo ""
  warn "Isso vai remover o gitai deste computador:"
  echo "  • binário gitai"
  echo "  • ${GITAI_CONFIG_DIR}/ (config, clone, usage, pricing…)"
  echo "  • entradas de PATH adicionadas pelo instalador"
  if should_remove_go; then
    echo "  • Go em ${GO_SDK_DIR} (instalado pelo install.sh)"
  fi
  echo ""
  read -r -p "Continuar? [y/N] " reply
  case "${reply:-}" in
    y|Y|yes|Yes|YES) return 0 ;;
    *) die "Cancelado." ;;
  esac
}

read_installed_go_sdk() {
  if [[ -f "$GO_SDK_MARKER" ]]; then
    tr -d '[:space:]' <"$GO_SDK_MARKER"
    return
  fi
  echo ""
}

should_remove_go() {
  if [[ "$REMOVE_GO" == "1" ]]; then
    return 0
  fi
  if [[ "$REMOVE_GO" == "0" ]]; then
    return 1
  fi
  local marked
  marked="$(read_installed_go_sdk)"
  [[ -n "$marked" ]] && [[ -d "$marked" ]]
}

strip_path_blocks() {
  local rc="$1"
  [[ -f "$rc" ]] || return 0

  local tmp
  tmp="$(mktemp)"
  awk '
    /^# gitai installer$/ { skip=2; next }
    /^# gitai \(Go bin\)$/ { skip=1; next }
    skip > 0 { skip--; next }
    { print }
  ' "$rc" >"$tmp"

  if ! cmp -s "$rc" "$tmp"; then
    mv "$tmp" "$rc"
    ok "PATH limpo em ${rc}"
  else
    rm -f "$tmp"
  fi
}

step_remove_binary() {
  log "1/4 Removendo binário"
  local bin_dir binary removed=0
  bin_dir="$(gopath_bin_dir)"
  binary="${bin_dir}/gitai"

  if [[ -f "$binary" ]]; then
    rm -f "$binary"
    ok "Removido ${binary}"
    removed=1
  fi

  if command -v gitai >/dev/null 2>&1; then
    local other
    other="$(command -v gitai)"
    if [[ "$other" != "$binary" ]] && [[ -f "$other" ]]; then
      warn "Outro binário encontrado: ${other} (não removido automaticamente)"
    fi
  fi

  if [[ "$removed" -eq 0 ]]; then
    warn "Binário não encontrado em ${binary}"
  fi
}

step_remove_go_sdk() {
  log "2/4 Verificando Go do instalador"
  if ! should_remove_go; then
    ok "Go em ${GO_SDK_DIR} mantido"
    return 0
  fi
  local dir
  dir="$(read_installed_go_sdk)"
  if [[ -z "$dir" ]]; then
    dir="$GO_SDK_DIR"
  fi
  if [[ -d "$dir" ]]; then
    rm -rf "$dir"
    ok "Removido ${dir}"
  else
    warn "Diretório Go não encontrado: ${dir}"
  fi
}

step_remove_config() {
  log "3/4 Removendo dados em ${GITAI_CONFIG_DIR}"
  if [[ -d "$GITAI_CONFIG_DIR" ]]; then
    rm -rf "$GITAI_CONFIG_DIR"
    ok "Diretório removido"
  else
    warn "Nada em ${GITAI_CONFIG_DIR}"
  fi
}

step_clean_shell_path() {
  log "4/4 Limpando PATH no shell"
  local found=0
  while IFS= read -r rc; do
    if grep -qF "$PATH_MARKER_INSTALLER" "$rc" 2>/dev/null || grep -qF "$PATH_MARKER_GOBIN" "$rc" 2>/dev/null; then
      strip_path_blocks "$rc"
      found=1
    fi
  done < <(shell_rc_files)

  if [[ "$found" -eq 0 ]]; then
    ok "Nenhum bloco gitai no shell rc"
  fi
}

finish() {
  echo ""
  ok "gitai removido deste computador."
  echo ""
  warn "Abra um novo terminal ou rode: source ~/.zshrc"
  echo ""
  echo "  Para reinstalar: curl -fsSL https://raw.githubusercontent.com/laerciocrestani/gitai/main/install.sh | bash"
  echo ""
}

main() {
  parse_args "$@"
  echo ""
  echo "  gitai uninstall"
  echo "  ─────────────────────────────────────"
  confirm_uninstall
  echo ""
  step_remove_binary
  step_remove_go_sdk
  step_remove_config
  step_clean_shell_path
  finish
}

main "$@"
