package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/DataManager-Go/libdatamanager"
	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/fatih/color"
)

// DownloadData information for downloading files
type DownloadData struct {
	FileName  string
	FileID    uint
	LocalPath string
	Preview   bool

	ProgressView *ProgressView
}

// ViewFile view file
func (cData *CommandData) ViewFile(downloadData *DownloadData) {
	resp, err := downloadData.doRequest(cData, downloadData.Preview)
	if err != nil {
		printResponseError(err, "viewing file")
		return
	}

	if downloadData.Preview {
		// Display file using a GUI application
		tmpFile := GetTempFile(resp.ServerFileName)

		// Shredder at the end
		defer ShredderFile(tmpFile, -1)

		// Write file
		if err = cData.writeFile(resp, tmpFile, nil, nil); err != nil {
			return
		}

		// Preview tempfile
		cData.previewFile(tmpFile)
	} else {
		// Display file in os.Stdout (terminal)
		if err = resp.SaveTo(os.Stdout, nil); err != nil {
			printError("downloading file", err.Error())
			return
		}

		// verify checksum and print an error if invalid
		if !resp.VerifyChecksum() {
			cData.printChecksumError(resp)
			if cData.VerifyFile {
				return
			}
		}
	}
}

// DownloadFile download a file specified by data
func (cData *CommandData) DownloadFile(downloadData *DownloadData) error {
	// Check output file
	if len(downloadData.LocalPath) == 0 {
		fmt.Println("You have to pass a local file")
	}

	// Do request but don't read the body yet
	resp, err := downloadData.doRequest(cData, true)
	if err != nil {
		printResponseError(err, "requesting file")
		return err
	}

	var bar *Bar
	if !cData.Quiet {
		// Create new progressview
		if downloadData.ProgressView == nil {
			downloadData.ProgressView = NewProgressView()
		}

		// Create and add bar
		bar = NewBar(DownloadTask, resp.Size, resp.ServerFileName, false)
		downloadData.ProgressView.AddBar(bar)
	}

	// Determine where the file should be stored in
	outFile := resolveOutputFile(resp.ServerFileName, downloadData.LocalPath)

	// Prevent accidentally overwriting the file
	// TODO add chechksum validation
	if gaw.FileExists(outFile) && !cData.Force && !strings.HasPrefix(outFile, "/dev/") {
		fmt.Printf("File '%s' already exists. Use -f to overwrite it or choose a different outputfile", outFile)
		return err
	}

	cancel := make(chan bool, 1)
	c := make(chan string, 1)

	go func() {
		// Save server file to local 'outFile'
		if err = cData.writeFile(resp, outFile, cancel, bar); err != nil {
			// Delete file on error. On checksum error only delete if --verify was passed
			if err != libdm.ErrChecksumNotMatch || cData.VerifyFile {
				ShredderFile(outFile, -1)
			}

			c <- err.Error()
			return
		}

		c <- ""
	}()

	// Wait for download to be done or delete file on interrupt
	awaitOrInterrupt(c, func(s os.Signal) {
		if bar != nil {
			bar.stop("Cancelled. Erasing file!")
		}

		cancel <- true

		// await shredder
		<-c
		os.Exit(1)
	}, func(s string) {
		var text string

		if len(s) > 0 {
			text = fmt.Sprintf("%s %s: %s", color.HiRedString("Error"), "downloading file", s)
		} else {
			text = fmt.Sprintf("saved '%s'", outFile)
		}

		// Print text
		if bar == nil {
			fmt.Println(text)
		} else {
			bar.doneTextChan <- text
		}
	})

	if downloadData.ProgressView != nil {
		// Wait for bars to complete
		for i := range downloadData.ProgressView.RawBars {
			for !downloadData.ProgressView.RawBars[i].done {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}

	return nil
}

func (downloadData *DownloadData) doRequest(cData *CommandData, showBar bool) (*libdm.FileDownloadResponse, error) {
	// Create new filerequest
	resp, err := cData.LibDM.NewFileRequest(downloadData.FileID, downloadData.FileName, cData.FileAttributes.Namespace).Do()
	if err != nil {
		return nil, err
	}

	// Build request and set bar if desired
	cData.handleDecryption(resp)
	return resp, nil
}

func (cData *CommandData) handleDecryption(resp *libdm.FileDownloadResponse) {
	// Set decryptionkey
	if cData.NoDecrypt {
		resp.DownloadRequest.DecryptWith(nil)
	} else {
		resp.DownloadRequest.DecryptWith(cData.determineDecryptionKey(resp.Response))
	}
}

// Write response to a given file
func (cData *CommandData) writeFile(resp *libdm.FileDownloadResponse, file string, cancel chan bool, bar *Bar) error {
	if bar != nil {
		resp.DownloadRequest.ReaderProxy = func(r io.Reader) io.Reader {
			// Proxy reader through bar
			return bar.bar.ProxyReader(r)
		}
	}

	// Save file to tempFile
	if err := resp.WriteToFile(file, 0600, cancel); err != nil {
		// Make error readable
		var errText string
		if err == libdm.ErrChecksumNotMatch {
			errText = cData.getChecksumError(resp)
		} else {
			errText = getError("downloading file", err.Error())
		}

		// View the error
		if bar != nil {
			bar.doneTextChan <- errText
		} else {
			fmt.Println(errText)
		}

		return err
	}

	return nil
}

// Download multiple files into a folder
func (cData *CommandData) downloadFiles(files []libdm.FileResponseItem, outDir string, parallelism uint, getSubDirName func(file libdm.FileResponseItem) string) {
	if len(files) == 0 {
		fmt.Println("No files found")
		return
	}

	// Prevent user stupidity
	if parallelism == 0 {
		parallelism = 1
	}

	// Reduce threads if files are less than threads
	if uint(len(files)) < parallelism {
		parallelism = uint(len(files))
	}

	// Use first files namespace as destination dir
	if len(outDir) == 0 {
		outDir = files[0].Attributes.Namespace
	}

	// Resolve path
	rootDir := gaw.ResolveFullPath(outDir)

	// Overwrite files
	cData.Force = true

	// Waitgroup to wait for all "threads" to be done
	wg := sync.WaitGroup{}
	// Channel for managing amount of parallel upload processes
	c := make(chan uint, 1)

	c <- parallelism
	var pos int

	totalfiles := len(files)

	// Use this context for progressview
	progressView := NewProgressView()

	// Start Downloader pool
	for pos < totalfiles {
		read := <-c
		for i := 0; i < int(read) && pos < totalfiles; i++ {
			wg.Add(1)

			go func(file libdatamanager.FileResponseItem) {
				// Build dest group dir name
				dir := getSubDirName(file)

				// Create dir if not exists
				path := filepath.Clean(filepath.Join(rootDir, dir))
				if _, err := os.Stat(path); err != nil {
					err := os.MkdirAll(path, 0750)
					if err != nil {
						printError("Creating dir", err.Error())
						os.Exit(1)
					}
				}

				// Download file
				err := cData.DownloadFile(&DownloadData{
					FileName:     file.Name,
					FileID:       file.ID,
					LocalPath:    filepath.Join(rootDir, dir),
					ProgressView: progressView,
				})

				if err != nil {
					os.Exit(1)
				}

				wg.Done()
				c <- 1
			}(files[pos])

			pos++
		}
	}

	// Wait for all threads
	// to be done
	wg.Wait()
}
