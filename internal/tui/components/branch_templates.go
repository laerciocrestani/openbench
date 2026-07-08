package components

// NewBranchTemplate describes a branch naming prefix or pattern.
type NewBranchTemplate struct {
	Prefix  string
	Icon    string
	Usage   string
	Example string
	Other   bool
}

// NewBranchTemplateItem is one row in the template picker.
type NewBranchTemplateItem struct {
	Template  NewBranchTemplate
	Separator bool
}

func (t NewBranchTemplate) Label() string {
	if t.Other {
		return t.Icon + " Outro"
	}
	return t.Icon + " " + t.Prefix
}

// ListLabel shows icon, usage and example separated by middle dots.
func (t NewBranchTemplate) ListLabel() string {
	return t.DetailLabel()
}

// DetailLabel formats icon + example · usage · example for list and reference rows.
func (t NewBranchTemplate) DetailLabel() string {
	if t.Other {
		return t.Icon + " Outro · " + t.Usage + " · " + t.Example
	}
	return t.Icon + " " + t.Example + " · " + t.Usage + " · " + t.Example
}

func (t NewBranchTemplate) PrefixColumn() string {
	if t.Other {
		return t.Icon + " Outro"
	}
	return t.Icon + " " + t.Prefix
}

// NameSeed returns the initial value for the name input field.
func (t NewBranchTemplate) NameSeed() string {
	if t.Other {
		return ""
	}
	return t.Prefix
}

func templatesMatch(a, b NewBranchTemplate) bool {
	if a.Other != b.Other {
		return false
	}
	if a.Other {
		return true
	}
	return a.Prefix == b.Prefix
}

// BranchTemplateCatalog returns all templates: common, rest, and Outro.
func BranchTemplateCatalog() []NewBranchTemplate {
	common := commonBranchTemplates()
	rest := restBranchTemplates()

	out := make([]NewBranchTemplate, 0, len(common)+len(rest)+1)
	out = append(out, common...)
	out = append(out, rest...)
	out = append(out, NewBranchTemplate{
		Icon:    "✏️",
		Usage:   "Nome personalizado",
		Example: "minha-branch",
		Other:   true,
	})
	return out
}

func commonBranchTemplates() []NewBranchTemplate {
	// Mais comuns — ordem alfabética por prefixo.
	return []NewBranchTemplate{
		{Prefix: "chore/", Icon: "🔧", Usage: "Tarefas técnicas sem impacto funcional", Example: "chore/update-dependencies"},
		{Prefix: "docs/", Icon: "📚", Usage: "Documentação", Example: "docs/api-reference"},
		{Prefix: "feature/", Icon: "✨", Usage: "Nova funcionalidade", Example: "feature/user-profile"},
		{Prefix: "fix/", Icon: "🐛", Usage: "Correção de bug", Example: "fix/login-error"},
		{Prefix: "hotfix/", Icon: "🚑", Usage: "Correção urgente em produção", Example: "hotfix/payment-timeout"},
		{Prefix: "refactor/", Icon: "♻️", Usage: "Refatoração sem mudança de comportamento", Example: "refactor/auth-service"},
		{Prefix: "release/", Icon: "🚀", Usage: "Preparação de uma versão", Example: "release/v2.4.0"},
		{Prefix: "test/", Icon: "🧪", Usage: "Testes", Example: "test/user-controller"},
	}
}

func restBranchTemplates() []NewBranchTemplate {
	// Demais prefixos — ordem alfabética.
	return []NewBranchTemplate{
		{Prefix: "bugfix/", Icon: "🪲", Usage: "Correção de bug (alternativa ao fix)", Example: "bugfix/memory-leak"},
		{Prefix: "build/", Icon: "📦", Usage: "Build e ferramentas", Example: "build/docker"},
		{Prefix: "ci/", Icon: "⚙️", Usage: "CI/CD", Example: "ci/github-actions"},
		{Prefix: "develop", Icon: "🌱", Usage: "Branch principal de desenvolvimento (GitFlow)", Example: "develop"},
		{Prefix: "experiment/", Icon: "🤖", Usage: "Experimentos/POCs", Example: "experiment/llm-provider"},
		{Prefix: "main", Icon: "🌿", Usage: "Branch principal de produção", Example: "main"},
		{Prefix: "master", Icon: "🌳", Usage: "Antigo nome da branch principal", Example: "master"},
		{Prefix: "perf/", Icon: "⚡", Usage: "Melhorias de performance", Example: "perf/query-cache"},
		{Prefix: "revert/", Icon: "↩️", Usage: "Reverter alterações", Example: "revert/pr-142"},
		{Prefix: "spike/", Icon: "🔬", Usage: "Pesquisa técnica", Example: "spike/openai-responses-api"},
		{Prefix: "style/", Icon: "💅", Usage: "Formatação/código (sem alterar lógica)", Example: "style/php-cs-fixer"},
	}
}

// BranchTemplateItems returns picker rows with a separator after the common group.
func BranchTemplateItems() []NewBranchTemplateItem {
	catalog := BranchTemplateCatalog()
	const commonCount = 8
	items := make([]NewBranchTemplateItem, 0, len(catalog)+1)
	for i, tpl := range catalog {
		if i == commonCount {
			items = append(items, NewBranchTemplateItem{Separator: true})
		}
		items = append(items, NewBranchTemplateItem{Template: tpl})
	}
	return items
}

// SelectableTemplateCount returns the number of non-separator template rows.
func SelectableTemplateCount(items []NewBranchTemplateItem) int {
	n := 0
	for _, item := range items {
		if !item.Separator {
			n++
		}
	}
	return n
}

// TemplateAtCursor returns the template for a selectable cursor index.
func TemplateAtCursor(items []NewBranchTemplateItem, cursor int) NewBranchTemplate {
	idx := 0
	for _, item := range items {
		if item.Separator {
			continue
		}
		if idx == cursor {
			return item.Template
		}
		idx++
	}
	return NewBranchTemplate{}
}
