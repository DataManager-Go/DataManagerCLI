package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DataManager-Go/DataManagerCLI/commands"
	libdm "github.com/DataManager-Go/libdatamanager"
)

// Generates a commands.Commanddata object based on the cli parameter
func buildCData(parsed string, appTrimName int) *commands.CommandData {
	// Command data
	commandData := commands.CommandData{
		Command: parsed,
		Config:  config,
		Details: uint8(*appDetails),
		FileAttributes: libdm.FileAttributes{
			Namespace: *appNamespace,
			Groups:    *appGroups,
			Tags:      *appTags,
		},
		Namespace:     *appNamespace,
		All:           *appAll,
		AllNamespaces: *appAllNamespaces,
		NoRedaction:   *appNoRedaction,
		OutputJSON:    *appOutputJSON,
		Yes:           *appYes,
		Force:         *appForce,
		NameLen:       appTrimName,

		Encryption: *appFileEncryption,

		NoDecrypt:  *appNoDecrypt,
		NoEmojis:   *appNoEmojis,
		RandKey:    *appFileEncrRandKey,
		Quiet:      *appQuiet,
		VerifyFile: *appVerify,
	}

	// Init cdata
	if !commandData.Init() {
		return nil
	}

	// Initialize encryption sources
	return initInputKey(commandData)
}

// ----- Init en/decryption ------

func initInputKey(cData commands.CommandData) *commands.CommandData {
	// --> RandKey
	randKeySize := *appFileEncrRandKey
	if randKeySize > 0 && cData.RequestedEncryptionInput() {
		// Check correct keylen for given encryption
		switch *appFileEncryption {
		case libdm.EncryptionCiphers[0]:
			// AES
			if !vaildAESkeylen(randKeySize) {
				fmt.Printf("The keysize %d is invalid\n", randKeySize)
				return nil
			}
		}

		// Generate key
		err := initRandomKey(&cData)
		if err != nil {
			log.Fatal(err)
		}
	}

	// --> Stdin
	if *appFileEncrKeyFromStdin {
		cData.EncryptionKey = readStdinWithTimeout(48)
	}

	// TODO password

	// --> Keyfile
	encrKeyFile := *appFileEncrKeyFile
	if len(encrKeyFile) > 0 {
		initKeyfile(encrKeyFile, &cData)
	}

	// FlagInput --key
	if len(*appFileEncrKey) > 0 {
		cData.EncryptionKey = []byte(*appFileEncrKey)
	}

	return &cData
}

// Generate and save a random key
func initRandomKey(cData *commands.CommandData) error {
	// Generate a random key
	cData.EncryptionKey = randKey(cData.RandKey)
	path := "./"

	// use keystorepath if keystore is enabled
	if keystore, _ := cData.GetKeystore(); keystore != nil {
		path = keystore.Path
	}

	// Generate file and save key
	cData.Keyfile = genFile(path, "key")
	return ioutil.WriteFile(cData.Keyfile, cData.EncryptionKey, 0600)
}

// Read keyfile to cData.EncryptionKey
func initKeyfile(encrKeyFile string, cData *commands.CommandData) {
	// Check if file exists
	_, err := os.Stat(encrKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	// Read key
	cData.EncryptionKey, err = ioutil.ReadFile(encrKeyFile)
	if err != nil {
		log.Fatal(err)
	}
}
