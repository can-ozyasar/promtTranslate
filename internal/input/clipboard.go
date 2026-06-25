package input

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/canoz/promttranslate/internal/env"
)

// ClipboardReader reads text from the system clipboard or primary selection.
type ClipboardReader struct {
	session env.Session
}

// NewClipboardReader creates a reader appropriate for the given session type.
func NewClipboardReader(session env.Session) *ClipboardReader {
	return &ClipboardReader{session: session}
}

// ReadPrimary reads the "primary selection" — the text currently highlighted
// by the mouse — without requiring an explicit Ctrl+C copy.
func (c *ClipboardReader) ReadPrimary() (string, error) {
	var cmd *exec.Cmd
	switch c.session {
	case env.SessionWayland:
		// wl-paste --primary reads the Wayland primary selection.
		cmd = exec.Command("wl-paste", "--primary", "--no-newline")
	default: // X11
		// xclip -selection primary reads the X11 PRIMARY selection.
		cmd = exec.Command("xclip", "-o", "-selection", "primary")
	}

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("clipboard: read primary: %w", err)
	}
	text := strings.TrimRight(string(out), "\n")
	if text == "" {
		return "", fmt.Errorf("clipboard: primary selection is empty")
	}
	return text, nil
}

// ReadClipboard reads the standard clipboard (Ctrl+C buffer).
func (c *ClipboardReader) ReadClipboard() (string, error) {
	var cmd *exec.Cmd
	switch c.session {
	case env.SessionWayland:
		cmd = exec.Command("wl-paste", "--no-newline")
	default:
		cmd = exec.Command("xclip", "-o", "-selection", "clipboard")
	}

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("clipboard: read clipboard: %w", err)
	}
	text := strings.TrimRight(string(out), "\n")
	if text == "" {
		return "", fmt.Errorf("clipboard: clipboard is empty")
	}
	return text, nil
}

// WriteClipboard copies text to the standard clipboard.
func (c *ClipboardReader) WriteClipboard(text string) error {
	var cmd *exec.Cmd
	switch c.session {
	case env.SessionWayland:
		cmd = exec.Command("wl-copy")
	default:
		cmd = exec.Command("xclip", "-in", "-selection", "clipboard")
	}

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("clipboard: write stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("clipboard: start write cmd: %w", err)
	}
	if _, err := fmt.Fprint(pipe, text); err != nil {
		return fmt.Errorf("clipboard: write text: %w", err)
	}
	pipe.Close()
	return cmd.Wait()
}
