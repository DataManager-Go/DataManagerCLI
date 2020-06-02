package commands

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/atotto/clipboard"
	"github.com/gosuri/uiprogress"
)

// UploadData data for uploads
type UploadData struct {
	Name          string
	Publicname    string
	FromStdIn     bool
	SetClip       bool
	Public        bool
	ReplaceFile   uint
	DeleteInvalid bool
	TotalFiles    int
	Progress      *uiprogress.Progress
}

// UploadFile uploads the given file to the server and set's its affiliations
func (cData *CommandData) UploadFile(uris []string, threads uint, uploadData *UploadData) {
	// Extract directories
	uris = parseURIArgUploadCommand(uris)
	if uris == nil {
		return
	}

	// Setup uploadData
	uploadData.TotalFiles = len(uris)
	uploadData.Progress = uiprogress.New()
	uploadData.Progress.Start()

	if uploadData.TotalFiles == 0 && !uploadData.FromStdIn {
		fmt.Println("Either specify one or more files or use --from-stdin to upload from stdin")
		return
	}

	// In case a user is dumb,
	// correct him
	if threads == 0 {
		threads = 1
	}

	// Verify combinations
	if uploadData.TotalFiles > 1 {
		if uploadData.FromStdIn {
			fmt.Println("Can't upload from stdin and files at the same time")
			return
		}
		if uploadData.SetClip {
			fmt.Println("You can't set clipboard while uploading multiple files")
			return
		}
		if len(uploadData.Publicname) > 0 {
			fmt.Println("You can't upload multiple files with the same public name")
		}
	}

	// Waitgroup to wait for all "threads" to be done
	wg := sync.WaitGroup{}
	// Channel for managing amount of parallel upload processes
	c := make(chan uint, 1)

	if threads > uint(uploadData.TotalFiles) {
		threads = uint(uploadData.TotalFiles)
	}

	c <- threads
	var pos int

	// Start Uploader pool
	for pos < uploadData.TotalFiles {
		read := <-c
		for i := 0; i < int(read) && pos < uploadData.TotalFiles; i++ {
			wg.Add(1)

			go func(uri string) {
				cData.upload(uploadData, uri)

				wg.Done()
				c <- 1
			}(uris[pos])

			pos++
		}
	}

	// Wait for all
	// threads to be done
	wg.Wait()
}

// Upload file uploads a file
func (cData *CommandData) uploadFile(uploadRequest *libdm.UploadRequest, uploadData *UploadData, uri string) (uploadResponse *libdm.UploadResponse, bar *uiprogress.Bar) {
	var r io.Reader
	var size int64
	var chsum string
	var err error
	var proxy libdm.WriterProxy
	done := make(chan string, 1)

	// Select upload source reader
	if !uploadData.FromStdIn {
		// Open file
		var f *os.File
		f, size = getFile(uri)
		defer f.Close()

		r = f
	} else {
		r = os.Stdin
	}

	// Progressbar setup
	if !cData.Quiet && !uploadData.FromStdIn {
		prefix := "Uploading " + uploadData.Name
		bar, proxy = buildProgressbar(prefix, uint(len(prefix)))

		// Setup proxy
		uploadRequest.ProxyWriter = proxy
		uploadRequest.SetFileSizeCallback(func(size int64) {
			bar.Total = int(size)
		})
	}

	// Start uploading
	go func() {
		c := make(chan string, 1)
		uploadResponse, err = uploadRequest.UploadFromReader(r, size, c, nil)
		done <- <-c
	}()

	if bar != nil {
		// Show bar after 500ms if upload
		// is not done by then
		go (func() {
			time.Sleep(500 * time.Millisecond)
			select {
			case <-done:
			default:
				uploadData.Progress.AddBar(bar)
			}
		})()
	}

	// Delete keyfile if upload was canceled
	awaitOrInterrupt(done, func(s os.Signal) {
		if !cData.Quiet {
			fmt.Println(s)
		}
		cData.deleteKeyfile()
		os.Exit(1)
	}, func(checksum string) {
		// On file upload done set chsum to received checksum
		chsum = checksum
	})

	// Handle upload errors
	if err != nil {
		printResponseError(err, "uploading file")
		return
	}

	// Checksum is not supposed to be empty If any known error
	// were thrown, this part wouldn't be executed
	if len(chsum) == 0 {
		fmt.Println("Unexpected error occured")
		return
	}

	// Verify checksum
	if !cData.verifyChecksum(chsum, uploadResponse.Checksum) {
		return
	}

	return uploadResponse, bar
}

// Upload uploads a file or a url
func (cData *CommandData) upload(uploadData *UploadData, uri string) (succ bool) {
	_, fileName := filepath.Split(uri)
	if len(uploadData.Name) != 0 {
		fileName = uploadData.Name
	}

	// Make public if public name was specified
	if len(uploadData.Publicname) > 0 {
		uploadData.Public = true
	}

	// Create upload request
	uploadRequest := cData.LibDM.NewUploadRequest(fileName, cData.FileAttributes)
	uploadRequest.ReplaceFileID = uploadData.ReplaceFile
	if len(cData.Encryption) > 0 {
		uploadRequest.Encrypted(cData.Encryption, cData.EncryptionKey)
	}
	if uploadData.Public {
		uploadRequest.MakePublic(uploadData.Publicname)
	}

	var uploadResponse *libdm.UploadResponse
	var bar *uiprogress.Bar

	// Do upload request
	if u, err := url.Parse(uri); err == nil && gaw.IsInStringArray(u.Scheme, []string{"http", "https"}) {
		// -----> Upload URL <------
		uploadResponse, err = uploadRequest.UploadURL(u)
		if err != nil {
			printResponseError(err, "uploading url")
			return
		}

		printSuccess("uploaded URL: %s", uri)
	} else {
		// -----> Upload file/stdin <-----
		uploadResponse, bar = cData.uploadFile(uploadRequest, uploadData, uri)
		if uploadResponse == nil {
			return
		}
	}

	// Set clipboard to public file if required
	if uploadData.SetClip && len(uploadResponse.PublicFilename) > 0 {
		cData.setClipboard(uploadResponse.PublicFilename)
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
		return true
	}

	// Render table with informations
	cData.printUploadResponse(uploadResponse, (cData.Quiet || uploadData.TotalFiles > 1), bar)
	return true
}

func (cData *CommandData) setClipboard(publicName string) bool {
	if clipboard.Unsupported {
		fmt.Println("Clipboard not supported on this OS")
		return false
	}
	err := clipboard.WriteAll(cData.Config.GetPreviewURL(publicName))
	if err != nil {
		printError("setting clipboard", err.Error())
		return false
	}
	return true
}
