package config

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Database DatabaseConfig `toml:"database"`
	Paste    PasteConfig    `toml:"paste"`
}

type DatabaseConfig struct {
	Path string `toml:"path"`
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
	return nil
}

func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if err != nil {
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
