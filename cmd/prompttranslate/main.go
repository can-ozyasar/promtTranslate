// promptTranslate — global hotkey translation daemon for Linux.
//
// Usage:
//
//	prompttranslate [flags]
//
// Flags:
//
//	-check    Run dependency check and exit
//	-config   Path to config file (default: XDG_CONFIG_HOME/prompttranslate/config.toml)
//	-once-write   Translate one prompt via rofi and exit (useful for testing)
//	-once-read    Read clipboard, translate, notify and exit (useful for testing)
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/canoz/promttranslate/internal/config"
	"github.com/canoz/promttranslate/internal/env"
	"github.com/canoz/promttranslate/internal/hotkey"
	"github.com/canoz/promttranslate/internal/injector"
	"github.com/canoz/promttranslate/internal/input"
	"github.com/canoz/promttranslate/internal/notify"
	"github.com/canoz/promttranslate/internal/translator"
)

func main() {
	checkFlag := flag.Bool("check", false, "run dependency check and exit")
	onceWrite := flag.Bool("once-write", false, "open rofi, translate once, inject, then exit")
	onceRead := flag.Bool("once-read", false, "read clipboard, translate once, notify, then exit")
	flag.Parse()

	// ── Load config ──────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// ── Logging ──────────────────────────────────────────────────────────────
	level := slog.LevelInfo
	switch cfg.General.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	// ── Detect environment ───────────────────────────────────────────────────
	session := env.Detect()
	slog.Info("detected session", "type", session)

	// ── Dependency check ─────────────────────────────────────────────────────
	if *checkFlag {
		ok := env.PrintCheckReport(session)
		if !ok {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// ── Validate config ───────────────────────────────────────────────────────
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n\n", err)
		fmt.Fprintln(os.Stderr, "Set your API key via environment variable, for example:")
		fmt.Fprintln(os.Stderr, "  export GROQ_API_KEY=gsk_...")
		fmt.Fprintln(os.Stderr, "Or add it to ~/.config/prompttranslate/config.toml")
		os.Exit(1)
	}

	// ── Build translator ─────────────────────────────────────────────────────
	var t translator.Translator
	switch cfg.Translator.Provider {
	case "groq":
		t = translator.NewGroqClient(cfg.Groq.APIKey, cfg.Groq.Model, cfg.Groq.BaseURL)
	case "deepl":
		t = translator.NewDeepLClient(cfg.DeepL.APIKey, cfg.DeepL.Pro)
	}
	if cfg.Translator.CacheSize > 0 {
		t = translator.NewCachedTranslator(t, cfg.Translator.CacheSize)
	}

	// ── Build sub-systems ─────────────────────────────────────────────────────
	launcher := input.NewRofiLauncher(cfg.Display.Launcher, cfg.Display.Theme)
	clipReader := input.NewClipboardReader(session)

	var inj injector.Injector
	switch session {
	case env.SessionWayland:
		inj = injector.NewYdotoolInjector(cfg.Injection.KeystrokeDelayMS)
	default:
		inj = injector.NewXdotoolInjector(cfg.Injection.KeystrokeDelayMS)
	}

	// ── One-shot modes ────────────────────────────────────────────────────────
	ctx := context.Background()

	if *onceWrite {
		if err := handleWriteMode(ctx, cfg, launcher, t, inj); err != nil {
			slog.Error("write mode failed", "err", err)
			notify.Error("promptTranslate", err.Error())
			os.Exit(1)
		}
		return
	}
	if *onceRead {
		if err := handleReadMode(ctx, cfg, clipReader, t); err != nil {
			slog.Error("read mode failed", "err", err)
			notify.Error("promptTranslate", err.Error())
			os.Exit(1)
		}
		return
	}

	// ── Daemon mode ───────────────────────────────────────────────────────────
	slog.Info("promptTranslate daemon starting",
		"provider", cfg.Translator.Provider,
		"session", session,
		"write_hotkey", cfg.Hotkeys.WriteMode,
		"read_hotkey", cfg.Hotkeys.ReadMode,
	)

	combos := hotkey.DefaultCombos()

	handler := func(comboName string) {
		slog.Debug("hotkey fired", "combo", comboName)
		switch comboName {
		case "write_mode":
			if err := handleWriteMode(ctx, cfg, launcher, t, inj); err != nil {
				slog.Error("write mode", "err", err)
				notify.Error("Çeviri Hatası", err.Error())
			}
		case "read_mode":
			if err := handleReadMode(ctx, cfg, clipReader, t); err != nil {
				slog.Error("read mode", "err", err)
				notify.Error("Çeviri Hatası", err.Error())
			}
		case "reload":
			slog.Info("config reload requested — restart daemon to apply")
			notify.Info("promptTranslate", "Yapılandırma yeniden yükleme için daemonu yeniden başlatın.")
		}
	}

	listener := hotkey.NewListener(combos, handler)

	// Handle OS signals for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		listener.Stop()
	}()

	notify.Info("promptTranslate başlatıldı",
		fmt.Sprintf("Yazma: %s | Okuma: %s", cfg.Hotkeys.WriteMode, cfg.Hotkeys.ReadMode))

	if err := listener.Start(); err != nil {
		slog.Error("listener failed", "err", err)
		notify.Error("promptTranslate", "Klavye dinleyicisi başlatılamadı: "+err.Error())
		os.Exit(1)
	}
}

// handleWriteMode: open rofi → get Turkish text → translate to English → inject.
func handleWriteMode(
	ctx context.Context,
	cfg *config.Config,
	launcher *input.RofiLauncher,
	t translator.Translator,
	inj injector.Injector,
) error {
	text, err := launcher.Prompt(ctx, "Çevir (TR→EN):")
	if err != nil {
		return fmt.Errorf("girdi alınamadı: %w", err)
	}
	if text == "" {
		slog.Debug("write mode: empty input or cancelled")
		return nil // user pressed Esc
	}

	slog.Debug("write mode: translating", "text", text)
	result, err := t.Translate(ctx, text, "TR", "EN")
	if err != nil {
		return fmt.Errorf("çeviri başarısız: %w", err)
	}

	slog.Debug("write mode: injecting", "result", result)
	if err := inj.Type(ctx, result); err != nil {
		return fmt.Errorf("metin enjekte edilemedi: %w", err)
	}
	return nil
}

// handleReadMode: read primary selection → translate to Turkish → notify.
func handleReadMode(
	ctx context.Context,
	cfg *config.Config,
	clipReader *input.ClipboardReader,
	t translator.Translator,
) error {
	// Try primary selection first (mouse-highlighted text), fall back to clipboard.
	text, err := clipReader.ReadPrimary()
	if err != nil {
		slog.Debug("primary selection empty, trying clipboard", "err", err)
		text, err = clipReader.ReadClipboard()
		if err != nil {
			return fmt.Errorf("panodan metin alınamadı: %w", err)
		}
	}

	slog.Debug("read mode: translating", "text", text)
	result, err := t.Translate(ctx, text, "EN", "TR")
	if err != nil {
		return fmt.Errorf("çeviri başarısız: %w", err)
	}

	// Also copy result to clipboard so user can paste it.
	_ = clipReader.WriteClipboard(result)

	notify.Translation(text, result)
	return nil
}
