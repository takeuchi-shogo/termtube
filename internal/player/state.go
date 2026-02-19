package player

// State はプレイヤーの再生状態を表す列挙型。
type State int

const (
	StateStopped State = iota
	StatePlaying
	StatePaused
)

// PlayerState はプレイヤーの現在の状態をまとめた構造体。
type PlayerState struct {
	State    State
	Title    string
	Position float64
	Duration float64
	Volume   int
	VideoID  string
}
