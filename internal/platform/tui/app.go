package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/phibot/phibot/internal/bot"
)

type chatMessage struct {
	sender    string
	content   string
	isUser    bool
	timestamp time.Time
}

type streamChunkMsg struct {
	content string
}

type streamDoneMsg struct {
	fullContent string
}

type errorMsg struct {
	err error
}

type ViewMode int

const (
	ViewChat ViewMode = iota
	ViewDebug
)

type Model struct {
	bot       *bot.Bot
	viewport  viewport.Model
	textInput textinput.Model
	messages  []chatMessage
	streaming bool
	streamBuf string
	width     int
	height    int
	ready     bool
	err       error
	eventCh   chan bot.Event

	debug     DebugPanel
	viewMode  ViewMode
}

func NewModel(b *bot.Bot) Model {
	ti := textinput.New()
	ti.Placeholder = "输入消息... (Ctrl+C 退出, Tab 切换视图)"
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = 80

	vp := viewport.New(80, 20)
	vp.SetContent("")

	eventCh := make(chan bot.Event, 100)

	m := Model{
		bot:       b,
		viewport:  vp,
		textInput: ti,
		messages:  make([]chatMessage, 0),
		eventCh:   eventCh,
		debug:     NewDebugPanel(eventCh),
		viewMode:  ViewChat,
	}

	handler := func(e bot.Event) {
		select {
		case eventCh <- e:
		default:
		}
	}

	b.EventBus.Subscribe(bot.EventStreamChunk, handler)
	b.EventBus.Subscribe(bot.EventStreamDone, handler)
	b.EventBus.Subscribe(bot.EventError, handler)
	b.EventBus.Subscribe(bot.EventMessageReceive, handler)
	b.EventBus.Subscribe(bot.EventMessageSend, handler)

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.waitForEvent(),
	)
}

func (m Model) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		event := <-m.eventCh
		switch event.Type {
		case bot.EventStreamChunk:
			if content, ok := event.Data.(string); ok {
				return debugLogAndReturn(debugEntry{
					timestamp: time.Now(),
					level:     "stream",
					message:   fmt.Sprintf("chunk: %q", truncate(content, 30)),
				}, streamChunkMsg{content: content})
			}
		case bot.EventStreamDone:
			if content, ok := event.Data.(string); ok {
				return debugLogAndReturn(debugEntry{
					timestamp: time.Now(),
					level:     "event",
					message:   fmt.Sprintf("stream done, %d chars", len(content)),
				}, streamDoneMsg{fullContent: content})
			}
		case bot.EventError:
			if err, ok := event.Data.(error); ok {
				return debugLogAndReturn(debugEntry{
					timestamp: time.Now(),
					level:     "error",
					message:   err.Error(),
				}, errorMsg{err: err})
			}
		case bot.EventMessageReceive:
			if event.Message != nil {
				return debugLogAndReturn(debugEntry{
					timestamp: time.Now(),
					level:     "event",
					message:   fmt.Sprintf("recv from %s: %s", event.Message.SenderName, truncate(event.Message.Content, 30)),
				}, nil)
			}
		case bot.EventMessageSend:
			if event.Message != nil {
				return debugLogAndReturn(debugEntry{
					timestamp: time.Now(),
					level:     "event",
					message:   fmt.Sprintf("send to %s: %s", event.Message.SenderName, truncate(event.Message.Content, 30)),
				}, nil)
			}
		}
		return m.waitForEvent()()
	}
}

type debugLogAndReturnMsg struct {
	entry debugEntry
	inner tea.Msg
}

func debugLogAndReturn(entry debugEntry, inner tea.Msg) tea.Msg {
	return debugLogAndReturnMsg{entry: entry, inner: inner}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case debugLogAndReturnMsg:
		m.debug.AddEntry(msg.entry.level, msg.entry.message)
		if msg.inner != nil {
			return m.Update(msg.inner)
		}
		return m, m.waitForEvent()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.viewMode = (m.viewMode + 1) % 2
			return m, nil
		case "enter":
			if m.viewMode == ViewChat && m.textInput.Value() != "" && !m.streaming {
				content := m.textInput.Value()
				m.textInput.SetValue("")

				m.messages = append(m.messages, chatMessage{
					sender:    "You",
					content:   content,
					isUser:    true,
					timestamp: time.Now(),
				})

				m.messages = append(m.messages, chatMessage{
					sender:    m.bot.Config.Bot.Nickname,
					content:   "",
					isUser:    false,
					timestamp: time.Now(),
				})
				m.streaming = true
				m.streamBuf = ""
				m.err = nil

				m.updateViewport()

				go func() {
					m.bot.SendMessage(&bot.Message{
						ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
						SessionID:  "default",
						SenderID:   "user",
						SenderName: "You",
						Content:    content,
						Type:       bot.MessageTypeText,
						Timestamp:  time.Now(),
					})
				}()

				return m, nil
			}
		}

	case streamChunkMsg:
		m.streamBuf += msg.content
		if len(m.messages) > 0 {
			m.messages[len(m.messages)-1].content = m.streamBuf
		}
		m.updateViewport()
		return m, m.waitForEvent()

	case streamDoneMsg:
		m.streaming = false
		m.streamBuf = ""
		m.updateViewport()
		return m, m.waitForEvent()

	case errorMsg:
		m.err = msg.err
		m.streaming = false
		m.updateViewport()
		return m, m.waitForEvent()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(m.width-4, m.height-10)
			m.viewport.YPosition = 2
			m.debug.SetSize(m.width-4, m.height-6)
			m.ready = true
		} else {
			m.viewport.Width = m.width - 4
			m.viewport.Height = m.height - 10
			m.debug.SetSize(m.width-4, m.height-6)
		}

		m.textInput.Width = m.width - 4
		m.updateViewport()
	}

	if m.viewMode == ViewChat {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	m.debug.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m *Model) updateViewport() {
	var b strings.Builder

	for _, msg := range m.messages {
		if msg.isUser {
			b.WriteString(UserMessageStyle.Render(fmt.Sprintf("You: %s", msg.content)))
		} else {
			content := msg.content
			if content == "" && m.streaming {
				content = StreamingIndicatorStyle.Render("思考中...")
			}
			b.WriteString(BotMessageStyle.Render(fmt.Sprintf("%s: %s", msg.sender, content)))
		}
		b.WriteString("\n\n")
	}

	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}

func (m Model) View() string {
	if !m.ready {
		return "\n  初始化中..."
	}

	var b strings.Builder

	title := TitleStyle.Render(fmt.Sprintf(" %s ", m.bot.Config.Bot.Nickname))
	b.WriteString(title)
	b.WriteString("\n")

	var tabs string
	if m.viewMode == ViewChat {
		tabs = TabActiveStyle.Render(" 聊天 ") + TabInactiveStyle.Render(" 调试 ")
	} else {
		tabs = TabInactiveStyle.Render(" 聊天 ") + TabActiveStyle.Render(" 调试 ")
	}
	b.WriteString(tabs)
	b.WriteString("\n\n")

	if m.viewMode == ViewChat {
		b.WriteString(m.viewport.View())
		b.WriteString("\n\n")
		b.WriteString(m.textInput.View())

		if m.err != nil {
			b.WriteString("\n")
			b.WriteString(ErrorStyle.Render(fmt.Sprintf("错误: %v", m.err)))
		}
	} else {
		b.WriteString(DebugHeaderStyle.Render(" 事件流 / 调试日志 "))
		b.WriteString("\n")
		b.WriteString(m.debug.View())
	}

	return AppStyle.Render(b.String())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
