package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primary   = lipgloss.Color("42")  // Spring green
	accent    = lipgloss.Color("212") // Soft pink
	highlight = lipgloss.Color("86")  // Cyan

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
		Foreground(lipgloss.Color("255"))

	Badge = lipgloss.NewStyle().
		Foreground(highlight).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Bold(true)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(1, 2).
		Margin(1, 0)

	// medium grey
	Dim = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	// bright green
	Success = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	// amber
	Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	Node = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(false)

	Target = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")) // Bright White

	// Compact badge specifically for MPB line decorators
	BarDoneText = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#04B575")).
			Render(" DONE ")

	BarPendingText = Dim.Render("  ⏳  ") // Match visual width of BarDoneText badge

)
