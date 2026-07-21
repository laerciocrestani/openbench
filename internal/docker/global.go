package docker

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	labelComposeProject     = "com.docker.compose.project"
	labelComposeService     = "com.docker.compose.service"
	labelComposeWorkingDir  = "com.docker.compose.project.working_dir"
	labelComposeConfigFiles = "com.docker.compose.project.config_files"
)

// GlobalContainer describes a daemon-wide container (any project).
type GlobalContainer struct {
	ID          string
	Name        string
	Image       string
	State       string
	Status      string
	Ports       string
	Project     string
	Service     string
	WorkingDir  string
	ComposeFile string
}

// ListAllContainers returns all containers from the Docker daemon (docker ps -a).
func ListAllContainers() ([]GlobalContainer, error) {
	if !HasDocker() {
		return nil, fmt.Errorf("docker CLI não encontrado")
	}
	if !DaemonRunning() {
		return nil, fmt.Errorf("Docker daemon não está rodando")
	}

	// Tab-separated fields; Label templates avoid ambiguous Labels CSV parsing.
	format := strings.Join([]string{
		"{{.ID}}",
		"{{.Names}}",
		"{{.Image}}",
		"{{.State}}",
		"{{.Status}}",
		"{{.Ports}}",
		`{{.Label "` + labelComposeProject + `"}}`,
		`{{.Label "` + labelComposeService + `"}}`,
		`{{.Label "` + labelComposeWorkingDir + `"}}`,
		`{{.Label "` + labelComposeConfigFiles + `"}}`,
	}, "\t")

	cmd := exec.Command("docker", "ps", "-a", "--format", format)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("docker ps: %s", msg)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && strings.TrimSpace(lines[0]) == "" {
		return nil, nil
	}

	containers := make([]GlobalContainer, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		for len(parts) < 10 {
			parts = append(parts, "")
		}
		workingDir := strings.TrimSpace(parts[8])
		configFiles := strings.TrimSpace(parts[9])
		containers = append(containers, GlobalContainer{
			ID:          strings.TrimSpace(parts[0]),
			Name:        firstContainerName(parts[1]),
			Image:       strings.TrimSpace(parts[2]),
			State:       strings.TrimSpace(parts[3]),
			Status:      strings.TrimSpace(parts[4]),
			Ports:       strings.TrimSpace(parts[5]),
			Project:     strings.TrimSpace(parts[6]),
			Service:     strings.TrimSpace(parts[7]),
			WorkingDir:  workingDir,
			ComposeFile: resolveComposeFromLabels(workingDir, configFiles),
		})
	}
	return containers, nil
}

func firstContainerName(names string) string {
	names = strings.TrimSpace(names)
	if names == "" {
		return ""
	}
	// docker may return "name" or "name,alias"
	if i := strings.IndexByte(names, ','); i >= 0 {
		names = names[:i]
	}
	return strings.TrimPrefix(names, "/")
}

func resolveComposeFromLabels(workingDir, configFiles string) string {
	configFiles = strings.TrimSpace(configFiles)
	if configFiles == "" {
		return ""
	}
	// Multiple files are comma-separated; prefer the first.
	first := strings.TrimSpace(strings.Split(configFiles, ",")[0])
	if first == "" {
		return ""
	}
	if filepath.IsAbs(first) {
		return first
	}
	workingDir = strings.TrimSpace(workingDir)
	if workingDir == "" {
		return first
	}
	return filepath.Join(workingDir, first)
}

// StartContainer starts a container by ID or name.
func StartContainer(idOrName string) error {
	return runDockerContainerAction("start", idOrName)
}

// StopContainer stops a container by ID or name.
func StopContainer(idOrName string) error {
	return runDockerContainerAction("stop", idOrName)
}

func runDockerContainerAction(action, idOrName string) error {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return fmt.Errorf("container não informado")
	}
	if !HasDocker() {
		return fmt.Errorf("docker CLI não encontrado")
	}
	cmd := exec.Command("docker", action, idOrName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("docker %s: %s", action, msg)
	}
	return nil
}

// FindGlobalContainer looks up a container by ID or name.
func FindGlobalContainer(idOrName string) (GlobalContainer, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return GlobalContainer{}, fmt.Errorf("container não informado")
	}
	all, err := ListAllContainers()
	if err != nil {
		return GlobalContainer{}, err
	}
	for _, c := range all {
		if c.ID == idOrName || c.Name == idOrName || strings.HasPrefix(c.ID, idOrName) {
			return c, nil
		}
	}
	return GlobalContainer{}, fmt.Errorf("container %q não encontrado", idOrName)
}
