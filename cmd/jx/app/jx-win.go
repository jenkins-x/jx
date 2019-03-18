// +build windows

package app

import (
	"os"
	"syscall"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
)

// Run runs the command
func Run() error {
	// if windows
	configureTerminalForAnsiEscapes()
	cmd := cmd.NewJXCommand(clients.NewFactory(), os.Stdin, os.Stdout, os.Stderr, nil)
	return cmd.Execute()
}

const (
	// https://docs.microsoft.com/en-us/windows/console/setconsolemode
	enableProcessedOutput = 0x1
	enableWrapAtEOLOutput = 0x2
	enableVirtualTermainalProcessing = 0x4

)
func configureTerminalForAnsiEscapes() {

	kernel32  := syscall.NewLazyDLL("kernel32.dll")
	kern32SetConsoleMode := kernel32.NewProc("SetConsoleMode")

	// stderr
	handle := syscall.Handle(os.Stderr.Fd())
	kern32SetConsoleMode.Call(uintptr(handle), enableProcessedOutput|enableWrapAtEOLOutput|enableVirtualTermainalProcessing)

	// stdout
	handle = syscall.Handle(os.Stdout.Fd())
	kern32SetConsoleMode.Call(uintptr(handle), enableProcessedOutput|enableWrapAtEOLOutput|enableVirtualTermainalProcessing)
}
