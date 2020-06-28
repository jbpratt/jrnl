package main

import (
	"bufio"
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
	"path"
	"time"

	keyring "github.com/99designs/keyring"
	"github.com/shurcooL/markdownfmt/markdown"
	"golang.org/x/crypto/ssh/terminal"
)

// TODO:
// - syncing
// - editing old entries

const configDir = "jrnl"

var editorNotSet = errors.New("EDITOR env variable not set")

type config struct {
	// Path to store journal entries
	Path string `json:"path"`
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	filename := path.Join(cfg.Path, "jrnl")
	encoded := doesExist(filename)

	file, err := ioutil.TempFile(os.TempDir(), "jrnl-")
	if err != nil {
		log.Fatal(err)
	}
	// remove decoded file
	defer os.Remove(file.Name())

	kr, err := keyring.Open(keyring.Config{
		ServiceName: "jrnl",
		AllowedBackends: []keyring.BackendType{
			keyring.KWalletBackend,
			keyring.PassBackend,
			keyring.SecretServiceBackend,
			keyring.KeychainBackend,
			keyring.WinCredBackend,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	var passphrase []byte
	if encoded {
		pswd, err := kr.Get("passphrase")
		if err != nil {
			log.Fatal(err)
		}
		passphrase = pswd.Data
		if err := decodeFile(filename, file.Name(), passphrase); err != nil {
			log.Fatal(err)
		}
	}

	if _, err := file.WriteString(
		fmt.Sprintf("\n### %s\n", time.Now().Format("01-02-2006 15:04:05 Mon")),
	); err != nil {
		log.Fatal(err)
	}

	file.Close()

	// Open a file named the current date. Insert the current time at the last line
	// handle inputting the time with other editors.
	// Eventually this should open a file in /tmp/ and handle things there
	// TODO: handle additional editors?
	if err := edit(file.Name()); err != nil {
		log.Fatal(err)
	}

	if len(passphrase) == 0 {
		fmt.Println("Passpharse (32 bytes): ")
		passphrase, err = terminal.ReadPassword(0)
		if err != nil {
			log.Fatal(err)
		}
		if err = kr.Set(keyring.Item{
			Description: "jrnl",
			Key:         "passphrase",
			Data:        passphrase,
		}); err != nil {
			log.Fatal(err)
		}
	}

	if err = fmtAndEncodeFile(file.Name(), filename, passphrase); err != nil {
		log.Fatal(err)
	}
}

func doesExist(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			log.Fatal(err)
		}
	}
	return true
}

func edit(cmds ...string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return editorNotSet
	}

	if editor == "vim" {
		cmds = append(cmds, "-c set syntax=markdown", "+$")
	} else if editor == "nano" {
		lc := countLines(cmds[0])
		if err := setupNanoSyntaxHighlighting(); err != nil {
			return err
		}
		cmds = append(cmds, "-Y markdown", fmt.Sprintf("+%d", lc))
	}

	cmd := exec.Command(editor, cmds...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor: %w with %v", err, cmds)
	}
	return nil
}

func setupNanoSyntaxHighlighting() error {
	confdir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	syntaxPath := path.Join(confdir, "nano", "syntax")
	if !doesExist(path.Join(syntaxPath, "markdown.nanorc")) {
		if err = os.MkdirAll(syntaxPath, os.ModePerm); err != nil {
			return err
		}

		// copy markdown syntax to user config
		source, err := os.Open(path.Join("config", "markdown.nanorc"))
		if err != nil {
			return err
		}

		mdpath := path.Join(syntaxPath, "markdown.nanorc")
		dest, err := os.OpenFile(mdpath, os.O_CREATE|os.O_RDWR, os.ModeAppend)
		if err != nil {
			return err
		}

		_, err = io.Copy(dest, source)
		if err != nil {
			return err
		}

		source.Close()
		dest.Close()
		fmt.Println(confdir)

		dir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		nanorcPath := path.Join(dir, ".nanorc")

		if !doesExist(nanorcPath) {
			nanorcPath = path.Join(confdir, "nano", "nanorc")
		}

		nanorc, err := os.OpenFile(nanorcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return err
		}

		if _, err = nanorc.WriteString(fmt.Sprintf("include %s\n", mdpath)); err != nil {
			return err
		}

		if err := nanorc.Close(); err != nil {
			return err
		}
	}

	return nil
}

// this probably isn't the fastest way to count lines but it works
func countLines(path string) int {
	file, _ := os.Open(path)
	fileScanner := bufio.NewScanner(file)
	lineCount := 0
	for fileScanner.Scan() {
		lineCount++
	}
	return lineCount
}

func getConfigPath(dir string) string {
	var response string
	fmt.Println("Where do you want to store your entries? (default $HOME/.config/jrnl/)")
	_, err := fmt.Scanln(&response)
	if err != nil {
		if err.Error() == "unexpected newline" {
			fmt.Println("Using default directory")
		} else {
			log.Fatal(err)
		}
	}

	var outpath string
	if response == "" {
		outpath = path.Join(dir, configDir)
	} else {
		outpath = response
	}

	return outpath
}

func loadConfig() (*config, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	cfgpath := path.Join(dir, "jrnl", "config.json")
	cfgdir := path.Join(dir, configDir)
	// first time, make dir and config file
	if _, err := os.Stat(cfgdir); os.IsNotExist(err) {
		if err = os.Mkdir(cfgdir, os.ModePerm); err != nil {
			return nil, err
		}

		file, err := os.Create(cfgpath)
		if err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		pth := getConfigPath(dir)

		if err = writeConfig(&config{pth}, cfgpath); err != nil {
			return nil, fmt.Errorf("failed to write the new config: %w", err)
		}
		if err = file.Close(); err != nil {
			return nil, fmt.Errorf("failed to close the new config file: %w", err)
		}
	}

	data, err := ioutil.ReadFile(cfgpath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config
	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &cfg, nil
}

func writeConfig(cfg *config, path string) error {
	data, err := json.Marshal(&config{})
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err = ioutil.WriteFile(path, data, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func decodeFile(encryptedFile, outputFilename string, passphrase []byte) error {
	encrypted, err := ioutil.ReadFile(encryptedFile)
	if err != nil {
		return fmt.Errorf("failed to read in encoded file: %w", err)
	}

	c, err := aes.NewCipher(passphrase)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return fmt.Errorf("failed to create gcm: %w", err)
	}

	ns := gcm.NonceSize()
	if len(encrypted) < ns {
		return fmt.Errorf("data not encrypted: %w", err)
	}
	nonce, ciphertext := encrypted[:ns], encrypted[ns:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	if err = ioutil.WriteFile(outputFilename, plaintext, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}
	return nil
}

func fmtAndEncodeFile(unencryptedFile, outputFilename string, passphrase []byte) error {
	unencrypted, err := ioutil.ReadFile(unencryptedFile)
	if err != nil {
		return fmt.Errorf("failed to load unencrypted file: %w", err)
	}

	fmtd, err := markdown.Process("", unencrypted, nil)
	if err != nil {
		return fmt.Errorf("failed to format jrnl entry: %w", err)
	}

	c, err := aes.NewCipher(passphrase)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return fmt.Errorf("failed to generating gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to populate nonce with random sequence: %w", err)
	}

	if err = ioutil.WriteFile(
		outputFilename,
		gcm.Seal(nonce, nonce, fmtd, nil),
		os.ModePerm,
	); err != nil {
		return fmt.Errorf("failed to write encrypted file (%q): %w", outputFilename, err)
	}
	return nil
}
