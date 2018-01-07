////////////////////////////////////////////////////////////////////////////////
//                          DO NOT MODIFY THIS FILE!
//
//  This file was automatically generated via the commands:
//
//      go get github.com/coryb/autoplay
//      autoplay -n autoplay/confirm.go go run confirm.go
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
	c := exec.Command("go", "run", "confirm.go")
	c.Stdin = tty
	c.Stdout = tty
	c.Stderr = tty
	c.Start()
	buf := bufio.NewReaderSize(fh, 1024)

	expect("Enter 'yes'\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[37m(y/N) \x1b[0m", buf)
	fh.Write([]byte("y"))
	expect("y", buf)
	fh.Write([]byte("e"))
	expect("e", buf)
	fh.Write([]byte("s"))
	expect("s", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[36mYes\x1b[0m\r\n", buf)
	expect("Answered true.\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("Enter 'no'\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[37m(y/N) \x1b[0m", buf)
	fh.Write([]byte("n"))
	expect("n", buf)
	fh.Write([]byte("o"))
	expect("o", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[36mNo\x1b[0m\r\n", buf)
	expect("Answered false.\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("default\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[37m(Y/n) \x1b[0m", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[36mYes\x1b[0m\r\n", buf)
	expect("Answered true.\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("not recognized (enter random letter)\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[37m(Y/n) \x1b[0m", buf)
	fh.Write([]byte("x"))
	expect("x", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[31m✘ Sorry, your reply was invalid: \"x\" is not a valid answer, please try again.\x1b[0m\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[37m(Y/n) \x1b[0m", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[36mYes\x1b[0m\r\n", buf)
	expect("Answered true.\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("no help - type '?'\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[37m(Y/n) \x1b[0m", buf)
	fh.Write([]byte("?"))
	expect("?", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[31m✘ Sorry, your reply was invalid: \"?\" is not a valid answer, please try again.\x1b[0m\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[37m(Y/n) \x1b[0m", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99myes: \x1b[0m\x1b[36mYes\x1b[0m\r\n", buf)
	expect("Answered true.\r\n", buf)
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
