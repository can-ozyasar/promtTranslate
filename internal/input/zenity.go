package input

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ZenityLauncher manages a zenity text-info dialog for multiline input.
type ZenityLauncher struct{}

// NewZenityLauncher creates a new Zenity launcher.
func NewZenityLauncher() *ZenityLauncher {
	return &ZenityLauncher{}
}

// Prompt opens a zenity text-info dialog. It saves and loads drafts automatically.
func (z *ZenityLauncher) Prompt(ctx context.Context, placeholder string) (string, error) {
	tmpFile := "/tmp/prompttranslate_draft.txt"

	// Ensure the file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		_ = os.WriteFile(tmpFile, []byte(""), 0644)
	}

	cmd := exec.CommandContext(ctx, "zenity",
		"--text-info",
		"--editable",
		"--title", placeholder,
		"--width", "600",
		"--height", "300",
		"--filename", tmpFile,
		"--ok-label", "Çevir",
		"--cancel-label", "İptal",
	)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// User cancelled. We don't clear the draft, so they can resume later if they want,
			// or we can clear it. Let's keep it.
			return "", nil
		}
		return "", fmt.Errorf("zenity failed: %w", err)
	}

	text := strings.TrimSpace(string(out))
	
	// Save what they typed as the draft. If it succeeds later, the handler will clear it.
	_ = os.WriteFile(tmpFile, []byte(text), 0644)

	return text, nil
}

// ClearDraft deletes the draft file. Called when translation succeeds.
func (z *ZenityLauncher) ClearDraft() {
	_ = os.Remove("/tmp/prompttranslate_draft.txt")
}

// DisplayResult shows a zenity window with the provided text.
func (z *ZenityLauncher) DisplayResult(ctx context.Context, title, text string) {
	cmd := exec.CommandContext(ctx, "zenity",
		"--text-info",
		"--title", title,
		"--width", "600",
		"--height", "300",
		"--ok-label", "Kapat",
		"--cancel-label", "İptal",
	)
	cmd.Stdin = strings.NewReader(text)
	_ = cmd.Run()
}
