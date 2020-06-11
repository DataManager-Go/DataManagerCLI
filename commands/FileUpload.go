package commands

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/DataManager-Go/libdatamanager"
	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/gosuri/uiprogress"
)

// UploadData data for uploads
type UploadData struct {
	Name          string
	PublicName    string
	FromStdIn     bool
	SetClip       bool
	Public        bool
	ReplaceFile   uint
	DeleteInvalid bool
	TotalFiles    int
	Progress      *uiprogress.Progress
	NoArchiving   bool
}

// UploadItems to the server and set's its affiliations
func (cData *CommandData) UploadItems(uris []string, threads uint, uploadData *UploadData) {
	// Stdin can only be used
	// without additional files
	if uploadData.FromStdIn {
		cData.uploadEntity(*uploadData, "")
		return
	}

	// Build new slice containing the
	// correct file/uri order
	uris = parseURIArgUploadCommand(uris, uploadData.NoArchiving)
	if uris == nil {
		return
	}

	// Setup uploadData
	uploadData.TotalFiles = len(uris)
	uploadData.Progress = uiprogress.New()
	uploadData.Progress.Start()

	// Check source(s)
	if uploadData.TotalFiles == 0 {
		fmt.Println("Either specify one or more files or use --from-stdin to upload from stdin")
		return
	}

	// Verify combinations
	if uploadData.TotalFiles > 1 {
		if uploadData.SetClip {
			fmt.Println("You can't set clipboard while uploading multiple files")
			return
		}

		if len(uploadData.PublicName) > 0 {
			fmt.Println("You can't upload multiple files with the same public name")
		}
	}

	// Upload Files
	cData.runUploadPool(uploadData, uris, threads)
}

// Run parallel Uploads
func (cData *CommandData) runUploadPool(uploadData *UploadData, uris []string, threads uint) {
	// In case a user is dumb,
	// correct him
	if threads == 0 {
		threads = 1
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
				cData.uploadEntity(*uploadData, uri)
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

// Upload upload a URI
func (cData *CommandData) uploadEntity(uploadData UploadData, uri string) (succ bool) {
	var uploadResponse *libdm.UploadResponse
	var err error
	var bar *uiprogress.Bar

	// Set name to filename if not set
	if len(uploadData.Name) == 0 {
		_, fileName := filepath.Split(uri)
		uploadData.Name = fileName
	}

	// Create uploadRequest
	uploadRequest := uploadData.toUploadRequest(cData)

	// Create Uploader
	execUploader := cData.newUploader(&uploadData, uri, uploadRequest, (!cData.Quiet && !uploadData.FromStdIn))

	// Do upload request
	if isHTTPURL(uri) {
		// -----> Upload URL <------

		// We checked if url.Parse is
		// successful in isHTTPURL
		u, _ := url.Parse(uri)
		uploadResponse, err = uploadRequest.UploadURL(u)
		if err != nil {
			printResponseError(err, "uploading url")
			return
		}

		printSuccess("uploaded URL: %s", uri)
	} else if !uploadData.FromStdIn {
		// Get uri info
		s, err := os.Stat(uri)
		if err != nil {
			printError(err, "reading file")
			return
		}

		// Call required lib func.
		// Since we replaced all dir-uris which shouldn't be uploaded
		// archived, we can safely upload all dirs as archive
		if s.IsDir() {
			// -----> Folder <-----
			uploadResponse, bar = execUploader.uploadArchivedFolder()
		} else {
			// -----> File <-----
			// Open file
			f, err := os.Open(uri)
			if err != nil {
				printError("opening file", err.Error())
				return
			}

			// Upload file
			uploadResponse, bar = execUploader.uploadFile(f)
		}
	} else {
		// -----> StdIn <-----
		uploadResponse, bar = execUploader.uploadFromStdin()
	}

	// Return on error
	if uploadResponse == nil {
		return
	}

	// Return result of postUpload
	return cData.runPostUpload(&uploadData, uploadResponse, bar)
}

// Build UploadRequest from UploadData
func (uploadData *UploadData) toUploadRequest(cData *CommandData) *libdatamanager.UploadRequest {
	// Make public if public name was specified
	if len(uploadData.PublicName) > 0 {
		uploadData.Public = true
	}

	// Create upload request
	uploadRequest := cData.LibDM.NewUploadRequest(uploadData.Name, cData.FileAttributes)
	uploadRequest.ReplaceFileID = uploadData.ReplaceFile

	// Encrypt file
	if len(cData.Encryption) > 0 {
		uploadRequest.Encrypted(cData.Encryption, cData.EncryptionKey)
	}

	// Publish file
	if uploadData.Public {
		uploadRequest.MakePublic(uploadData.PublicName)
	}

	return uploadRequest
}

// Hit clipboard, keystore and output trigger
func (cData *CommandData) runPostUpload(uploadData *UploadData, uploadResponse *libdatamanager.UploadResponse, bar *uiprogress.Bar) bool {
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

// Upload helper
type uploader struct {
	cData         *CommandData         // CLI informations
	uploadRequest *libdm.UploadRequest // Prepared uploadrequest
	uri           string               // URI to be uploaded
	uploadData    *UploadData          // Data containing information for the uploaded fil
	showProgress  bool                 // Use a progressbar
}

// Hook func
type uploadFunc func(done chan string, uri string) (*libdm.UploadResponse, error)

// Create new uploader
func (cData *CommandData) newUploader(uploadData *UploadData, uri string, uploadRequest *libdatamanager.UploadRequest, useProgressbar bool) *uploader {
	return &uploader{
		cData:         cData,
		uploadData:    uploadData,
		uploadRequest: uploadRequest,
		uri:           uri,
		showProgress:  useProgressbar,
	}
}

// Upload the uri
func (uploader uploader) upload(uploadFunc uploadFunc) (uploadResponse *libdm.UploadResponse, bar *uiprogress.Bar) {
	var chsum string
	var err error
	var proxy libdm.WriterProxy
	done := make(chan string, 1)

	if uploader.showProgress {
		prefix := "Uploading " + uploader.uploadData.Name
		bar, proxy = buildProgressbar(prefix, uint(len(prefix)))

		// Setup proxy
		uploader.uploadRequest.ProxyWriter = proxy
		uploader.uploadRequest.SetFileSizeCallback(func(size int64) {
			bar.Total = int(size)
		})
	}

	// Call upload hook in background
	go func() {
		c := make(chan string, 1)
		uploadResponse, err = uploadFunc(c, uploader.uri)
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
				uploader.uploadData.Progress.AddBar(bar)
			}
		})()
	}

	// Delete keyfile if upload was canceled
	awaitOrInterrupt(done, func(s os.Signal) {
		if !uploader.cData.Quiet {
			fmt.Println(s)
		}
		uploader.cData.deleteKeyfile()
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

	// Verify checksum
	if !uploader.cData.verifyChecksum(chsum, uploadResponse.Checksum) {
		return
	}

	return uploadResponse, bar
}

// Upload from reader
func (uploader uploader) uploadFromReader(r io.Reader, size int64) (*libdm.UploadResponse, *uiprogress.Bar) {
	return uploader.upload(func(done chan string, uri string) (*libdm.UploadResponse, error) {
		return uploader.uploadRequest.UploadFromReader(r, size, done, nil)
	})
}

// Upload a file
func (uploader uploader) uploadFile(file *os.File) (*libdm.UploadResponse, *uiprogress.Bar) {
	// Get fileinfo
	s, err := file.Stat()
	if err != nil {
		return nil, nil
	}
	defer file.Close()

	// Upload from file reader
	return uploader.uploadFromReader(file, s.Size())
}

// Upload from stdin
func (uploader uploader) uploadFromStdin() (*libdm.UploadResponse, *uiprogress.Bar) {
	return uploader.uploadFromReader(os.Stdin, 0)
}

// Upload archived folder
func (uploader uploader) uploadArchivedFolder() (*libdm.UploadResponse, *uiprogress.Bar) {
	return uploader.upload(func(done chan string, uri string) (*libdm.UploadResponse, error) {
		return uploader.uploadRequest.UploadCompressedFolder(uri, done, nil)
	})
}
