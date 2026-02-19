package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PlaylistVideo はプレイリスト内の1動画を表す。
type PlaylistVideo struct {
	VideoID string `json:"video_id"`
	Title   string `json:"title"`
	Channel string `json:"channel"`
}

// Playlist はプレイリストを表す。
type Playlist struct {
	Name   string          `json:"name"`
	Videos []PlaylistVideo `json:"videos"`
}

// PlaylistStore はプレイリストのファイルシステムバックエンドを管理する。
// 各プレイリストは dir 内の個別の JSON ファイルとして保存される。
type PlaylistStore struct {
	dir string
}

// NewPlaylistStore は新しい PlaylistStore を作成する。
func NewPlaylistStore(dir string) *PlaylistStore {
	return &PlaylistStore{dir: dir}
}

// filenameForName はプレイリスト名からファイル名を生成する。
func (s *PlaylistStore) filenameForName(name string) string {
	// スペースやスラッシュをアンダースコアに変換して安全なファイル名にする
	safe := strings.ReplaceAll(name, " ", "_")
	safe = strings.ReplaceAll(safe, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	return filepath.Join(s.dir, safe+".json")
}

// Save はプレイリストを JSON ファイルに保存する。
func (s *PlaylistStore) Save(pl Playlist) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filenameForName(pl.Name), data, 0o644)
}

// Load は指定された名前のプレイリストを読み込む。
func (s *PlaylistStore) Load(name string) (Playlist, error) {
	path := s.filenameForName(name)

	data, err := os.ReadFile(path)
	if err != nil {
		return Playlist{}, fmt.Errorf("playlist %q not found: %w", name, err)
	}

	var pl Playlist
	if err := json.Unmarshal(data, &pl); err != nil {
		return Playlist{}, err
	}
	return pl, nil
}

// List は保存されている全プレイリストの名前一覧を返す。
func (s *PlaylistStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// ファイルを読み込んでプレイリスト名を取得
		path := filepath.Join(s.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var pl Playlist
		if err := json.Unmarshal(data, &pl); err != nil {
			continue
		}
		names = append(names, pl.Name)
	}

	return names, nil
}

// AddVideo はプレイリストに動画を追加する。
func (s *PlaylistStore) AddVideo(name string, video PlaylistVideo) error {
	pl, err := s.Load(name)
	if err != nil {
		return err
	}

	pl.Videos = append(pl.Videos, video)
	return s.Save(pl)
}
