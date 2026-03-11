package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Station struct {
	name string
	url  string
}

func (s Station) Title() string       { return s.name }
func (s Station) Description() string { return s.url }
func (s Station) FilterValue() string { return s.name }

var stations = []list.Item{
	Station{"Nightride FM", "https://stream.nightride.fm/nightride.mp3"},
	Station{"Chillsynth", "https://stream.nightride.fm/chillsynth.mp3"},
	Station{"Darksynth", "https://stream.nightride.fm/darksynth.mp3"},
	Station{"Nightwave Plaza", "https://radio.plaza.one/mp3"},
}

type appState int

const (
	picking appState = iota
	playing
)

type model struct {
	state   appState
	list    list.Model
	station Station
	cmd     *exec.Cmd
	conn    net.Conn
	paused  bool
	volume  int
}

func initialModel() model {
	l := list.New(stations, list.NewDefaultDelegate(), 40, 14)
	l.Title = "solar-radio"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginLeft(2)
	return model{list: l, volume: 80}
}

func startMpv(url string) (*exec.Cmd, error) {
	cmd := exec.Command("mpv",
		"--no-video",
		"--really-quiet",
		"--no-terminal",
		"--input-ipc-server=/tmp/mpv.sock",
		url,
	)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func dialMpv() (net.Conn, error) {
	time.Sleep(300 * time.Millisecond)
	return net.Dial("unix", "/tmp/mpv.sock")
}

func ipcSend(conn net.Conn, msg string) {
	if conn != nil {
		conn.Write([]byte(msg + "\n"))
	}
}

func stopMpv(m *model) {
	if m.conn != nil {
		ipcSend(m.conn, `{"command":["quit"]}`)
		m.conn.Close()
		m.conn = nil
	}
	if m.cmd != nil {
		m.cmd.Wait()
		m.cmd = nil
	}
	os.Remove("/tmp/mpv.sock")
}

type mpvStartedMsg struct {
	cmd  *exec.Cmd
	conn net.Conn
	err  error
}

func launchMpv(url string, vol int) tea.Cmd {
	return func() tea.Msg {
		cmd, err := startMpv(url)
		if err != nil {
			return mpvStartedMsg{err: err}
		}
		conn, err := dialMpv()
		if err != nil {
			cmd.Process.Kill()
			return mpvStartedMsg{err: err}
		}
		ipcSend(conn, fmt.Sprintf(`{"command":["set_property","volume",%d]}`, vol))
		return mpvStartedMsg{cmd: cmd, conn: conn}
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case picking:
		return m.updatePicking(msg)
	case playing:
		return m.updatePlaying(msg)
	}
	return m, nil
}

func (m model) updatePicking(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			selected, ok := m.list.SelectedItem().(Station)
			if !ok {
				break
			}
			m.station = selected
			m.state = playing
			m.paused = false
			return m, launchMpv(selected.url, m.volume)
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) updatePlaying(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case mpvStartedMsg:
		if msg.err != nil {
			stopMpv(&m)
			m.state = picking
			return m, nil
		}
		m.cmd = msg.cmd
		m.conn = msg.conn
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			stopMpv(&m)
			return m, tea.Quit

		case " ":
			m.paused = !m.paused
			val := "false"
			if m.paused {
				val = "true"
			}
			ipcSend(m.conn, fmt.Sprintf(`{"command":["set_property","pause",%s]}`, val))

		case "+", "=":
			if m.volume < 100 {
				m.volume += 5
				if m.volume > 100 {
					m.volume = 100
				}
				ipcSend(m.conn, fmt.Sprintf(`{"command":["set_property","volume",%d]}`, m.volume))
			}

		case "-":
			if m.volume > 0 {
				m.volume -= 5
				if m.volume < 0 {
					m.volume = 0
				}
				ipcSend(m.conn, fmt.Sprintf(`{"command":["set_property","volume",%d]}`, m.volume))
			}

		case "b":
			stopMpv(&m)
			m.state = picking
		}
	}
	return m, nil
}

var (
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).MarginBottom(1)
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	valueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
)

func (m model) View() string {
	if m.state == picking {
		return "\n" + m.list.View()
	}

	status := "playing"
	if m.paused {
		status = "paused"
	}

	bar := fmt.Sprintf("[%s%s]", lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(
		repeatStr("█", m.volume/5)), repeatStr("░", 20-m.volume/5))

	return fmt.Sprintf("\n  %s\n\n  %s %s\n  %s %s\n  %s %s\n%s",
		titleStyle.Render("solar-radio"),
		labelStyle.Render("station:"), valueStyle.Render(m.station.name),
		labelStyle.Render("status: "), valueStyle.Render(status),
		labelStyle.Render("volume: "), bar,
		helpStyle.Render("  space: pause/resume  +/-: volume  b: back  q: quit"),
	)
}

func repeatStr(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
