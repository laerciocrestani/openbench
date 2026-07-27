package desktop

import (
	"fmt"
	"strings"

	dockerpkg "github.com/laerciocrestani/openbench/internal/docker"
)

// DockerActionResult is returned after a docker mutate action.
type DockerActionResult struct {
	Action    string       `json:"action"`
	Message   string       `json:"message"`
	Output    string       `json:"output,omitempty"`
	OK        bool         `json:"ok"`
	Dashboard *Dashboard   `json:"dashboard"`
	Docker    DockerStatus `json:"docker"`
}

// DockerProgressHooks wires live compose output into the UI.
type DockerProgressHooks struct {
	OnLine    func(line string)
	OnService func(name, status, detail string)
}

// LoadDockerStatus returns docker status for a project path.
func LoadDockerStatus(projectPath string) (DockerStatus, error) {
	if strings.TrimSpace(projectPath) == "" {
		return DockerStatus{}, fmt.Errorf("no project open")
	}
	ov := dockerpkg.LoadOverview(projectPath)
	return mapDocker(ov, dockerpkg.HasDocker()), nil
}

// DockerUp runs compose up -d for the project.
func DockerUp(projectPath string, build bool, hooks DockerProgressHooks) (*DockerActionResult, error) {
	action := "up"
	if build {
		action = "up --build"
	}
	return runDockerAction(projectPath, action, hooks, func(compose string, onLine func(string)) (dockerpkg.RunResult, error) {
		return dockerpkg.UpResult(dockerpkg.UpOptions{
			ComposeFile: compose,
			Build:       build,
			OnLine:      onLine,
		})
	})
}

// DockerDown runs compose down for the project.
func DockerDown(projectPath string, hooks DockerProgressHooks) (*DockerActionResult, error) {
	return runDockerAction(projectPath, "down", hooks, func(compose string, onLine func(string)) (dockerpkg.RunResult, error) {
		return dockerpkg.DownResult(dockerpkg.DownOptions{
			ComposeFile: compose,
			OnLine:      onLine,
		})
	})
}

// DockerStop stops all running compose services (or named ones).
func DockerStop(projectPath string, services []string, hooks DockerProgressHooks) (*DockerActionResult, error) {
	return runDockerAction(projectPath, "stop", hooks, func(compose string, onLine func(string)) (dockerpkg.RunResult, error) {
		svcs := services
		if len(svcs) == 0 {
			ov := dockerpkg.LoadOverview(projectPath)
			for _, c := range ov.Containers {
				if strings.EqualFold(c.State, "running") {
					svcs = append(svcs, c.Service)
				}
			}
		}
		if len(svcs) == 0 {
			return dockerpkg.RunResult{}, fmt.Errorf("nenhum serviço running para stop")
		}
		return dockerpkg.StopResult(dockerpkg.ServiceOptions{
			ComposeFile: compose,
			Services:    svcs,
			OnLine:      onLine,
		})
	})
}

// DockerStart starts compose services (defaults to all listed containers).
func DockerStart(projectPath string, services []string, hooks DockerProgressHooks) (*DockerActionResult, error) {
	return runDockerAction(projectPath, "start", hooks, func(compose string, onLine func(string)) (dockerpkg.RunResult, error) {
		svcs := services
		if len(svcs) == 0 {
			ov := dockerpkg.LoadOverview(projectPath)
			for _, c := range ov.Containers {
				if c.Service != "" {
					svcs = append(svcs, c.Service)
				}
			}
		}
		if len(svcs) == 0 {
			return dockerpkg.RunResult{}, fmt.Errorf("nenhum serviço para start")
		}
		return dockerpkg.StartResult(dockerpkg.ServiceOptions{
			ComposeFile: compose,
			Services:    svcs,
			OnLine:      onLine,
		})
	})
}

// DockerRecreate force-recreates a service (default: first/default service).
func DockerRecreate(projectPath, service string, hooks DockerProgressHooks) (*DockerActionResult, error) {
	return runDockerAction(projectPath, "recreate", hooks, func(compose string, onLine func(string)) (dockerpkg.RunResult, error) {
		svc := strings.TrimSpace(service)
		if svc == "" {
			ov := dockerpkg.LoadOverview(projectPath)
			svc = ov.DefaultService()
		}
		if svc == "" {
			return dockerpkg.RunResult{}, fmt.Errorf("informe o serviço para recreate")
		}
		return dockerpkg.RecreateResult(compose, svc, false, onLine)
	})
}

func runDockerAction(
	projectPath, action string,
	hooks DockerProgressHooks,
	fn func(compose string, onLine func(string)) (dockerpkg.RunResult, error),
) (*DockerActionResult, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	ov := dockerpkg.LoadOverview(projectPath)
	if !ov.Available {
		return nil, fmt.Errorf("docker CLI não encontrado")
	}
	if !ov.DaemonRunning {
		return nil, fmt.Errorf("Docker daemon não está rodando")
	}
	if ov.ComposeFile == "" {
		return nil, fmt.Errorf("compose file não encontrado no projeto")
	}

	onLine := func(line string) {
		if hooks.OnLine != nil {
			hooks.OnLine(line)
		}
		if hooks.OnService == nil {
			return
		}
		if name, status, detail := parseComposeServiceLine(line); name != "" {
			hooks.OnService(name, status, detail)
		}
	}

	runRes, runErr := fn(ov.ComposeFile, onLine)

	dash, err := LoadDashboard(projectPath)
	if err != nil {
		if runErr != nil {
			return &DockerActionResult{
				Action:  action,
				Message: runErr.Error(),
				Output:  runRes.Output,
				OK:      false,
			}, runErr
		}
		return nil, err
	}
	docker, err := LoadDockerStatus(projectPath)
	if err != nil {
		if runErr != nil {
			return &DockerActionResult{
				Action:    action,
				Message:   runErr.Error(),
				Output:    runRes.Output,
				OK:        false,
				Dashboard: dash,
			}, runErr
		}
		return nil, err
	}
	dash.Docker = docker
	dash.HasDocker = docker.Available

	res := &DockerActionResult{
		Action:    action,
		Output:    runRes.Output,
		OK:        runErr == nil,
		Dashboard: dash,
		Docker:    docker,
	}
	if runErr != nil {
		msg := runErr.Error()
		if strings.TrimSpace(runRes.Output) != "" {
			msg = summarizeDockerError(runRes.Output, runErr)
		}
		res.Message = msg
		// Prefer returning the result so the UI can show output; still signal failure.
		return res, fmt.Errorf("%s", msg)
	}
	res.Message = fmt.Sprintf("docker %s ok", action)
	return res, nil
}

func summarizeDockerError(output string, err error) string {
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") ||
			strings.Contains(lower, "failed") ||
			strings.Contains(lower, "port is already") ||
			strings.Contains(lower, "address already in use") ||
			strings.Contains(lower, "bind for") {
			return line
		}
	}
	if err != nil {
		return err.Error()
	}
	return "docker action failed"
}

// parseComposeServiceLine extracts a service/container progress hint from a compose log line.
func parseComposeServiceLine(line string) (name, status, detail string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", ""
	}
	lower := strings.ToLower(line)

	// "Container project-service-1  Starting" / "Created" / "Started" / "Stopping" / "Stopped" / "Removing" / "Removed"
	if strings.HasPrefix(lower, "container ") {
		rest := strings.TrimSpace(line[len("Container "):])
		parts := strings.Fields(rest)
		if len(parts) >= 2 {
			cname := parts[0]
			verb := strings.ToLower(parts[len(parts)-1])
			svc := serviceNameFromContainer(cname)
			switch verb {
			case "creating", "starting", "stopping", "removing", "recreating":
				return svc, "running", line
			case "created", "started", "healthy", "stopped", "removed", "exited":
				return svc, "ok", line
			case "error", "failed":
				return svc, "error", line
			}
		}
	}

	if strings.Contains(lower, "error") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "port is already") ||
		strings.Contains(lower, "address already in use") ||
		strings.Contains(lower, "bind for") {
		if svc := extractServiceHint(line); svc != "" {
			return svc, "error", line
		}
		return "_", "error", line
	}
	return "", "", ""
}

func serviceNameFromContainer(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	// compose default: project-service-N
	parts := strings.Split(name, "-")
	if len(parts) >= 3 {
		return parts[len(parts)-2]
	}
	return name
}

func extractServiceHint(line string) string {
	lower := strings.ToLower(line)
	for _, key := range []string{`service "`, `service '`} {
		i := strings.Index(lower, key)
		if i < 0 {
			continue
		}
		rest := line[i+len(key):]
		end := strings.IndexAny(rest, "\"' \t,")
		if end > 0 {
			return rest[:end]
		}
	}
	return ""
}
