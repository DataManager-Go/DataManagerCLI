package commands

import (
	"fmt"
	"os"
	"strings"

	libdm "github.com/DataManager-Go/libdatamanager"
	dmConfig "github.com/DataManager-Go/libdatamanager/config"
	"github.com/JojiiOfficial/gaw"
	"github.com/fatih/color"
	"github.com/gosuri/uiprogress"
	"github.com/sbani/go-humanizer/units"
	clitable "gopkg.in/benweidig/cli-table.v2"
)

// CommandData data for commands
type CommandData struct {
	LibDM   *libdm.LibDM
	Command string
	Config  *dmConfig.Config

	// Encryption
	keystore            *libdm.Keystore
	EncryptionKey       []byte
	Encryption, Keyfile string
	RandKey             int

	Namespace               string
	UnmodifiedNamespace     string
	FileAttributes          libdm.FileAttributes
	Details                 uint8
	NameLen                 int
	All, AllNamespaces      bool
	NoRedaction, OutputJSON bool
	Yes, Force, Quiet       bool
	NoDecrypt, NoEmojis     bool
	VerifyFile              bool
	NoCompression           bool
}

// Init init CommandData
func (cData *CommandData) Init() bool {
	// Get requestconfig
	// Allow setup, register and login command to continue without
	// handling the error

	var config *libdm.RequestConfig
	if cData.Config != nil {
		var err error
		config, err = cData.Config.ToRequestConfig()
		if err != nil && !gaw.IsInStringArray(cData.Command, []string{"setup", "register", "login"}) {
			fmt.Println(err)
			return false
		}
	}

	// Create new dmanager lib object
	cData.LibDM = libdm.NewLibDM(config)

	// return success
	return true
}

func (cData *CommandData) deleteKeyfile() {
	if len(cData.Keyfile) > 0 {
		ShredderFile(cData.Keyfile, -1)
		if !cData.Quiet {
			fmt.Println("Deleting unused key", cData.Keyfile)
		}
	}
}

// RequestedEncryptionInput determine if encryption input was requested
func (cData *CommandData) RequestedEncryptionInput() bool {
	return len(cData.Encryption) > 0
}

// GetKeystore returns the keystore for user
func (cData *CommandData) GetKeystore() (*libdm.Keystore, error) {
	// Check if keystore is valid
	if cData.Config.KeystoreDirValid() != nil || !cData.Config.KeystoreEnabled() {
		return nil, nil
	}

	// If keystore is nil, try to open it
	if cData.keystore == nil {
		// Check if keystore is enabled
		if cData.Config.KeystoreEnabled() {
			// Check if keystore config is valid
			if err := cData.Config.KeystoreDirValid(); err != nil {
				return nil, err
			}

			// Open and set keystore
			var err error
			cData.keystore, err = cData.Config.GetKeystore()
			if err != nil {
				return nil, err
			}
		}
	}

	return cData.keystore, nil
}

// CloseKeystore closes keystoree
func (cData *CommandData) CloseKeystore() {
	cData.keystore.Close()
}

// HasKeystoreSupport return true if kesytore is set up
// correctly and is enabled
func (cData *CommandData) HasKeystoreSupport() bool {
	ks, err := cData.GetKeystore()
	return ks != nil && err == nil
}

// Print nice output for a file upload
// If total files is > 1 only a summary is shown
func (cData CommandData) printUploadResponse(ur *libdm.UploadResponse, short bool, bar *uiprogress.Bar) {
	if short {
		var text string
		if len(ur.PublicFilename) > 0 {
			text = fmt.Sprintf("%s %d; %s %s;\t%s %s", color.HiGreenString("ID:"), ur.FileID, color.HiGreenString("Name:"), ur.Filename, color.HiGreenString("Public url:"), cData.Config.GetPreviewURL(ur.PublicFilename))
		} else {
			text = fmt.Sprintf("%s %d; %s %s", color.HiGreenString("ID"), ur.FileID, color.HiGreenString("Name:"), ur.Filename)
		}

		printBar(text, bar)

		return
	}

	table := clitable.New()
	table.ColSeparator = " "
	table.Padding = 4

	table.AddRow([]interface{}{color.HiGreenString("FileID:"), ur.FileID}...)
	if len(ur.PublicFilename) > 0 {
		table.AddRow([]interface{}{color.HiGreenString("Public url:"), cData.Config.GetPreviewURL(ur.PublicFilename)}...)
	}

	table.AddRow([]interface{}{color.HiGreenString("File name:"), ur.Filename}...)

	table.AddRow([]interface{}{color.HiGreenString("Namespace:"), ur.Namespace}...)
	table.AddRow([]interface{}{color.HiGreenString("Size:"), units.BinarySuffix(float64(ur.FileSize))}...)
	table.AddRow([]interface{}{color.HiGreenString("Checksum:"), ur.Checksum}...)

	printBar(table.String(), bar)
}

func namespaceOverwritten() bool {
	if len(os.Args) < 2 {
		return false
	}

	cmdj := strings.Join(os.Args[1:], ",")
	return (strings.Contains(cmdj, "--namespace,") || strings.Contains(cmdj, "-n,"))
}

// Get real set namespace. If no --namespace was provided "" will be returned
func (cData *CommandData) getRealNamespace() string {
	if !namespaceOverwritten() {
		return ""
	}

	return cData.UnmodifiedNamespace
}
