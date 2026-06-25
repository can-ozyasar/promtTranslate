package injector

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// WaylandInjector uses wl-copy to copy text to the clipboard.
// This is more reliable on Wayland than ydotool.
type WaylandInjector struct{}

// NewWaylandInjector creates a new WaylandInjector.
func NewWaylandInjector() Injector {
	return &WaylandInjector{}
}

// Type uses wl-copy to copy text to the clipboard.
// Instead of injecting keystrokes, it places the result on the clipboard
// and notifies the user to paste it.
func (w *WaylandInjector) Type(ctx context.Context, text string) error {
	// 1. Show the translation in a Zenity window so the user can read it.
	zenityCmd := exec.CommandContext(ctx, "zenity",
		"--text-info",
		"--title", "Çeviri Sonucu (Kapatınca Kopyalanır)",
		"--width", "600",
		"--height", "300",
		"--ok-label", "Kopyala ve Kapat",
		"--cancel-label", "İptal",
	)
	zenityCmd.Stdin = strings.NewReader(text)
	_ = zenityCmd.Run() // Wait for user to read it

	// 2. Copy to clipboard
	cmd := exec.CommandContext(ctx, "wl-copy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy failed: %w", err)
	}

	// 3. Optional small notification
	msg := "Metin panoya kopyalandı. <b>Ctrl+V</b> ile yapıştırabilirsiniz."
	notify := exec.CommandContext(ctx, "notify-send", "-a", "promptTranslate", "-i", "accessories-dictionary", "Panoya Kopyalandı 📋", msg)
	_ = notify.Run()

	return nil
}
