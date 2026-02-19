package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Player.DefaultVolume != 80 {
		t.Errorf("expected default volume 80, got %d", cfg.Player.DefaultVolume)
	}
	if cfg.Player.DefaultMode != "focus" {
		t.Errorf("expected default mode 'focus', got %q", cfg.Player.DefaultMode)
	}
	if cfg.Player.SixelQuality != "medium" {
		t.Errorf("expected sixel quality 'medium', got %q", cfg.Player.SixelQuality)
	}
	if cfg.Chat.Enabled != true {
		t.Errorf("expected chat enabled true, got %v", cfg.Chat.Enabled)
	}
	if cfg.Chat.MaxLines != 50 {
		t.Errorf("expected chat max lines 50, got %d", cfg.Chat.MaxLines)
	}
	if cfg.Search.ResultsPerPage != 20 {
		t.Errorf("expected results per page 20, got %d", cfg.Search.ResultsPerPage)
	}
	if cfg.Search.Region != "JP" {
		t.Errorf("expected region 'JP', got %q", cfg.Search.Region)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := DefaultConfig()
	cfg.Player.DefaultVolume = 50
	cfg.Player.DefaultMode = "bgv"
	cfg.Search.Region = "US"

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Player.DefaultVolume != 50 {
		t.Errorf("expected volume 50, got %d", loaded.Player.DefaultVolume)
	}
	if loaded.Player.DefaultMode != "bgv" {
		t.Errorf("expected mode 'bgv', got %q", loaded.Player.DefaultMode)
	}
	if loaded.Search.Region != "US" {
		t.Errorf("expected region 'US', got %q", loaded.Search.Region)
	}
}

func TestLoadConfigCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.toml")

	// File should not exist yet
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected config file to not exist")
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("failed to load config from nonexistent path: %v", err)
	}

	// Should return defaults
	if cfg.Player.DefaultVolume != 80 {
		t.Errorf("expected default volume 80, got %d", cfg.Player.DefaultVolume)
	}
	if cfg.Player.DefaultMode != "focus" {
		t.Errorf("expected default mode 'focus', got %q", cfg.Player.DefaultMode)
	}
	if cfg.Search.Region != "JP" {
		t.Errorf("expected default region 'JP', got %q", cfg.Search.Region)
	}

	// File should now exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected config file to be created")
	}
}
