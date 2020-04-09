package commands

import (
	"fmt"
	"os"

	"github.com/DataManager-Go/DataManagerServer/constants"
	libdm "github.com/DataManager-Go/libdatamanager"
	dmConfig "github.com/DataManager-Go/libdatamanager/config"
	"github.com/JojiiOfficial/gaw"
)

// CommandData data for commands
type CommandData struct {
	LibDM                     *libdm.LibDM
	Keystore                  *libdm.Keystore
	Command                   string
	Config                    *dmConfig.Config
	FileAttributes            libdm.FileAttributes
	Encryption, EncryptionKey string
	Namespace, Keyfile        string
	Details                   uint8
	NameLen, RandKey          int
	All, AllNamespaces        bool
	NoRedaction, OutputJSON   bool
	Yes, Force, Quiet         bool
	EncryptionPassKey         bool
	NoDecrypt, NoEmojis       bool
	EncryptionFromStdin       bool
	VerifyFile                bool
}

// Init init CommandData
func (cData *CommandData) Init() bool {
	// Validate cipher
	if len(cData.Encryption) > 0 && !constants.IsValidCipher(cData.Encryption) {
		fmt.Println("Invalid encryption cipter")
		return false
	}

	// Setup keystore if required
	if cData.Config.KeystoreEnabled() && cData.needKeystore() {
		err := cData.Config.KeystoreDirValid()
		if err != nil {
			if !cData.Config.Client.SkipKeystoreCheck {
				printError("opening keystore", err.Error())
				fmt.Println(`To fix: Be sure the folder is correct or remove "keystoredir" from the config`)
				return false
			}
			if !cData.Quiet && !cData.Config.Client.HideKeystoreWarnings {
				printWarning("opening keystore", err.Error())
			}
		} else {
			// Open keystore
			err = cData.setupKeystore()
			if err != nil {
				printError("opening keystore", err.Error())
				return false
			}
		}
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
		if !cData.GenerateKey() {
			return false
		}
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
	config, err := cData.Config.ToRequestConfig()
	cData.LibDM = libdm.NewLibDM(config)

	// Allow setup, register and login command to continue without
	// noticing the error
	if err != nil && !gaw.IsInStringArray(cData.Command, []string{"setup", "register", "login"}) {
		fmt.Println(err)
		return false
	}

	return true
}

// GenerateKey generates a random key
func (cData *CommandData) GenerateKey() bool {
	if !isValidAESLen(cData.RandKey) {
		fmt.Println("Invalid Keysize", cData.RandKey)
		return false
	}

	// Generate key
	b := randKey(cData.RandKey)
	if b == nil {
		return false
	}

	// Store key in keypath if desired
	path := ""
	if cData.Keystore != nil {
		path = cData.Keystore.Path
	}
	keyFile := genFileName(path, "key")

	// Save keyfile
	err := saveFile(b, keyFile)
	if err != nil {
		printError("saving key", err.Error())
		return false
	}

	fmt.Printf("File %s saved\n", keyFile)

	cData.EncryptionKey = string(b)
	cData.Keyfile = keyFile

	return true
}

func (cData *CommandData) setupKeystore() error {
	// Create new keystore
	kstore := libdm.NewKeystore(cData.Config.Client.KeyStoreDir)
	err := kstore.Open()
	if err != nil {
		return err
	}

	// On success, set keystore
	cData.Keystore = kstore
	return nil
}

// Return true if current command needs a key input`
func (cData *CommandData) supportRandKey() bool {
	return gaw.IsInStringArray(cData.Command, []string{"upload"}) && len(cData.Encryption) > 0
}

// Return true if current command needs a key input
func (cData *CommandData) supportInputKey() bool {
	if cData.supportRandKey() {
		return true
	}

	return gaw.IsInStringArray(cData.Command, []string{"file view", "file download", "file edit"})
}

// Return true if current command needs a key input
func (cData *CommandData) needKeystore() bool {
	if cData.supportInputKey() {
		return true
	}
	return gaw.IsInStringArray(cData.Command, []string{"file rm", "file delete", "rm"})
}
