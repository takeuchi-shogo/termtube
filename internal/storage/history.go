package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const maxHistoryEntries = 100

// HistoryEntry は視聴履歴の1エントリを表す。
type HistoryEntry struct {
	VideoID   string    `json:"video_id"`
	Title     string    `json:"title"`
	Channel   string    `json:"channel"`
	WatchedAt time.Time `json:"watched_at"`
	Duration  int       `json:"duration"`
	Position  int       `json:"position"`
}

// HistoryStore は視聴履歴の JSON ファイルバックエンドを管理する。
type HistoryStore struct {
	path string
}

// NewHistoryStore は新しい HistoryStore を作成する。
func NewHistoryStore(path string) *HistoryStore {
	return &HistoryStore{path: path}
}

// Load は履歴ファイルからエントリを読み込む。
// ファイルが存在しない場合は空スライスを返す。
func (s *HistoryStore) Load() ([]HistoryEntry, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, err
	}

	var entries []HistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// Add は履歴にエントリを追加する。
// 同じ video_id が既に存在する場合は古いエントリを削除して先頭に新しいエントリを追加する。
// 最大 100 件まで保持し、超過分は古い順に削除される。
func (s *HistoryStore) Add(entry HistoryEntry) error {
	entries, err := s.Load()
	if err != nil {
		return err
	}

	// 重複を除去
	filtered := make([]HistoryEntry, 0, len(entries))
	for _, e := range entries {
		if e.VideoID != entry.VideoID {
			filtered = append(filtered, e)
		}
	}

	// 先頭に追加（newest first）
	entries = append([]HistoryEntry{entry}, filtered...)

	// 最大件数に制限
	if len(entries) > maxHistoryEntries {
		entries = entries[:maxHistoryEntries]
	}

	return s.save(entries)
}

func (s *HistoryStore) save(entries []HistoryEntry) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o644)
}
