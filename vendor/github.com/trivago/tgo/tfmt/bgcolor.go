package tfmt

import (
	"strconv"
)

// BackgroundColor provides ANSI background color support. As Color implements Stringer
//  you can simply add a color to your print commands like
// fmt.Print(BgRed,"Hello world").
type BackgroundColor int

const (
	// NoBackground is used to not change the current background color
	NoBackground = BackgroundColor(0)
	// BlackBackground color
	BlackBackground = BackgroundColor(40)
	// RedBackground color
	RedBackground = BackgroundColor(41)
	// GreenBackground color
	GreenBackground = BackgroundColor(42)
	// YellowBackground color
	YellowBackground = BackgroundColor(43)
	// BlueBackground color
	BlueBackground = BackgroundColor(44)
	// PurpleBackground color
	PurpleBackground = BackgroundColor(45)
	// CyanBackground color
	CyanBackground = BackgroundColor(46)
	// GrayBackground color
	GrayBackground = BackgroundColor(47)
)

// String implements the stringer interface for color
func (c BackgroundColor) String() string {
	if c == NoBackground {
		return ""
	}
	return "\x1b[" + strconv.Itoa(int(c)) + "m"
}
