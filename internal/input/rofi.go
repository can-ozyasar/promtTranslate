package input

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RofiLauncher manages a rofi or wofi dmenu subprocess.
type RofiLauncher struct {
	launcher string // "rofi" or "wofi"
	theme    string
}

// NewRofiLauncher creates a launcher preferring `preferred`; falls back to the other.
func NewRofiLauncher(preferred, theme string) *RofiLauncher {
	launcher := preferred
	if _, err := exec.LookPath(preferred); err != nil {
		// Try the other one
		alt := "wofi"
		if preferred == "wofi" {
			alt = "rofi"
		}
		if _, err2 := exec.LookPath(alt); err2 == nil {
			launcher = alt
		}
	}
	return &RofiLauncher{launcher: launcher, theme: theme}
}

// Prompt opens a dmenu prompt with the given placeholder text and returns
// the user's input. Returns ("", nil) if the user cancelled (pressed Esc).
func (r *RofiLauncher) Prompt(ctx context.Context, placeholder string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute) // generous timeout for typing
	defer cancel()

	var cmd *exec.Cmd
	switch r.launcher {
	case "wofi":
		cmd = exec.CommandContext(ctx,
			"wofi",
			"--dmenu",
			"--prompt", placeholder,
			"--lines", "1",
			"--width", "600",
			"--height", "50",
			"--normal-window", // Enables window dragging
			"--style", wofiCSS(),
			"--allow-markup=false",
			"--insensitive",
		)
	default: // rofi
		args := []string{
			"-dmenu",
			"-p", placeholder,
			"-lines", "0",
			"-width", "50",
			"-theme-str", rofiThemeInline(r.theme),
		}
		cmd = exec.CommandContext(ctx, "rofi", args...)
	}

	out, err := cmd.Output()
	if err != nil {
		// Exit code 1 means the user cancelled — not an error for us.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("rofi: %w", err)
	}

	text := strings.TrimRight(string(out), "\n")
	return text, nil
}

// rofiThemeInline returns a minimal inline rofi theme string for the dark
// prompttranslate style. This avoids requiring an external theme file.
func rofiThemeInline(theme string) string {
	if theme != "prompttranslate" {
		return ""
	}
	return `
* {
    background-color: #1a1b26;
    text-color:       #c0caf5;
    border-color:     #7aa2f7;
    font:             "JetBrains Mono 13";
}
window {
    width:            50%;
    border:           2px;
    border-radius:    8px;
    padding:          12px;
}
inputbar {
    children: [prompt, entry];
    padding:  8px;
    spacing:  8px;
}
prompt {
    text-color: #7aa2f7;
}
entry {
    text-color: #e0e0ff;
}
`
}

// wofiCSS returns a file path containing the CSS string for wofi.
// Wofi accepts CSS styling. We'll use a basic dark theme.
func wofiCSS() string {
	css := `
window {
    background-color: #1a1b26;
    color: #c0caf5;
    border: 2px solid #7aa2f7;
    border-radius: 8px;
    font-family: "JetBrains Mono", monospace;
    font-size: 13px;
}
input {
    background-color: #1a1b26;
    color: #e0e0ff;
    border: none;
    padding: 8px;
}
`
	tmpFile := filepath.Join(os.TempDir(), "prompttranslate-wofi.css")
	_ = os.WriteFile(tmpFile, []byte(css), 0644)
	return tmpFile
}
