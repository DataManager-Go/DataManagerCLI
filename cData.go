package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

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
		Namespace:   *appNamespace,
		All:         *appAll,
		NoRedaction: *appNoRedaction,
		OutputJSON:  *appOutputJSON,
		Yes:         *appYes,
		Force:       *appForce,
		NameLen:     appTrimName,

		Encryption: *appFileEncryption,

		NoDecrypt:           *appNoDecrypt,
		NoEmojis:            *appNoEmojis,
		RandKey:             *appFileEncrRandKey,
		Quiet:               *appQuiet,
		VerifyFile:          *appVerify,
		UnmodifiedNamespace: unmodifiedNS,
		Compression:         *appDisableCompression,
		Extract:             *appDecompress,
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

			// TODO add age key generation
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
		switch *appFileEncryption {
		case libdm.EncryptionCiphers[0]:
			if !vaildAESkeylen(randKeySize) {
				fmt.Printf("The keysize %d is invalid\n", len(*appFileEncrKey))
				return nil
			}
		case libdm.EncryptionCiphers[1]:
			if len(*appFileEncrKey) != 62 {
				fmt.Printf("The key \"%s\" is invalid (Invalid keysize)\n", *appFileEncrKey)

				if strings.HasPrefix(*appFileEncrKey, "/") || strings.HasPrefix(*appFileEncrKey, "~/") || strings.HasPrefix(*appFileEncrKey, "./") {
					fmt.Println("\nDid you want to pass a file? use --keyfile")
				}
				return nil
			}
		}
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
	if !fileExists(encrKeyFile) {
		log.Fatal("Keyfile does not exists!")
	}

	// Read key
	var err error
	cData.EncryptionKey, err = ioutil.ReadFile(filepath.Clean(encrKeyFile))
	if err != nil {
		log.Fatal(err)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
