package pr

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/laerciocrestani/gitia/internal/formatter"
)

type Client struct {
	dir string
}

func New() (*Client, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &Client{dir: dir}, nil
}

func (c *Client) run(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Dir = c.dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (c *Client) Exists() (bool, string, error) {
	out, err := c.run("pr", "view", "--json", "url", "-q", ".url")
	if err != nil {
		if strings.Contains(err.Error(), "no pull requests") ||
			strings.Contains(err.Error(), "could not find") ||
			strings.Contains(err.Error(), "not found") {
			return false, "", nil
		}
		return false, "", err
	}
	return true, out, nil
}

func (c *Client) Create(title, commitMessage, base string, draft bool) (string, error) {
	exists, url, err := c.Exists()
	if err != nil {
		return "", err
	}
	if exists {
		return url, fmt.Errorf("PR já existe: %s", url)
	}

	body := buildPRBody(commitMessage)

	args := []string{
		"pr", "create",
		"--title", title,
		"--body", body,
		"--base", base,
	}
	if draft {
		args = append(args, "--draft")
	}

	return c.run(args...)
}

func buildPRBody(commitMessage string) string {
	bullets := formatter.BodyBullets(commitMessage)

	var b strings.Builder
	b.WriteString("## Summary\n")
	if len(bullets) == 0 {
		b.WriteString("- ")
		b.WriteString(formatter.TitleLine(commitMessage))
		b.WriteString("\n")
	} else {
		for _, bullet := range bullets {
			b.WriteString("- ")
			b.WriteString(bullet)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n## Test plan\n")
	b.WriteString("- [ ] Verificar alterações localmente\n")
	b.WriteString("- [ ] Confirmar que testes passam\n")

	return b.String()
}

func (c *Client) PreviewCreate(title, commitMessage, base string, draft bool) string {
	body := buildPRBody(commitMessage)
	draftFlag := ""
	if draft {
		draftFlag = " --draft"
	}
	return fmt.Sprintf("gh pr create --title %q --body %q --base %q%s",
		title, body, base, draftFlag)
}
