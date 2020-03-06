package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	file, err := ioutil.TempFile("", "testing.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	now := time.Now().Format("15:04:05")
	want := fmt.Sprintf("\n### %s\n", now)

	if err := run(
		file.Name(),
		"-c", ":$put _",
		"-c", fmt.Sprintf("$ s/^/### %s/", now),
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
