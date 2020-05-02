package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestEditor(t *testing.T) {
	if err := os.Setenv("EDITOR", "vim"); err != nil {
		t.Fatal(err)
	}

	file, err := ioutil.TempFile("", "testing.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	now := time.Now().Format("15:04:05")
	want := fmt.Sprintf("\n### %s\n", now)

	if err := edit(
		file.Name(),
		"-c", fmt.Sprintf(":call append(line('$'), '### %s')", time.Now().Format("15:04:05")),
		"-c", ":wq",
	); err != nil {
		t.Fatal(err)
	}

	got, err := ioutil.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != want {
		t.Fatalf("incorrect data from vim command. got=%q; want=%q", string(got), want)
	}
}

func TestEditorNotSet(t *testing.T) {
	if err := os.Setenv("EDITOR", ""); err != nil {
		t.Fatal(err)
	}

	file, err := ioutil.TempFile("", "testing.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	if err := edit(
		file.Name(),
		"-c", ":wq",
	); err != editorNotSet {
		t.Fatalf("excepted to fail when editor not set. got=%s", err.Error())
	}
}
