package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
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

	// if path not set, run through setup
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
	filename := fmt.Sprintf("%s/%s", cfg.Path, time.Now().Format("2006-01-02"))
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			encoded = false
		} else {
			log.Fatal(err)
		}
	}

	file, err := ioutil.TempFile(os.TempDir(), "jrnl-")
	if err != nil {
		log.Fatal(err)
	}
	// remove decoded file
	defer os.Remove(file.Name())

	var passphrase string
	if encoded {
		fmt.Println("Decrypting today's entry...")
		fmt.Println("Passpharse (32 bytes): ")
		bytePass, err := terminal.ReadPassword(0)
		if err != nil {
			log.Fatal(err)
		}
		passphrase = strings.TrimSpace(string(bytePass))
		// decode file if encoded and write to tmpfile
		if err := decodeFile(filename, file.Name(), passphrase); err != nil {
			log.Fatal(err)
		}
	}

	file.Close()
	// Open a file named the current date. Insert the current time at the last line
	// handle inputting the time with other editors.
	// Eventually this should open a file in /tmp/ and handle things there
	// TODO: handle additional editors?
	if err := edit(
		file.Name(),
		"-c", "set syntax=markdown",
		"-c", fmt.Sprintf(":call append(line('$'), '### %s')",
			time.Now().Format("15:04:05")),
		"+$",
	); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Encrypting today's entry...")
	if passphrase == "" {
		fmt.Println("Passpharse (32 bytes): ")
		bytePass, err := terminal.ReadPassword(0)
		if err != nil {
			log.Fatal(err)
		}
		passphrase = strings.TrimSpace(string(bytePass))
	}

	if err = encodeFile(file.Name(), filename, passphrase); err != nil {
		log.Fatal(err)
	}
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

func decodeFile(encryptedFile, outputFilename, passphrase string) error {
	encrypted, err := ioutil.ReadFile(encryptedFile)
	if err != nil {
		return fmt.Errorf("failed to read in encoded file: %v", err)
	}

	c, err := aes.NewCipher([]byte(passphrase))
	if err != nil {
		return fmt.Errorf("failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return fmt.Errorf("failed to create gcm: %v", err)
	}

	ns := gcm.NonceSize()
	if len(encrypted) < ns {
		return fmt.Errorf("data not encrypted: %v", err)
	}
	nonce, ciphertext := encrypted[:ns], encrypted[ns:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decode ciphertext: %v", err)
	}
	if err = ioutil.WriteFile(outputFilename, plaintext, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write decrypted file: %v", err)
	}
	return nil
}

func encodeFile(unencryptedFile, outputFilename, passphrase string) error {
	unencrypted, err := ioutil.ReadFile(unencryptedFile)
	if err != nil {
		return fmt.Errorf("failed to load unencrypted file: %v", err)
	}

	c, err := aes.NewCipher([]byte(passphrase))
	if err != nil {
		return fmt.Errorf("failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return fmt.Errorf("failed to generating gcm: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to populate nonce with random sequence: %v", err)
	}

	if err = ioutil.WriteFile(
		outputFilename,
		gcm.Seal(nonce, nonce, unencrypted, nil),
		os.ModePerm,
	); err != nil {
		return fmt.Errorf("failed to write encrypted file (%q): %v", outputFilename, err)
	}
	return nil
}
