package injector

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// XdotoolInjector uses xdotool to type text into the active X11 window.
type XdotoolInjector struct {
	delayMS int
}

// NewXdotoolInjector creates an injector with the given per-keystroke delay.
func NewXdotoolInjector(delayMS int) Injector {
	if delayMS <= 0 {
		delayMS = 12
	}
	return &XdotoolInjector{delayMS: delayMS}
}

// Type uses xdotool type to inject text. It:
//   - uses --clearmodifiers to release stuck modifier keys (Alt, Shift…)
//   - uses --delay to pace keystrokes (some terminals drop events if too fast)
//   - uses -- to prevent text starting with "-" being parsed as a flag
//   - adds a small pre-injection pause so the launcher window has time to close
func (x *XdotoolInjector) Type(ctx context.Context, text string) error {
	// Give the rofi/wofi window time to close so the correct window receives focus.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(120 * time.Millisecond):
	}

	cmd := exec.CommandContext(ctx,
		"xdotool", "type",
		"--clearmodifiers",
		"--delay", strconv.Itoa(x.delayMS),
		"--", text,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("xdotool: %w — %s", err, string(out))
	}
	return nil
}
