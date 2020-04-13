package commands

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/fatih/color"
)

// determineDecryptionKey  gets the correct decryption key from either the arguments of
// the command or from the keystore
func (cData *CommandData) determineDecryptionKey(resp *http.Response) []byte {
	key := []byte(cData.EncryptionKey)

	// If keystore is enabled and no key was passed, try
	// search in keystore for matching key and use it
	if cData.HasKeystoreSupport() && len(key) == 0 {
		keystore, _ := cData.GetKeystore()
		// Get fileID from header
		fileid, err := strconv.ParseUint(resp.Header.Get(libdm.HeaderFileID), 10, 32)
		if err == nil {
			// Search Key in keystore
			k, err := keystore.GetKey(uint(fileid))
			if err == nil {
				return k
			}
			if strings.HasSuffix(err.Error(), "no such file or directory") {
				fmt.Println("-> Key is in keystore but file was not found!")
			}
		}
	}

	return key
}

// Returns a file and its size. Exit on error
func getFile(uri string) (*os.File, int64) {
	// Open file
	f, err := os.Open(uri)
	if err != nil {
		printError("opening file", err.Error())
		os.Exit(1)
		return nil, 0
	}

	// Get it's stats
	stat, err := f.Stat()
	if err != nil {
		printError("reading file", err.Error())
		os.Exit(1)
	}

	return f, stat.Size()
}

// verifyChecksum return true on success
func (cData *CommandData) verifyChecksum(localCs, remoteCs string) bool {
	// Verify checksum
	if localCs != remoteCs {
		if cData.VerifyFile {
			fmtError("checksums don't match!")
			return false
		}

		fmt.Printf("%s checksums don't match!\n", color.YellowString("Warning"))
		if !cData.Quiet {
			fmt.Printf("Local CS:\t%s\n", localCs)
			fmt.Printf("Rem. CS:\t%s\n", remoteCs)
		}
	}

	return true
}

func (cData *CommandData) printChecksumError(resp *libdm.FileDownloadResponse) {
	fmt.Printf("%s checksums don't match!\n", color.YellowString("Warning"))
	if !cData.Quiet {
		fmt.Printf("Local CS:\t%s\n", resp.LocalChecksum)
		fmt.Printf("Rem. CS:\t%s\n", resp.ServerChecksum)
	}
}

func editFile(file string) bool {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "/usr/bin/nano"
	}

	// Check editor
	if _, err := os.Stat(editor); err != nil {
		fmtError("finding editor. Either install nano or set $EDITOR to your desired editor")
		return false
	}

	// Launch editor
	cmd := exec.Command(editor, file)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	// Wait for it to finish
	err := cmd.Run()

	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

func parseURIArgUploadCommand(uris []string) []string {
	var newURIList []string
	for i := range uris {
		// Skip urls
		if u, err := url.Parse(uris[i]); err == nil && gaw.IsInStringArray(u.Scheme, []string{"http", "https"}) {
			newURIList = append(newURIList, uris[i])
			continue
		}

		s, err := os.Stat(uris[i])
		if err != nil {
			fmt.Println("Skipping", uris[i], err.Error())
			continue
		}

		if s.IsDir() {
			dd, err := ioutil.ReadDir(uris[i])
			if err != nil {
				printError("reading directory", err.Error())
				return nil
			}
			for j := range dd {
				if !dd[j].IsDir() {
					newURIList = append(newURIList, filepath.Join(uris[i], dd[j].Name()))
				}
			}
		} else {
			newURIList = append(newURIList, uris[i])
		}
	}

	return newURIList
}
