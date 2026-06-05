package oat

import (
	"bytes"
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// SetClipboard writes text to the system clipboard.
//
// Supported platforms:
//   - macOS:   pbcopy
//   - Linux:   xclip (X11), xsel (X11), wl-copy (Wayland) — tried in order
//   - Windows: clip
//
// Returns an error if no clipboard command is available or if the command fails.
func SetClipboard(text string) error {
	cmd := clipWriteCmd()
	if cmd == nil {
		return errors.New("clipboard: no supported clipboard command found")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// GetClipboard reads text from the system clipboard.
//
// See SetClipboard for platform support details.
func GetClipboard() (string, error) {
	cmd := clipReadCmd()
	if cmd == nil {
		return "", errors.New("clipboard: no supported clipboard command found")
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimRight(out.String(), "\n\r"), nil
}

func clipWriteCmd() *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("pbcopy")
	case "linux":
		if found("xclip") {
			return exec.Command("xclip", "-selection", "clipboard")
		}
		if found("xsel") {
			return exec.Command("xsel", "--clipboard", "--input")
		}
		if found("wl-copy") {
			return exec.Command("wl-copy")
		}
	case "windows":
		return exec.Command("clip")
	}
	return nil
}

func clipReadCmd() *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("pbpaste")
	case "linux":
		if found("xclip") {
			return exec.Command("xclip", "-selection", "clipboard", "-o")
		}
		if found("xsel") {
			return exec.Command("xsel", "--clipboard", "--output")
		}
		if found("wl-paste") {
			return exec.Command("wl-paste", "--no-newline")
		}
	case "windows":
		return exec.Command("powershell", "-command", "Get-Clipboard")
	}
	return nil
}

func found(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
