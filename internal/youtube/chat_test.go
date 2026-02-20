package youtube

import (
	"testing"
)

func TestParseChatMessage(t *testing.T) {
	jsonData := `{"author":"user1","message":"こんにちは","timestamp":1708300000}`
	msg, err := ParseChatMessage([]byte(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Author != "user1" {
		t.Errorf("expected user1, got %s", msg.Author)
	}
	if msg.Message != "こんにちは" {
		t.Errorf("expected こんにちは, got %s", msg.Message)
	}
	if msg.Timestamp != 1708300000 {
		t.Errorf("expected 1708300000, got %d", msg.Timestamp)
	}
}

func TestParseChatMessageInvalidJSON(t *testing.T) {
	_, err := ParseChatMessage([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseChatMessageEmptyFields(t *testing.T) {
	jsonData := `{"author":"","message":"","timestamp":0}`
	msg, err := ParseChatMessage([]byte(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Author != "" {
		t.Errorf("expected empty author, got %s", msg.Author)
	}
	if msg.Message != "" {
		t.Errorf("expected empty message, got %s", msg.Message)
	}
	if msg.Timestamp != 0 {
		t.Errorf("expected 0 timestamp, got %d", msg.Timestamp)
	}
}
