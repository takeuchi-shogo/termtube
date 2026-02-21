package storage

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config はアプリケーション全体の設定を保持する。
type Config struct {
	Player PlayerConfig `toml:"player"`
	Chat   ChatConfig   `toml:"chat"`
	Search SearchConfig `toml:"search"`
}

// PlayerConfig はプレイヤー関連の設定。
type PlayerConfig struct {
	DefaultMode   string `toml:"default_mode"`
	DefaultVolume int    `toml:"default_volume"`
	VideoOutput   string `toml:"video_output"`
}

// ChatConfig はチャット関連の設定。
type ChatConfig struct {
	Enabled  bool `toml:"enabled"`
	MaxLines int  `toml:"max_lines"`
}

// SearchConfig は検索関連の設定。
type SearchConfig struct {
	ResultsPerPage int    `toml:"results_per_page"`
	Region         string `toml:"region"`
}

// DefaultConfig はデフォルト設定を返す。
func DefaultConfig() Config {
	return Config{
		Player: PlayerConfig{
			DefaultMode:   "focus",
			DefaultVolume: 80,
			VideoOutput:   "kitty",
		},
		Chat: ChatConfig{
			Enabled:  true,
			MaxLines: 50,
		},
		Search: SearchConfig{
			ResultsPerPage: 20,
			Region:         "JP",
		},
	}
}

// SaveConfig は設定を TOML 形式でファイルに保存する。
func SaveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(cfg); err != nil {
		return err
	}

	return atomicWriteFile(path, buf.Bytes(), 0o644)
}

// LoadConfig は TOML ファイルから設定を読み込む。
// ファイルが存在しない場合はデフォルト設定を作成して返す。
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// ファイルが存在しない場合はデフォルトを保存して返す
		if err := SaveConfig(path, cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
