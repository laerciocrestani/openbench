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
wails3 package APP_VERSION=0.2.1
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

Injetada no build via `-ldflags` (`internal/version.BuildVersion`).

```bash
wails3 package APP_VERSION=0.2.1
```

Sem `APP_VERSION`, o Taskfile deriva `0.1.(commits-1)` do git (útil em dev). O app empacotado **não** depende de `.git` em runtime — se a versão não for injetada, cai no fallback `0.1.0` (bug que fazia o updater sempre achar update).

Para releases, alinhe `APP_VERSION`, a tag GitHub (`v0.2.1`) e o `-version` do `wails3 updater manifest`.
