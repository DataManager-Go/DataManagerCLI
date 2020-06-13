package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/DataManager-Go/libdatamanager"
	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
)

// DownloadData information for downloading files
type DownloadData struct {
	FileName  string
	FileID    uint
	LocalPath string
	Preview   bool
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
		if err = cData.writeFile(resp, tmpFile, nil); err != nil {
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
		printResponseError(err, "downloading file")
		return err
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
		if err = cData.writeFile(resp, outFile, cancel); err != nil {
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
		cancel <- true

		// await shredder
		<-c
		os.Exit(1)
	}, func(s string) {
		if len(s) > 0 {
			// printBar(fmt.Sprintf("%s %s: %s", color.HiRedString("Error"), "downloading file", s), data.bar)
			os.Exit(1)
		} else {
			// printBar(sPrintSuccess("saved '%s'", outFile))
		}
	})

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

func (cData *CommandData) writeFile(resp *libdm.FileDownloadResponse, file string, cancel chan bool) error {
	// Save file to tempFile
	if err := resp.WriteToFile(file, 0600, cancel); err != nil {
		if err == libdm.ErrChecksumNotMatch {
			// printBar(cData.getChecksumError(resp))
		} else {
			// printBar(fmt.Sprintf("%s %s: %s", color.HiRedString("Error"), "downloading file", err.Error()))
		}
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
					FileName:  file.Name,
					FileID:    file.ID,
					LocalPath: filepath.Join(rootDir, dir),
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
