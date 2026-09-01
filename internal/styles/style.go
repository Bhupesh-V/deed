package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primary   = lipgloss.Color("42")  // Spring green
	accent    = lipgloss.Color("212") // Soft pink
	highlight = lipgloss.Color("86")  // Cyan
	white     = lipgloss.Color("#FFFFFF")
	green     = lipgloss.Color("#04B575")
	amber     = lipgloss.Color("214")
	darkGrey  = lipgloss.Color("236")

	// Styles
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(primary).
		MarginBottom(1)

	TitleText = lipgloss.NewStyle().
			Bold(true).
			Foreground(primary)

	Label = lipgloss.NewStyle().
		Bold(true).
		Foreground(accent).
		Width(10)

	Value = lipgloss.NewStyle().
		Foreground(white)

	Badge = lipgloss.NewStyle().
		Foreground(highlight).
		Background(darkGrey).
		Padding(0, 1).
		Bold(true)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(1, 2).
		Margin(1, 0)

	// medium grey
	Dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	Success = lipgloss.NewStyle().Foreground(primary)
	Warning = lipgloss.NewStyle().Foreground(amber).Bold(true)
	Node    = lipgloss.NewStyle().Foreground(white).Bold(false)
	Target  = lipgloss.NewStyle().Foreground(accent).Bold(true)

	// Compact badge specifically for MPB line decorators
	BarDoneText = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(green).
			Render(" DONE ")

	BarPendingText = Dim.Render("  ⏳  ") // Match visual width of BarDoneText badge

)
