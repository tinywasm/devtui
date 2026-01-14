package devtui

import (
	"github.com/charmbracelet/lipgloss"
)

// Layout Constants - Single Source of Truth for Alignment
// UIColumnWidth determines the width of the left column (Header Title, Footer Label, Body Metadata)
// Based on "TINYWASM/BUILD" (14) + padding, and body requirements.
// 24 provides a balanced width:
// - Header: "TINYWASM/BUILD" (14) fits comfortably
// - Footer: "Compiler Mode" (13) fits comfortably
// - Body: Timestamp (9) + HandlerName (15) = 24
const (
	// Master Width - Single Source of Truth for Left Column Alignment
	UIColumnWidth = 24

	// Fixed Element Widths
	TimestampColumnWidth  = 9 // "HH:MM:SS "
	PaginationColumnWidth = 5 // " 1/ 4"
	FooterSpacerWidth     = 1 // Spacer between pagination and label
	FooterExtraPadding    = 2 // Extra padding from having two distinct blocks (Pag+Label) vs one (Header)

	// Derived Widths (Calculated automatically)
	HandlerNameWidth = UIColumnWidth - TimestampColumnWidth
	FooterLabelWidth = UIColumnWidth - PaginationColumnWidth - FooterSpacerWidth - FooterExtraPadding
)

type ColorPalette struct {
	// Base (2 colores)
	Foreground string // #F4F4F4
	Background string // #000000

	// Accent (2 colores)
	Primary   string // #FF6600 (tu actual Primary)
	Secondary string // #666666 (tu actual Secondary)

	// Semantic (4 colores)
	Success string // #00FF00
	Warning string // #FFFF00
	Error   string // #FF0000
	Info    string // #00FFFF

	// UI (2-4 colores adicionales)
	Border   string // #444444
	Muted    string // #999999
	Selected string // Derivado de Primary
	Hover    string // Derivado de Primary
}

type tuiStyle struct {
	*ColorPalette

	contentBorder    lipgloss.Border
	headerTitleStyle lipgloss.Style
	labelWidth       int // Ancho estándar para etiquetas
	labelStyle       lipgloss.Style

	footerInfoStyle    lipgloss.Style
	paginationStyle    lipgloss.Style // NEW: For pagination indicators
	fieldLineStyle     lipgloss.Style
	fieldSelectedStyle lipgloss.Style
	fieldEditingStyle  lipgloss.Style
	fieldReadOnlyStyle lipgloss.Style // NEW: For readonly fields (empty label)

	textContentStyle  lipgloss.Style
	lineHeadFootStyle lipgloss.Style // header right and footer left line

	// Estilos globales mensajes
	successStyle lipgloss.Style
	errStyle     lipgloss.Style
	warnStyle    lipgloss.Style
	infoStyle    lipgloss.Style
	normStyle    lipgloss.NoColor
	timeStyle    lipgloss.Style
}

func newTuiStyle(palette *ColorPalette) *tuiStyle {
	if palette == nil {
		palette = DefaultPalette()
	}

	t := &tuiStyle{
		ColorPalette: palette,
		// LABEL uses the remaining space after pagination and spacer
		labelWidth: FooterLabelWidth,
	}

	t.labelStyle = lipgloss.NewStyle().
		Width(t.labelWidth).
		Align(lipgloss.Left).
		Padding(0, 0)

	// El borde del contenido necesita conectarse con las pestañas
	t.contentBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	t.headerTitleStyle = lipgloss.NewStyle().
		Padding(0, 1).
		BorderForeground(lipgloss.Color(palette.Primary)).
		Background(lipgloss.Color(palette.Primary)).
		Foreground(lipgloss.Color(palette.Foreground))

	t.footerInfoStyle = t.headerTitleStyle

	t.paginationStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color(palette.Primary)).
		Foreground(lipgloss.Color(palette.Foreground))

	t.fieldLineStyle = lipgloss.NewStyle().
		Padding(0, 2)

	t.fieldSelectedStyle = t.fieldLineStyle
	t.fieldSelectedStyle = t.fieldSelectedStyle.
		Bold(true).
		Background(lipgloss.Color(palette.Primary)).
		Foreground(lipgloss.Color(palette.Foreground))

	t.fieldEditingStyle = t.fieldSelectedStyle.
		Foreground(lipgloss.Color(palette.Background))

	// NEW: Readonly style - highlight background with clear text for readonly fields (empty label)
	t.fieldReadOnlyStyle = t.fieldSelectedStyle.
		Background(lipgloss.Color(palette.Primary)).
		Foreground(lipgloss.Color(palette.Foreground))

	// Estilo para los mensajes - VISUAL UPGRADE: Padding interno para mejor legibilidad
	t.textContentStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(palette.Foreground)).
		PaddingLeft(1).
		PaddingRight(1)

	t.lineHeadFootStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(palette.Primary))

	// Inicializar los estilos que antes eran globales
	t.successStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(palette.Success))

	t.errStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(palette.Error))

	t.warnStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(palette.Warning))

	t.infoStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(palette.Info))

	t.normStyle = lipgloss.NoColor{}

	t.timeStyle = lipgloss.NewStyle().Foreground(
		lipgloss.Color(palette.Secondary),
	)

	return t
}

func DefaultPalette() *ColorPalette {
	return &ColorPalette{
		Foreground: "#F4F4F4",
		Background: "#000000",
		Primary:    "#00ADD8", // Gopher blue oficial de Go
		Secondary:  "#666666",
		Success:    "#00AA00",
		Warning:    "#FFAA00",
		Error:      "#FF0000",
		Info:       "#0088FF",
		Border:     "#444444",
		Muted:      "#999999",
	}
}
