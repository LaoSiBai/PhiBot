package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Bot   BotConfig   `toml:"bot"`
	LLM   LLMConfig   `toml:"llm"`
	TUI   TUIConfig   `toml:"tui"`
}

type BotConfig struct {
	Nickname string `toml:"nickname"`
	DataDir  string `toml:"data_dir"`
}

type LLMConfig struct {
	Provider    string  `toml:"provider"`
	BaseURL     string  `toml:"base_url"`
	APIKey      string  `toml:"api_key"`
	Model       string  `toml:"model"`
	Temperature float64 `toml:"temperature"`
	MaxTokens   int     `toml:"max_tokens"`
}

type TUIConfig struct {
	Theme string `toml:"theme"`
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Bot.DataDir == "" {
		cfg.Bot.DataDir = filepath.Dir(path)
	}

	return cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Bot: BotConfig{
			Nickname: "PhiBot",
			DataDir:  ".",
		},
		LLM: LLMConfig{
			Provider:    "openai",
			BaseURL:     "https://api.openai.com/v1",
			Model:       "gpt-4o-mini",
			Temperature: 0.7,
			MaxTokens:   2048,
		},
		TUI: TUIConfig{
			Theme: "default",
		},
	}
}
