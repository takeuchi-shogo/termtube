package storage

import (
	"testing"
)

func TestCreateAndLoadPlaylist(t *testing.T) {
	dir := t.TempDir()
	store := NewPlaylistStore(dir)

	pl := Playlist{
		Name: "My Favorites",
		Videos: []PlaylistVideo{
			{VideoID: "vid1", Title: "Video 1", Channel: "Ch1"},
			{VideoID: "vid2", Title: "Video 2", Channel: "Ch2"},
		},
	}

	if err := store.Save(pl); err != nil {
		t.Fatalf("failed to create playlist: %v", err)
	}

	loaded, err := store.Load("My Favorites")
	if err != nil {
		t.Fatalf("failed to load playlist: %v", err)
	}

	if loaded.Name != "My Favorites" {
		t.Errorf("expected name 'My Favorites', got %q", loaded.Name)
	}
	if len(loaded.Videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(loaded.Videos))
	}
	if loaded.Videos[0].VideoID != "vid1" {
		t.Errorf("expected first video 'vid1', got %q", loaded.Videos[0].VideoID)
	}
	if loaded.Videos[1].Title != "Video 2" {
		t.Errorf("expected second video title 'Video 2', got %q", loaded.Videos[1].Title)
	}
}

func TestListPlaylists(t *testing.T) {
	dir := t.TempDir()
	store := NewPlaylistStore(dir)

	pl1 := Playlist{
		Name:   "Playlist A",
		Videos: []PlaylistVideo{},
	}
	pl2 := Playlist{
		Name:   "Playlist B",
		Videos: []PlaylistVideo{},
	}

	if err := store.Save(pl1); err != nil {
		t.Fatalf("failed to save playlist A: %v", err)
	}
	if err := store.Save(pl2); err != nil {
		t.Fatalf("failed to save playlist B: %v", err)
	}

	names, err := store.List()
	if err != nil {
		t.Fatalf("failed to list playlists: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 playlists, got %d", len(names))
	}

	// Check both names exist (order may vary)
	found := map[string]bool{}
	for _, name := range names {
		found[name] = true
	}
	if !found["Playlist A"] {
		t.Error("expected 'Playlist A' in list")
	}
	if !found["Playlist B"] {
		t.Error("expected 'Playlist B' in list")
	}
}

func TestAddVideoToPlaylist(t *testing.T) {
	dir := t.TempDir()
	store := NewPlaylistStore(dir)

	pl := Playlist{
		Name:   "Empty PL",
		Videos: []PlaylistVideo{},
	}
	if err := store.Save(pl); err != nil {
		t.Fatalf("failed to save empty playlist: %v", err)
	}

	video := PlaylistVideo{
		VideoID: "newvid",
		Title:   "New Video",
		Channel: "New Channel",
	}
	if err := store.AddVideo("Empty PL", video); err != nil {
		t.Fatalf("failed to add video: %v", err)
	}

	loaded, err := store.Load("Empty PL")
	if err != nil {
		t.Fatalf("failed to load playlist: %v", err)
	}

	if len(loaded.Videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(loaded.Videos))
	}
	if loaded.Videos[0].VideoID != "newvid" {
		t.Errorf("expected video ID 'newvid', got %q", loaded.Videos[0].VideoID)
	}
	if loaded.Videos[0].Title != "New Video" {
		t.Errorf("expected title 'New Video', got %q", loaded.Videos[0].Title)
	}
}
