package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

var editorNotSet = errors.New("EDITOR env variable not set")

func main() {
	// Open a file named the current date. Insert the current time at the last line
	// handle inputting the time with other editors. This seems to not work with vi
	if err := run(
		fmt.Sprintf("%s.md", time.Now().Format("2006-01-02")),
		"-c", ":$put _",
		"-c", fmt.Sprintf("$ s/^/### %s/", time.Now().Format("15:04:05")),
	); err != nil {
		panic(err)
	}
}

func run(cmds ...string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return editorNotSet
	}

	cmd := exec.Command(editor, cmds...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// TODO:
// - encryption
// - syncing
// - editing old entries

// Notes:
// For testing, search for some examples in https://github.com/golang/go/blob/master/src/os/exec/exec_test.go
