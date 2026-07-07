package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/aymanbagabas/go-osc52/v2"
)

// CopyToClipboard copies text to the system clipboard.
func CopyToClipboard(text string) error {
	if text == "" {
		return fmt.Errorf("nothing to copy")
	}

	if _, err := fmt.Fprint(os.Stderr, osc52.New(text)); err == nil {
		return nil
	}

	return copyFallback(text)
}

func copyFallback(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("clipboard unavailable")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("clipboard unavailable")
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := stdin.Write([]byte(text)); err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		return err
	}
	if err := stdin.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}
	return cmd.Wait()
}
