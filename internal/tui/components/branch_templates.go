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
		return t.Icon + " Other"
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
		return t.Icon + " Other · " + t.Usage + " · " + t.Example
	}
	return t.Icon + " " + t.Example + " · " + t.Usage + " · " + t.Example
}

func (t NewBranchTemplate) PrefixColumn() string {
	if t.Other {
		return t.Icon + " Other"
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

// BranchTemplateCatalog returns all templates: common, rest, and Other.
func BranchTemplateCatalog() []NewBranchTemplate {
	common := commonBranchTemplates()
	rest := restBranchTemplates()

	out := make([]NewBranchTemplate, 0, len(common)+len(rest)+1)
	out = append(out, common...)
	out = append(out, rest...)
	out = append(out, NewBranchTemplate{
		Icon:    "✏️",
		Usage:   "Custom branch name",
		Example: "my-branch",
		Other:   true,
	})
	return out
}

func commonBranchTemplates() []NewBranchTemplate {
	return []NewBranchTemplate{
		{Prefix: "chore/", Icon: "🔧", Usage: "Technical tasks without functional impact", Example: "chore/update-dependencies"},
		{Prefix: "docs/", Icon: "📚", Usage: "Documentation", Example: "docs/api-reference"},
		{Prefix: "feature/", Icon: "✨", Usage: "New feature", Example: "feature/user-profile"},
		{Prefix: "fix/", Icon: "🐛", Usage: "Bug fix", Example: "fix/login-error"},
		{Prefix: "hotfix/", Icon: "🚑", Usage: "Urgent production fix", Example: "hotfix/payment-timeout"},
		{Prefix: "refactor/", Icon: "♻️", Usage: "Refactor without behavior change", Example: "refactor/auth-service"},
		{Prefix: "release/", Icon: "🚀", Usage: "Release preparation", Example: "release/v2.4.0"},
		{Prefix: "test/", Icon: "🧪", Usage: "Tests", Example: "test/user-controller"},
	}
}

func restBranchTemplates() []NewBranchTemplate {
	return []NewBranchTemplate{
		{Prefix: "bugfix/", Icon: "🪲", Usage: "Bug fix (alternative to fix)", Example: "bugfix/memory-leak"},
		{Prefix: "build/", Icon: "📦", Usage: "Build and tooling", Example: "build/docker"},
		{Prefix: "ci/", Icon: "⚙️", Usage: "CI/CD", Example: "ci/github-actions"},
		{Prefix: "develop", Icon: "🌱", Usage: "Main development branch (GitFlow)", Example: "develop"},
		{Prefix: "experiment/", Icon: "🤖", Usage: "Experiments and POCs", Example: "experiment/llm-provider"},
		{Prefix: "main", Icon: "🌿", Usage: "Production main branch", Example: "main"},
		{Prefix: "master", Icon: "🌳", Usage: "Legacy main branch name", Example: "master"},
		{Prefix: "perf/", Icon: "⚡", Usage: "Performance improvements", Example: "perf/query-cache"},
		{Prefix: "revert/", Icon: "↩️", Usage: "Revert changes", Example: "revert/pr-142"},
		{Prefix: "spike/", Icon: "🔬", Usage: "Technical research", Example: "spike/openai-responses-api"},
		{Prefix: "style/", Icon: "💅", Usage: "Formatting/style (no logic change)", Example: "style/php-cs-fixer"},
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
