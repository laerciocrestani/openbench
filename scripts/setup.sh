#!/usr/bin/env bash
# Wrapper de compatibilidade — prefira os comandos nativos do gitai.
#
#   go run ./cmd/gitai install   (primeira vez)
#   gitai config
#   gitai update

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

run_gitai() {
  if command -v gitai >/dev/null 2>&1; then
    gitai "$@"
    return
  fi
  if ! command -v go >/dev/null 2>&1; then
    echo "✗ Go não encontrado. Instale: https://go.dev/dl/" >&2
    exit 1
  fi
  (cd "$REPO_ROOT" && go run ./cmd/gitai "$@")
}

usage() {
  cat <<'EOF'
gitai-setup — wrapper de compatibilidade

Prefira os comandos nativos:
  go run ./cmd/gitai install   Instala binário + PATH (primeira vez)
  gitai config                 Wizard de configuração
  gitai update                 git pull + reinstala

Este script ainda aceita:
  ./scripts/setup.sh install
  ./scripts/setup.sh config
  ./scripts/setup.sh update
EOF
}

main() {
  local cmd="${1:-help}"
  case "$cmd" in
    install) (cd "$REPO_ROOT" && go run ./cmd/gitai install) ;;
    config)  run_gitai config ;;
    update)  run_gitai update ;;
    help|-h|--help) usage ;;
    *) echo "✗ Comando desconhecido: $cmd" >&2; usage >&2; exit 1 ;;
  esac
}

main "$@"
