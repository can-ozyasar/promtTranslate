package injector

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// YdotoolInjector uses ydotool to type text on both X11 and Wayland.
// This version of ydotool (Ubuntu 24.04 noble) works directly via
// /dev/uinput — no separate ydotoold daemon is required.
type YdotoolInjector struct {
	delayMS int
}

// NewYdotoolInjector creates an injector with the given per-keystroke delay.
func NewYdotoolInjector(delayMS int) Injector {
	if delayMS <= 0 {
		delayMS = 12
	}
	return &YdotoolInjector{delayMS: delayMS}
}

// Type uses ydotool type to inject text into the active window.
// Waits 150ms after hotkey so rofi has time to close and focus returns
// to the target terminal before keystrokes are sent.
func (y *YdotoolInjector) Type(ctx context.Context, text string) error {
	// Give the launcher window time to close so focus returns to the terminal.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(150 * time.Millisecond):
	}

	// ydotool type --next-delay <ms> -- <text>
	cmd := exec.CommandContext(ctx,
		"ydotool", "type",
		"--next-delay", strconv.Itoa(y.delayMS),
		"--", text,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ydotool: %w — %s\nHint: kullanıcının 'input' grubunda olduğundan emin olun: sudo usermod -aG input $USER", err, string(out))
	}
	return nil
}
