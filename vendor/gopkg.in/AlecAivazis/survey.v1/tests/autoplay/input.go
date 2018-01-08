////////////////////////////////////////////////////////////////////////////////
//                          DO NOT MODIFY THIS FILE!
//
//  This file was automatically generated via the commands:
//
//      go get github.com/coryb/autoplay
//      autoplay -n autoplay/input.go go run input.go
//
////////////////////////////////////////////////////////////////////////////////
package main

import (
	"bufio"
	"fmt"
	"github.com/kr/pty"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	RED   = "\033[31m"
	RESET = "\033[0m"
)

func main() {
	fh, tty, _ := pty.Open()
	defer tty.Close()
	defer fh.Close()
	c := exec.Command("go", "run", "input.go")
	c.Stdin = tty
	c.Stdout = tty
	c.Stderr = tty
	c.Start()
	buf := bufio.NewReaderSize(fh, 1024)

	expect("no default\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mHello world \x1b[0m", buf)
	fh.Write([]byte("a"))
	expect("a", buf)
	fh.Write([]byte("l"))
	expect("l", buf)
	fh.Write([]byte("e"))
	expect("e", buf)
	fh.Write([]byte("c"))
	expect("c", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mHello world \x1b[0m\x1b[36malec\x1b[0m\r\n", buf)
	expect("Answered alec.\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("default\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mHello world \x1b[0m\x1b[37m(default) \x1b[0m", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mHello world \x1b[0m\x1b[36mdefault\x1b[0m\r\n", buf)
	expect("Answered default.\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("no help, send '?'\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mHello world \x1b[0m", buf)
	fh.Write([]byte("?"))
	expect("?", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mHello world \x1b[0m\x1b[36m?\x1b[0m\r\n", buf)
	expect("Answered ?.\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("input text in random location\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mHello \x1b[0m", buf)
	fh.Write([]byte("h"))
	expect("h", buf)
	fh.Write([]byte("e"))
	expect("e", buf)
	fh.Write([]byte("l"))
	expect("l", buf)
	fh.Write([]byte("l"))
	expect("l", buf)
	fh.Write([]byte("o"))
	expect("o", buf)
	fh.Write([]byte(" "))
	expect(" ", buf)
	fh.Write([]byte("w"))
	expect("w", buf)
	fh.Write([]byte("o"))
	expect("o", buf)
	fh.Write([]byte("r"))
	expect("r", buf)
	fh.Write([]byte("l"))
	expect("l", buf)
	fh.Write([]byte("d"))
	expect("d", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("D"))
	expect("\x1b[1D", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("B"))
	fh.Write([]byte("a"))
	expect("\x1b[0Kad\x1b[1D", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("D"))
	expect("\x1b[1D", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("D"))
	expect("\x1b[1D", buf)
	fh.Write([]byte("a"))
	expect("\x1b[0Kalad\x1b[3D", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("D"))
	expect("\x1b[1D", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("D"))
	expect("\x1b[1D", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("D"))
	expect("\x1b[1D", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("D"))
	expect("\x1b[1D", buf)
	fh.Write([]byte("\u007f"))
	expect("\x1b[1D\x1b[0Kworalad\x1b[7D", buf)
	fh.Write([]byte("\u007f"))
	expect("\x1b[1D\x1b[0Kworalad\x1b[7D", buf)
	fh.Write([]byte("\u007f"))
	expect("\x1b[1D\x1b[0Kworalad\x1b[7D", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mHello \x1b[0m\x1b[36mhelworalad\x1b[0m\r\n", buf)
	expect("Answered helworalad.\r\n", buf)
	expect("---------------------\r\n", buf)

	c.Wait()
	tty.Close()
	fh.Close()
}

func expect(expected string, buf *bufio.Reader) {
	sofar := []rune{}
	for _, r := range expected {
		got, _, _ := buf.ReadRune()
		sofar = append(sofar, got)
		if got != r {
			fmt.Fprintln(os.Stderr, RESET)

			// we want to quote the string but we also want to make the unexpected character RED
			// so we use the strconv.Quote function but trim off the quoted characters so we can
			// merge multiple quoted strings into one.
			expStart := strings.TrimSuffix(strconv.Quote(expected[:len(sofar)-1]), "\"")
			expMiss := strings.TrimSuffix(strings.TrimPrefix(strconv.Quote(string(expected[len(sofar)-1])), "\""), "\"")
			expEnd := strings.TrimPrefix(strconv.Quote(expected[len(sofar):]), "\"")

			fmt.Fprintf(os.Stderr, "Expected: %s%s%s%s%s\n", expStart, RED, expMiss, RESET, expEnd)

			// read the rest of the buffer
			p := make([]byte, buf.Buffered())
			buf.Read(p)

			gotStart := strings.TrimSuffix(strconv.Quote(string(sofar[:len(sofar)-1])), "\"")
			gotMiss := strings.TrimSuffix(strings.TrimPrefix(strconv.Quote(string(sofar[len(sofar)-1])), "\""), "\"")
			gotEnd := strings.TrimPrefix(strconv.Quote(string(p)), "\"")

			fmt.Fprintf(os.Stderr, "Got:      %s%s%s%s%s\n", gotStart, RED, gotMiss, RESET, gotEnd)
			panic(fmt.Errorf("Unexpected Rune %q, Expected %q\n", got, r))
		} else {
			fmt.Printf("%c", r)
		}
	}
}
