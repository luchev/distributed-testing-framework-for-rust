package logging

// Escape sequence
const Escape = "\x1b"

// Custom terminal background colors
const (
	Reset = iota
	Bold
	Faint
	Italic
	Underline
	BlinkSlow
	BlinkRapid
	ReverseVideo
	Concealed
	CrossedOut
)

// Custom terminal foreground text colors
const (
	FgBlack = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)
