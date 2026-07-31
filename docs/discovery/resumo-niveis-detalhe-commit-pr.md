# Resumo do Entendimento — Níveis de detalhe Commit / PR

## Problema e objetivo

Hoje a geração de mensagem de commit e de PR via IA usa um único “modo” de profundidade (contexto truncado em `max_diff_bytes` + prompts com body/seções em densidade fixa). Usuários precisam escolher **quão profundo** deve ser o uso de contexto e o texto gerado — às vezes economizar tokens, às vezes cobrir diffs grandes com mais densidade.

O objetivo é oferecer **3 níveis** (`Minimal` / `Standard` / `Thorough`) **separados** para Commit e para PR, configuráveis no dialog de Configurações (desktop) e via flag na CLI, com preferência **por projeto** no `.openbench.yaml` local.

## Sistema existente

- CLI (`ob`) e app desktop (Wails): `SuggestCommit` / `SuggestPR` com prompts JSON em `internal/ai`.
- Diff: `diff --stat` sempre completo; patch truncado em `MaxDiffBytes` (default 120_000).
- Config efetiva: `.openbench.yaml` no projeto (prioridade) ou `~/.config/openbench/config.yaml`; `.openbench.yaml` já está no `.gitignore` do repo openbench (config local por usuário, pode conter `api_key`).
- Prefs de UI desktop (`desktop.yaml`): `validate_commit` / `validate_pr` — ortogonais a esta feature.
- Índice de contexto do diff (discovery separada): alerta de diff grande; **não** substitui a escolha de nível.

**O que NÃO muda nesta feature:** fluxo de revisão humana antes de commit/PR; providers de IA; obrigação de `api_key`; estrutura de seções do PR (title, summary, changes, test_plan, notes).

## Restrições organizacionais

- Entrega incremental no produto atual (CLI + desktop).
- Sem deadline formal.
- Escopo: ambos os canais (CLI e app).

## Atores

- **Dev (usuário local):** escolhe nível em Configurações ou por flag; revisa/edita o texto gerado antes de confirmar.
- **Time (fora do escopo de sync):** preferência é por máquina/usuário no `.openbench.yaml` local — **não** é política versionada no git do projeto do usuário.

## Requisitos funcionais

1. Dois seletores independentes: **nível de Commit** e **nível de PR**, cada um com `Minimal` | `Standard` | `Thorough`.
2. Default de ambos: **Standard** (= comportamento atual), inclusive quando o projeto não tem a chave configurada.
3. Persistência por projeto em **`.openbench.yaml`** (local ao working copy; tipicamente não versionado).
4. Desktop: dialog **Configurações** permite ler/alterar os dois níveis (grava no `.openbench.yaml` do projeto aberto).
5. CLI: flag `--detail=minimal|standard|thorough` em `ob commit` e `ob pr`; a flag **sobrescreve** o valor do arquivo **somente naquela execução**.
6. Todos os níveis geram **title + body** (commit) e, no PR, as **mesmas seções** atuais; o que muda é a **densidade** do texto e o **quanto do patch** entra no prompt.
7. Usuário **pode editar** o texto gerado antes de confirmar (inalterado em relação ao fluxo atual).
8. Labels na UI/docs/flag: **Minimal / Standard / Thorough** (não “Curto/Padrão/Detalhado”).

## Requisitos não funcionais

- **Custo/tokens:** Minimal deve gastar pouco de propósito; Standard mantém o equilíbrio atual; Thorough usa o teto de contexto disponível sem inventar limite novo — diffs grandes implicam consumo maior (aceito).
- **Sem teto hard adicional** além do `max_diff_bytes` já existente (e limites do provider).
- **Latência:** Thorough pode ser mais lento em diffs grandes; aceitável.
- **Compatibilidade:** projetos sem a nova config continuam iguais (Standard silencioso).

## Regras de negócio

### Significado dos níveis (CONFIRMADO)

| Nível | Entrada (diff) | Saída |
|--------|----------------|--------|
| **Minimal** | Truncamento mais agressivo (~25% de `max_diff_bytes`); overview via `--stat` | Body/seções curtos (poucos bullets, alto nível) |
| **Standard** | Comportamento atual (`max_diff_bytes` + prompts atuais) | Densidade atual |
| **Thorough** | Usa o teto de `max_diff_bytes` (sem teto novo); instrução para cobrir mais áreas com mais densidade | Body/seções mais densos; notes quando fizer sentido |

- Commit: sempre `type`/`scope`/`title`/`body` (e `notes` quando Thorough/relevante).
- PR: sempre title + summary + changes + test_plan (+ notes quando relevante); só varia densidade.
- Precedência CLI: flag `--detail` > valor no `.openbench.yaml` do projeto > default Standard.
- Precedência desktop: valor salvo no `.openbench.yaml` do projeto aberto; sem override “só desta vez” na v1 (salvo se a UI reutilizar a mesma ideia depois).
- `validate_commit` / `validate_pr` continuam independentes do nível de detalhe.

## Fluxos principais

1. **Configurar (desktop):** abrir projeto → Configurações → escolher nível Commit e/ou PR → salvar em `.openbench.yaml`.
2. **Configurar (CLI/manual):** editar `.openbench.yaml` com as chaves de nível (nomes exatos na arquitetura).
3. **Gerar commit:** carregar nível efetivo → montar contexto (truncate conforme nível) → prompt com densidade correspondente → usuário revisa/edita → confirma.
4. **Gerar PR:** idem com nível de PR e seções atuais.
5. **Override CLI:** `ob commit --detail=thorough` (ou minimal) ignora o arquivo só nessa run; próxima run volta ao arquivo/default.
6. **Projeto sem chave:** Standard, sem migração nem warning obrigatório.

## Integrações externas

- **Providers de IA** (OpenAI / OpenRouter / Gemini): mesmo contrato; mudam tamanho do prompt e instruções de densidade.
- **Git local:** fonte do diff/stat/log (inalterado na origem; só política de truncamento/uso).
- **Arquivo `.openbench.yaml`:** persistência local por projeto.
- **GitHub CLI (`gh`):** fora do escopo do nível (só consome o texto do PR já gerado).

## Restrições e premissas

- **CONFIRMADO:** eixo principal = quanto contexto a IA recebe + densidade da saída (não remover seções).
- **CONFIRMADO:** seletores separados Commit vs PR.
- **CONFIRMADO:** default Standard = comportamento atual.
- **CONFIRMADO:** persistência em `.openbench.yaml` por projeto / por usuário (não versionar keys; arquivo local).
- **CONFIRMADO:** CLI com `--detail=...` sobrescrevendo o arquivo na execução.
- **CONFIRMADO:** PR mantém estrutura de seções; só densidade.
- **CONFIRMADO:** usuário pode editar o resultado.
- **PREMISSA:** fator ~25% de `max_diff_bytes` para Minimal é ponto de partida; pode calibrar na implementação/testes.
- **PREMISSA:** Thorough não aumenta `max_diff_bytes` além do configurado; diferencia-se de Standard sobretudo pelas **instruções de densidade** (e uso pleno do teto já permitido).
- **PREMISSA v1 desktop:** sem override “só desta vez” na UI (só Settings + valor persistido).
- **FORA DE ESCOPO v1:** sync de preferência entre membros do time via git; níveis para chat/Docker/CI fix.

## Riscos identificados

- **Thorough + diff enorme:** custo/latência altos e possível truncamento ainda assim — mitigação: manter `max_diff_bytes`; índice de contexto (feature irmã) alerta o usuário; Minimal/Standard como alternativas.
- **Minimal com pouco patch:** bullets genéricos ou omissão de áreas — mitigação: manter `--stat` completo + instrução explícita de cobrir áreas do resumo mesmo com patch curto.
- **Standard vs Thorough pouco distintos na entrada:** se ambos usam o mesmo teto de bytes, a diferença cai no prompt — mitigação aceita (PREMISSA); calibrar copy dos prompts na implementação.
- **Confusão `.openbench.yaml` com secrets:** arquivo já gitignored no ecossistema openbench; Settings deve deixar claro que é config local do projeto. `.openbench/` (docker presets) é versionável (ADR-006).
- **Dois canais (CLI + desktop) divergirem:** mesma leitura de config e mesma função de montagem de prompt/truncate — mitigação na arquitetura (um único caminho no core Go).

## Lacunas / decisões pendentes

- ~~Nomes exatos das chaves YAML~~ → `commit_detail` / `pr_detail`.
- ~~Copy final dos prompts por nível~~ → implementado em `internal/ai`.
- ~~Settings sem projeto aberto~~ → seletores desabilitados; exige projeto aberto.

---

**Confirmado pelo usuário (2026-07-31). Implementação entregue.**
`)