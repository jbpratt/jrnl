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

	"golang.org/x/crypto/ssh/terminal"
)

// TODO:
// - syncing
// - editing old entries

const configDir = "/jrnl/"

var editorNotSet = errors.New("EDITOR env variable not set")

type config struct {
	// Path to store journal entries
	Path string `json:"path"`
	// age encryption method
	EncryptionMethod string `json:"encryption_method"`
}

func main() {
	// check for userconfigdir for jrnl
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := loadConfig(dir)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.EncryptionMethod == "" && cfg.Path == "" {
		fmt.Println(`
Which age encryption method would you like to go with?
1.) Encrypt to a PEM encoded format.
2.) Encrypt with a passphrase.
3.) Encrypt to the specified RECIPIENT. Can be repeated.`)
		// setup encryption recipient
		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			log.Fatal(err)
		}

		switch response {
		case "1":
			cfg.EncryptionMethod = "PEM"
		case "2":
			cfg.EncryptionMethod = "PASSPHRASE"
		case "3":
			cfg.EncryptionMethod = "RECIPIENT"
		default:
			log.Fatal("Error: invalid input")
		}

		fmt.Println("Where do you want to store your entries? (default ~/.config/jrnl/)")
		response = ""
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
	filename := fmt.Sprintf(cfg.Path+"%s.md", time.Now().Format("2006-01-02"))
	if _, err := os.Stat(filename + ".age"); err != nil {
		if os.IsNotExist(err) {
			encoded = false
		}
	}
	var pass []byte
	if encoded {
		switch cfg.EncryptionMethod {
		case "PASSPHRASE":
			// TODO: autogenerate using age feature
			fmt.Println("Enter passphrase:")
			pass, err = terminal.ReadPassword(0)
			if err != nil {
				log.Fatal(err)
			}
		case "PEM":
		case "RECIPIENT":
			fmt.Println("decoding")
		default:
			log.Fatal("unknown encryption method")
		}
		// decode file
		_ = pass
	}

	// Open a file named the current date. Insert the current time at the last line
	// handle inputting the time with other editors.
	if err := editor(
		filename,
		"-c", fmt.Sprintf(":call append(line('$'), '### %s')", time.Now().Format("15:04:05")),
		"+$",
	); err != nil {
		log.Fatal(err)
	}

	// reencode the file

	// remove decoded file
}

func editor(cmds ...string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return editorNotSet
	}

	cmd := exec.Command(editor, cmds...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return cmd.Run()
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
			return nil, err
		}
		defer file.Close()
		if err := writeConfig(&config{}, path); err != nil {
			return nil, err
		}
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func writeConfig(cfg *config, path string) error {
	data, err := json.Marshal(&config{})
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(path, data, os.ModePerm); err != nil {
		return err
	}
	return nil
}
