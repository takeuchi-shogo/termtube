package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

// MpvCommand は mpv の JSON IPC へ送信するコマンドを表す。
type MpvCommand struct {
	Command   []interface{} `json:"command"`
	RequestID int           `json:"request_id"`
}

// MpvResponse は mpv の JSON IPC から返されるレスポンスを表す。
type MpvResponse struct {
	Error     string      `json:"error"`
	Data      interface{} `json:"data"`
	RequestID int         `json:"request_id"`
}

// MpvEvent は mpv の JSON IPC から送られるイベントを表す。
type MpvEvent struct {
	Event string `json:"event"`
}

// MpvPlayer は mpv プロセスを子プロセスとして管理し、
// JSON IPC over UNIX ソケットで制御する。
type MpvPlayer struct {
	socketPath    string
	conn          net.Conn
	cmd           *exec.Cmd
	state         PlayerState
	mu            sync.RWMutex
	onStateChange func(PlayerState)
	requestID     int
	done          chan struct{}
	doneOnce      sync.Once
	videoOutput   string
}

// NewMpvPlayer は新しい MpvPlayer を生成する。
// ソケットパスは /tmp/termtube-mpv-{pid}.sock、デフォルト音量は 80。
func NewMpvPlayer() *MpvPlayer {
	return &MpvPlayer{
		socketPath:  fmt.Sprintf("/tmp/termtube-mpv-%d.sock", os.Getpid()),
		videoOutput: "kitty",
		state: PlayerState{
			State:  StateStopped,
			Volume: 80,
		},
		done: make(chan struct{}),
	}
}

// SetVideoOutput は mpv の映像出力方式を設定する（null, kitty, tct 等）。
func (p *MpvPlayer) SetVideoOutput(vo string) {
	p.videoOutput = vo
}

// IsTerminalVO はターミナル内映像出力（kitty, tct, sixel）かどうかを返す。
func (p *MpvPlayer) IsTerminalVO() bool {
	switch p.videoOutput {
	case "kitty", "tct", "sixel":
		return true
	default:
		return false
	}
}

// BuildForegroundCmd はターミナル VO 用のフォアグラウンド再生コマンドを構築する。
// IPC ソケット（--input-ipc-server）と --no-terminal は使用しない。
// tea.ExecProcess 経由で Bubble Tea が mpv に stdin/stdout/stderr を直接渡すため、
// mpv がターミナルを全面制御し、キーボード入力・映像出力を直接処理する。
func (p *MpvPlayer) BuildForegroundCmd(url string) *exec.Cmd {
	return exec.Command("mpv",
		fmt.Sprintf("--vo=%s", p.videoOutput),
		fmt.Sprintf("--volume=%d", p.state.Volume),
		"--really-quiet",
		"--", url,
	)
}

// nextRequestID はスレッドセーフに requestID をインクリメントして返す。
func (p *MpvPlayer) nextRequestID() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requestID++
	return p.requestID
}

// Play は指定された URL の再生を開始する。
// 既に再生中であれば先に停止してから新しく mpv プロセスを起動する。
func (p *MpvPlayer) Play(url string) error {
	// 既存プロセスがあれば停止
	p.Stop()

	p.mu.Lock()
	p.done = make(chan struct{})
	p.doneOnce = sync.Once{}
	p.mu.Unlock()

	// mpv プロセスを起動
	cmd := exec.Command("mpv",
		fmt.Sprintf("--vo=%s", p.videoOutput),
		"--no-terminal",
		fmt.Sprintf("--input-ipc-server=%s", p.socketPath),
		fmt.Sprintf("--volume=%d", p.state.Volume),
		"--", url,
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mpv の起動に失敗しました: %w", err)
	}

	p.mu.Lock()
	p.cmd = cmd
	p.mu.Unlock()

	// ソケット接続を待つ (最大 50 回、100ms 間隔)
	var conn net.Conn
	var err error
	for i := 0; i < 50; i++ {
		conn, err = net.Dial("unix", p.socketPath)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		p.Stop()
		return fmt.Errorf("mpv ソケットへの接続に失敗しました: %w", err)
	}

	p.mu.Lock()
	p.conn = conn
	p.state.State = StatePlaying
	p.mu.Unlock()

	// イベントリスナーを開始
	go p.listenEvents()

	p.notifyStateChange()

	return nil
}

// sendCommand は mpv にコマンドを送信する。
func (p *MpvPlayer) sendCommand(cmd MpvCommand) error {
	p.mu.RLock()
	conn := p.conn
	p.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("mpv に接続されていません")
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("コマンドのシリアライズに失敗しました: %w", err)
	}

	data = append(data, '\n')
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("コマンドの送信に失敗しました: %w", err)
	}

	return nil
}

// TogglePause は再生/一時停止を切り替える。
func (p *MpvPlayer) TogglePause() error {
	id := p.nextRequestID()
	return p.sendCommand(MpvCommand{
		Command:   []interface{}{"cycle", "pause"},
		RequestID: id,
	})
}

// Seek は現在位置から指定秒数シークする。
func (p *MpvPlayer) Seek(seconds float64) error {
	id := p.nextRequestID()
	return p.sendCommand(MpvCommand{
		Command:   []interface{}{"seek", seconds, "relative"},
		RequestID: id,
	})
}

// SetVolume は音量を設定する。
func (p *MpvPlayer) SetVolume(vol int) error {
	p.mu.Lock()
	p.state.Volume = vol
	p.mu.Unlock()

	id := p.nextRequestID()
	return p.sendCommand(MpvCommand{
		Command:   []interface{}{"set_property", "volume", vol},
		RequestID: id,
	})
}

// GetState は現在のプレイヤー状態を返す。
func (p *MpvPlayer) GetState() PlayerState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// Stop は mpv プロセスを停止し、リソースをクリーンアップする。
func (p *MpvPlayer) Stop() {
	// conn と cmd をロック下で取得し、フィールドを nil にする
	p.mu.Lock()
	conn := p.conn
	cmd := p.cmd
	p.conn = nil
	p.cmd = nil
	p.mu.Unlock()

	// quit コマンドを送信（接続がある場合）
	if conn != nil {
		data, _ := json.Marshal(MpvCommand{
			Command:   []interface{}{"quit"},
			RequestID: 0,
		})
		data = append(data, '\n')
		_, _ = conn.Write(data)
	}

	// done チャネルを閉じて listenEvents を停止（sync.Once で二重 close を防止）
	p.doneOnce.Do(func() {
		close(p.done)
	})

	// コネクションを閉じる
	if conn != nil {
		_ = conn.Close()
	}

	// プロセスを終了
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}

	// ソケットファイルを削除
	_ = os.Remove(p.socketPath)

	p.mu.Lock()
	p.state.State = StateStopped
	p.mu.Unlock()
}

// OnStateChange は状態変更時に呼ばれるコールバックを設定する。
func (p *MpvPlayer) OnStateChange(fn func(PlayerState)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onStateChange = fn
}

// listenEvents は mpv ソケットからイベントを読み取り処理する。
// 同時に定期的に再生位置・再生時間をポーリングして更新する。
func (p *MpvPlayer) listenEvents() {
	p.mu.RLock()
	conn := p.conn
	p.mu.RUnlock()
	if conn == nil {
		return
	}

	// property を observe して自動通知を受け取る
	p.observeProperty(1, "time-pos")
	p.observeProperty(2, "duration")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		select {
		case <-p.done:
			return
		default:
		}

		line := scanner.Bytes()

		// property-change イベントを処理
		var propChange struct {
			Event string      `json:"event"`
			ID    int         `json:"id"`
			Name  string      `json:"name"`
			Data  interface{} `json:"data"`
		}
		if err := json.Unmarshal(line, &propChange); err == nil {
			if propChange.Event == "property-change" {
				p.handlePropertyChange(propChange.Name, propChange.Data)
				continue
			}
		}

		// 通常のイベントを処理
		var event MpvEvent
		if err := json.Unmarshal(line, &event); err == nil && event.Event != "" {
			p.handleEvent(event)
		}
	}
}

// observeProperty は mpv に property の変更通知を登録する。
func (p *MpvPlayer) observeProperty(id int, name string) {
	_ = p.sendCommand(MpvCommand{
		Command:   []interface{}{"observe_property", id, name},
		RequestID: p.nextRequestID(),
	})
}

// handlePropertyChange は mpv の property-change イベントを処理する。
func (p *MpvPlayer) handlePropertyChange(name string, data interface{}) {
	if data == nil {
		return
	}
	val, ok := data.(float64)
	if !ok {
		return
	}

	p.mu.Lock()
	switch name {
	case "time-pos":
		p.state.Position = val
	case "duration":
		p.state.Duration = val
	}
	p.mu.Unlock()

	p.notifyStateChange()
}

// handleEvent は mpv イベントに応じてプレイヤー状態を更新する。
func (p *MpvPlayer) handleEvent(event MpvEvent) {
	p.mu.Lock()
	switch event.Event {
	case "pause":
		p.state.State = StatePaused
	case "unpause":
		p.state.State = StatePlaying
	case "end-file":
		p.state.State = StateStopped
	}
	p.mu.Unlock()

	p.notifyStateChange()
}

// notifyStateChange は登録されたコールバックに状態変更を通知する。
func (p *MpvPlayer) notifyStateChange() {
	p.mu.RLock()
	fn := p.onStateChange
	state := p.state
	p.mu.RUnlock()

	if fn != nil {
		fn(state)
	}
}
