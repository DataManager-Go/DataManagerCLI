package commands

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/DataManagerServer/constants"
	"github.com/JojiiOfficial/gaw"
	"github.com/Yukaru-san/DataManager_Client/models"
	"golang.org/x/crypto/ssh/terminal"
)

// CommandData data for commands
type CommandData struct {
	LibDM                     *libdm.LibDM
	Command                   string
	Config                    *models.Config
	FileAttributes            models.FileAttributes
	Encryption, EncryptionKey string
	Namespace                 string
	Details                   uint8
	NameLen, RandKey          int
	All, AllNamespaces        bool
	NoRedaction, OutputJSON   bool
	Yes, Force, Bench, Quiet  bool
	EncryptionPassKey         bool
	NoDecrypt, NoEmojis       bool
	BenchDone                 chan time.Time
}

// Init init CommandData
func (cData *CommandData) Init() bool {
	// Validate cipher
	if len(cData.Encryption) > 0 && !constants.IsValidCipher(cData.Encryption) {
		fmt.Println("Invalid encryption cipter")
		return false
	}

	// Read encryptionkey from stdin
	if cData.EncryptionPassKey && cData.supportInputKey() {
		cData.EncryptionKey = readPassword("Key password")
		if len(cData.EncryptionKey) == 0 {
			return false
		}
	}

	// Generate random key
	if cData.RandKey > 0 && !cData.EncryptionPassKey && cData.supportRandKey() {
		switch cData.RandKey {
		case 16, 24, 32:
		default:
			fmt.Println("Invalid Keysize", cData.RandKey)
			return false
		}

		b := make([]byte, cData.RandKey)
		_, err := rand.Read(b)
		if err != nil {
			fmt.Println("error:", err)
			return false
		}

		keyFile := genFileName("key")
		f, err := os.Create(keyFile)
		defer f.Close()

		if err != nil {
			fmt.Println(err)
			return false
		}

		_, err = f.Write(b)
		if err != nil {
			fmt.Println(err)
			return false
		}

		fmt.Printf("File %s saved\n", keyFile)
		cData.EncryptionKey = string(b)
	}

	// Create and set RequestConfig
	cData.LibDM = libdm.NewLibDM(cData.Config.ToRequestConfig())

	return true
}

// Return true if current command needs a key input
func (cData *CommandData) supportRandKey() bool {
	return gaw.IsInStringArray(cData.Command, []string{"upload"})
}

// Return true if current command needs a key input
func (cData *CommandData) supportInputKey() bool {
	if cData.supportRandKey() {
		return true
	}
	return gaw.IsInStringArray(cData.Command, []string{"file view"})
}

// Gen filename for args
func genFileName(prefix string) string {
	var name string
	for {
		name = prefix + gaw.RandString(4)
		_, err := os.Stat(name)
		if err != nil {
			break
		}
	}
	return name
}

// Read password/key from stdin
func readPassword(message string) string {
	fmt.Print(message + "> ")

	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalln("Error:", err.Error())
		return ""
	}

	var pass string

	for _, a := range bytePassword {
		if int(a) != 0 && int(a) != 32 {
			pass += string(a)
		}
	}

	return strings.TrimSpace(pass)
}
