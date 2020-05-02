package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
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
		"-c", ":$put _",
		"-c", "$ s/^/### testing/",
		"-c", ":wq",
	); err != editorNotSet {
		t.Fatalf("excepted to fail when editor not set. got=%s", err.Error())
	}
}

func TestLoadConfigNotExists(t *testing.T) {
	dir, err := ioutil.TempDir("", ".config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cfg, err := loadConfig(dir)
	if err != nil {
		t.Fatalf("loadConfig(%q) failed with %s", dir, err.Error())
	}

	if cfg.Path != "" {
		t.Fatalf("excepted new, empty configuration. got=%v", cfg)
	}
}

func TestLoadConfigExists(t *testing.T) {
	dir, err := ioutil.TempDir("", ".config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	// make config and save it, then load it
	want := &config{Path: "testing"}
	if err = os.Mkdir(dir+configDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}

	if err = ioutil.WriteFile(path.Join(dir, "jrnl", "config.json"), data, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	got, err := loadConfig(dir)
	if err != nil {
		t.Fatalf("loadConfig(%q) failed with %s", dir, err.Error())
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loadConfig failed. got=%v; want=%v", got, want)
	}
}
