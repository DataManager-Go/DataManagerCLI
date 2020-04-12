package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/gosuri/uiprogress"
)

// DownloadData information for downloading files
type DownloadData struct {
	FileName  string
	FileID    uint
	LocalPath string
	Preview   bool
	NoPreview bool
	bar       *uiprogress.Bar
}

func (downloadData *DownloadData) needPreview() bool {
	return downloadData.Preview && !downloadData.NoPreview
}

func buildRequest(cData *CommandData, resp *libdm.FileDownloadResponse, showBar bool) (bar *uiprogress.Bar) {
	// Set decryptionkey
	if cData.NoDecrypt {
		resp.DownloadRequest.DecryptWith(nil)
	} else {
		resp.DownloadRequest.DecryptWith(cData.determineDecryptionKey(resp.Response))
	}

	if !cData.Quiet && showBar {
		uiprogress.Start()

		prefix := "Downloading " + resp.ServerFileName
		bar, proxy := buildProgressbar(prefix, uint(len(prefix)))
		bar.Total = int(resp.Size)
		resp.DownloadRequest.Proxy = proxy
		uiprogress.AddBar(bar)
		return bar
	}

	return nil
}

func (downloadData *DownloadData) doRequest(cData *CommandData, showBar bool) (*libdm.FileDownloadResponse, error) {
	resp, err := cData.LibDM.NewFileRequest(downloadData.FileID, downloadData.FileName, cData.FileAttributes.Namespace).Do()
	if err != nil {
		return nil, err
	}

	bar := buildRequest(cData, resp, showBar)

	if bar != nil {
		downloadData.bar = bar
	}

	return resp, nil
}

// ViewFile view file
func (cData *CommandData) ViewFile(data *DownloadData) {
	resp, err := data.doRequest(cData, data.needPreview())
	if err != nil {
		printError("viewing file", err.Error())
		return
	}

	if data.needPreview() {
		// Display file using a GUI application
		tmpFile := GetTempFile(resp.ServerFileName)

		// Shredder at the end
		defer ShredderFile(tmpFile, -1)

		// Write file
		if err = writeFile(cData, resp, tmpFile); err != nil {
			return
		}

		// Preview tempfile
		previewFile(tmpFile)
	} else {
		// Display file in os.Stdout (cli)
		err = resp.SaveTo(os.Stdout)
		if err != nil {
			printError("viewing file", err.Error())
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

// DownloadFile view file
func (cData *CommandData) DownloadFile(data *DownloadData) {
	if len(data.LocalPath) == 0 {
		fmt.Println("You have to pass a local file")
	}

	// Do request but don't read the body yet
	resp, err := data.doRequest(cData, true)
	if err != nil {
		printError("downloading file", err.Error())
		return
	}

	// Determine where the file should be stored in
	outFile := data.LocalPath
	if strings.HasSuffix(data.LocalPath, "/") {
		// If no special file was choosen
		outFile = filepath.Join(data.LocalPath, resp.ServerFileName)
	} else {
		stat, err := os.Stat(outFile)
		if err == nil && stat.IsDir() {
			outFile = filepath.Join(outFile, resp.ServerFileName)
		}
	}

	// Prevent accitentally overwrite the file
	if gaw.FileExists(outFile) && !cData.Force {
		fmt.Printf("File '%s' already exists. Use -f to overwrite it or choose a different outputfile", outFile)
		return
	}

	err = writeFile(cData, resp, outFile)
	if err != nil {
		// Delete file on error. On checksum error only delete if --verify was passed
		if err != libdm.ErrChecksumNotMatch || cData.VerifyFile {
			ShredderFile(outFile, -1)
		}

		return
	}

	printSuccess("saved '%s'", outFile)
}

func writeFile(cData *CommandData, resp *libdm.FileDownloadResponse, file string) error {
	// Save file to tempFile
	err := resp.WriteToFile(file, 0600)
	if err != nil {
		if err == libdm.ErrChecksumNotMatch {
			cData.printChecksumError(resp)
		} else {
			printError("downloading file", err.Error())
		}
	}

	return err
}
