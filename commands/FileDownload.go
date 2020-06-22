package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/JojiiOfficial/gopool"
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
func (cData *CommandData) DownloadFile(downloadData *DownloadData) (*libdm.FileDownloadResponse, error) {
	doBench := cData.Config.Client.BenchResult == 0
	var benchChan chan int

	// Run the bench in background
	if doBench {
		benchChan = make(chan int, 1)
		hashTest := NewHashBench()
		go func() {
			benchChan <- hashTest.DoTest()
		}()
	}

	// Check output file
	if len(downloadData.LocalPath) == 0 {
		fmt.Println("You have to pass a local file")
		return nil, nil
	}

	// Do request but don't read the body yet
	resp, err := downloadData.doRequest(cData, true)
	if err != nil {
		printResponseError(err, "requesting file")
		return resp, err
	}

	// Determine where the file should be stored in
	outFile := resolveOutputFile(resp.ServerFileName, downloadData.LocalPath)

	// Wait for bench result
	// and save it to config
	if doBench {
		res := <-benchChan
		cData.Config.Client.BenchResult = res
		if err := cData.Config.Save(); err != nil {
			printError("Saving config", err.Error())
			return resp, err
		}
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

	// Prevent accidentally overwriting the file
	// TODO add chechksum validation
	if gaw.FileExists(outFile) && !cData.Force && !strings.HasPrefix(outFile, "/dev/") {
		fmt.Printf("File '%s' already exists. Use -f to overwrite it or choose a different outputfile", outFile)
		return resp, err
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
		if bar != nil {
			bar.done = true
			bar.doneText = "Stopped"
		}

		fmt.Println(text)
	})

	if downloadData.ProgressView != nil {
		// Wait for bars to complete
		for i := range downloadData.ProgressView.RawBars {
			for !downloadData.ProgressView.RawBars[i].done {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}

	return resp, nil
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
func (cData *CommandData) downloadFiles(files []libdm.FileResponseItem, outDir string, threads int, getSubDirName func(file libdm.FileResponseItem) string) {
	cData.LibDM.MaxConnectionsPerHost = threads

	if len(files) == 0 {
		fmt.Println("No files found")
		return
	}

	// Use first files namespace as destination dir
	if len(outDir) == 0 {
		outDir = files[0].Attributes.Namespace
	}

	// Resolve path
	rootDir := gaw.ResolveFullPath(outDir)

	// Use this context for progressview
	progressView := NewProgressView()

	// Overwrite files
	cData.Force = true

	// Create and execute a new pool
	gopool.New(len(files), threads, func(wg *sync.WaitGroup, pos, total, workerID int) interface{} {
		file := files[pos]

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
		if _, err := cData.DownloadFile(&DownloadData{
			FileName:     file.Name,
			FileID:       file.ID,
			LocalPath:    filepath.Join(rootDir, dir),
			ProgressView: progressView,
		}); err != nil {
			printError("downloading namespace", err.Error())
			os.Exit(1)
		}

		return nil
	}).Run().Wait()
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
