package terminal

import (
	"os"
	"syscall"
	"unsafe"
)

var COORDINATE_SYSTEM_BEGIN Short = 0
// shared variable to save the cursor location from CursorSave()
var cursorLoc Coord

func CursorUp(n int) {
	cursorMove(0, n)
}

func CursorDown(n int) {
	cursorMove(0, -1*n)
}

func CursorForward(n int) {
	cursorMove(n, 0)
}

func CursorBack(n int) {
	cursorMove(-1*n, 0)
}

// save the cursor location
func CursorSave() {
	cursorLoc, _ = CursorLocation()
}

func CursorRestore() {
	handle := syscall.Handle(os.Stdout.Fd())
	// restore it to the original position
	procSetConsoleCursorPosition.Call(uintptr(handle), uintptr(*(*int32)(unsafe.Pointer(&cursorLoc))))
}

func (cur Coord) CursorIsAtLineEnd(size *Coord) bool {
	return cur.X == size.X
}

func (cur Coord) CursorIsAtLineBegin() bool {
	return cur.X == 0
}

func cursorMove(x int, y int) {
	handle := syscall.Handle(os.Stdout.Fd())

	var csbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))

	var cursor Coord
	cursor.X = csbi.cursorPosition.X + Short(x)
	cursor.Y = csbi.cursorPosition.Y + Short(y)

	procSetConsoleCursorPosition.Call(uintptr(handle), uintptr(*(*int32)(unsafe.Pointer(&cursor))))
}

func CursorNextLine(n int) {
	CursorUp(n)
	CursorHorizontalAbsolute(0)
}

func CursorPreviousLine(n int) {
	CursorDown(n)
	CursorHorizontalAbsolute(0)
}

// for comparability purposes between windows
// in windows we don't have to print out a new line
func CursorMoveNextLine(cur Coord, terminalSize *Coord) {
	CursorNextLine(1)
}

func CursorHorizontalAbsolute(x int) {
	handle := syscall.Handle(os.Stdout.Fd())

	var csbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))

	var cursor Coord
	cursor.X = Short(x)
	cursor.Y = csbi.cursorPosition.Y

	if csbi.size.X < cursor.X {
		cursor.X = csbi.size.X
	}

	procSetConsoleCursorPosition.Call(uintptr(handle), uintptr(*(*int32)(unsafe.Pointer(&cursor))))
}

func CursorShow() {
	handle := syscall.Handle(os.Stdout.Fd())

	var cci consoleCursorInfo
	procGetConsoleCursorInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&cci)))
	cci.visible = 1

	procSetConsoleCursorInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&cci)))
}

func CursorHide() {
	handle := syscall.Handle(os.Stdout.Fd())

	var cci consoleCursorInfo
	procGetConsoleCursorInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&cci)))
	cci.visible = 0

	procSetConsoleCursorInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&cci)))
}

func CursorLocation() (Coord, error) {
	handle := syscall.Handle(os.Stdout.Fd())

	var csbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))

	return csbi.cursorPosition, nil
}

func Size() (*Coord, error) {
	handle := syscall.Handle(os.Stdout.Fd())

	var csbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))
	// windows' coordinate system begins at (0, 0)
	csbi.size.X--
	csbi.size.Y--
	return &csbi.size, nil
}
