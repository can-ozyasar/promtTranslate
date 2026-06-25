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
	// GNOME Wayland security blocks virtual keyboards heavily.
	// The most reliable cross-compositor way is copying to the clipboard.
	cmd := exec.CommandContext(ctx, "wl-copy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy failed: %w", err)
	}

	// Show a nice desktop notification
	msg := fmt.Sprintf("Çeviri panoya kopyalandı, <b>Ctrl+V</b> ile yapıştırabilirsiniz.\n\n<i>%s</i>", text)
	notify := exec.CommandContext(ctx, "notify-send", "-a", "promptTranslate", "-i", "accessories-dictionary", "Çeviri Hazır! 📋", msg)
	_ = notify.Run()

	return nil
}
