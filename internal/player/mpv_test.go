package player_test

import (
	"encoding/json"
	"testing"

	"github.com/takeuchishougo/termtube/internal/player"
)

func TestMpvCommandJSON(t *testing.T) {
	cmd := player.MpvCommand{
		Command:   []interface{}{"set_property", "pause", true},
		RequestID: 1,
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"command":["set_property","pause",true],"request_id":1}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestParseMpvEvent(t *testing.T) {
	eventJSON := `{"event":"playback-restart"}`
	var event player.MpvEvent
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		t.Fatal(err)
	}
	if event.Event != "playback-restart" {
		t.Errorf("expected playback-restart, got %s", event.Event)
	}
}
