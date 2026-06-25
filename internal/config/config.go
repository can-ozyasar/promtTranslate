package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the root configuration structure.
type Config struct {
	General    GeneralConfig    `toml:"general"`
	Hotkeys    HotkeysConfig    `toml:"hotkeys"`
	Display    DisplayConfig    `toml:"display"`
	Translator TranslatorConfig `toml:"translator"`
	Groq       GroqConfig       `toml:"groq"`
	DeepL      DeepLConfig      `toml:"deepl"`
	Injection  InjectionConfig  `toml:"injection"`
}

type GeneralConfig struct {
	LogLevel string `toml:"log_level"`
	LogFile  string `toml:"log_file"`
}

type HotkeysConfig struct {
	WriteMode string `toml:"write_mode"`
	ReadMode  string `toml:"read_mode"`
	Reload    string `toml:"reload"`
}

type DisplayConfig struct {
	Launcher string `toml:"launcher"`
	Theme    string `toml:"theme"`
}

type TranslatorConfig struct {
	Provider   string `toml:"provider"`
	CacheSize  int    `toml:"cache_size"`
	TimeoutSec int    `toml:"timeout_sec"`
	MaxRetries int    `toml:"max_retries"`
}

type GroqConfig struct {
	APIKey  string `toml:"api_key"`
	Model   string `toml:"model"`
	BaseURL string `toml:"base_url"`
}

type DeepLConfig struct {
	APIKey string `toml:"api_key"`
	Pro    bool   `toml:"pro"`
}

type InjectionConfig struct {
	KeystrokeDelayMS int `toml:"keystroke_delay_ms"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			LogLevel: "info",
			LogFile:  "",
		},
		Hotkeys: HotkeysConfig{
			WriteMode: "alt+space",
			ReadMode:  "alt+shift+space",
			Reload:    "alt+shift+c",
		},
		Display: DisplayConfig{
			Launcher: "rofi",
			Theme:    "prompttranslate",
		},
		Translator: TranslatorConfig{
			Provider:   "groq",
			CacheSize:  50,
			TimeoutSec: 10,
			MaxRetries: 3,
		},
		Groq: GroqConfig{
			Model:   "llama-3.1-8b-instant",
			BaseURL: "https://api.groq.com/openai/v1",
		},
		DeepL: DeepLConfig{
			Pro: false,
		},
		Injection: InjectionConfig{
			KeystrokeDelayMS: 12,
		},
	}
}

// ConfigPath returns the path to the config file, honouring XDG_CONFIG_HOME.
func ConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "prompttranslate", "config.toml")
}

// Load reads the config file from the XDG location, merging it with
// defaults. If the file does not exist the defaults are returned unchanged.
func Load() (*Config, error) {
	cfg := DefaultConfig()
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil // no config file → use defaults
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	// Prefer environment variables over file values.
	if v := os.Getenv("GROQ_API_KEY"); v != "" {
		cfg.Groq.APIKey = v
	}
	if v := os.Getenv("DEEPL_API_KEY"); v != "" {
		cfg.DeepL.APIKey = v
	}

	return cfg, nil
}

// Validate checks that the mandatory fields are present given the chosen provider.
func (c *Config) Validate() error {
	switch c.Translator.Provider {
	case "groq":
		if c.Groq.APIKey == "" {
			return errors.New("config: groq.api_key is required (or set GROQ_API_KEY)")
		}
	case "deepl":
		if c.DeepL.APIKey == "" {
			return errors.New("config: deepl.api_key is required (or set DEEPL_API_KEY)")
		}
	default:
		return fmt.Errorf("config: unknown provider %q (must be groq or deepl)", c.Translator.Provider)
	}
	return nil
}
