package commands

import (
	"fmt"
	"os"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/fatih/color"
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
	progress  *uiprogress.Progress
}

func (downloadData *DownloadData) needPreview() bool {
	return downloadData.Preview && !downloadData.NoPreview
}

func buildRequest(cData *CommandData, resp *libdm.FileDownloadResponse, showBar bool, progress *uiprogress.Progress) (bar *uiprogress.Bar) {
	// Set decryptionkey
	if cData.NoDecrypt {
		resp.DownloadRequest.DecryptWith(nil)
	} else {
		resp.DownloadRequest.DecryptWith(cData.determineDecryptionKey(resp.Response))
	}

	if !cData.Quiet && showBar {
		if progress == nil {
			uiprogress.Start()
		}

		bar, proxy := createDownloadBar(progress, resp.ServerFileName, resp.Size)
		resp.DownloadRequest.Proxy = proxy
		return bar
	}

	return nil
}

func createDownloadBar(progress *uiprogress.Progress, fileName string, fileSize int64) (*uiprogress.Bar, libdm.WriterProxy) {
	prefix := "Downloading " + fileName
	bar, proxy := buildProgressbar(prefix, uint(len(prefix)))
	bar.Total = int(fileSize)

	if progress == nil {
		uiprogress.AddBar(bar)
	} else {
		progress.AddBar(bar)
	}

	return bar, proxy
}

func (downloadData *DownloadData) doRequest(cData *CommandData, showBar bool, progress *uiprogress.Progress) (*libdm.FileDownloadResponse, error) {
	resp, err := cData.LibDM.NewFileRequest(downloadData.FileID, downloadData.FileName, cData.FileAttributes.Namespace).Do()
	if err != nil {
		return nil, err
	}

	bar := buildRequest(cData, resp, showBar, progress)

	if bar != nil {
		downloadData.bar = bar
	}

	return resp, nil
}

// ViewFile view file
func (cData *CommandData) ViewFile(data *DownloadData, progress *uiprogress.Progress) {
	resp, err := data.doRequest(cData, data.needPreview(), progress)
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
		if err = writeFile(cData, resp, tmpFile, nil, data.bar); err != nil {
			return
		}

		// Preview tempfile
		previewFile(tmpFile)
	} else {
		// Display file in os.Stdout (cli)
		err = resp.SaveTo(os.Stdout, nil)
		if err != nil {
			printResponseError(err, "downloading file")
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
func (cData *CommandData) DownloadFile(data *DownloadData, progress *uiprogress.Progress) error {
	if len(data.LocalPath) == 0 {
		fmt.Println("You have to pass a local file")
	}

	// Do request but don't read the body yet
	resp, err := data.doRequest(cData, true, progress)
	if err != nil {
		printResponseError(err, "downloading file")
		return err
	}

	// Determine where the file should be stored in
	outFile := determineLocalOutputfile(resp.ServerFileName, data.LocalPath)

	// Prevent accidentally overwriting the file
	// TODO add chechksum validation
	if gaw.FileExists(outFile) && !cData.Force {
		fmt.Printf("File '%s' already exists. Use -f to overwrite it or choose a different outputfile", outFile)
		return err
	}

	cancel := make(chan bool, 1)
	c := make(chan string, 1)

	go func() {
		err = writeFile(cData, resp, outFile, cancel, data.bar)
		if err != nil {
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
			printBar(fmt.Sprintf("%s %s: %s", color.HiRedString("Error"), "downloading file", s), data.bar)
			os.Exit(1)
		} else {
			printBar(sPrintSuccess("saved '%s'", outFile), data.bar)
		}
	})

	return nil
}

func writeFile(cData *CommandData, resp *libdm.FileDownloadResponse, file string, cancel chan bool, bar *uiprogress.Bar) error {
	// Save file to tempFile
	err := resp.WriteToFile(file, 0600, cancel)
	if err != nil {
		if err == libdm.ErrChecksumNotMatch {
			printBar(cData.getChecksumError(resp), bar)
		} else {
			printBar(fmt.Sprintf("%s %s: %s", color.HiRedString("Error"), "downloading file", err.Error()), bar)
		}
	}

	return err
}
