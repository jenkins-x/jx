package tfmt

import (
	"strconv"
)

// CursorUp provides ANSI cursor movement support. As CursorUp implements
// Stringer you can simply add a it to your print commands like
// fmt.Print(CursorUp(10),"Hello world").
type CursorUp int

// CursorDown provides ANSI cursor movement support. As CursorDown implements
// Stringer you can simply add a it to your print commands like
// fmt.Print(CursorDown(10),"Hello world").
type CursorDown int

// CursorRight provides ANSI cursor movement support. As CursorRight implements
// Stringer you can simply add a it to your print commands like
// fmt.Print(CursorRight(10),"Hello world").
type CursorRight int

// CursorLeft provides ANSI cursor movement support. As CursorLeft implements
// Stringer you can simply add a it to your print commands like
// fmt.Print(CursorLeft(10),"Hello world").
type CursorLeft int

// CursorPosition provides ANSI cursor movement support. As CursorPosition
// implements Stringer you can simply add a it to your print commands like
// fmt.Print(CursorPosition{x:0,y:0},"Hello world").
type CursorPosition struct {
	X int
	Y int
}

// CursorAction provides ANSI cursor management support. As CursorAction
// implements Stringer you can simply add a it to your print commands.
type CursorAction int

const (
	// CursorSave stores the current cursor position
	CursorSave = CursorAction(0)
	// CursorRestore restores the cursor position saved by CursorSave
	CursorRestore = CursorAction(1)
	// CursorClearLine clears the rest of the line
	CursorClearLine = CursorAction(2)
	// CursorClearScreen clears the screen and positions the cursor at 0,0
	CursorClearScreen = CursorAction(3)
)

// String implements the stringer interface for CursorUp
func (c CursorUp) String() string {
	return "\x1b[" + strconv.Itoa(int(c)) + "A"
}

// String implements the stringer interface for CursorDown
func (c CursorDown) String() string {
	return "\x1b[" + strconv.Itoa(int(c)) + "B"
}

// String implements the stringer interface for CursorRight
func (c CursorRight) String() string {
	return "\x1b[" + strconv.Itoa(int(c)) + "C"
}

// String implements the stringer interface for CursorLeft
func (c CursorLeft) String() string {
	return "\x1b[" + strconv.Itoa(int(c)) + "D"
}

// String implements the stringer interface for CursorPosition
func (c CursorPosition) String() string {
	return "\x1b[" + strconv.Itoa(c.X) + ";" + strconv.Itoa(c.Y) + "H"
}

// String implements the stringer interface for CursorAction
func (c CursorAction) String() string {
	switch c {
	case CursorSave:
		return "\x1b[s"
	case CursorRestore:
		return "\x1b[u"
	case CursorClearLine:
		return "\x1b[K"
	case CursorClearScreen:
		return "\x1b[2J"
	}
	return ""
}
