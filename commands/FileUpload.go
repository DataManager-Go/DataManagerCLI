package commands

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/atotto/clipboard"
	"github.com/gosuri/uiprogress"
)

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

// Upload file uploads a file
func uploadFile(cData *CommandData, uploadRequest *libdm.UploadRequest, uri string, fromStdin bool, totalFiles int, progress *uiprogress.Progress) (uploadResponse *libdm.UploadResponse) {
	var r io.Reader
	var size int64
	var chsum string
	fsDer := make(chan int64, 1)
	done := make(chan string, 1)
	proxy := libdm.NoProxyWriter
	var err error
	var bar *uiprogress.Bar

	// Select upload source reader
	if !fromStdin {
		// Open file
		var f *os.File
		f, size = getFile(uri)
		defer f.Close()

		r = f
	} else {
		r = os.Stdin
	}

	// Progressbar setup
	if !cData.Quiet && !fromStdin {
		prefix := "Uploading " + uri
		bar, proxy = buildProgressbar(prefix, uint(len(prefix)))
	}

	// Start uploading
	go func() {
		c := make(chan string, 1)
		uploadResponse, err = uploadRequest.UploadFromReader(r, size, fsDer, proxy, c)
		done <- <-c
	}()

	if bar != nil {
		bar.Total = int(<-fsDer)

		// Show bar after 500ms if upload
		// wasn't canceled until then
		go (func() {
			time.Sleep(500 * time.Millisecond)
			select {
			case <-done:
			default:
				progress.AddBar(bar)
			}
		})()
	}

	// make channel to listen for kill signals
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt, os.Kill, syscall.SIGTERM)

	select {
	case killsig := <-kill:
		// Delete keyfile if upload was canceled
		if !cData.Quiet {
			fmt.Println(killsig)
		}
		cData.deleteKeyfile()
		os.Exit(1)
		return
	case chsum = <-done:
	}

	if err != nil {
		printError("uploading file", err.Error())
		return
	}

	// Checksum is not supposed to be empty
	// If any known error were thrown, this
	// part wouldn't be executed
	if len(chsum) == 0 {
		fmt.Println("Unexpected error occured")
		return
	}

	// Verify checksum
	if !cData.verifyChecksum(chsum, uploadResponse.Checksum) {
		return
	}

	return uploadResponse
}

// Upload uploads a file or a url
func upload(cData *CommandData, uri string, name, publicName string, public, fromStdin, setClip bool, replaceFile uint, deletInvalid bool, totalFiles int, progress *uiprogress.Progress) {
	_, fileName := filepath.Split(uri)
	if len(name) != 0 {
		fileName = name
	}

	// Make public if public name was specified
	if len(publicName) > 0 {
		public = true
	}

	// Create upload request
	uploadRequest := cData.LibDM.NewUploadRequest(fileName, cData.FileAttributes)
	uploadRequest.ReplaceFileID = replaceFile
	if len(cData.Encryption) > 0 {
		uploadRequest.Encrypted(cData.Encryption, cData.EncryptionKey)
	}
	if public {
		uploadRequest.MakePublic(publicName)
	}

	var uploadResponse *libdm.UploadResponse

	// Do upload request
	if u, err := url.Parse(uri); err == nil && gaw.IsInStringArray(u.Scheme, []string{"http", "https"}) {
		// -----> Upload URL <------
		uploadResponse, err = uploadRequest.UploadURL(u)
		if err != nil {
			printError("uploading url", err.Error())
			return
		}

		printSuccess("uploaded URL: %s", uri)
	} else {
		// -----> Upload file/stdin <-----
		uploadResponse = uploadFile(cData, uploadRequest, uri, fromStdin, totalFiles, progress)
		if uploadResponse == nil {
			return
		}
	}

	// Set clipboard to public file if required
	if setClip && len(uploadResponse.PublicFilename) > 0 {
		if clipboard.Unsupported {
			fmt.Println("Clipboard not supported on this OS")
		} else {
			err := clipboard.WriteAll(cData.Config.GetPreviewURL(uploadResponse.PublicFilename))
			if err != nil {
				printError("setting clipboard", err.Error())
			}
		}
	}

	// Add key to keystore
	if cData.HasKeystoreSupport() && len(cData.Keyfile) > 0 {
		keystore, _ := cData.GetKeystore()
		err := keystore.AddKey(uploadResponse.FileID, cData.Keyfile)
		if err != nil {
			printError("writing keystore", err.Error())
		}
	}

	// Print output
	// Print response as json
	if cData.OutputJSON {
		fmt.Println(toJSON(uploadResponse))
		return
	}
	// Render table with informations
	cData.printUploadResponse(uploadResponse, (cData.Quiet || totalFiles > 1))
}
