package config

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Database DatabaseConfig `toml:"database"`
	HTTP     HTTPConfig     `toml:"http"`
	Log      LogConfig      `toml:"log"`
	Paste    PasteConfig    `toml:"paste"`
}

type LogConfig struct {
	Level string `toml:"level"`
}

type DatabaseConfig struct {
	Path string `toml:"path"`
}

type HTTPConfig struct {
	BaseURL        string  `toml:"base_url"`
	ClientIPHeader *string `toml:"client_ip_header"`
}

type PasteConfig struct {
	IDLength int                 `toml:"id_length"`
	Defaults PasteDefaultsConfig `toml:"defaults"`
}

type PasteDefaultsConfig struct {
	Anonymous     PasteTierConfig `toml:"anonymous"`
	Authenticated PasteTierConfig `toml:"authenticated"`
}

type PasteTierConfig struct {
	ExpiryLength Duration `toml:"expiry_length"`
	MaxSize      ByteSize `toml:"max_size"`
}

func defaults() *Config {
	return &Config{
		Database: DatabaseConfig{
			Path: "data/yaps.db",
		},
		HTTP: HTTPConfig{
			BaseURL:        "http://localhost:3000",
			ClientIPHeader: nil,
		},
		Log: LogConfig{
			Level: "info",
		},
		Paste: PasteConfig{
			IDLength: 10,
			Defaults: PasteDefaultsConfig{
				Anonymous: PasteTierConfig{
					ExpiryLength: mustParseDuration("720h"),
					MaxSize:      mustParseByteSize("512KB"),
				},
				Authenticated: PasteTierConfig{
					ExpiryLength: mustParseDuration("8765h"),
					MaxSize:      mustParseByteSize("2MB"),
				},
			},
		},
	}
}

func (c *Config) Validate() error {
	if c.Database.Path == "" {
		return fmt.Errorf("database.path must not be empty")
	}
	if c.HTTP.BaseURL == "" {
		return fmt.Errorf("http.base_url must not be empty")
	}
	if c.Paste.IDLength <= 0 {
		return fmt.Errorf("paste.id_length must be greater than zero, got %d", c.Paste.IDLength)
	}
	for name, tier := range map[string]PasteTierConfig{
		"anonymous":     c.Paste.Defaults.Anonymous,
		"authenticated": c.Paste.Defaults.Authenticated,
	} {
		if tier.MaxSize <= 0 {
			return fmt.Errorf("paste.defaults.%s.max_size must be greater than zero", name)
		}
		if tier.ExpiryLength <= 0 {
			return fmt.Errorf("paste.defaults.%s.expiry_length must be greater than zero", name)
		}
	}

	if _, err := parseLogLevel(c.Log.Level); err != nil {
		return fmt.Errorf("log.level: %w", err)
	}

	return nil
}

func (c *Config) LogLevel() slog.Level {
	lvl, err := parseLogLevel(c.Log.Level)
	if err != nil {
		return slog.LevelInfo
	}
	return lvl
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level %q (want debug|info|warn|error)", s)
	}
}

func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err = cfg.Validate(); err != nil {
				return nil, fmt.Errorf("failed to validate config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err = dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return cfg, nil
}

type Duration time.Duration

func (d *Duration) UnmarshalText(b []byte) error {
	x, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = Duration(x)
	return nil
}

func (d Duration) String() string {
	return time.Duration(d).String()
}

func mustParseDuration(s string) Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(fmt.Sprintf("invalid hardcoded duration default %q: %v", s, err))
	}
	return Duration(d)
}

type ByteSize int64

func (b *ByteSize) UnmarshalText(text []byte) error {
	n, err := humanize.ParseBytes(string(text))
	if err != nil {
		return err
	}
	*b = ByteSize(n)
	return nil
}

func (b ByteSize) String() string {
	return humanize.Bytes(uint64(b))
}

func mustParseByteSize(s string) ByteSize {
	n, err := humanize.ParseBytes(s)
	if err != nil {
		panic(fmt.Sprintf("invalid hardcoded byte size default %q: %v", s, err))
	}
	return ByteSize(n)
}
