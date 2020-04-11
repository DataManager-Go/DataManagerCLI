package main

import (
	"fmt"
	"io/ioutil"

	"github.com/DataManager-Go/DataManagerCLI/commands"
	libdm "github.com/DataManager-Go/libdatamanager"
)

// generates a commands.Commanddata object based on the cli parameter
func buildCData(parsed string, appTrimName int) *commands.CommandData {
	// Generate  file attributes
	fileAttributes := libdm.FileAttributes{
		Namespace: *appNamespace,
		Groups:    *appGroups,
		Tags:      *appTags,
	}

	// Command data
	commandData := commands.CommandData{
		Command:             parsed,
		Config:              config,
		Details:             uint8(*appDetails),
		FileAttributes:      fileAttributes,
		Namespace:           *appNamespace,
		All:                 *appAll,
		AllNamespaces:       *appAllNamespaces,
		NoRedaction:         *appNoRedaction,
		OutputJSON:          *appOutputJSON,
		Yes:                 *appYes,
		Force:               *appForce,
		NameLen:             appTrimName,
		Encryption:          *appFileEncryption,
		EncryptionPassKey:   *appFileEncryptionPassKey,
		NoDecrypt:           *appNoDecrypt,
		NoEmojis:            *appNoEmojis,
		RandKey:             *appFileEncryptionRandKey,
		Quiet:               *appQuiet,
		EncryptionFromStdin: *appFileEncryptionFromStdin,
		VerifyFile:          *appVerify,
	}

	// Set encryptionKey
	if len(*appFileEncryptionKeyFile) > 0 {
		var err error
		commandData.EncryptionKey, err = ioutil.ReadFile(*appFileEncryptionKeyFile)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		commandData.Keyfile = *appFileEncryptionKeyFile
	} else if len(*appFileEncryptionKey) > 0 {
		commandData.EncryptionKey = []byte(*appFileEncryptionKey)
	}

	if parsed != setupCmd.FullCommand() {
		if !commandData.Init() {
			return nil
		}

	}

	return &commandData
}
