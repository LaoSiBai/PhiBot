package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/phibot/phibot/internal/bot"
)

type debugEntry struct {
	timestamp time.Time
	level     string
	message   string
}

type debugLogMsg struct {
	entry debugEntry
}

type DebugPanel struct {
	viewport viewport.Model
	entries  []debugEntry
	eventCh  chan bot.Event
	ready    bool
}

func NewDebugPanel(eventCh chan bot.Event) DebugPanel {
	vp := viewport.New(40, 20)
	vp.SetContent("")

	return DebugPanel{
		viewport: vp,
		entries:  make([]debugEntry, 0),
		eventCh:  eventCh,
	}
}

func (d *DebugPanel) AddEntry(level, message string) {
	entry := debugEntry{
		timestamp: time.Now(),
		level:     level,
		message:   message,
	}
	d.entries = append(d.entries, entry)
	if len(d.entries) > 500 {
		d.entries = d.entries[len(d.entries)-500:]
	}
	d.updateContent()
}

func (d *DebugPanel) updateContent() {
	var b strings.Builder
	for _, e := range d.entries {
		ts := e.timestamp.Format("15:04:05")
		var line string
		switch e.level {
		case "stream":
			line = SystemMessageStyle.Render(fmt.Sprintf("[%s] STREAM: %s", ts, e.message))
		case "error":
			line = ErrorStyle.Render(fmt.Sprintf("[%s] ERROR: %s", ts, e.message))
		case "event":
			line = BotMessageStyle.Render(fmt.Sprintf("[%s] EVENT: %s", ts, e.message))
		default:
			line = fmt.Sprintf("[%s] %s", ts, e.message)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	d.viewport.SetContent(b.String())
	d.viewport.GotoBottom()
}

func (d *DebugPanel) SetSize(w, h int) {
	d.viewport.Width = w
	d.viewport.Height = h
	d.ready = true
	d.updateContent()
}

func (d *DebugPanel) Update(msg tea.Msg) {
	d.viewport, _ = d.viewport.Update(msg)
}

func (d DebugPanel) View() string {
	return d.viewport.View()
}
