package commands

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/DataManager-Go/libdatamanager"
	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gopool"
)

// UploadData data for uploads
type UploadData struct {
	Name            string
	PublicName      string
	FromStdIn       bool
	SetClip         bool
	Public          bool
	ReplaceFileID   uint
	ReplaceSameName bool
	All             bool
	DeleteInvalid   bool
	TotalFiles      int
	ProgressView    *ProgressView
	NoArchiving     bool

	customName      bool
	uploadAsArchive bool
	maxItemLen      int
}

// UploadItems to the server and set's its affiliations
func (cData *CommandData) UploadItems(uris []string, threads int, uploadData *UploadData) {
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

	uploadData.maxItemLen = getLongestItem(uris)

	// Setup Total files
	uploadData.TotalFiles = len(uris)

	// Create ProgressView
	uploadData.ProgressView = NewProgressView()

	// Check source(s)
	if uploadData.TotalFiles == 0 {
		// We already handled upload from stdin
		fmt.Println("Either specify one or more files or use --from-stdin to upload from stdin")
		return
	}

	if uploadData.ReplaceFileID > 0 && uploadData.ReplaceSameName {
		fmt.Println("Can't handle two replace options at once")
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
	if cData.runUploadPool(uploadData, uris, threads) {
		uploadData.ProgressView.ProgressContainer.Wait()

		for i := range uploadData.ProgressView.Bars {
			if i >= len(uploadData.ProgressView.RawBars) {
				continue
			}

			for !uploadData.ProgressView.RawBars[i].done {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// Run parallel Uploads
func (cData *CommandData) runUploadPool(uploadData *UploadData, uris []string, threads int) bool {
	// Set max connections to amouth of threads
	cData.LibDM.MaxConnectionsPerHost = threads

	// Write all results into resultChan
	resultChan := make(chan interface{}, uploadData.TotalFiles)

	// Create pool
	pool := gopool.New(uploadData.TotalFiles, threads, func(wg *sync.WaitGroup, pos, total, workerID int) interface{} {
		return cData.uploadEntity(*uploadData, uris[pos])
	})

	// use custom result channel
	pool.WithResultChan(resultChan)

	// Start pool and wait for it to complete
	pool.Run().Wait()

	for i := 0; i < uploadData.TotalFiles; i++ {
		b := (<-resultChan).(bool)
		if !b {
			close(resultChan)
			return false
		}
	}

	return true
}

// Upload upload a URI
func (cData *CommandData) uploadEntity(uploadData UploadData, uri string) (succ bool) {
	var uploadResponse *libdm.UploadResponse
	var err error

	// Set name to filename if not set
	if len(uploadData.Name) == 0 {
		if uploadData.ReplaceFileID == 0 {
			_, fileName := filepath.Split(uri)
			uploadData.Name = fileName
		}
	} else {
		uploadData.customName = true
	}

	// Determine if uri is an http url
	isURL := isHTTPURL(uri)

	// Get uri info
	if !isURL {
		s, err := os.Stat(uri)
		if err != nil {
			printError(err, "reading file")
			return
		}

		uploadData.uploadAsArchive = s.IsDir()
	}

	// Create uploadRequest
	uploadRequest := uploadData.toUploadRequest(cData)

	// Create Uploader
	execUploader := cData.newUploader(&uploadData, uri, uploadRequest, (!cData.Quiet && !uploadData.FromStdIn))

	// Do upload request
	if isURL {
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
		// Call required lib func.
		if uploadData.uploadAsArchive {
			// -----> Folder <-----
			uploadResponse = execUploader.uploadArchivedFolder()
		} else {
			// -----> File <-----
			// Open file
			f, err := os.Open(uri)
			if err != nil {
				printError("opening file", err.Error())
				return
			}

			// Upload file
			uploadResponse = execUploader.uploadFile(f)
			f.Close()
		}
	} else {
		// -----> StdIn <-----
		uploadResponse = execUploader.uploadFromStdin()
	}

	// Return on error
	if uploadResponse == nil {
		return
	}

	// Return result of postUpload
	return cData.runPostUpload(&uploadData, uploadResponse, execUploader)
}

// Build UploadRequest from UploadData
func (uploadData *UploadData) toUploadRequest(cData *CommandData) *libdatamanager.UploadRequest {
	// Make public if public name was specified
	if len(uploadData.PublicName) > 0 {
		uploadData.Public = true
	}

	// Add correct ending if name is not set
	if !uploadData.customName {
		// Handle archiving
		if uploadData.uploadAsArchive {
			// Append .tar ending
			if !strings.HasSuffix(uploadData.Name, ".tar") && !strings.HasSuffix(uploadData.Name, ".tar.gz") {
				uploadData.Name += ".tar"
			}
		}

		// Append .gz ending
		if cData.Compression && !strings.HasSuffix(uploadData.Name, ".gz") {
			uploadData.Name += ".gz"
		}
	}

	// Use empty namespace for upload since we want
	// to update it if the user specifies one
	if uploadData.ReplaceFileID > 0 {
		cData.FileAttributes.Namespace = cData.getRealNamespace()
	}

	// Create upload request
	uploadRequest := cData.LibDM.NewUploadRequest(uploadData.Name, cData.FileAttributes)
	uploadRequest.ReplaceFileID = uploadData.ReplaceFileID

	if uploadData.ReplaceSameName {
		uploadRequest.ReplaceFileWithSameName()
	}

	if cData.All {
		uploadRequest.HandleAll()
	}

	// Encrypt file
	if len(cData.Encryption) > 0 {
		uploadRequest.Encrypted(cData.Encryption, cData.EncryptionKey)
	}

	// Publish file
	if uploadData.Public {
		uploadRequest.MakePublic(uploadData.PublicName)
	}

	// Set compression
	if cData.Compression {
		uploadRequest.Compress()
	}

	return uploadRequest
}

// Hit clipboard, keystore and output trigger
func (cData *CommandData) runPostUpload(uploadData *UploadData, uploadResponse *libdatamanager.UploadResponse, uploader *uploader) bool {
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
	text := cData.printUploadResponse(uploadResponse, uploadData, (cData.Quiet || uploadData.TotalFiles > 1), uploader.bar)

	// On quietMode (no bar is shown)
	// just print the output
	if cData.Quiet {
		fmt.Println(text)
	}

	return true
}

// Upload helper
type uploader struct {
	cData         *CommandData         // CLI informations
	uploadRequest *libdm.UploadRequest // Prepared uploadrequest
	uri           string               // URI to be uploaded
	uploadData    *UploadData          // Data containing information for the uploaded fil
	showProgress  bool                 // Use a progressbar
	bar           *Bar                 // Progressbar generated if desired
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
func (uploader *uploader) upload(uploadFunc uploadFunc) (uploadResponse *libdm.UploadResponse) {
	var chsum string
	var err error
	done := make(chan string, 1)

	if uploader.showProgress {
		name := uploader.uploadData.Name
		// Create progressbar
		uploader.bar = NewBar(UploadTask, 0, name, (uploader.uploadData.TotalFiles == 1), uploader.uploadData.maxItemLen)
		uploader.uploadData.ProgressView.AddBar(uploader.bar)

		// Setup proxy
		uploader.uploadRequest.ProxyReader = func(r io.Reader) io.Reader {
			return uploader.bar.bar.ProxyReader(r)
		}

		// Callback if filesize is known
		uploader.uploadRequest.SetFileSizeCallback(func(size int64) {
			uploader.bar.bar.SetTotal(size, false)
		})
	}

	// Call upload hook in background
	go func() {
		c := make(chan string, 1)
		uploadResponse, err = uploadFunc(c, uploader.uri)
		done <- <-c
	}()

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
	if err != nil || uploadResponse == nil {
		printResponseError(err, "uploading file")
		uploader.bar.stop()
		return
	}

	// Verify checksum
	if !uploader.cData.verifyChecksum(chsum, uploadResponse.Checksum) {
		uploader.bar.stop()
		return
	}

	return uploadResponse
}

// Upload from reader
func (uploader *uploader) uploadFromReader(r io.Reader, size int64) *libdm.UploadResponse {
	return uploader.upload(func(done chan string, uri string) (*libdm.UploadResponse, error) {
		return uploader.uploadRequest.UploadFromReader(r, size, done, nil)
	})
}

// Upload a file
func (uploader *uploader) uploadFile(file *os.File) *libdm.UploadResponse {
	// Get fileinfo
	s, err := file.Stat()
	if err != nil {
		return nil
	}

	// Upload from file reader
	return uploader.uploadFromReader(file, s.Size())
}

// Upload from stdin
func (uploader *uploader) uploadFromStdin() *libdm.UploadResponse {
	return uploader.uploadFromReader(os.Stdin, 0)
}

// Upload archived folder
func (uploader *uploader) uploadArchivedFolder() *libdm.UploadResponse {
	return uploader.upload(func(done chan string, uri string) (*libdm.UploadResponse, error) {
		return uploader.uploadRequest.UploadArchivedFolder(uri, done, nil)
	})
}
