#!/usr/bin/env bash
# Wrapper de compatibilidade — prefira os comandos nativos do gitia.
#
#   go run ./cmd/gitia install   (primeira vez)
#   gitia config
#   gitia update

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

run_gitia() {
  if command -v gitia >/dev/null 2>&1; then
    gitia "$@"
    return
  fi
  if ! command -v go >/dev/null 2>&1; then
    echo "✗ Go não encontrado. Instale: https://go.dev/dl/" >&2
    exit 1
  fi
  (cd "$REPO_ROOT" && go run ./cmd/gitia "$@")
}

usage() {
  cat <<'EOF'
gitia-setup — wrapper de compatibilidade

Prefira os comandos nativos:
  go run ./cmd/gitia install   Instala binário + PATH (primeira vez)
  gitia config                 Wizard de configuração
  gitia update                 git pull + reinstala

Este script ainda aceita:
  ./scripts/setup.sh install
  ./scripts/setup.sh config
  ./scripts/setup.sh update
EOF
}

main() {
  local cmd="${1:-help}"
  case "$cmd" in
    install) (cd "$REPO_ROOT" && go run ./cmd/gitia install) ;;
    config)  run_gitia config ;;
    update)  run_gitia update ;;
    help|-h|--help) usage ;;
    *) echo "✗ Comando desconhecido: $cmd" >&2; usage >&2; exit 1 ;;
  esac
}

main "$@"
