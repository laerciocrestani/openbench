#!/usr/bin/env bash
# Wrapper de compatibilidade — prefira install.sh na raiz do repositório.
#
#   curl -fsSL https://raw.githubusercontent.com/laerciocrestani/gitai/main/install.sh | bash
#   ./install.sh
#   gitai update

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

usage() {
  cat <<'EOF'
gitai-setup — wrapper de compatibilidade

Prefira:
  ./install.sh                 Instalação completa (Go + gitai + PATH + config)
  ./uninstall.sh               Remove gitai, config e PATH do instalador
  gitai config                 Wizard de configuração
  gitai update                 git pull + reinstala

Este script ainda aceita:
  ./scripts/setup.sh install   → ./install.sh
  ./scripts/setup.sh uninstall → ./uninstall.sh
  ./scripts/setup.sh config
  ./scripts/setup.sh update
EOF
}

run_gitai() {
  if command -v gitai >/dev/null 2>&1; then
    gitai "$@"
    return
  fi
  if [[ -x "${HOME}/go/bin/gitai" ]]; then
    "${HOME}/go/bin/gitai" "$@"
    return
  fi
  if ! command -v go >/dev/null 2>&1; then
    echo "✗ Go não encontrado. Rode: ./install.sh" >&2
    exit 1
  fi
  (cd "$REPO_ROOT" && go run ./cmd/gitai "$@")
}

main() {
  local cmd="${1:-help}"
  case "$cmd" in
    install)   exec "$REPO_ROOT/install.sh" "${@:2}" ;;
    uninstall) exec "$REPO_ROOT/uninstall.sh" "${@:2}" ;;
    config)    run_gitai config ;;
    update)  run_gitai update ;;
    help|-h|--help) usage ;;
    *) echo "✗ Comando desconhecido: $cmd" >&2; usage >&2; exit 1 ;;
  esac
}

main "$@"
