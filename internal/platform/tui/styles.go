package tui

import "github.com/charmbracelet/lipgloss"

var (
	cWhite   = lipgloss.Color("#FFFFFF")
	cDimGray = lipgloss.Color("#444444")
	cIceBlue = lipgloss.Color("#89DCEB")
	cGreen   = lipgloss.Color("#A6E3A1")
	cError   = lipgloss.Color("#EF4444")

	StyleLogo = lipgloss.NewStyle().Foreground(cWhite).Bold(true)

	StyleMenuHover  = lipgloss.NewStyle().Foreground(cWhite).Bold(true)
	StyleMenuDim    = lipgloss.NewStyle().Foreground(cDimGray)
	StyleMenuActive = lipgloss.NewStyle().Foreground(cIceBlue).Bold(true)

	StyleDivider = lipgloss.NewStyle().Foreground(cDimGray)

	StyleLabel      = lipgloss.NewStyle().Foreground(cDimGray).Width(18)
	StyleValueHover = lipgloss.NewStyle().Foreground(cWhite)
	StyleValueDim   = lipgloss.NewStyle().Foreground(cDimGray)
	StyleValueFocus = lipgloss.NewStyle().Foreground(cWhite).Bold(true)

	StyleChatUserPrefix = lipgloss.NewStyle().Foreground(cDimGray)
	StyleChatUserText   = lipgloss.NewStyle().Foreground(cWhite)

	StyleChatBotPrefix = lipgloss.NewStyle().Foreground(cIceBlue)
	StyleChatBotText   = lipgloss.NewStyle().Foreground(cDimGray)

	StyleInputLine = lipgloss.NewStyle().Foreground(cDimGray)

	StyleFooter = lipgloss.NewStyle().Foreground(cDimGray)

	StyleStatusDot = lipgloss.NewStyle().Foreground(cGreen)
	StyleBarFilled = lipgloss.NewStyle().Foreground(cIceBlue)
	StyleBarEmpty  = lipgloss.NewStyle().Foreground(cDimGray)
)
