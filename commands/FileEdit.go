package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// EditFile edits a file
func (cData *CommandData) EditFile(name string, id uint, editor string) {
	if !checkEditor(editor) {
		fmt.Printf("Can't find editor %s\n", editor)
		return
	}

	name, id = GetFileCommandData(name, id)

	// Use force to overwrite a local version
	// If the checksums match, the file won't be
	// downloaded again
	cData.Force = true
	resp, err := cData.DownloadFile(&DownloadData{
		FileName:  name,
		FileID:    id,
		LocalPath: os.TempDir(),
	})

	if err != nil {
		// We already printed an error
		// in the cData.DlFile func
		return
	}

	// Get output file
	filePath := resolveOutputFile(resp.ServerFileName, os.TempDir())

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

	// Replace file on server with new version
	cData.UploadItems([]string{filePath}, 1, &UploadData{
		ReplaceFile: resp.FileID,
		Name:        resp.ServerFileName,
	})
}

// Edit file in some way
func editFile(file, editor string) bool {
	// Different OS may handle file editing different
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd":
		{
			return editLinux(file, editor)
		}
	case "windows":
		{
			return editWindows(file)
		}
	default:
		fmt.Printf("No support for %s at the moment\n", runtime.GOOS)
	}

	return false
}

// Edit a given file on a linux based os
func editLinux(file, editor string) bool {
	if len(editor) == 0 {
		editor = os.Getenv("EDITOR")
		if len(editor) == 0 {
			editor = "/usr/bin/nano"
		}
	}

	// Check editor
	editor, found := findEditor(editor)
	if !found {
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

// Edit a file in windows
func editWindows(file string) bool {
	cmd := exec.Command("start", "/B", "/wait", file)
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

var tuiEditors = []string{"vim", "vi", "emacs", "nano"}

// Were gonna need this if there is a
// problem with the progressbar and a
// tui based text editor
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

// Check if given editor can be found
func checkEditor(editor string) bool {
	if runtime.GOOS == "windows" {
		return true
	}

	_, ok := findEditor(editor)
	return ok
}

// Find editor
func findEditor(editor string) (string, bool) {
	// Check if path is already valid
	if strings.HasPrefix(editor, "/") {
		if _, err := os.Stat(editor); err == nil {
			return editor, true
		}
	}

	var paths []string

	// Prefer $PATH
	path, has := os.LookupEnv("PATH")
	if !has {
		paths = append(paths, strings.Split(path, ":")...)
	} else {
		// Alternatively use a custom $PATH
		paths = []string{"/usr/bin", "/usr/local/bin", "/usr/sbin", "/sbin", "/bin", "/usr/local/sbin"}
	}

	// Find editor
	for _, path := range paths {
		e := filepath.Join(path, editor)
		if _, err := os.Stat(e); err == nil {
			return e, true
		}
	}

	return "", false
}
