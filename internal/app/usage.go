package app

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/laerciocrestani/openbench/internal/ai"
	"github.com/laerciocrestani/openbench/internal/config"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
	"github.com/laerciocrestani/openbench/internal/usage"
)

func recordAIUsage(command string, cfg *config.Config, summary ai.UsageSummary) {
	project := "unknown"
	if repo, err := gitpkg.New(); err == nil {
		project = repo.ProjectName()
	}
	RecordAIUsageForProject(command, project, cfg, summary)
}

// RecordAIUsageForProject appends usage ledger entries for an explicit project name/path.
func RecordAIUsageForProject(command, projectPath string, cfg *config.Config, summary ai.UsageSummary) {
	if len(summary.Records) == 0 || cfg == nil {
		return
	}

	project := "unknown"
	if strings.TrimSpace(projectPath) != "" {
		if repo, err := gitpkg.Open(projectPath); err == nil {
			project = repo.ProjectName()
		} else {
			project = filepath.Base(strings.TrimRight(projectPath, `/\`))
		}
	}

	for _, r := range summary.Records {
		model := r.Model
		if model == "" {
			model = cfg.Model
		}
		_ = usage.Log(usage.Entry{
			Timestamp:    time.Now().UTC(),
			Command:      command,
			Project:      project,
			Provider:     string(cfg.Provider),
			Model:        model,
			Label:        r.Label,
			InputTokens:  r.PromptTokens,
			OutputTokens: r.CompletionTokens,
			CostUSD:      r.CostUSD,
		})
	}
}
