package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrPlaylistNotFound はプレイリストが見つからない場合のエラー。
var ErrPlaylistNotFound = errors.New("playlist not found")

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
	mu  sync.Mutex
}

// NewPlaylistStore は新しい PlaylistStore を作成する。
func NewPlaylistStore(dir string) *PlaylistStore {
	return &PlaylistStore{dir: dir}
}

// filenameForName はプレイリスト名からファイル名を生成する。
// パス走査攻撃（../ 等）を防止する。
func (s *PlaylistStore) filenameForName(name string) string {
	// スペースやスラッシュをアンダースコアに変換して安全なファイル名にする
	safe := strings.ReplaceAll(name, " ", "_")
	safe = strings.ReplaceAll(safe, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	safe = strings.ReplaceAll(safe, "..", "_")
	// filepath.Base でディレクトリ成分を除去
	safe = filepath.Base(safe)
	if safe == "." || safe == "" {
		safe = "unnamed"
	}
	return filepath.Join(s.dir, safe+".json")
}

// Save はプレイリストを JSON ファイルにアトミックに保存する。
func (s *PlaylistStore) Save(pl Playlist) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return err
	}

	return atomicWriteFile(s.filenameForName(pl.Name), data, 0o644)
}

// Load は指定された名前のプレイリストを読み込む。
// ファイルが存在しない場合は ErrPlaylistNotFound を返す。
func (s *PlaylistStore) Load(name string) (Playlist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked(name)
}

// loadLocked は mutex 取得済みの状態でプレイリストを読み込む。
func (s *PlaylistStore) loadLocked(name string) (Playlist, error) {
	path := s.filenameForName(name)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Playlist{}, fmt.Errorf("%w: %s", ErrPlaylistNotFound, name)
		}
		return Playlist{}, fmt.Errorf("failed to read playlist %q: %w", name, err)
	}

	var pl Playlist
	if err := json.Unmarshal(data, &pl); err != nil {
		return Playlist{}, fmt.Errorf("failed to parse playlist %q: %w", name, err)
	}
	return pl, nil
}

// List は保存されている全プレイリストの名前一覧を返す。
func (s *PlaylistStore) List() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
// 既に同じ VideoID の動画が存在する場合はスキップする。
func (s *PlaylistStore) AddVideo(name string, video PlaylistVideo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pl, err := s.loadLocked(name)
	if err != nil {
		return err
	}

	// 重複チェック
	for _, v := range pl.Videos {
		if v.VideoID == video.VideoID {
			return nil // 既に存在するのでスキップ
		}
	}

	pl.Videos = append(pl.Videos, video)

	writeData, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return err
	}

	return atomicWriteFile(s.filenameForName(name), writeData, 0o644)
}
