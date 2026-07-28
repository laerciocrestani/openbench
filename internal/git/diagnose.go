package git

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// HealthLevel indica a gravidade de um achado ou do panorama geral.
type HealthLevel string

const (
	HealthOK       HealthLevel = "ok"
	HealthWarn     HealthLevel = "warn"
	HealthCritical HealthLevel = "critical"
)

// CommitAnalysis resume um commit local em relação a uma referência remota.
type CommitAnalysis struct {
	Hash               string
	Subject            string
	FileCount          int
	BuildArtifactFiles int
	LikelyDiscardable  bool
}

// DivergenceReport descreve divergência entre duas refs (ex.: main vs origin/main).
type DivergenceReport struct {
	LocalRef       string
	RemoteRef      string
	MergeBase      string
	LocalAhead     int
	RemoteAhead    int
	LocalCommits   []string
	RemoteCommits  []string
	LocalAnalyses  []CommitAnalysis
	LocalDiffStat  string
}

// HealthSnapshot agrega dados Git brutos para diagnóstico.
type HealthSnapshot struct {
	Branch             string
	Base               string
	OnBase             bool
	IsDirty            bool
	Staged             int
	Modified           int
	Untracked          int
	Ahead              int
	Behind             int
	Diverged           bool
	CommitsAheadOfBase int
	BaseDivergence     *DivergenceReport
	BranchDivergence   *DivergenceReport
	UnpushedBlockers   *UnpushedBlockers
}

var buildArtifactSegments = []string{
	"node_modules/",
	".pnpm-store/",
	"/dist/",
	"/build/",
	".next/",
	".astro/",
	"__pycache__/",
	"/.cache/",
	"/coverage/",
	".turbo/",
	"/target/",
	"/vendor/bundle/",
}

// CollectHealthSnapshot reúne o estado Git relevante para o doctor.
func (r *Repo) CollectHealthSnapshot(base string) (*HealthSnapshot, error) {
	resolvedBase, err := r.ResolveBase(base)
	if err != nil {
		return nil, err
	}

	localBase := strings.TrimPrefix(resolvedBase, "origin/")
	current, err := r.CurrentBranch()
	if err != nil {
		return nil, err
	}

	staged, modified, untracked, err := r.worktreeCounts()
	if err != nil {
		return nil, err
	}

	snap := &HealthSnapshot{
		Branch:     current,
		Base:       localBase,
		OnBase:     current == localBase,
		IsDirty:    staged+modified+untracked > 0,
		Staged:     staged,
		Modified:   modified,
		Untracked:  untracked,
	}

	if count, err := r.run("rev-list", "--count", fmt.Sprintf("%s..%s", resolvedBase, current)); err == nil {
		snap.CommitsAheadOfBase, _ = strconv.Atoi(strings.TrimSpace(count))
	}

	if upstream, err := r.run("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err == nil {
		if ahead, behind, err := r.aheadBehind(upstream); err == nil {
			snap.Ahead = ahead
			snap.Behind = behind
			snap.Diverged = ahead > 0 && behind > 0
		}
		if snap.Diverged {
			snap.BranchDivergence, _ = r.DivergenceReport(current, upstream)
		}
	}

	remoteBase := "origin/" + localBase
	if _, err := r.run("rev-parse", "--verify", remoteBase); err == nil {
		if snap.OnBase {
			div, err := r.DivergenceReport(localBase, remoteBase)
			if err == nil && (div.LocalAhead > 0 || div.RemoteAhead > 0) {
				snap.BaseDivergence = div
			}
			if err == nil && div != nil && div.LocalAhead > 0 {
				if blockers, scanErr := r.ScanUnpushedBlockers(remoteBase, localBase); scanErr == nil {
					snap.UnpushedBlockers = blockers
				}
			}
		}
	}

	return snap, nil
}

// DivergenceReport compara duas refs e analisa commits exclusivos locais.
func (r *Repo) DivergenceReport(localRef, remoteRef string) (*DivergenceReport, error) {
	ahead, behind, err := r.BranchAheadBehind(localRef, remoteRef)
	if err != nil {
		return nil, err
	}

	mergeBase, err := r.run("merge-base", localRef, remoteRef)
	if err != nil {
		return nil, err
	}

	report := &DivergenceReport{
		LocalRef:      localRef,
		RemoteRef:     remoteRef,
		MergeBase:     mergeBase,
		LocalAhead:    ahead,
		RemoteAhead:   behind,
	}

	if ahead > 0 {
		report.LocalCommits, err = r.logOnelineRange(remoteRef, localRef, maxPruneCommitPreview)
		if err != nil {
			return nil, err
		}
		report.LocalAnalyses, err = r.analyzeCommitRange(remoteRef, localRef, maxPruneCommitPreview)
		if err != nil {
			return nil, err
		}
		report.LocalDiffStat, err = r.diffStatRange(remoteRef, localRef, 40)
		if err != nil {
			return nil, err
		}
	}

	if behind > 0 {
		report.RemoteCommits, err = r.logOnelineRange(localRef, remoteRef, maxPruneCommitPreview)
		if err != nil {
			return nil, err
		}
	}

	return report, nil
}

func (r *Repo) analyzeCommitRange(baseRef, headRef string, limit int) ([]CommitAnalysis, error) {
	out, err := r.run("log", fmt.Sprintf("%s..%s", baseRef, headRef), "--format=%H %s", "--no-decorate", "-n", strconv.Itoa(limit))
	if err != nil {
		return nil, err
	}

	var analyses []CommitAnalysis
	for _, line := range splitLines(out) {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 0 {
			continue
		}
		hash := parts[0]
		subject := ""
		if len(parts) > 1 {
			subject = parts[1]
		}
		analysis, err := r.analyzeCommit(hash, subject)
		if err != nil {
			continue
		}
		analyses = append(analyses, analysis)
	}
	return analyses, nil
}

func (r *Repo) analyzeCommit(hash, subject string) (CommitAnalysis, error) {
	short, _ := r.run("rev-parse", "--short", hash)
	stat, err := r.run("show", "--stat", "--format=", hash)
	if err != nil {
		return CommitAnalysis{Hash: short, Subject: subject}, err
	}

	fileCount := 0
	buildCount := 0
	for _, line := range strings.Split(stat, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, " ") {
			continue
		}
		pipe := strings.Index(line, "|")
		if pipe < 0 {
			continue
		}
		path := strings.TrimSpace(line[:pipe])
		if path == "" {
			continue
		}
		fileCount++
		if IsBuildArtifactPath(path) {
			buildCount++
		}
	}

	ratio := 0.0
	if fileCount > 0 {
		ratio = float64(buildCount) / float64(fileCount)
	}

	return CommitAnalysis{
		Hash:               short,
		Subject:            subject,
		FileCount:          fileCount,
		BuildArtifactFiles: buildCount,
		LikelyDiscardable:  fileCount >= 20 && ratio >= 0.7,
	}, nil
}

func (r *Repo) diffStatRange(baseRef, headRef string, maxLines int) (string, error) {
	out, err := r.run("diff", "--stat", fmt.Sprintf("%s..%s", baseRef, headRef))
	if err != nil {
		return "", err
	}
	lines := splitLines(out)
	if len(lines) > maxLines {
		lines = append(lines[:maxLines], fmt.Sprintf("… +%d linha(s) omitida(s)", len(lines)-maxLines))
	}
	return strings.Join(lines, "\n"), nil
}

// IsBuildArtifactPath detecta caminhos típicos de artefatos de build/cache.
func IsBuildArtifactPath(path string) bool {
	norm := filepath.ToSlash(strings.ToLower(path))
	for _, seg := range buildArtifactSegments {
		if strings.Contains(norm, seg) {
			return true
		}
	}
	base := filepath.Base(norm)
	if strings.HasSuffix(base, ".min.js") || strings.HasSuffix(base, ".min.css") {
		return true
	}
	return false
}
