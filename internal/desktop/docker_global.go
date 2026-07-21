package desktop

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/laerciocrestani/openbench/internal/app"
	dockerpkg "github.com/laerciocrestani/openbench/internal/docker"
)

// GlobalDockerView is the home-screen Docker Desktop–style summary.
type GlobalDockerView struct {
	Available     bool                  `json:"available"`
	DaemonRunning bool                  `json:"daemonRunning"`
	Summary       string                `json:"summary"`
	Running       int                   `json:"running"`
	Total         int                   `json:"total"`
	Projects      []GlobalDockerProject `json:"projects"`
	Error         string                `json:"error,omitempty"`
}

// GlobalDockerProject groups containers by compose project (or standalone).
type GlobalDockerProject struct {
	Name        string                  `json:"name"`
	WorkingDir  string                  `json:"workingDir,omitempty"`
	ComposeFile string                  `json:"composeFile,omitempty"`
	CanCompose  bool                    `json:"canCompose"`
	Running     int                     `json:"running"`
	Total       int                     `json:"total"`
	Containers  []GlobalDockerContainer `json:"containers"`
}

// GlobalDockerContainer is one daemon container for the home panel.
type GlobalDockerContainer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Image       string `json:"image,omitempty"`
	State       string `json:"state"`
	Status      string `json:"status,omitempty"`
	Ports       string `json:"ports,omitempty"`
	Project     string `json:"project,omitempty"`
	Service     string `json:"service,omitempty"`
	WorkingDir  string `json:"workingDir,omitempty"`
	ComposeFile string `json:"composeFile,omitempty"`
	CanCompose  bool   `json:"canCompose"`
}

// GlobalDockerActionResult is returned after a global docker mutate action.
type GlobalDockerActionResult struct {
	Action  string           `json:"action"`
	Message string           `json:"message"`
	Docker  GlobalDockerView `json:"docker"`
}

// LoadGlobalDocker lists all daemon containers grouped by compose project.
func LoadGlobalDocker() GlobalDockerView {
	view := GlobalDockerView{
		Available: dockerpkg.HasDocker(),
		Projects:  []GlobalDockerProject{},
	}
	if !view.Available {
		view.Summary = "n/a"
		view.Error = "docker CLI não encontrado no PATH"
		return view
	}

	view.DaemonRunning = dockerpkg.DaemonRunning()
	if !view.DaemonRunning {
		view.Summary = "off"
		view.Error = "Docker daemon não está rodando"
		return view
	}

	containers, err := dockerpkg.ListAllContainers()
	if err != nil {
		view.Summary = "error"
		view.Error = err.Error()
		return view
	}

	groups := map[string]*GlobalDockerProject{}
	order := make([]string, 0)

	for _, c := range containers {
		key := c.Project
		title := c.Project
		if key == "" {
			key = "standalone:" + c.ID
			title = "Standalone"
		}
		proj, ok := groups[key]
		if !ok {
			proj = &GlobalDockerProject{
				Name:        title,
				WorkingDir:  c.WorkingDir,
				ComposeFile: c.ComposeFile,
				CanCompose:  c.ComposeFile != "",
				Containers:  []GlobalDockerContainer{},
			}
			groups[key] = proj
			order = append(order, key)
		}
		if proj.ComposeFile == "" && c.ComposeFile != "" {
			proj.ComposeFile = c.ComposeFile
			proj.CanCompose = true
		}
		if proj.WorkingDir == "" && c.WorkingDir != "" {
			proj.WorkingDir = c.WorkingDir
		}
		item := mapGlobalContainer(c)
		proj.Containers = append(proj.Containers, item)
		proj.Total++
		view.Total++
		if dockerpkg.IsRunningState(c.State) {
			proj.Running++
			view.Running++
		}
	}

	view.Projects = make([]GlobalDockerProject, 0, len(order))
	for _, key := range order {
		view.Projects = append(view.Projects, *groups[key])
	}

	switch {
	case view.Total == 0:
		view.Summary = "vazio"
	case view.Running == 0:
		view.Summary = "stopped"
	default:
		view.Summary = fmt.Sprintf("%d/%d", view.Running, view.Total)
	}
	return view
}

func mapGlobalContainer(c dockerpkg.GlobalContainer) GlobalDockerContainer {
	return GlobalDockerContainer{
		ID:          c.ID,
		Name:        c.Name,
		Image:       c.Image,
		State:       c.State,
		Status:      c.Status,
		Ports:       c.Ports,
		Project:     c.Project,
		Service:     c.Service,
		WorkingDir:  c.WorkingDir,
		ComposeFile: c.ComposeFile,
		CanCompose:  c.ComposeFile != "",
	}
}

// GlobalDockerStart starts a container by ID/name.
func GlobalDockerStart(idOrName string) (*GlobalDockerActionResult, error) {
	if err := dockerpkg.StartContainer(idOrName); err != nil {
		return nil, err
	}
	return globalDockerResult("start", "container iniciado")
}

// GlobalDockerStop stops a container by ID/name.
func GlobalDockerStop(idOrName string) (*GlobalDockerActionResult, error) {
	if err := dockerpkg.StopContainer(idOrName); err != nil {
		return nil, err
	}
	return globalDockerResult("stop", "container parado")
}

// GlobalDockerRecreate force-recreates a compose service for the container.
func GlobalDockerRecreate(idOrName string) (*GlobalDockerActionResult, error) {
	c, err := dockerpkg.FindGlobalContainer(idOrName)
	if err != nil {
		return nil, err
	}
	if c.ComposeFile == "" || c.Service == "" {
		return nil, fmt.Errorf("recreate só está disponível para containers Compose com serviço identificado")
	}
	if err := app.RunDockerRecreate(app.DockerOptions{
		WorkDir:     c.WorkingDir,
		ComposeFile: c.ComposeFile,
		Service:     c.Service,
	}); err != nil {
		return nil, err
	}
	return globalDockerResult("recreate", fmt.Sprintf("serviço %s recriado", c.Service))
}

// GlobalDockerUp runs compose up -d for a compose file path.
func GlobalDockerUp(composeFile string, build bool) (*GlobalDockerActionResult, error) {
	composeFile = strings.TrimSpace(composeFile)
	if composeFile == "" {
		return nil, fmt.Errorf("compose file não informado")
	}
	workDir := filepath.Dir(composeFile)
	if err := app.RunDockerUp(app.DockerOptions{
		WorkDir:     workDir,
		ComposeFile: composeFile,
		Build:       build,
	}); err != nil {
		return nil, err
	}
	return globalDockerResult("up", "compose up ok")
}

// GlobalDockerDown runs compose down for a compose file path.
func GlobalDockerDown(composeFile string) (*GlobalDockerActionResult, error) {
	composeFile = strings.TrimSpace(composeFile)
	if composeFile == "" {
		return nil, fmt.Errorf("compose file não informado")
	}
	workDir := filepath.Dir(composeFile)
	if err := app.RunDockerDown(app.DockerOptions{
		WorkDir:     workDir,
		ComposeFile: composeFile,
	}); err != nil {
		return nil, err
	}
	return globalDockerResult("down", "compose down ok")
}

func globalDockerResult(action, message string) (*GlobalDockerActionResult, error) {
	return &GlobalDockerActionResult{
		Action:  action,
		Message: message,
		Docker:  LoadGlobalDocker(),
	}, nil
}
