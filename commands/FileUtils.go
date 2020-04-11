package commands

import (
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
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

// Saves data from r to file. Shows progressbar after 500ms if still saving
func saveFileToFile(outFile, encryption string, key []byte, r io.Reader, c chan error, bar *pb.ProgressBar) (chan string, *os.File) {
	f, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		c <- err
		return nil, nil
	}

	return writeFileToWriter(f, encryption, key, r, c, bar), f
}

// Saves data from r to file. Shows progressbar after 500ms if still saving
func writeFileToWriter(wr io.Writer, encryption string, key []byte, r io.Reader, c chan error, bar *pb.ProgressBar) chan string {
	doneChan := make(chan string, 1)

	go (func() {
		buf := make([]byte, 10*1024)
		hash := crc32.NewIEEE()
		var err error
		// Write file
		switch encryption {
		case constants.EncryptionCiphers[0]:
			{
				err = libdm.Decrypt(r, wr, hash, key, buf)
			}
		case "":
			{
				w := io.MultiWriter(wr, hash)
				_, err = io.CopyBuffer(w, r, buf)
			}
		}
		if err != nil {
			fmt.Println(err)
			c <- err
			return
		}

		doneChan <- hex.EncodeToString(hash.Sum(nil))
	})()

	// Show bar if desired
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
	doneCh, f := saveFileToFile(file, encryption, determineDecryptionKey(cData, resp), respData, errCh, bar)
	defer f.Close()

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

	if !cData.verifyChecksum(chsum, checksum) {
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

func uploadFileCommand(cData *CommandData, uploadRequest *libdm.UploadRequest, uri string, fromStdin bool) (uploadResponse *libdm.UploadResponse) {
	var r io.Reader
	var size int64
	var bar *pb.ProgressBar
	var chsum string
	fsDer := make(chan int64, 1)
	done := make(chan string, 1)
	proxy := libdm.NoProxyWriter
	var err error

	if !fromStdin {
		// Open file
		f, err := os.Open(uri)
		if err != nil {
			printError("opening file", err.Error())
			return
		}

		stat, err := f.Stat()
		if err != nil {
			printError("reading file", err.Error())
		}

		size = stat.Size()
		r = f
	} else {
		r = os.Stdin
	}

	if !cData.Quiet && !fromStdin {
		bar = pb.New64(0).SetMaxWidth(100)
		proxy = func(w io.Writer) io.Writer {
			return bar.NewProxyWriter(w)
		}
	}

	// Start uploading
	go func() {
		c := make(chan string, 1)
		uploadResponse, err = uploadRequest.UploadFromReader(r, size, fsDer, proxy, c)
		done <- <-c
	}()

	if bar != nil {
		bar.SetTotal(<-fsDer)

		// Show bar after 500ms if uploa
		// wasn't canceled until then
		go (func() {
			time.Sleep(500 * time.Millisecond)
			select {
			case <-done:
			default:
				bar.Start()
			}
		})()
	}

	// make channel to listen for kill signals
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt, os.Kill, syscall.SIGTERM)

	select {
	case killsig := <-kill:
		// Delete keyfile if upload was canceled
		if bar != nil {
			bar.Finish()
		}
		if !cData.Quiet {
			fmt.Println(killsig)
		}
		cData.deleteKeyfile()
		return
	case chsum = <-done:
	}

	if bar != nil {
		bar.Finish()
	}

	if len(chsum) == 0 {
		fmt.Println("Unexpected error occured")
		return
	}

	if err != nil {
		printError("uploading file", err.Error())
		return
	}

	if !cData.verifyChecksum(chsum, uploadResponse.Checksum) {
		return
	}

	return uploadResponse
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
