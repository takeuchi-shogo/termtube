package youtube

import (
	"testing"
)

func TestParseMetadata(t *testing.T) {
	input := []byte(`{"id":"xyz789","title":"Metadata Test","channel":"Meta Channel","channel_id":"UC_META","duration":600,"duration_string":"10:00","view_count":50000,"webpage_url":"https://www.youtube.com/watch?v=xyz789","thumbnail":"https://i.ytimg.com/vi/xyz789/maxresdefault.jpg","is_live":false}`)

	video, err := ParseMetadata(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if video.ID != "xyz789" {
		t.Errorf("ID = %q, want %q", video.ID, "xyz789")
	}
	if video.Title != "Metadata Test" {
		t.Errorf("Title = %q, want %q", video.Title, "Metadata Test")
	}
	if video.Channel != "Meta Channel" {
		t.Errorf("Channel = %q, want %q", video.Channel, "Meta Channel")
	}
	if video.ChannelID != "UC_META" {
		t.Errorf("ChannelID = %q, want %q", video.ChannelID, "UC_META")
	}
	if video.Duration != 600 {
		t.Errorf("Duration = %v, want %v", video.Duration, 600.0)
	}
	if video.DurationStr != "10:00" {
		t.Errorf("DurationStr = %q, want %q", video.DurationStr, "10:00")
	}
	if video.ViewCount != 50000 {
		t.Errorf("ViewCount = %d, want %d", video.ViewCount, 50000)
	}
	if video.URL != "https://www.youtube.com/watch?v=xyz789" {
		t.Errorf("URL = %q, want %q", video.URL, "https://www.youtube.com/watch?v=xyz789")
	}
	if video.Thumbnail != "https://i.ytimg.com/vi/xyz789/maxresdefault.jpg" {
		t.Errorf("Thumbnail = %q, want %q", video.Thumbnail, "https://i.ytimg.com/vi/xyz789/maxresdefault.jpg")
	}
	if video.IsLive != false {
		t.Errorf("IsLive = %v, want %v", video.IsLive, false)
	}
}

func TestParseMetadataInvalidJSON(t *testing.T) {
	_, err := ParseMetadata([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
