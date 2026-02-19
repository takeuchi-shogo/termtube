package storage

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestAddAndLoadHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	store := NewHistoryStore(path)

	entry := HistoryEntry{
		VideoID:   "abc123",
		Title:     "Test Video",
		Channel:   "Test Channel",
		WatchedAt: time.Now(),
	}

	if err := store.Add(entry); err != nil {
		t.Fatalf("failed to add history entry: %v", err)
	}

	entries, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load history: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].VideoID != "abc123" {
		t.Errorf("expected video ID 'abc123', got %q", entries[0].VideoID)
	}
	if entries[0].Title != "Test Video" {
		t.Errorf("expected title 'Test Video', got %q", entries[0].Title)
	}
	if entries[0].Channel != "Test Channel" {
		t.Errorf("expected channel 'Test Channel', got %q", entries[0].Channel)
	}
}

func TestHistoryMaxEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	store := NewHistoryStore(path)

	for i := 0; i < 150; i++ {
		entry := HistoryEntry{
			VideoID:   fmt.Sprintf("video_%d", i),
			Title:     fmt.Sprintf("Video %d", i),
			Channel:   "Channel",
			WatchedAt: time.Now(),
		}
		if err := store.Add(entry); err != nil {
			t.Fatalf("failed to add entry %d: %v", i, err)
		}
	}

	entries, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load history: %v", err)
	}

	if len(entries) != 100 {
		t.Errorf("expected max 100 entries, got %d", len(entries))
	}

	// Newest entry should be first
	if entries[0].VideoID != "video_149" {
		t.Errorf("expected newest entry 'video_149' first, got %q", entries[0].VideoID)
	}
}

func TestHistoryDuplicateReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	store := NewHistoryStore(path)

	entry1 := HistoryEntry{
		VideoID:   "abc123",
		Title:     "Original Title",
		Channel:   "Channel",
		WatchedAt: time.Now().Add(-1 * time.Hour),
	}
	if err := store.Add(entry1); err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	entry2 := HistoryEntry{
		VideoID:   "other_video",
		Title:     "Other Video",
		Channel:   "Channel",
		WatchedAt: time.Now().Add(-30 * time.Minute),
	}
	if err := store.Add(entry2); err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	// Re-add same video with updated title
	entry3 := HistoryEntry{
		VideoID:   "abc123",
		Title:     "Updated Title",
		Channel:   "Channel",
		WatchedAt: time.Now(),
	}
	if err := store.Add(entry3); err != nil {
		t.Fatalf("failed to add duplicate entry: %v", err)
	}

	entries, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load history: %v", err)
	}

	// Should still be 2 entries (duplicate replaced)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries after duplicate, got %d", len(entries))
	}

	// Newest (re-added) should be first with updated title
	if entries[0].VideoID != "abc123" {
		t.Errorf("expected 'abc123' first, got %q", entries[0].VideoID)
	}
	if entries[0].Title != "Updated Title" {
		t.Errorf("expected 'Updated Title', got %q", entries[0].Title)
	}
}
