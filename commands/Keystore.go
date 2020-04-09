package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/JojiiOfficial/shred"
	"github.com/dustin/go-humanize/english"
	"github.com/fatih/color"
	"github.com/pkg/errors"
)

var (
	// ErrAbortDeletion if user canceled interaction
	ErrAbortDeletion = errors.New("aborted")
)

// CreateKeystore create a keystore
func CreateKeystore(cData CommandData, path string, overwrite bool) {
	// Check if valid keystore is available
	if err := cData.Config.KeystoreDirValid(); err == nil && cData.Config.KeystoreEnabled() {
		fmt.Println("You have already create a keystore. You have to delete it before you can cerate a new keystore!")
		return
	}

	// Create keystore dir if not already exists
	err := os.MkdirAll(path, 0700)
	if err != nil {
		printError("creating keystore", err.Error())
		return
	}

	// Only allow non empty directories as keystore
	isempty, err := isEmpty(path)
	if err != nil {
		printError("reading directory", err.Error())
		return
	}

	if !isempty {
		fmtError("You can't use a non empty directory as keystore!")
		return
	}

	// Open keystore and create the database
	keystore := libdm.NewKeystore(path)
	err = keystore.Open()
	defer keystore.Close()
	if err != nil {
		printError("opening keystore", err.Error())
		return
	}

	// Set new keystore and save config
	err = cData.Config.SetKeystoreDir(path)
	if err != nil {
		printError("saving config", err.Error())
		return
	}

	fmt.Printf("%s created keystore\n", color.HiGreenString("Successfully"))
}

// KeystoreInfo shows info for keystore
func KeystoreInfo(cData CommandData) {
	if !checkKeystoreAvailable(&cData) {
		return
	}

	// Open keystore
	keystore, err := cData.Config.GetKeystore()
	defer keystore.Close()
	if err != nil {
		printError("opening keystore", err.Error())
		return
	}

	// Retrieve data
	items, err := keystore.GetValidKeyCount()
	if err != nil {
		printError("getting info", err.Error())
		return
	}

	// Print info output
	fmt.Printf("Keystore:\t%s\n", keystore.Path)
	fmt.Printf("Keys:\t\t%d\n", items)
}

// KeystoreDelete delete a keystore
func KeystoreDelete(cData CommandData, shredderCount uint) {
	if !checkKeystoreAvailable(&cData) {
		return
	}

	// Request confirmation
	if !cData.Yes {
		y, _ := gaw.ConfirmInput("Do you want to delete your current keystore and all it's keys? (y/n)> ", bufio.NewReader(os.Stdin))
		if !y {
			return
		}
	}

	fmt.Printf("Going to shredder the directory '%s' and all its subdirectories!!\nWaiting 4 seconds. Press Crtl+c to cancel\n", cData.Config.Client.KeyStoreDir)
	time.Sleep(4 * time.Second)

	// Open keystore
	keystore, err := cData.Config.GetKeystore()
	defer keystore.Close()

	// If keystore directory was not found, remove it from the config
	if strings.HasSuffix(err.Error(), "no such file or directory") {
		err := cData.Config.UnsetKeystoreDir()
		if err != nil {
			printError("saving config", err.Error())
		}
		return
	}

	if err != nil {
		printError("opening keystore", err.Error())
		return
	}

	// Shredder all keys and key DB file
	shredder := shred.Shredder{}
	shredderConf := shred.NewShredderConf(&shredder, shred.WriteRand|shred.WriteRandSecure|shred.WriteZeros, int(shredderCount), true)
	err = shredderConf.ShredDir(keystore.Path)
	if err != nil {
		printError("shreddering files", err.Error())
		return
	}

	// Remove old keystore directory
	err = os.Remove(keystore.Path)
	if err != nil {
		printError("removing path", err.Error())
	}

	// Remove path from config
	err = cData.Config.UnsetKeystoreDir()
	if err != nil {
		printError("updating config", err.Error())
		return
	}

	fmt.Printf("%s deleted your keystore", color.HiGreenString("Successfully"))
}

// KeystoreCleanup cleansup a keystore
func KeystoreCleanup(cData CommandData, shredderCount uint) {
	if !checkKeystoreAvailable(&cData) {
		return
	}

	// Opening keystore
	keystore, err := cData.Config.GetKeystore()
	defer keystore.Close()
	if err != nil {
		printError("opening keystore", err.Error())
		return
	}

	// Get all keystore files
	files, err := keystore.GetFiles()
	if err != nil {
		printError("getting files", err.Error())
		return
	}

	var cleanedFiles int

	// Find entries without a valid file
	// and remove them
	for i := range files {
		path := keystore.GetKeystoreFile(files[i].Key)
		// Delete entry if file wasn't found
		if _, err := os.Stat(path); err != nil {
			keystore.DeleteKey(files[i].FileID)
			cleanedFiles++
		}
	}

	if cleanedFiles == 0 {
		fmt.Println("Nothing to do")
	} else {
		fmt.Printf("%s cleaned %s", color.HiGreenString("Successfully"), english.Plural(cleanedFiles, "entry", "entries"))
	}
}

func printNoKeystoreFound() {
	fmt.Println("You don't have a valid keystore!")
}

// checkKeystoreAvailable checks if a keystore is available
// returns true if a valid keystore was found
func checkKeystoreAvailable(cData *CommandData) bool {
	// Check keystore dir setting
	if !cData.Config.KeystoreEnabled() {
		printNoKeystoreFound()
		return false
	}

	// If found, check if valid
	if err := cData.Config.KeystoreDirValid(); err != nil {
		printError("verifying keystore", err.Error())
		return false
	}

	return true
}
