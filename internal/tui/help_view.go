package tui

import "strings"

func helpContent() string {
	lines := []string{
		"Atalhos do dashboard",
		"",
		"  a       Adicionar arquivos ao stage (um, vários ou git add .)",
		"  c       Commit com IA (preview → e editar → Enter confirma)",
		"  p       Push para origin (preview → Enter confirma)",
		"  P       Criar Pull Request com IA (preview → Enter confirma)",
		"  d       Ver diff (working tree ou branch)",
		"  b       Trocar de branch (lista + contexto)",
		"  y       Copiar hash do commit",
		"  l       Ver log de commits",
		"  s       Sync com origin (quando behind)",
		"  o       Abrir PR no browser",
		"  u       Relatório de uso/custo de IA",
		"  r       Atualizar dashboard manualmente",
		"  ?       Esta ajuda",
		"  q       Sair",
		"",
		"Auto-refresh (dashboard e diff)",
		"  Mudanças em arquivos são detectadas em ~400ms (fsnotify)",
		"  git add/reset/branch externo: polling a cada ui_auto_refresh_seconds",
		"",
		"Na tela de diff/report/branches/add",
		"  ↑↓      Scroll / navegar branches",
		"  esc     Voltar",
		"",
		"Na tela de add",
		"  space   Marcar/desmarcar arquivo",
		"  A       Marcar/desmarcar todos",
		"  Enter   git add nos selecionados (ou no cursor se nenhum)",
		"  .       git add . (todos os arquivos)",
		"",
		"No preview de commit/push/PR",
		"  e       Editar mensagem/título/corpo",
		"  Enter   Confirmar",
		"  esc     Cancelar (ou voltar ao preview ao editar)",
		"",
		"No modal de PR",
		"  d       Alternar draft",
		"  tab     Alternar título/corpo (ao editar)",
		"",
		"Preferências em config.yaml",
		"  interactive_ui          TUI ao rodar gitai (padrão: sim)",
		"  ui_color                Cores na CLI e TUI (padrão: sim)",
		"  ui_auto_refresh_seconds Polling fallback em segundos (padrão: 5, 0=off)",
		"  ui_watch_files          Observar filesystem (padrão: sim)",
		"",
		"Variáveis de ambiente (sobrescrevem config)",
		"  GITAI_NO_UI=1   Força overview CLI em vez da TUI",
		"  NO_COLOR=1      Sem cores (convenção Unix; ver no-color.org)",
		"  CI=1            Sem TUI nem cores",
	}

	var b strings.Builder
	b.WriteString(styleSection.Render("Ajuda"))
	b.WriteString("\n\n")
	for _, line := range lines {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		if !strings.HasPrefix(line, "  ") {
			b.WriteString(styleTitle.Render(line))
		} else {
			b.WriteString(styleHint.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func helpHelpLine() string {
	return styleKey.Render("esc") + " ou " + styleKey.Render("?") + " fechar"
}
