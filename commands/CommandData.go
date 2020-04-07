package commands

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/DataManager-Go/DataManagerServer/constants"
	libdm "github.com/DataManager-Go/libdatamanager"
	dmConfig "github.com/DataManager-Go/libdatamanager/config"
	"github.com/JojiiOfficial/gaw"
	"golang.org/x/crypto/ssh/terminal"
)

// CommandData data for commands
type CommandData struct {
	LibDM                     *libdm.LibDM
	Command                   string
	Config                    *dmConfig.Config
	FileAttributes            libdm.FileAttributes
	Encryption, EncryptionKey string
	Namespace                 string
	Details                   uint8
	NameLen, RandKey          int
	All, AllNamespaces        bool
	NoRedaction, OutputJSON   bool
	Yes, Force, Quiet         bool
	EncryptionPassKey         bool
	NoDecrypt, NoEmojis       bool
	EncryptionFromStdin       bool
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
		if !isValidAESLen(cData.RandKey) {
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
		f, err := os.OpenFile(keyFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
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

	// Read and set encryptionkey from stdin
	if cData.EncryptionFromStdin {
		cData.EncryptionKey = readFullStdin(48)
		if !isValidAESLen(len(cData.EncryptionKey)) {
			fmtError("Invaild key length")
			os.Exit(1)
		}
	}

	// Create and set RequestConfig
	cData.LibDM = libdm.NewLibDM(cData.Config.MustGetRequestConfig())

	return true
}

func isValidAESLen(l int) bool {
	switch l {
	case 16, 24, 32:
		return true
	default:
		return false
	}
}

// Read from stdin with a timeout of 2s
func readFullStdin(bufferSize int) string {
	c := make(chan []byte, 1)

	// Read in background to allow using a select for a timeout
	go (func() {
		r := bufio.NewReader(os.Stdin)
		buf := make([]byte, bufferSize)

		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		c <- buf[:n]
	})()

	select {
	case b := <-c:
		return string(b)
	// Timeout
	case <-time.After(2 * time.Second):
		fmtError("No input received")
		os.Exit(1)
		return ""
	}
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
