////////////////////////////////////////////////////////////////////////////////
//                          DO NOT MODIFY THIS FILE!
//
//  This file was automatically generated via the commands:
//
//      go get github.com/coryb/autoplay
//      autoplay -n autoplay/help.go go run help.go
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
	c := exec.Command("go", "run", "help.go")
	c.Stdin = tty
	c.Stdout = tty
	c.Stderr = tty
	c.Start()
	buf := bufio.NewReaderSize(fh, 1024)

	expect("confirm\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mIs it raining? \x1b[0m\x1b[36m[? for help]\x1b[0m \x1b[37m(y/N) \x1b[0m", buf)
	fh.Write([]byte("?"))
	expect("?", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[36mⓘ Go outside, if your head becomes wet the answer is probably 'yes'\x1b[0m\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mIs it raining? \x1b[0m\x1b[37m(y/N) \x1b[0m", buf)
	fh.Write([]byte("y"))
	expect("y", buf)
	fh.Write([]byte("e"))
	expect("e", buf)
	fh.Write([]byte("s"))
	expect("s", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mIs it raining? \x1b[0m\x1b[36mYes\x1b[0m\r\n", buf)
	expect("Answered true.\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("input\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mWhat is your phone number: \x1b[0m\x1b[36m[? for help]\x1b[0m ", buf)
	fh.Write([]byte("?"))
	expect("?", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[36mⓘ Phone number should include the area code, parentheses optional\x1b[0m\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mWhat is your phone number: \x1b[0m", buf)
	fh.Write([]byte("1"))
	expect("1", buf)
	fh.Write([]byte("2"))
	expect("2", buf)
	fh.Write([]byte("3"))
	expect("3", buf)
	fh.Write([]byte("-"))
	expect("-", buf)
	fh.Write([]byte("1"))
	expect("1", buf)
	fh.Write([]byte("2"))
	expect("2", buf)
	fh.Write([]byte("3"))
	expect("3", buf)
	fh.Write([]byte("-"))
	expect("-", buf)
	fh.Write([]byte("1"))
	expect("1", buf)
	fh.Write([]byte("2"))
	expect("2", buf)
	fh.Write([]byte("3"))
	expect("3", buf)
	fh.Write([]byte("4"))
	expect("4", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mWhat is your phone number: \x1b[0m\x1b[36m123-123-1234\x1b[0m\r\n", buf)
	expect("Answered 123-123-1234.\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("select\r\n", buf)
	expect("\x1b[?25l\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mWhat days are you available:\x1b[0m \x1b[36m[? for help]\x1b[0m\r\n", buf)
	expect("\x1b[36m❯\x1b[0m\x1b[32m ◉ \x1b[0m Monday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Tuesday\r\n", buf)
	expect(" \x1b[1;99m ◯ \x1b[0m Wednesday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Thursday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Friday\r\n", buf)
	fh.Write([]byte("?"))
	expect("\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[36mⓘ We are closed weekends and avaibility is limited on Wednesday\x1b[0m\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mWhat days are you available:\x1b[0m\r\n", buf)
	expect("\x1b[36m❯\x1b[0m\x1b[32m ◉ \x1b[0m Monday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Tuesday\r\n", buf)
	expect(" \x1b[1;99m ◯ \x1b[0m Wednesday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Thursday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Friday\r\n", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("B"))
	expect("\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[36mⓘ We are closed weekends and avaibility is limited on Wednesday\x1b[0m\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mWhat days are you available:\x1b[0m\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Monday\r\n", buf)
	expect("\x1b[36m❯\x1b[0m\x1b[32m ◉ \x1b[0m Tuesday\r\n", buf)
	expect(" \x1b[1;99m ◯ \x1b[0m Wednesday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Thursday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Friday\r\n", buf)
	fh.Write([]byte(" "))
	expect("\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[36mⓘ We are closed weekends and avaibility is limited on Wednesday\x1b[0m\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mWhat days are you available:\x1b[0m\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Monday\r\n", buf)
	expect("\x1b[36m❯\x1b[0m\x1b[1;99m ◯ \x1b[0m Tuesday\r\n", buf)
	expect(" \x1b[1;99m ◯ \x1b[0m Wednesday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Thursday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Friday\r\n", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("B"))
	expect("\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[36mⓘ We are closed weekends and avaibility is limited on Wednesday\x1b[0m\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mWhat days are you available:\x1b[0m\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Monday\r\n", buf)
	expect(" \x1b[1;99m ◯ \x1b[0m Tuesday\r\n", buf)
	expect("\x1b[36m❯\x1b[0m\x1b[1;99m ◯ \x1b[0m Wednesday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Thursday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Friday\r\n", buf)
	fh.Write([]byte("\x1b"))
	fh.Write([]byte("["))
	fh.Write([]byte("B"))
	expect("\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[36mⓘ We are closed weekends and avaibility is limited on Wednesday\x1b[0m\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mWhat days are you available:\x1b[0m\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Monday\r\n", buf)
	expect(" \x1b[1;99m ◯ \x1b[0m Tuesday\r\n", buf)
	expect(" \x1b[1;99m ◯ \x1b[0m Wednesday\r\n", buf)
	expect("\x1b[36m❯\x1b[0m\x1b[32m ◉ \x1b[0m Thursday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Friday\r\n", buf)
	fh.Write([]byte(" "))
	expect("\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[36mⓘ We are closed weekends and avaibility is limited on Wednesday\x1b[0m\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mWhat days are you available:\x1b[0m\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Monday\r\n", buf)
	expect(" \x1b[1;99m ◯ \x1b[0m Tuesday\r\n", buf)
	expect(" \x1b[1;99m ◯ \x1b[0m Wednesday\r\n", buf)
	expect("\x1b[36m❯\x1b[0m\x1b[1;99m ◯ \x1b[0m Thursday\r\n", buf)
	expect(" \x1b[32m ◉ \x1b[0m Friday\r\n", buf)
	fh.Write([]byte("\r"))
	expect("\x1b[?25h\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mWhat days are you available:\x1b[0m\x1b[36m Monday, Friday\x1b[0m\r\n", buf)
	expect("Answered [Monday Friday].\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("select\r\n", buf)
	expect("\x1b[0G\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mChoose a color:\x1b[0m \x1b[36m[? for help]\x1b[0m\r\n", buf)
	expect("\x1b[1;99m  red\x1b[0m\r\n", buf)
	expect("\x1b[1;36m❯ blue\x1b[0m\r\n", buf)
	expect("\x1b[1;99m  green\x1b[0m\r\n", buf)
	expect("\x1b[?25l", buf)
	fh.Write([]byte("?"))
	expect("\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[36mⓘ Blue is the best color, but it is your choice\x1b[0m\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mChoose a color:\x1b[0m\r\n", buf)
	expect("\x1b[1;99m  red\x1b[0m\r\n", buf)
	expect("\x1b[1;36m❯ blue\x1b[0m\r\n", buf)
	expect("\x1b[1;99m  green\x1b[0m\r\n", buf)
	fh.Write([]byte("\r"))
	expect("\x1b[?25h\x1b[0G\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1F\x1b[2K\x1b[1;92m? \x1b[0m\x1b[1;99mChoose a color:\x1b[0m\x1b[36m blue\x1b[0m\r\n", buf)
	expect("Answered blue.\r\n", buf)
	expect("---------------------\r\n", buf)
	expect("password\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mEnter a secret: \x1b[0m\x1b[36m[? for help]\x1b[0m ", buf)
	fh.Write([]byte("?"))
	expect("*", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("\x1b[1F\x1b[0G\x1b[2K\x1b[36mⓘ Don't really enter a secret, this is just for testing\x1b[0m\r\n", buf)
	expect("\x1b[1;92m? \x1b[0m\x1b[1;99mEnter a secret: \x1b[0m", buf)
	fh.Write([]byte("f"))
	expect("*", buf)
	fh.Write([]byte("o"))
	expect("*", buf)
	fh.Write([]byte("o"))
	expect("*", buf)
	fh.Write([]byte("b"))
	expect("*", buf)
	fh.Write([]byte("a"))
	expect("*", buf)
	fh.Write([]byte("r"))
	expect("*", buf)
	fh.Write([]byte("\r"))
	expect("\r\r\n", buf)
	expect("Answered foobar.\r\n", buf)
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
