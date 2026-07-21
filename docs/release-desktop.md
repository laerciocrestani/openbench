# Release desktop (auto-update)

O app usa **Wails v3 Updater** + **GitHub Releases** (`laerciocrestani/openbench`).

## Chaves

- Pública (commitada): `build/updater/updater.key.pub` — embutida no binário.
- Privada (nunca no git): `build/updater/updater.key` — gerar com:

```bash
wails3 updater genkey -out build/updater/updater.key
```

Guarde a privada no CI/secrets. Se regenerar, atualize a `.pub` e publique um release que use a nova chave.

## Build + package

```bash
wails3 build
wails3 package   # gera artefato por OS
```

Artefatos esperados (ver docs Wails updater):

- macOS: `.zip` do `.app`
- Windows: `.exe` ou `.zip`
- Linux: binário ou `.tar.gz`

## Manifest assinado

```bash
mkdir -p bin/updates
# copie os artefatos de package para bin/updates/

wails3 updater manifest \
  -version 0.2.0 \
  -channel stable \
  -key build/updater/updater.key \
  -notes-file notes.md \
  -url-prefix "https://github.com/laerciocrestani/openbench/releases/download/v0.2.0" \
  bin/updates/

wails3 updater verify \
  -manifest manifest.json \
  -publickey build/updater/updater.key.pub
```

Publique no GitHub Release (tag **sem** ou **com** `v` — a versão em `updater.Config.CurrentVersion` é sem `v`, ex.: `0.2.0`):

1. Crie release `v0.2.0` (ou `0.2.0` conforme o provider esperar — teste com `Check`).
2. Anexe artefatos + `manifest.json`.

## App

- Checagem automática a cada **6h**. Se houver update, aparece um dialog com **Atualizar agora** / **Depois**.
- Manual: **Settings → Atualizações → Verificar atualizações**.

## Versão atual do binário

Derivada de `internal/version` (contagem de commits → `v0.1.N`). Para releases, alinhe a tag ao semver desejado ou injete `-ldflags` com a versão de release no pipeline.
