# Resumo do Entendimento — Docker: Corrigir com IA

## Problema e objetivo

Quando uma ação Docker no app desktop falha (ex.: `docker start` com porta já alocada), o usuário vê o erro no `DockerActionDialog`, mas hoje só consegue fechar e investigar manualmente. O objetivo é oferecer um fluxo **Corrigir com IA**: a partir do erro depositado no dialog, a IA (orientada pela skill `docker-debug` refinada) analisa o problema, propõe uma resolução revisável e, se o usuário aceitar, executa a correção e **reexecuta automaticamente** a ação Docker original.

## Sistema existente

- Dialog de progresso/resultado: `frontend/src/components/docker-action-dialog.tsx` (hoje rodapé só com “Fechar”).
- Skills de chat: `internal/aiskills` + builtin `docker-debug` (`internal/aiskills/builtin/docker-debug.skill.md`).
- Referência de UX/fluxo de plano + execução: Doctor Fix (`DoctorFixDialog`, `PlanDoctorFix` / runner) — padrão semelhante (plano → revisão → executar), domínio diferente.
- Fechar do dialog de falha já cobre X e click fora; botão “Fechar” no rodapé torna-se redundante no estado de falha.

## Restrições organizacionais

- Feature incremental no app desktop Openbench (Wails + React).
- Sem deadline/orçamento explícitos nesta descoberta.

## Atores

- **Desenvolvedor (usuário do desktop):** dispara ação Docker, vê falha, solicita correção com IA, revisa/edita/recusa plano, autoriza execução.
- **IA (via provider configurado + skill `docker-debug`):** interpreta erro + contexto Docker, monta plano (problema + resolução + passos/arquivos).
- **Openbench (runtime):** coleta contexto do erro, orquestra geração do plano, aplica passos aprovados, reexecuta a ação Docker original.

## Requisitos funcionais

1. No `DockerActionDialog`, em estado de **falha** (`ok === false` e não running), o rodapé exibe **somente** o botão **Corrigir com IA** (sem “Fechar” — dismiss via X / click fora).
2. O botão **não** aparece em sucesso nem durante execução.
3. Ao clicar, abre um **segundo dialog** de correção que:
   - mostra **Problema** (causa observada a partir do erro depositado);
   - mostra **Resolução** (plano de ação);
   - permite **recusar** (fechar sem executar) ou **editar** o plano antes de executar;
   - no rodapé, botão **Executar correção** (quando houver plano válido).
4. A IA usa o **erro depositado** (message, lines, services em erro, ação Docker) e o cenário Docker do projeto; toda sugestão deve ser coerente com esse contexto.
5. Refinar a skill builtin **`docker-debug`** (não criar skill nova) para cobrir bem o pós-falha de orquestração (ex.: porta ocupada por outro container/processo).
6. Se a correção incluir **edição de arquivos**, o plano deve listar **todos** os arquivos que serão modificados e **por quê**; o usuário aceita ou recusa a execução com essa informação explícita.
7. Após **Executar correção** com sucesso: fechar o dialog de correção (e o de falha, conforme fluxo) e **reexecutar automaticamente** a ação Docker original.
8. Se a IA **não conseguir montar um plano**: o dialog de correção permanece aberto com mensagem de falha clara; dismiss via X / click fora (sem forçar execução).

## Requisitos não funcionais

- Respostas e UI em **pt-BR**.
- Não executar mutações sem ação explícita do usuário no botão **Executar correção**.
- Preferir menor ação segura (alinhado ao espírito atual de `docker-debug`).
- Geração do plano e execução devem falhar de forma observável (loading + erro), sem travar o dialog de origem de forma irreversível.
- PREMISSA: latência da chamada de IA aceitável para UX de modal (loading no segundo dialog); sem SLA formal.

## Regras de negócio

- “Corrigir com IA” só em falha Docker do dialog de ação.
- Plano é **proposta** até o usuário confirmar execução.
- Usuário pode **editar** ou **recusar** o plano.
- Edições de arquivo no plano exigem transparência (lista completa + motivo) e consentimento na execução.
- Comandos destrutivos (`down -v`, prune, apagar volumes) só com aviso claro no plano (PREMISSA alinhada à skill atual).
- PREMISSA: ações Docker no escopo do projeto (stop/rm de containers conflitantes, recreate, etc.) e, quando necessário, remediação de **processo no host** que ocupa a porta, desde que constem no plano revisável. Escopo exato de kill no host a validar na arquitetura.

## Fluxos principais

1. **Happy path:** ação Docker falha → usuário clica Corrigir com IA → loading → dialog com Problema + Resolução → usuário revisa/edita → Executar correção → sucesso → fecha dialogs → reexecuta ação Docker original.
2. **Recusa:** usuário fecha o dialog de correção sem executar → permanece no dialog de falha (ou o fecha via X); nenhuma mutação.
3. **Edição:** usuário altera o plano (texto/passos/arquivos conforme UX definida na arquitetura) → Executar correção aplica a versão editada.
4. **IA sem plano:** segundo dialog mostra erro de geração → usuário fecha (X / fora); sem execução.
5. **Falha na execução do plano:** dialog de correção reporta erro; não reexecuta a ação Docker original automaticamente.
6. **Reexecução pós-fix falha de novo:** novo ciclo (Corrigir com IA de novo), sem loop automático infinito.

## Integrações externas

- **Provider de IA** (config Openbench / `OB_API_KEY`): geração do plano de correção.
- **Docker CLI / Compose**: inspeção e execução dos passos aprovados.
- **Filesystem do projeto** (quando o plano incluir edits): leitura/escrita com lista explícita no plano.
- Skill **`docker-debug`**: conteúdo injetado/orientando o comportamento da IA neste fluxo.

## Restrições e premissas

- Reusar/refinar `docker-debug`; não criar skill paralela.
- Padrão mental próximo ao Doctor Fix (plano → revisão → executar), implementação pode ser dedicada ao domínio Docker.
- PREMISSA: o payload enviado à IA inclui no mínimo `action`, `message`, `lines`, `services` (nome/status/detail) e contexto Docker já disponível no app (compose path / snapshot, se existir).
- PREMISSA: “editar o plano” inclui pelo menos edição do texto da resolução / passos apresentados; granularidade de UI (checkbox por passo vs textarea) a definir na arquitetura.
- Lacuna aceita na descoberta: formato exato do plano estruturado (JSON steps vs markdown) fica para a fase de arquitetura.

## Riscos identificados

- **IA alucinar serviço/porta:** mitigação — ancorar no erro depositado + snapshot Docker; skill exige evidência.
- **Executar kill no host / stop de container errado:** mitigação — plano revisável + listagem explícita de alvos; preferir leitura antes de mutação.
- **Edit de compose/env quebrar outros serviços:** mitigação — listar arquivos + motivo; usuário pode recusar.
- **Reexecução automática mascara falha parcial do plano:** mitigação — só reexecutar se a execução do plano reportar sucesso completo.
- **Sobreposição com chat genérico:** risco de duplicar UX; mitigação — fluxo modal dedicado a partir do dialog de falha, skill compartilhada.

## Lacunas / decisões pendentes

Resolvidas na arquitetura — ver [`docs/architecture/docker-corrigir-com-ia.md`](../architecture/docker-corrigir-com-ia.md):

- Plano JSON tipado (`problem` / `resolution` / `steps` / `files`); edição MVP = toggles + recusar dialog.
- Kill no host **fora do MVP** (só `notes`); Docker Compose allowlist + `write_file`.
- Serviço novo `DockerFix*` (híbrido CI Fix geração + Doctor Fix Begin/Advance).
- Dialog de falha permanece atrás; ambos fecham no sucesso + reexecução.

---

**Descoberta confirmada.** Arquitetura proposta em `docs/architecture/docker-corrigir-com-ia.md`.
