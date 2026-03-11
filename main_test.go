package main

import (
	"errors"
	"io"
	"net"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Pure function tests ---

func TestRepeatStr(t *testing.T) {
	if got := repeatStr("x", 3); got != "xxx" {
		t.Errorf("got %q, want %q", got, "xxx")
	}
	if got := repeatStr("x", 0); got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestStationInterface(t *testing.T) {
	s := Station{name: "Test FM", url: "https://example.com/stream.mp3"}
	if s.Title() != "Test FM" {
		t.Errorf("Title() = %q", s.Title())
	}
	if s.Description() != "https://example.com/stream.mp3" {
		t.Errorf("Description() = %q", s.Description())
	}
	if s.FilterValue() != "Test FM" {
		t.Errorf("FilterValue() = %q", s.FilterValue())
	}
}

// --- initialModel ---

func TestInitialModel(t *testing.T) {
	m := initialModel()
	if m.volume != 80 {
		t.Errorf("volume = %d, want 80", m.volume)
	}
	if m.state != picking {
		t.Errorf("state = %d, want picking", m.state)
	}
	if m.paused {
		t.Error("paused should be false initially")
	}
}

// --- helpers ---

// playingModel returns a model in the playing state with an in-memory conn.
// The remote end is drained in a goroutine so writes to m.conn never block.
func playingModel(t *testing.T, vol int) model {
	t.Helper()
	// net.Pipe gives two connected conns with no OS socket file.
	c1, c2 := net.Pipe()
	// Drain c2 so ipcSend writes to c1 don't block the test.
	go func() { io.Copy(io.Discard, c2); c2.Close() }()
	t.Cleanup(func() { c1.Close() })
	l := list.New(stations, list.NewDefaultDelegate(), 40, 14)
	return model{
		state:   playing,
		list:    l,
		station: Station{name: "Nightride FM", url: "https://example.com"},
		conn:    c1,
		volume:  vol,
	}
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func specialKeyMsg(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

// --- Picking state transitions ---

func TestPickingQuit(t *testing.T) {
	m := initialModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected a quit cmd, got nil")
	}
	// Execute the cmd and check it returns tea.QuitMsg.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestPickingEnterTransitionsToPlaying(t *testing.T) {
	m := initialModel()
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := result.(model)
	if updated.state != playing {
		t.Errorf("state = %d, want playing", updated.state)
	}
	if cmd == nil {
		t.Error("expected launchMpv cmd, got nil")
	}
}

// --- Playing state transitions ---

func TestPlayingMpvStartedSuccess(t *testing.T) {
	m := playingModel(t, 80)

	c1, c2 := net.Pipe()
	go func() { io.Copy(io.Discard, c2); c2.Close() }()

	result, _ := m.Update(mpvStartedMsg{conn: c1})
	updated := result.(model)
	if updated.conn == nil {
		t.Error("conn should be set after mpvStartedMsg success")
	}
	updated.conn.Close()
}

func TestPlayingMpvStartedError(t *testing.T) {
	m := playingModel(t, 80)
	result, _ := m.Update(mpvStartedMsg{err: errors.New("mpv not found")})
	updated := result.(model)
	if updated.state != picking {
		t.Errorf("state = %d, want picking after mpv error", updated.state)
	}
}

func TestPlayingSpaceTogglesPause(t *testing.T) {
	m := playingModel(t, 80)
	result, _ := m.Update(keyMsg(" "))
	updated := result.(model)
	if !updated.paused {
		t.Error("paused should be true after first space")
	}

	result, _ = updated.Update(keyMsg(" "))
	updated = result.(model)
	if updated.paused {
		t.Error("paused should be false after second space")
	}
}

func TestPlayingVolumeUp(t *testing.T) {
	m := playingModel(t, 80)
	result, _ := m.Update(keyMsg("+"))
	if result.(model).volume != 85 {
		t.Errorf("volume = %d, want 85", result.(model).volume)
	}
}

func TestPlayingVolumeDown(t *testing.T) {
	m := playingModel(t, 80)
	result, _ := m.Update(keyMsg("-"))
	if result.(model).volume != 75 {
		t.Errorf("volume = %d, want 75", result.(model).volume)
	}
}

func TestPlayingVolumeClampMax(t *testing.T) {
	m := playingModel(t, 100)
	result, _ := m.Update(keyMsg("+"))
	if result.(model).volume != 100 {
		t.Errorf("volume = %d, want 100 (clamped)", result.(model).volume)
	}
}

func TestPlayingVolumeClampMin(t *testing.T) {
	m := playingModel(t, 0)
	result, _ := m.Update(keyMsg("-"))
	if result.(model).volume != 0 {
		t.Errorf("volume = %d, want 0 (clamped)", result.(model).volume)
	}
}

func TestPlayingBackReturnsToPicking(t *testing.T) {
	m := playingModel(t, 80)
	result, _ := m.Update(keyMsg("b"))
	updated := result.(model)
	if updated.state != picking {
		t.Errorf("state = %d, want picking after b", updated.state)
	}
}
