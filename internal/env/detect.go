package env

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Session represents the detected display server environment.
type Session string

const (
	SessionWayland Session = "wayland"
	SessionX11     Session = "x11"
	SessionUnknown Session = "unknown"
)

// Detect returns the current display session type.
func Detect() Session {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return SessionWayland
	}
	if os.Getenv("DISPLAY") != "" {
		return SessionX11
	}
	// Fallback: check XDG_SESSION_TYPE
	switch strings.ToLower(os.Getenv("XDG_SESSION_TYPE")) {
	case "wayland":
		return SessionWayland
	case "x11":
		return SessionX11
	}
	return SessionUnknown
}

// Dependency represents an external tool required at runtime.
type Dependency struct {
	Name     string
	Required bool
	ForX11   bool
	ForWayland bool
	InstallHint string
}

func allDeps() []Dependency {
	return []Dependency{
		{
			Name: "notify-send", Required: true,
			ForX11: true, ForWayland: true,
			InstallHint: "sudo apt install libnotify-bin",
		},
		{
			Name: "rofi", Required: false,
			ForX11: true, ForWayland: true,
			InstallHint: "sudo apt install rofi",
		},
		{
			Name: "wofi", Required: false,
			ForX11: false, ForWayland: true,
			InstallHint: "sudo apt install wofi",
		},
		{
			Name: "xclip", Required: false,
			ForX11: true, ForWayland: false,
			InstallHint: "sudo apt install xclip",
		},
		{
			Name: "xdotool", Required: false,
			ForX11: true, ForWayland: false,
			InstallHint: "sudo apt install xdotool",
		},
		{
			Name: "wl-paste", Required: false,
			ForX11: false, ForWayland: true,
			InstallHint: "sudo apt install wl-clipboard",
		},
		{
			Name: "wl-copy", Required: false,
			ForX11: false, ForWayland: true,
			InstallHint: "sudo apt install wl-clipboard",
		},
		{
			Name: "ydotool", Required: false,
			ForX11: false, ForWayland: true,
			InstallHint: "sudo apt install ydotool",
		},
	}
}

// CheckResult holds the result of a single dependency check.
type CheckResult struct {
	Name    string
	Found   bool
	Path    string
	Hint    string
}

// CheckDependencies verifies all runtime tools relevant to the given session.
// It returns warnings for missing optional tools and errors for missing required ones.
func CheckDependencies(session Session) (warnings []CheckResult, errs []CheckResult) {
	for _, dep := range allDeps() {
		relevant := dep.Required
		if !relevant {
			switch session {
			case SessionX11:
				relevant = dep.ForX11
			case SessionWayland:
				relevant = dep.ForWayland
			default:
				relevant = dep.ForX11 || dep.ForWayland
			}
		}
		if !relevant {
			continue
		}

		path, err := exec.LookPath(dep.Name)
		r := CheckResult{Name: dep.Name, Found: err == nil, Path: path, Hint: dep.InstallHint}
		if err == nil {
			continue
		}
		if dep.Required {
			errs = append(errs, r)
		} else {
			warnings = append(warnings, r)
		}
	}
	return
}

// CheckYdotoold verifies that the ydotoold daemon socket is reachable.
func CheckYdotoold() bool {
	cmd := exec.Command("ydotool", "--help")
	return cmd.Run() == nil
}

// PrintCheckReport prints a human-readable dependency report to stderr.
func PrintCheckReport(session Session) (ok bool) {
	warnings, errs := CheckDependencies(session)
	ok = len(errs) == 0

	fmt.Fprintf(os.Stderr, "promptTranslate — dependency check (%s)\n", session)
	fmt.Fprintln(os.Stderr, strings.Repeat("─", 50))

	if len(errs) == 0 && len(warnings) == 0 {
		fmt.Fprintln(os.Stderr, "✅  All required dependencies found.")
		return
	}

	for _, r := range errs {
		fmt.Fprintf(os.Stderr, "❌  MISSING (required): %-16s → %s\n", r.Name, r.Hint)
	}
	for _, r := range warnings {
		fmt.Fprintf(os.Stderr, "⚠️   missing (optional): %-16s → %s\n", r.Name, r.Hint)
	}
	return
}
