package youtube

import (
	"testing"
)

func TestParseSearchResults(t *testing.T) {
	input := []byte(`{"id":"abc123","title":"Test Video 1","channel":"Channel A","channel_id":"UC_A","duration":120,"duration_string":"2:00","view_count":1000,"webpage_url":"https://www.youtube.com/watch?v=abc123","thumbnail":"https://i.ytimg.com/vi/abc123/default.jpg","is_live":false}
{"id":"def456","title":"Test Video 2","channel":"Channel B","channel_id":"UC_B","duration":300,"duration_string":"5:00","view_count":2000,"webpage_url":"https://www.youtube.com/watch?v=def456","thumbnail":"https://i.ytimg.com/vi/def456/default.jpg","is_live":true}
`)

	videos, err := ParseSearchResults(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(videos))
	}

	// Verify first video
	v1 := videos[0]
	if v1.ID != "abc123" {
		t.Errorf("video[0].ID = %q, want %q", v1.ID, "abc123")
	}
	if v1.Title != "Test Video 1" {
		t.Errorf("video[0].Title = %q, want %q", v1.Title, "Test Video 1")
	}
	if v1.Channel != "Channel A" {
		t.Errorf("video[0].Channel = %q, want %q", v1.Channel, "Channel A")
	}
	if v1.ChannelID != "UC_A" {
		t.Errorf("video[0].ChannelID = %q, want %q", v1.ChannelID, "UC_A")
	}
	if v1.Duration != 120 {
		t.Errorf("video[0].Duration = %d, want %d", v1.Duration, 120)
	}
	if v1.DurationStr != "2:00" {
		t.Errorf("video[0].DurationStr = %q, want %q", v1.DurationStr, "2:00")
	}
	if v1.ViewCount != 1000 {
		t.Errorf("video[0].ViewCount = %d, want %d", v1.ViewCount, 1000)
	}
	if v1.URL != "https://www.youtube.com/watch?v=abc123" {
		t.Errorf("video[0].URL = %q, want %q", v1.URL, "https://www.youtube.com/watch?v=abc123")
	}
	if v1.Thumbnail != "https://i.ytimg.com/vi/abc123/default.jpg" {
		t.Errorf("video[0].Thumbnail = %q, want %q", v1.Thumbnail, "https://i.ytimg.com/vi/abc123/default.jpg")
	}
	if v1.IsLive != false {
		t.Errorf("video[0].IsLive = %v, want %v", v1.IsLive, false)
	}

	// Verify second video
	v2 := videos[1]
	if v2.ID != "def456" {
		t.Errorf("video[1].ID = %q, want %q", v2.ID, "def456")
	}
	if v2.Title != "Test Video 2" {
		t.Errorf("video[1].Title = %q, want %q", v2.Title, "Test Video 2")
	}
	if v2.Channel != "Channel B" {
		t.Errorf("video[1].Channel = %q, want %q", v2.Channel, "Channel B")
	}
	if v2.ViewCount != 2000 {
		t.Errorf("video[1].ViewCount = %d, want %d", v2.ViewCount, 2000)
	}
	if v2.IsLive != true {
		t.Errorf("video[1].IsLive = %v, want %v", v2.IsLive, true)
	}
}

func TestParseSearchResultsEmpty(t *testing.T) {
	videos, err := ParseSearchResults([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 0 {
		t.Fatalf("expected 0 videos, got %d", len(videos))
	}
}
