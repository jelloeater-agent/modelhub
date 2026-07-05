package tui

import "github.com/charmbracelet/lipgloss"

var (
	primary   = lipgloss.Color("#7C3AED")
	secondary = lipgloss.Color("#10B981")
	warning   = lipgloss.Color("#F59E0B")
	errColor  = lipgloss.Color("#EF4444")
	muted     = lipgloss.Color("#6B7280")
	highlight = lipgloss.Color("#3B82F6")
	border    = lipgloss.Color("#374151")

	AppStyle = lipgloss.NewStyle().
			Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Background(primary).
			Foreground(lipgloss.Color("#F9FAFB")).
			Bold(true).
			Padding(0, 2)

	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#111827")).
			Foreground(muted).
			Padding(0, 1)

	mutedStyle = lipgloss.NewStyle().Foreground(muted)
	errorStyle = lipgloss.NewStyle().Foreground(errColor)

	DetailStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primary).
			Padding(1, 2)

	SectionTitle = lipgloss.NewStyle().
			Foreground(primary).
			Bold(true).
			Underline(true)

	LabelStyle = lipgloss.NewStyle().
			Foreground(muted).
			Width(20).
			Align(lipgloss.Left)

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB"))

	CapEnabled  = lipgloss.NewStyle().Foreground(secondary).Bold(true)
	CapDisabled = lipgloss.NewStyle().Foreground(muted)

	SourceBifrost   = "Bifrost"
	SourceModelsDev = "models.dev"
	SourceAA        = "Artificial Analysis"

	FilterStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(border).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(muted).
			Italic(true)

	ItalicStyle = lipgloss.NewStyle().Italic(true).Foreground(muted)
)
