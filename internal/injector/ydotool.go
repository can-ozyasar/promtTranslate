package injector

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// YdotoolInjector uses ydotool to type text on both X11 and Wayland.
// It requires ydotoold to be running (see install.sh).
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
// It checks that ydotoold is reachable before attempting injection.
func (y *YdotoolInjector) Type(ctx context.Context, text string) error {
	// Verify ydotoold socket is available.
	if err := checkYdotoold(ctx); err != nil {
		return err
	}

	// Give the launcher window time to close so focus returns to the terminal.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(120 * time.Millisecond):
	}

	// ydotool type --next-delay <ms> -- <text>
	cmd := exec.CommandContext(ctx,
		"ydotool", "type",
		"--next-delay", strconv.Itoa(y.delayMS),
		"--", text,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ydotool: %w — %s", err, string(out))
	}
	return nil
}

// checkYdotoold verifies the ydotoold daemon is running by running a no-op command.
func checkYdotoold(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "ydotool", "key") // no args → ydotoold connects, does nothing
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ydotoold does not appear to be running — " +
			"start it with: systemctl --user start ydotoold")
	}
	return nil
}
