package notify

import (
	"fmt"
	"os/exec"
	"time"
)

// Urgency controls the notification urgency level.
type Urgency string

const (
	UrgencyLow      Urgency = "low"
	UrgencyNormal   Urgency = "normal"
	UrgencyCritical Urgency = "critical"
)

// Send displays a desktop notification using notify-send.
func Send(title, body string, urgency Urgency, expire time.Duration) error {
	ms := int(expire.Milliseconds())
	if ms <= 0 {
		ms = 8000
	}

	cmd := exec.Command("notify-send",
		"--urgency", string(urgency),
		"--expire-time", fmt.Sprintf("%d", ms),
		"--icon", "accessories-dictionary",
		"--app-name", "promptTranslate",
		title,
		body,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("notify-send: %w — %s", err, string(out))
	}
	return nil
}

// Info sends a normal-urgency notification.
func Info(title, body string) {
	Send(title, body, UrgencyNormal, 8*time.Second) //nolint:errcheck
}

// Warn sends a low-urgency warning notification.
func Warn(title, body string) {
	Send(title, body, UrgencyLow, 6*time.Second) //nolint:errcheck
}

// Error sends a critical-urgency error notification.
func Error(title, body string) {
	Send(title, body, UrgencyCritical, 12*time.Second) //nolint:errcheck
}

// Translation sends the standard translation-result notification (Okuma modu).
func Translation(original, translated string) {
	// Truncate very long originals for the subtitle.
	preview := original
	if len(preview) > 60 {
		preview = preview[:57] + "…"
	}
	body := fmt.Sprintf("%s\n\n📋 %s", preview, translated)
	Send("📖 Çeviri", body, UrgencyNormal, 10*time.Second) //nolint:errcheck
}
