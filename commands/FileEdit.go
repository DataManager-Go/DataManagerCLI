package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// EditFile edits a file
func (cData *CommandData) EditFile(id uint, editor string) {
	// Do file Request
	resp, err := cData.LibDM.NewFileRequestByID(id).Do()
	if err != nil {
		printError("downloading file", err.Error())
		return
	}

	// Generate temp-filePath
	filePath := GetTempFile(resp.ServerFileName)
	fmt.Println(resp.ServerFileName)

	cData.handleDecryption(resp)

	if resp.FileID == 0 {
		fmt.Println("Unexpected error occured, received File Id is invalid")
		return
	}

	// Save File
	err = resp.WriteToFile(filePath, 0600, nil)
	if err != nil {
		printError("downloading file", err.Error())
		return
	}

	// Shredder temp file at the end
	defer func() {
		ShredderFile(filePath, -1)
	}()

	// Generate md5 of original file
	fileOldMd5 := fileMd5(filePath)

	// Edit file. Return on error
	if !editFile(filePath, editor) {
		return
	}

	// Generate md5 of original file
	fileNewMd5 := fileMd5(filePath)

	// Check for file changes
	if fileNewMd5 == fileOldMd5 {
		fmt.Println("Nothing changed")
		return
	}

	// Set encryption to keep its encrypted state
	if len(resp.Encryption) != 0 {
		cData.Encryption = resp.Encryption
	}

	// Set key and encryption to use for upload
	cData.EncryptionKey = resp.DownloadRequest.Key
	cData.Encryption = resp.Encryption

	// Replace file on server with new file
	cData.UploadItems([]string{filePath}, 1, &UploadData{
		ReplaceFile: resp.FileID,
	})
}

func editFile(file, editor string) bool {
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd":
		{
			return editLinux(file, editor)
		}
	case "windows":
		{

		}
	default:
		fmt.Printf("No support for %s at the moment\n", runtime.GOOS)
	}

	return false
}

func editLinux(file, editor string) bool {
	if len(editor) == 0 {
		editor = os.Getenv("EDITOR")
		if len(editor) == 0 {
			editor = "/usr/bin/nano"
		}
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

var tuiEditors = []string{"vim", "vi", "emacs", "nano"}

func isTUIeditor(editor string) bool {
	editor = strings.ToLower(editor)

	if runtime.GOOS == "windows" {
		// Windows is too cool for TUI
		// based editors (i'am aware of)
		return false
	}

	for i := range tuiEditors {
		if tuiEditors[i] == editor ||
			strings.HasSuffix(editor, tuiEditors[i]) {
			return true
		}
	}

	return false
}
