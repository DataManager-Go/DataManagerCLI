package commands

import (
	"fmt"
	"net/http"
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

func (cData *CommandData) getChecksumError(resp *libdm.FileDownloadResponse) string {
	var s string
	s += fmt.Sprintf("%s checksums don't match!\n", color.YellowString("Warning"))
	if !cData.Quiet {
		s += fmt.Sprintf("Local CS:\t%s\n", resp.LocalChecksum)
		s += fmt.Sprintf("Rem. CS:\t%s\n", resp.ServerChecksum)
	}
	return s
}

func (cData *CommandData) printChecksumError(resp *libdm.FileDownloadResponse) {
	fmt.Println(cData.getChecksumError(resp))
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

func parseURIArgUploadCommand(uris []string, noCompress bool) []string {
	var newURIList []string
	for i := range uris {
		uriPath := gaw.ResolveFullPath(uris[i])

		// Skip urls
		if isHTTPURL(uris[i]) {
			newURIList = append(newURIList, uris[i])
			continue
		}

		s, err := os.Stat(uriPath)
		if err != nil {
			fmt.Println("Skipping", uriPath, err.Error())
			continue
		}

		// Using --no-compress means uploading all
		// files inside a given folder
		if s.IsDir() && noCompress {
			// Get all files in uriPath
			files, err := gaw.ListDir(uriPath, true)
			if err != nil {
				printError("Listing dir", err.Error())
				return nil
			}

			newURIList = append(newURIList, files...)
		} else {
			newURIList = append(newURIList, uriPath)
		}
	}

	return newURIList
}

func resolveOutputFile(fileName, outputFile string) string {
	localFile := gaw.ResolveFullPath(outputFile)

	// Replace fileSeparators to prevent writing file to an other directory
	fileName = strings.ReplaceAll(fileName, string(filepath.Separator), "-")

	// Append original filename to
	// specified local path
	if strings.HasSuffix(outputFile, "/") {
		// If no special file was choosen
		localFile = filepath.Join(outputFile, fileName)
	} else {
		stat, err := os.Stat(localFile)
		if err == nil && stat.IsDir() {
			localFile = filepath.Join(localFile, fileName)
		}
	}

	return localFile
}

func sortFiles(sOrder string, files []*libdm.FileResponseItem) bool {
	// Order output
	if len(sOrder) > 0 {
		if order := FileOrderFromString(sOrder); order != nil {
			// Sort
			NewFileSorter(files).
				Reversed(IsOrderReversed(sOrder)).
				SortBy(*order)
		} else {
			fmtError(fmt.Sprintf("sort by '%s' not supporded", sOrder))
			return false
		}
	} else {
		// By default sort by creation desc
		NewFileSorter(files).Reversed(true).SortBy(CreatedOrder)
	}

	return true
}

func fileSliceToRef(inpItems []libdm.FileResponseItem) []*libdm.FileResponseItem {
	var respsl []*libdm.FileResponseItem

	for i := range inpItems {
		respsl = append(respsl, &inpItems[i])
	}

	return respsl
}
