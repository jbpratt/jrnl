package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

// TODO:
// - encryption
// - syncing
// - editing old entries

const configDir = "/jrnl/"

var editorNotSet = errors.New("EDITOR env variable not set")

type config struct {
	// Path to store journal entries
	Path string `json:"path"`
}

func main() {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := loadConfig(dir)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.Path == "" {
		var response string
		fmt.Println("Where do you want to store your entries? (default ~/.config/jrnl/)")
		_, err = fmt.Scanln(&response)
		if err != nil {
			if err.Error() == "unexpected newline" {
				fmt.Println("Using default directory")
			} else {
				log.Fatal(err)
			}
		}

		if response == "" {
			cfg.Path = dir + configDir
		} else {
			cfg.Path = response
		}

		if err := writeConfig(cfg, dir+"/jrnl/config.json"); err != nil {
			log.Fatal(err)
		}
	}

	encoded := true
	filename := fmt.Sprintf("%s/%s.md", cfg.Path, time.Now().Format("2006-01-02"))
	if _, err := os.Stat(filename + ".age"); err != nil {
		if os.IsNotExist(err) {
			encoded = false
		} else {
			log.Fatal(err)
		}
	}

	// create the markdown file but don't encode
	/*
		file, err := ioutil.TempFile(os.TempDir(), "jrnl-")
		if err != nil {
			log.Fatal(err)
		}
		// remove decoded file
		defer os.Remove(file.Name())
	*/

	if encoded {
		// decode file if encoded
		log.Fatal(errors.New("decoding not implemented"))
	}

	// file.Close()
	// Open a file named the current date. Insert the current time at the last line
	// handle inputting the time with other editors.
	// Eventually this should open a file in /tmp/ and handle things there
	// TODO: handle additional editors?
	if err := edit(
		filename,
		"-c", fmt.Sprintf(":call append(line('$'), '### %s')", time.Now().Format("15:04:05")),
		"+$",
	); err != nil {
		log.Fatal(err)
	}

	// reencode the file
}

func edit(cmds ...string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return editorNotSet
	}

	cmd := exec.Command(editor, cmds...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor: %v with %v", err, cmds)
	}
	return nil
}

func loadConfig(dir string) (*config, error) {
	path := dir + "/jrnl/config.json"
	// first time, make dir and config file
	if _, err := os.Stat(dir + configDir); os.IsNotExist(err) {
		if err = os.Mkdir(dir+configDir, os.ModePerm); err != nil {
			return nil, err
		}

		file, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create config directory: %v", err)
		}
		if err = writeConfig(&config{}, path); err != nil {
			return nil, fmt.Errorf("failed to write the new config: %v", err)
		}
		if err = file.Close(); err != nil {
			return nil, fmt.Errorf("failed to close the new config file: %v", err)
		}
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var cfg config
	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return &cfg, nil
}

func writeConfig(cfg *config, path string) error {
	data, err := json.Marshal(&config{})
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	if err = ioutil.WriteFile(path, data, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	return nil
}
