package commands

import (
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DataManager-Go/DataManagerServer/constants"
	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
)

// determineDecryptionKey  gets the correct decryption key from either the arguments of
// the command or from the keystore
func determineDecryptionKey(cData *CommandData, resp *http.Response) []byte {
	key := []byte(cData.EncryptionKey)

	// If keystore is enabled and no key was passed, try
	// search in keystore for matching key and use it
	if cData.Config.KeystoreEnabled() && len(key) == 0 && cData.Keystore != nil {
		// Get fileID from header
		fileid, err := strconv.ParseUint(resp.Header.Get(libdm.HeaderFileID), 10, 32)
		if err == nil {
			// Search Key in keystore
			k, err := cData.Keystore.GetKey(uint(fileid))
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

// Saves data from r to file. Shows progressbar after 500ms if still saving
func saveFileFromStream(outFile, encryption string, key []byte, r io.Reader, c chan error, bar *pb.ProgressBar) chan string {
	doneChan := make(chan string, 1)

	go (func() {
		// Create or truncate file
		f, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		defer f.Close()
		if err != nil {
			c <- err
			return
		}

		buf := make([]byte, 10*1024)
		hash := crc32.NewIEEE()

		// Write file
		switch encryption {
		case constants.EncryptionCiphers[0]:
			{
				err = libdm.Decrypt(r, f, hash, key, buf)
			}
		case "":
			{
				w := io.MultiWriter(f, hash)
				_, err = io.CopyBuffer(w, r, buf)
			}
		}
		if err != nil {
			c <- err
			return
		}

		fmt.Println("write chsum")
		doneChan <- hex.EncodeToString(hash.Sum(nil))
	})()

	// Show bar if desired
	// and not already done after 500ms
	if bar != nil {
		go (func() {
			time.Sleep(500 * time.Millisecond)
			select {
			case <-c:
			default:
				bar.Start()
			}
		})()
	}

	return doneChan
}

func guiPreview(cData *CommandData, serverFileName, encryption, checksum string, resp *http.Response, respData io.Reader, bar *pb.ProgressBar) string {
	done := make(chan bool)
	errCh := make(chan error)

	// Generate tempfile
	file := GetTempFile(serverFileName)

	// Save stream and decrypt if necessary
	doneCh := saveFileFromStream(file, encryption, determineDecryptionKey(cData, resp), respData, errCh, bar)

	// Show bar only if uploading takes more than 500ms
	if bar != nil {
		go (func() {
			time.Sleep(500 * time.Millisecond)
			select {
			case <-done:
				break
			default:
				bar.Start()
			}
		})()
	}

	var chsum string

	// Wait for download to be finished
	// or an error to occur
	select {
	case err := <-errCh:
		if err = <-errCh; err != nil {
			printError("while downloading", err.Error())
			fmt.Println(err)
			return ""
		}
	case chsum = <-doneCh:
	}

	if !verifyChecksum(cData, chsum, checksum) {
		return ""
	}

	// Close bar if open
	if bar != nil {
		bar.Finish()
	}

	// Preview file
	previewFile(file)
	return file
}

// verifyChecksum return true on success
func verifyChecksum(cData *CommandData, localCs, remoteCs string) bool {
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
