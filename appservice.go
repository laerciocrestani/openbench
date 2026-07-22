package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/laerciocrestani/openbench/internal/desktop"
	"github.com/laerciocrestani/openbench/internal/version"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// AppService exposes desktop bindings to the frontend.
type AppService struct {
	mu          sync.RWMutex
	app         *application.App
	projectPath string
	trayRefresh func()
	hub         *desktop.StatusHub
	term        *desktop.TerminalSession
	chatCancel  context.CancelFunc
	pendingTool *pendingChatTool
	repoWatch   *desktop.RepoWatcher
}

func (s *AppService) setApp(app *application.App) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.app = app
	s.hub = desktop.NewStatusHub(func(st desktop.ProjectStatus) {
		app.Event.Emit("project:status", st)
	})
	s.hub.Start()
	s.syncHubFromPrefsLocked()
}

func (s *AppService) setTrayRefresh(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trayRefresh = fn
}

func (s *AppService) currentPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.projectPath
}

func (s *AppService) setProjectPath(path string) {
	s.mu.Lock()
	prev := s.projectPath
	s.projectPath = path
	if s.hub != nil {
		s.hub.SetActive(path)
	}
	changed := !desktop.SamePath(prev, path)
	// Restart shell / chat / watcher when project changes (including close).
	if changed {
		s.stopTerminalLocked()
		s.stopRepoWatchLocked()
		if s.chatCancel != nil {
			s.chatCancel()
			s.chatCancel = nil
		}
		s.clearPendingToolLocked()
	}
	s.mu.Unlock()

	if changed && strings.TrimSpace(path) != "" {
		s.startRepoWatch(path)
	}
}

func (s *AppService) syncHubFromPrefs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncHubFromPrefsLocked()
}

func (s *AppService) syncHubFromPrefsLocked() {
	if s.hub == nil {
		return
	}
	prefs, err := desktop.LoadPrefs()
	if err != nil {
		return
	}
	s.hub.SetPinned(prefs.Pinned, s.projectPath)
}

// Ping is a health check used by the shell.
func (s *AppService) Ping() string {
	return "ok"
}

// Version returns the openbench version string.
func (s *AppService) Version() string {
	return version.DisplayCurrent()
}

// AppName returns the product name.
func (s *AppService) AppName() string {
	return "openbench"
}

// CurrentProject returns the open project path, or empty string.
func (s *AppService) CurrentProject() string {
	return s.currentPath()
}

// GetPrefs returns desktop UI preferences.
func (s *AppService) GetPrefs() (desktop.Prefs, error) {
	return desktop.LoadPrefs()
}

// PrefsPathString returns the desktop.yaml path for Settings UI.
func (s *AppService) PrefsPathString() (string, error) {
	return desktop.PrefsPath()
}

// SetPinnedAlias updates the display alias for a pinned project.
func (s *AppService) SetPinnedAlias(path, alias string) error {
	prefs, err := desktop.LoadPrefs()
	if err != nil {
		return err
	}
	found := false
	for i, p := range prefs.Pinned {
		if desktop.SamePath(p.Path, path) {
			prefs.Pinned[i].Alias = alias
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("projeto não está pinned")
	}
	if err := desktop.SavePrefs(prefs); err != nil {
		return err
	}
	s.syncHubFromPrefs()
	s.refreshTray()
	return nil
}

// ListProjectStatuses returns cached statuses for pinned projects.
func (s *AppService) ListProjectStatuses() []desktop.ProjectStatus {
	s.mu.RLock()
	hub := s.hub
	s.mu.RUnlock()
	if hub == nil {
		return nil
	}
	return hub.Snapshot()
}

// RefreshProjectStatuses forces a hub poll and returns statuses.
func (s *AppService) RefreshProjectStatuses() []desktop.ProjectStatus {
	s.mu.RLock()
	hub := s.hub
	s.mu.RUnlock()
	if hub == nil {
		return nil
	}
	return hub.RefreshNow()
}

// SetValidateCommit updates the validate_commit preference.
func (s *AppService) SetValidateCommit(enabled bool) error {
	prefs, err := desktop.LoadPrefs()
	if err != nil {
		return err
	}
	prefs.ValidateCommit = enabled
	return desktop.SavePrefs(prefs)
}

// SetValidatePR updates the validate_pr preference.
func (s *AppService) SetValidatePR(enabled bool) error {
	prefs, err := desktop.LoadPrefs()
	if err != nil {
		return err
	}
	prefs.ValidatePR = enabled
	return desktop.SavePrefs(prefs)
}

// OpenProjectDialog opens a native folder picker and loads the project.
func (s *AppService) OpenProjectDialog() (*desktop.Dashboard, error) {
	s.mu.RLock()
	app := s.app
	s.mu.RUnlock()
	if app == nil {
		return nil, fmt.Errorf("app not ready")
	}

	path, err := app.Dialog.OpenFile().
		SetTitle("Open project").
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return nil, fmt.Errorf("cancelled")
	}
	return s.OpenProject(path)
}

// PinProject adds path to pinned shortcuts (does not open the project).
func (s *AppService) PinProject(path string) error {
	if _, err := desktop.PinProject(path, ""); err != nil {
		return err
	}
	s.syncHubFromPrefs()
	s.refreshTray()
	return nil
}

// OpenProject validates and opens a git repository at path (updates recent; pin is explicit).
func (s *AppService) OpenProject(path string) (*desktop.Dashboard, error) {
	dash, err := desktop.LoadDashboard(path)
	if err != nil {
		return nil, err
	}
	s.setProjectPath(dash.Path)
	_, _ = desktop.RememberProject(dash.Path)
	s.syncHubFromPrefs()
	s.refreshTray()
	return dash, nil
}

// SwitchProject focuses an already pinned (or any) project path.
func (s *AppService) SwitchProject(path string) (*desktop.Dashboard, error) {
	dash, err := desktop.LoadDashboard(path)
	if err != nil {
		return nil, err
	}
	s.setProjectPath(dash.Path)
	_, _ = desktop.RememberProject(dash.Path)
	s.syncHubFromPrefs()
	s.refreshTray()
	return dash, nil
}

// UnpinProject removes a project from pinned tabs.
func (s *AppService) UnpinProject(path string) (*desktop.Dashboard, error) {
	_, err := desktop.UnpinProject(path)
	if err != nil {
		return nil, err
	}
	s.syncHubFromPrefs()

	cur := s.currentPath()
	if cur != "" && desktop.SamePath(cur, path) {
		s.setProjectPath("")
		s.refreshTray()
		return nil, nil
	}
	s.refreshTray()
	if cur == "" {
		return nil, nil
	}
	return s.RefreshDashboard()
}

// RefreshDashboard reloads status for the current project.
func (s *AppService) RefreshDashboard() (*desktop.Dashboard, error) {
	path := s.currentPath()
	if path == "" {
		return nil, fmt.Errorf("no project open")
	}
	dash, err := desktop.LoadDashboard(path)
	if err != nil {
		return nil, err
	}
	s.setProjectPath(dash.Path)
	s.syncHubFromPrefs()
	s.refreshTray()
	return dash, nil
}

// GetUsageReport returns AI token usage for the chart UI (ledger-based, like `ob report`).
// periodKey: "24h" (default), "7d", "30d", "90d", "month", "all".
func (s *AppService) GetUsageReport(periodKey string) (*desktop.UsageReportView, error) {
	return desktop.LoadUsageReport(periodKey)
}

// RefreshDockerStatus loads Docker/compose status for the open project (slow path).
func (s *AppService) RefreshDockerStatus() (desktop.DockerStatus, error) {
	return desktop.LoadDockerStatus(s.currentPath())
}

// RefreshOpenPR loads the open PR for the current branch (gh CLI; slow path).
func (s *AppService) RefreshOpenPR() (*desktop.PRStatus, error) {
	return desktop.LoadOpenPR(s.currentPath())
}

// LoadCommitActivity returns a GitHub-style commit calendar for the open project.
// authorOnly=true filters by local git user.email.
func (s *AppService) LoadCommitActivity(authorOnly bool) (*desktop.CommitActivityView, error) {
	return desktop.LoadCommitActivity(s.currentPath(), authorOnly)
}

// LoadTimeline returns a unified commit + PR activity timeline for the open project.
func (s *AppService) LoadTimeline(limit int) (*desktop.TimelineView, error) {
	return desktop.LoadTimeline(s.currentPath(), limit)
}

// RevertTimelineCommit creates a git revert commit for the given hash.
func (s *AppService) RevertTimelineCommit(hash string, isMerge bool) (*desktop.HistoryActionResult, error) {
	res, err := desktop.RevertTimelineCommit(s.currentPath(), hash, isMerge)
	if err != nil {
		return nil, err
	}
	if res != nil && res.Dashboard != nil {
		s.setProjectPath(res.Dashboard.Path)
	}
	s.syncHubFromPrefs()
	s.refreshTray()
	return res, nil
}

// ResetTimelineCommit runs git reset --soft|mixed|hard to hash (must be ancestor of HEAD).
func (s *AppService) ResetTimelineCommit(hash, mode string) (*desktop.HistoryActionResult, error) {
	res, err := desktop.ResetTimelineCommit(s.currentPath(), hash, mode)
	if err != nil {
		return nil, err
	}
	if res != nil && res.Dashboard != nil {
		s.setProjectPath(res.Dashboard.Path)
	}
	s.syncHubFromPrefs()
	s.refreshTray()
	return res, nil
}

// DeleteTimelineBranch deletes a local branch from the timeline context menu.
func (s *AppService) DeleteTimelineBranch(name string, force bool) (*desktop.HistoryActionResult, error) {
	res, err := desktop.DeleteTimelineBranch(s.currentPath(), name, force)
	if err != nil {
		return nil, err
	}
	if res != nil && res.Dashboard != nil {
		s.setProjectPath(res.Dashboard.Path)
	}
	s.syncHubFromPrefs()
	s.refreshTray()
	return res, nil
}

// FileDiff returns before/after content for a changed file in the open project.
func (s *AppService) FileDiff(path string) (*desktop.FileDiffView, error) {
	return desktop.LoadFileDiff(s.currentPath(), path)
}

// SyncModes returns sync presets for the desktop dialog.
func (s *AppService) SyncModes() []desktop.SyncModeView {
	return desktop.SyncModes()
}

// RunSync synchronizes the base branch (and optionally prunes) for the open project.
func (s *AppService) RunSync(mode, base string) (*desktop.SyncResult, error) {
	res, err := desktop.RunSync(s.currentPath(), mode, base)
	if err != nil {
		return nil, err
	}
	if res != nil && res.Dashboard != nil {
		s.setProjectPath(res.Dashboard.Path)
	}
	s.syncHubFromPrefs()
	s.refreshTray()
	return res, nil
}

// RunPull fetches origin and fast-forwards the current branch / local base.
func (s *AppService) RunPull(base string) (*desktop.PullResult, error) {
	res, err := desktop.RunPull(s.currentPath(), base)
	if err != nil {
		return nil, err
	}
	if res != nil && res.Dashboard != nil {
		s.setProjectPath(res.Dashboard.Path)
	}
	s.syncHubFromPrefs()
	s.refreshTray()
	return res, nil
}

// MarkPRReady marks the open draft PR as ready for review.
func (s *AppService) MarkPRReady() (*desktop.PRStatus, error) {
	return desktop.MarkPRReady(s.currentPath())
}

// MergePR merges the open PR (method: squash|merge|rebase).
func (s *AppService) MergePR(method string) (*desktop.PROutcome, error) {
	out, err := desktop.MergePR(s.currentPath(), method)
	if err != nil {
		return nil, err
	}
	s.syncHubFromPrefs()
	s.refreshTray()
	return out, nil
}

// ListBranches returns local branches for the open project.
func (s *AppService) ListBranches() ([]desktop.BranchView, error) {
	return desktop.ListBranches(s.currentPath())
}

// CheckoutBranch switches to the given local branch and refreshes the dashboard.
func (s *AppService) CheckoutBranch(name string) (*desktop.Dashboard, error) {
	dash, err := desktop.CheckoutBranch(s.currentPath(), name)
	if err != nil {
		return nil, err
	}
	s.setProjectPath(dash.Path)
	s.syncHubFromPrefs()
	s.refreshTray()
	return dash, nil
}

// CloseProject clears the current project selection (keeps pins).
func (s *AppService) CloseProject() {
	s.setProjectPath("")
	s.syncHubFromPrefs()
	s.refreshTray()
}

// PreviewCommit generates a commit message via AI for the open project.
func (s *AppService) PreviewCommit() (*desktop.CommitPreview, error) {
	return desktop.PreviewCommit(context.Background(), s.currentPath())
}

// ConfirmCommit creates the commit with the reviewed message.
func (s *AppService) ConfirmCommit(message string) (*desktop.CommitOutcome, error) {
	return desktop.ConfirmCommit(context.Background(), s.currentPath(), message)
}

// ConfirmCommitAndPush commits with the reviewed message and pushes to origin.
func (s *AppService) ConfirmCommitAndPush(message string) (*desktop.CommitOutcome, error) {
	out, err := desktop.ConfirmCommitAndPush(context.Background(), s.currentPath(), message)
	if err != nil {
		return nil, err
	}
	s.syncHubFromPrefs()
	s.refreshTray()
	return out, nil
}

// CreateBranch creates and checks out a new local branch, then refreshes the dashboard.
func (s *AppService) CreateBranch(name, from string) (*desktop.Dashboard, error) {
	dash, err := desktop.CreateBranch(s.currentPath(), name, from)
	if err != nil {
		return nil, err
	}
	s.setProjectPath(dash.Path)
	s.syncHubFromPrefs()
	s.refreshTray()
	return dash, nil
}

// CheckOnboarding returns setup status for config/gh/remote.
func (s *AppService) CheckOnboarding() (*desktop.OnboardingStatus, error) {
	return desktop.CheckOnboarding(s.currentPath())
}

// SaveAIConfig persists provider, API key and model to local config.
func (s *AppService) SaveAIConfig(provider, apiKey, model string) error {
	return desktop.SaveAIConfig(provider, apiKey, model)
}

// PreviewPR generates PR title/body via AI (dry-run).
func (s *AppService) PreviewPR(draft bool) (*desktop.PRPreview, error) {
	return desktop.PreviewPR(context.Background(), s.currentPath(), draft)
}

// ConfirmPR pushes and creates the PR with reviewed fields.
func (s *AppService) ConfirmPR(title, body string, draft bool) (*desktop.PROutcome, error) {
	return desktop.ConfirmPR(context.Background(), s.currentPath(), title, body, draft)
}

// DockerUp starts compose services for the open project.
func (s *AppService) DockerUp(build bool) (*desktop.DockerActionResult, error) {
	res, err := desktop.DockerUp(s.currentPath(), build)
	if err != nil {
		return nil, err
	}
	s.afterDocker(res)
	return res, nil
}

// DockerDown stops and removes compose services.
func (s *AppService) DockerDown() (*desktop.DockerActionResult, error) {
	res, err := desktop.DockerDown(s.currentPath())
	if err != nil {
		return nil, err
	}
	s.afterDocker(res)
	return res, nil
}

// DockerStop stops running compose services.
func (s *AppService) DockerStop() (*desktop.DockerActionResult, error) {
	res, err := desktop.DockerStop(s.currentPath(), nil)
	if err != nil {
		return nil, err
	}
	s.afterDocker(res)
	return res, nil
}

// DockerStart starts compose services.
func (s *AppService) DockerStart() (*desktop.DockerActionResult, error) {
	res, err := desktop.DockerStart(s.currentPath(), nil)
	if err != nil {
		return nil, err
	}
	s.afterDocker(res)
	return res, nil
}

// DockerRecreate force-recreates the default (or given) service.
func (s *AppService) DockerRecreate(service string) (*desktop.DockerActionResult, error) {
	res, err := desktop.DockerRecreate(s.currentPath(), service)
	if err != nil {
		return nil, err
	}
	s.afterDocker(res)
	return res, nil
}

// ListDockerPresets returns project docker command presets.
func (s *AppService) ListDockerPresets() ([]desktop.DockerPresetView, error) {
	return desktop.ListDockerPresets(s.currentPath())
}

// ListDockerKits returns built-in kits available for import.
func (s *AppService) ListDockerKits() ([]desktop.DockerKitView, error) {
	return desktop.ListDockerKits()
}

// ImportDockerKit merges a built-in kit into the open project's presets.
func (s *AppService) ImportDockerKit(kitID string) (*desktop.DockerImportResult, error) {
	return desktop.ImportDockerKit(s.currentPath(), kitID)
}

// DockerRunPreset runs a one-shot preset (or signals interactive shell via result.interactive).
func (s *AppService) DockerRunPreset(service, presetID string) (*desktop.DockerExecResult, error) {
	return desktop.RunDockerPreset(s.currentPath(), service, presetID)
}

// DockerExecCommand runs an arbitrary one-shot command in a service.
func (s *AppService) DockerExecCommand(service, command string) (*desktop.DockerExecResult, error) {
	return desktop.RunDockerExecCommand(s.currentPath(), service, command)
}

// ListGlobalDocker returns all daemon containers for the home panel (no project required).
func (s *AppService) ListGlobalDocker() desktop.GlobalDockerView {
	return desktop.LoadGlobalDocker()
}

// GlobalDockerStart starts a container by ID/name (home panel).
func (s *AppService) GlobalDockerStart(idOrName string) (*desktop.GlobalDockerActionResult, error) {
	return desktop.GlobalDockerStart(idOrName)
}

// GlobalDockerStop stops a container by ID/name (home panel).
func (s *AppService) GlobalDockerStop(idOrName string) (*desktop.GlobalDockerActionResult, error) {
	return desktop.GlobalDockerStop(idOrName)
}

// GlobalDockerRecreate force-recreates a compose service for the container (home panel).
func (s *AppService) GlobalDockerRecreate(idOrName string) (*desktop.GlobalDockerActionResult, error) {
	return desktop.GlobalDockerRecreate(idOrName)
}

// GlobalDockerUp runs compose up -d for a compose file (home panel).
func (s *AppService) GlobalDockerUp(composeFile string, build bool) (*desktop.GlobalDockerActionResult, error) {
	return desktop.GlobalDockerUp(composeFile, build)
}

// GlobalDockerDown runs compose down for a compose file (home panel).
func (s *AppService) GlobalDockerDown(composeFile string) (*desktop.GlobalDockerActionResult, error) {
	return desktop.GlobalDockerDown(composeFile)
}

func (s *AppService) afterDocker(res *desktop.DockerActionResult) {
	if res != nil && res.Dashboard != nil {
		s.setProjectPath(res.Dashboard.Path)
	}
	s.syncHubFromPrefs()
	s.refreshTray()
}

func (s *AppService) refreshTray() {
	s.mu.RLock()
	fn := s.trayRefresh
	s.mu.RUnlock()
	if fn != nil {
		fn()
	}
}
