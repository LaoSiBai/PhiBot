package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/phibot/phibot/internal/bot"
)

type chatMessage struct {
	sender  string
	content string
	isUser  bool
}

type streamChunkMsg struct{ content string }
type streamDoneMsg struct{ fullContent string }
type errorMsg struct{ err error }
type tickMsg time.Time
type clearToastMsg struct{ id int }

type dialogKind int

const (
	dialogNone dialogKind = iota
	dialogExit
	dialogRestart
	dialogShutdown
)

type Model struct {
	bot *bot.Bot

	width, height  int
	mouseX, mouseY int
	mouseDown      bool

	tabIdx     int
	inputFocus int // -1: none, 0: chat, 1..6: settings

	chatVp    viewport.Model
	chatInput textinput.Model
	messages  []chatMessage
	spinner   spinner.Model
	streaming bool
	streamBuf string

	toastErr string
	toastID  int

	setInputs []textinput.Model
	setValues []string

	dialog    dialogKind
	dialogYes bool

	eventCh   chan bot.Event
	startTime time.Time
}

func NewModel(b *bot.Bot) Model {
	ti := textinput.New()
	ti.Prompt = " ✎ Ask PhiBot... "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(cDimGray)
	ti.TextStyle = lipgloss.NewStyle().Foreground(cDimGray)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(cWhite)
	ti.Placeholder = ""
	ti.CharLimit = 2000

	vp := viewport.New(60, 10)

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(cIceBlue)

	inputs := make([]textinput.Model, 6)
	values := make([]string, 6)
	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Prompt = ""
		inputs[i].TextStyle = lipgloss.NewStyle().Foreground(cDimGray)
		inputs[i].Cursor.Style = lipgloss.NewStyle().Foreground(cWhite)
	}

	values[0] = b.Config.Bot.Nickname
	values[1] = b.Config.LLM.Provider
	values[2] = b.Config.LLM.BaseURL
	values[3] = b.Config.LLM.APIKey
	values[4] = b.Config.LLM.Model
	values[5] = fmt.Sprintf("%d", b.Config.LLM.MaxTokens)

	for i, v := range values {
		inputs[i].SetValue(v)
		if i == 3 {
			inputs[i].EchoMode = textinput.EchoPassword
		}
	}

	ch := make(chan bot.Event, 100)

	m := Model{
		bot:        b,
		tabIdx:     0,
		inputFocus: -1,
		chatVp:     vp,
		chatInput:  ti,
		spinner:    sp,
		setInputs:  inputs,
		setValues:  values,
		eventCh:    ch,
		startTime:  time.Now(),
	}

	handler := func(e bot.Event) {
		select {
		case ch <- e:
		default:
		}
	}
	b.EventBus.Subscribe(bot.EventStreamChunk, handler)
	b.EventBus.Subscribe(bot.EventStreamDone, handler)
	b.EventBus.Subscribe(bot.EventError, handler)

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick, m.waitForEvent(), m.tick())
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		e := <-m.eventCh
		switch e.Type {
		case bot.EventStreamChunk:
			if c, ok := e.Data.(string); ok {
				return streamChunkMsg{content: c}
			}
		case bot.EventStreamDone:
			if c, ok := e.Data.(string); ok {
				return streamDoneMsg{fullContent: c}
			}
		case bot.EventError:
			if err, ok := e.Data.(error); ok {
				return errorMsg{err: err}
			}
		}
		return m.waitForEvent()()
	}
}

func isHit(mx, my, tx, ty, w int) bool {
	return my == ty && mx >= tx && mx < tx+w
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		chatH := m.height - 14 - 3 // total height - header - footer - input area
		if chatH < 5 {
			chatH = 5
		}
		m.chatVp.Width = m.width - 16
		m.chatVp.Height = chatH
		m.chatInput.Width = m.width - 16
		for i := range m.setInputs {
			m.setInputs[i].Width = m.width - 40
		}
		m.refreshChat()

	case tickMsg:
		return m, m.tick()

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case streamChunkMsg:
		m.streamBuf += msg.content
		if len(m.messages) > 0 {
			m.messages[len(m.messages)-1].content = m.streamBuf
		}
		m.refreshChat()
		return m, m.waitForEvent()

	case streamDoneMsg:
		m.streaming = false
		m.streamBuf = ""
		m.refreshChat()
		return m, m.waitForEvent()

	case errorMsg:
		m.toastErr = msg.err.Error()
		m.toastID++
		m.streaming = false
		m.refreshChat()
		
		tid := m.toastID
		return m, tea.Batch(
			m.waitForEvent(),
			func() tea.Msg {
				time.Sleep(3 * time.Second)
				return clearToastMsg{id: tid}
			},
		)

	case clearToastMsg:
		if m.toastID == msg.id {
			m.toastErr = ""
		}
		return m, nil

	case tea.MouseMsg:
		m.mouseX = msg.X
		m.mouseY = msg.Y

		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			m.mouseDown = true
			m.handleClick()
		} else if msg.Action == tea.MouseActionRelease {
			m.mouseDown = false
			m.syncSettings()
		} else if msg.Action == tea.MouseActionMotion && m.mouseDown {
			m.handleDrag()
		}

		if m.tabIdx == 1 {
			if msg.Button == tea.MouseButtonWheelUp {
				m.chatVp.LineUp(3)
			} else if msg.Button == tea.MouseButtonWheelDown {
				m.chatVp.LineDown(3)
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.inputFocus == -1 && msg.String() == "q" {
			return m, tea.Quit
		}

		// 只处理焦点输入框的键盘事件
		if m.inputFocus == 0 {
			if msg.String() == "enter" {
				if m.chatInput.Value() != "" && !m.streaming {
					return m.sendChat()
				}
			} else {
				m.chatInput, cmd = m.chatInput.Update(msg)
				cmds = append(cmds, cmd)
			}
		} else if m.inputFocus > 0 {
			idx := m.inputFocus - 1
			m.setInputs[idx], cmd = m.setInputs[idx].Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleClick() {
	// Tabs: Dashboard (8,6,9), Chat (21,6,10), Settings (35,6,8)
	if isHit(m.mouseX, m.mouseY, 8, 6, 9) {
		m.tabIdx = 0
		m.blurAll()
	} else if isHit(m.mouseX, m.mouseY, 21, 6, 10) {
		m.tabIdx = 1
		m.blurAll()
	} else if isHit(m.mouseX, m.mouseY, 35, 6, 8) {
		m.tabIdx = 2
		m.blurAll()
	}

	// Content areas
	if m.tabIdx == 1 {
		chatH := m.height - 14 - 3
		inputY := 10 + chatH + 1
		if m.mouseY == inputY || m.mouseY == inputY+1 {
			m.blurAll()
			m.inputFocus = 0
			m.chatInput.Focus()
			m.chatInput.PromptStyle = lipgloss.NewStyle().Foreground(cWhite)
			m.chatInput.TextStyle = lipgloss.NewStyle().Foreground(cWhite)
		} else {
			m.blurAll()
		}
	} else if m.tabIdx == 2 {
		clickedInput := false
		for i := 0; i < 7; i++ {
			y := 10 + i*2
			if i == 5 {
				// Slider click
				if isHit(m.mouseX, m.mouseY, 28, y, 20) {
					clickedInput = true
					m.blurAll()
					m.updateTempSlider()
				}
			} else {
				idx := i
				if i > 5 {
					idx = i - 1
				}
				if isHit(m.mouseX, m.mouseY, 28, y, 40) {
					clickedInput = true
					m.blurAll()
					m.inputFocus = idx + 1
					m.setInputs[idx].Focus()
					m.setInputs[idx].TextStyle = lipgloss.NewStyle().Foreground(cWhite).Bold(true)
				}
			}
		}
		// Max Tokens (index 6, Y=22)
		if isHit(m.mouseX, m.mouseY, 28, 22, 40) {
			clickedInput = true
			m.blurAll()
			m.inputFocus = 6
			m.setInputs[5].Focus()
			m.setInputs[5].TextStyle = lipgloss.NewStyle().Foreground(cWhite).Bold(true)
		}

		if !clickedInput {
			m.blurAll()
		}
	}
}

func (m *Model) handleDrag() {
	if m.tabIdx == 2 && m.mouseY == 20 { // Y=20 is Temperature slider
		if m.mouseX >= 28 && m.mouseX <= 48 {
			m.updateTempSlider()
		}
	}
}

func (m *Model) updateTempSlider() {
	val := float64(m.mouseX-28) / 20.0 * 2.0
	val = math.Max(0.0, math.Min(2.0, val))
	m.bot.Config.LLM.Temperature = val
}

func (m *Model) blurAll() {
	m.inputFocus = -1
	m.chatInput.Blur()
	m.chatInput.PromptStyle = lipgloss.NewStyle().Foreground(cDimGray)
	m.chatInput.TextStyle = lipgloss.NewStyle().Foreground(cDimGray)
	for i := range m.setInputs {
		m.setInputs[i].Blur()
		m.setInputs[i].TextStyle = lipgloss.NewStyle().Foreground(cDimGray)
	}
	m.syncSettings()
}

func (m *Model) syncSettings() {
	cfg := m.bot.Config
	if m.setInputs[0].Value() != "" {
		cfg.Bot.Nickname = m.setInputs[0].Value()
	}
	if m.setInputs[1].Value() != "" {
		cfg.LLM.Provider = m.setInputs[1].Value()
	}
	if m.setInputs[2].Value() != "" {
		cfg.LLM.BaseURL = m.setInputs[2].Value()
	}
	if m.setInputs[3].Value() != "" {
		cfg.LLM.APIKey = m.setInputs[3].Value()
	}
	if m.setInputs[4].Value() != "" {
		cfg.LLM.Model = m.setInputs[4].Value()
	}
	if m.setInputs[5].Value() != "" {
		fmt.Sscanf(m.setInputs[5].Value(), "%d", &cfg.LLM.MaxTokens)
	}
}

func (m Model) sendChat() (tea.Model, tea.Cmd) {
	content := m.chatInput.Value()
	m.chatInput.SetValue("")

	m.messages = append(m.messages,
		chatMessage{sender: "User", content: content, isUser: true},
		chatMessage{sender: m.bot.Config.Bot.Nickname, content: "", isUser: false},
	)
	m.streaming = true
	m.streamBuf = ""
	m.toastErr = ""
	m.refreshChat()

	go func() {
		m.bot.SendMessage(&bot.Message{
			ID: fmt.Sprintf("%d", time.Now().UnixNano()), SessionID: "default",
			SenderID: "user", SenderName: "User",
			Content: content, Type: bot.MessageTypeText, Timestamp: time.Now(),
		})
	}()
	return m, nil
}

func (m *Model) refreshChat() {
	var b strings.Builder
	for i, msg := range m.messages {
		if msg.isUser {
			b.WriteString(StyleChatUserPrefix.Render("User ›"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().PaddingLeft(2).Render(StyleChatUserText.Render(msg.content)))
		} else {
			b.WriteString(StyleChatBotPrefix.Render(fmt.Sprintf("%s ›", msg.sender)))
			b.WriteString("\n")
			content := msg.content
			if content == "" && m.streaming {
				content = m.spinner.View()
			} else if m.streaming && i == len(m.messages)-1 {
				content += " " + m.spinner.View()
			}
			b.WriteString(lipgloss.NewStyle().PaddingLeft(2).Render(StyleChatBotText.Render(content)))
		}
		b.WriteString("\n\n")
	}
	m.chatVp.SetContent(b.String())
	m.chatVp.GotoBottom()
}

func (m Model) View() string {
	var b strings.Builder

	// Y=0,1,2,3
	b.WriteString("\n\n\n\n")

	// Helper
	writeLine := func(s string) {
		b.WriteString(strings.Repeat(" ", 8))
		b.WriteString(s)
		b.WriteString("\n")
	}

	// Y=4
	writeLine(StyleLogo.Render("P H I B O T"))
	
	// Y=5
	writeLine("")

	// Y=6 Tabs
	var tabs []string
	
	// Dashboard
	if m.tabIdx == 0 {
		tabs = append(tabs, StyleMenuActive.Render("Dashboard"))
	} else if isHit(m.mouseX, m.mouseY, 8, 6, 9) {
		tabs = append(tabs, StyleMenuHover.Render("Dashboard"))
	} else {
		tabs = append(tabs, StyleMenuDim.Render("Dashboard"))
	}

	// Local Chat
	if m.tabIdx == 1 {
		tabs = append(tabs, StyleMenuActive.Render("Local Chat"))
	} else if isHit(m.mouseX, m.mouseY, 21, 6, 10) {
		tabs = append(tabs, StyleMenuHover.Render("Local Chat"))
	} else {
		tabs = append(tabs, StyleMenuDim.Render("Local Chat"))
	}

	// Settings
	if m.tabIdx == 2 {
		tabs = append(tabs, StyleMenuActive.Render("Settings"))
	} else if isHit(m.mouseX, m.mouseY, 35, 6, 8) {
		tabs = append(tabs, StyleMenuHover.Render("Settings"))
	} else {
		tabs = append(tabs, StyleMenuDim.Render("Settings"))
	}

	writeLine(strings.Join(tabs, "    "))

	// Y=7
	writeLine("")

	// Y=8
	dividerW := m.width - 16
	if dividerW < 0 { dividerW = 0 }
	writeLine(StyleDivider.Render(strings.Repeat("─", dividerW)))

	// Y=9
	writeLine("")

	// Y=10+ Content
	contentStr := ""
	switch m.tabIdx {
	case 0:
		contentStr = m.renderDashboard()
	case 1:
		contentStr = m.renderChat()
	case 2:
		contentStr = m.renderSettings()
	}

	for _, line := range strings.Split(contentStr, "\n") {
		writeLine(line)
	}

	// Footer Y = height - 3
	linesSoFar := strings.Count(b.String(), "\n")
	padLines := m.height - 3 - linesSoFar
	if padLines > 0 {
		b.WriteString(strings.Repeat("\n", padLines))
	}

	if m.toastErr != "" {
		writeLine(lipgloss.NewStyle().Foreground(cError).Render("⚠ " + m.toastErr))
	} else {
		writeLine(StyleFooter.Render("● Engine Active  │  🖱️ Mouse Driver Mode"))
	}

	full := b.String()

	if m.dialog != dialogNone {
		full = m.renderDialogOverlay(full)
	}

	return full
}

func (m Model) renderDialogOverlay(base string) string {
	title := "Confirmation"
	msg := ""
	switch m.dialog {
	case dialogExit:
		msg = "Exit PhiBot?"
	case dialogRestart:
		msg = "Restart Bot? History will be cleared."
	case dialogShutdown:
		msg = "Shutdown Bot?"
	}

	yesBtn := StyleValueDim.Render("  Y  ")
	noBtn := StyleValueDim.Render("  N  ")
	
	if m.dialogYes {
		yesBtn = StyleValueFocus.Render("[ Y ]")
	} else {
		noBtn = StyleValueFocus.Render("[ N ]")
	}

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cDimGray).Padding(1, 4).Render(
		lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true).Render(title) + "\n\n" +
		StyleWhite(msg) + "\n\n" +
		lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, "   ", noBtn),
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box, lipgloss.WithWhitespaceChars("░"), lipgloss.WithWhitespaceForeground(cDimGray))
}

func (m Model) renderDashboard() string {
	var b strings.Builder
	
	valStyle := func(s string) string {
		return StyleValueHover.Render(s)
	}

	b.WriteString(StyleLabel.Render("Engine") + valStyle(m.bot.Config.LLM.Provider) + "\n\n")
	b.WriteString(StyleLabel.Render("Memory") + valStyle(fmt.Sprintf("%d Tokens", m.bot.Config.LLM.MaxTokens)) + "\n\n")
	
	uptime := time.Since(m.startTime).Round(time.Second)
	b.WriteString(StyleLabel.Render("Uptime") + StyleStatusDot.Render("● ") + valStyle(fmt.Sprintf("Running (%s)", uptime)) + "\n\n\n\n")

	tps := m.bot.Stats.LastTPS
	tpsStr := fmt.Sprintf("%.1f", tps)
	if tps == 0 { tpsStr = "-.-" }
	b.WriteString(StyleLabel.Render("Tokens/Sec") + valStyle(tpsStr) + "\n")

	fillCount := int((tps / 100.0) * 15)
	if fillCount > 15 { fillCount = 15 }
	if fillCount < 0 { fillCount = 0 }
	emptyCount := 15 - fillCount
	
	bar := StyleBarFilled.Render(strings.Repeat("■", fillCount)) + StyleBarEmpty.Render(strings.Repeat("░", emptyCount))
	b.WriteString(StyleLabel.Render("") + bar)

	return b.String()
}

func (m Model) renderChat() string {
	var b strings.Builder

	b.WriteString(m.chatVp.View())
	b.WriteString("\n\n")

	b.WriteString(m.chatInput.View() + "\n")
	
	lineColor := StyleInputLine
	if m.inputFocus == 0 { lineColor = lipgloss.NewStyle().Foreground(cWhite) }
	
	divW := m.width - 16
	if divW < 0 { divW = 0 }
	b.WriteString(lineColor.Render(strings.Repeat("─", divW)))

	return b.String()
}

func (m Model) renderSettings() string {
	var b strings.Builder

	labels := []string{"Bot Nickname", "API Provider", "Base URL", "API Key", "Model", "Temperature", "Max Tokens"}

	for i, lbl := range labels {
		b.WriteString(StyleLabel.Render(lbl))

		if i == 5 { // Temperature Slider
			val := m.bot.Config.LLM.Temperature
			fillCount := int((val / 2.0) * 20)
			if fillCount > 20 { fillCount = 20 }
			if fillCount < 0 { fillCount = 0 }
			emptyCount := 20 - fillCount

			bar := StyleBarFilled.Render(strings.Repeat("■", fillCount)) + StyleBarEmpty.Render(strings.Repeat("░", emptyCount))
			
			hoverText := ""
			if isHit(m.mouseX, m.mouseY, 28, 10+i*2, 20) {
				hoverText = StyleWhite(fmt.Sprintf(" %.2f", val))
			} else {
				hoverText = StyleDimGray(fmt.Sprintf(" %.2f", val))
			}
			b.WriteString(bar + hoverText)
		} else {
			idx := i
			if i > 5 { idx = i - 1 }

			isFocus := m.inputFocus == idx+1

			if isFocus {
				b.WriteString(m.setInputs[idx].View())
			} else {
				valStr := m.setInputs[idx].Value()
				if i == 3 && len(valStr) > 6 {
					valStr = valStr[:3] + "..." + valStr[len(valStr)-3:]
				}
				if valStr == "" { valStr = "-" }
				
				isHover := isHit(m.mouseX, m.mouseY, 28, 10+i*2, 40)
				if isHover {
					b.WriteString(StyleValueHover.Render(valStr))
				} else {
					b.WriteString(StyleValueDim.Render(valStr))
				}
			}
		}
		b.WriteString("\n\n")
	}

	return b.String()
}

func StyleWhite(s string) string { return lipgloss.NewStyle().Foreground(cWhite).Render(s) }
func StyleDimGray(s string) string { return lipgloss.NewStyle().Foreground(cDimGray).Render(s) }
