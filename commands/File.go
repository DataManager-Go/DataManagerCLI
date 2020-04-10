package commands

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
	humanTime "github.com/sbani/go-humanizer/time"
	"github.com/sbani/go-humanizer/units"
	clitable "gopkg.in/benweidig/cli-table.v2"
)

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(cData CommandData, path, name, publicName string, public, fromStdin bool, replaceFile uint, deletInvalid bool) {
	if len(path) == 0 && !fromStdin {
		fmt.Println("Either specify a path or use --from-stdin to upload from stdin")
		return
	}

	_, fileName := filepath.Split(path)
	if len(name) != 0 {
		fileName = name
	}

	// Make public if public name was specified
	if len(publicName) > 0 {
		public = true
	}

	// Declare var
	var err error
	var uploadResponse *libdm.UploadResponse
	var bar *pb.ProgressBar
	done := make(chan bool, 1)
	kill := make(chan os.Signal, 1)
	c := make(chan int64, 1)
	var checksum string

	// Init progressbar and proxy
	proxy := libdm.NoProxyWriter
	if !cData.Quiet {
		bar = pb.New64(0).SetMaxWidth(100)
		proxy = func(w io.Writer) io.Writer {
			return bar.NewProxyWriter(w)
		}
	}

	// Start upload
	go (func(path, fileName string, public bool, replaceFile uint, fileattributes libdm.FileAttributes, proxy func(io.Writer) io.Writer, c chan int64, publicName, encryption, encryptionKey string) {
		uploadResponse, checksum, err = cData.LibDM.UploadFile(path, fileName, public, replaceFile, fileattributes, proxy, c, publicName, encryption, encryptionKey)
		done <- true
	})(path, fileName, public, replaceFile, cData.FileAttributes, proxy, c, publicName, cData.Encryption, cData.EncryptionKey)

	// Read filesize and set bars total to filesize
	fileSize := <-c
	if bar != nil {
		bar.SetTotal(fileSize)

		// Show bar only if uploading takes more than 500ms
		go (func() {
			time.Sleep(500 * time.Millisecond)
			select {
			case <-done:
			default:
				bar.Start()
			}
		})()
	}

	// Set notify for signal to do something on cancel
	signal.Notify(kill, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Wait for upload to be finished or upload to
	// be canceled
	select {
	case killsig := <-kill:
		// Stop bar
		if bar != nil {
			bar.Finish()
		}
		fmt.Println(killsig)
		cData.deleteKeyfile()
		os.Exit(0)
		return
	case <-done:
	}

	// Stop bar
	if bar != nil {
		bar.Finish()
	}

	if err != nil {
		printResponseError(err, "uploading file")
		return
	}

	if len(checksum) == 0 || uploadResponse == nil {
		fmtError("Unexpected error while uploading")
		return
	}

	// Update keystore
	if cData.Keystore != nil && len(cData.Keyfile) > 0 {
		cData.Keystore.AddKey(uploadResponse.FileID, cData.Keyfile)
	}

	checksumMatch := uploadResponse.Checksum == checksum

	// If requested, do json output
	if cData.OutputJSON {
		if !checksumMatch {
			printJSONError("invalid checksum")
			os.Exit(1)
		} else {
			fmt.Println(toJSON(uploadResponse))
		}
		return
	}

	// Verify checksums
	if deletInvalid {
		cData.VerifyFile = true
	}
	if !verifyChecksum(&cData, checksum, uploadResponse.Checksum) {
		if deletInvalid {
			DeleteFile(cData, "", uploadResponse.FileID)
		}
	}

	// Print nice human output
	if len(uploadResponse.PublicFilename) != 0 {
		fmt.Printf("Public name: %s\nName: %s\nID %d\n", cData.Config.GetPreviewURL(uploadResponse.PublicFilename), fileName, uploadResponse.FileID)
	} else {
		fmt.Printf("Name: %s\nID: %d\n", fileName, uploadResponse.FileID)
	}

}

// DeleteFile deletes the desired file(s)
func DeleteFile(cData CommandData, name string, id uint) {
	// Convert input
	name, id = getFileCommandData(name, id)

	if len(strings.TrimSpace(name)) == 0 && id <= 0 {
		fmtError("Missing a valid parameter. Provide fileID or Filename")
		return
	}

	// Confirm 'delete everything'
	if strings.TrimSpace(name) == "%" && !cData.Yes && cData.All {
		if i, _ := gaw.ConfirmInput("Do you really want to delete all files in "+cData.Namespace+"? (y/n)> ", bufio.NewReader(os.Stdin)); !i {
			return
		}
	}

	resp, err := cData.LibDM.DeleteFile(name, id, cData.All, cData.FileAttributes)
	if err != nil {
		printResponseError(err, "deleting file")
		return
	}

	if len(resp.IDs) > 1 {
		fmt.Printf("Deleted %d files %s\n", len(resp.IDs), color.HiGreenString("successfully"))
	} else {
		fmt.Printf("The file has been %s\n", color.HiGreenString("successfully deleted"))
	}

	// rm keys from keystore
	if cData.Keystore != nil {
		rmFilesFromkeystore(cData.Keystore, resp.IDs)
	}
}

// ListFiles lists the files corresponding to the args
func ListFiles(cData CommandData, name string, id uint, sOrder string) {
	// Convert input
	name, id = getFileCommandData(name, id)

	resp, err := cData.LibDM.ListFiles(name, id, cData.AllNamespaces, cData.FileAttributes, cData.Details)
	if err != nil {
		printResponseError(err, "listing files")
		return
	}

	if uint16(len(resp.Files)) > cData.Config.Client.MinFilesToDisplay && !cData.Yes {
		y, _ := gaw.ConfirmInput("Do you want to view all? (y/n) > ", bufio.NewReader(os.Stdin))
		if !y {
			return
		}
	}

	// Print as json if desired
	if cData.OutputJSON {
		fmt.Println(toJSON(resp.Files))
	} else {
		if len(resp.Files) == 0 {
			fmt.Printf("No files in namespace %s\n", cData.Namespace)
			return
		}

		headingColor := color.New(color.FgHiGreen, color.Underline, color.Bold)

		// Table setup
		table := clitable.New()
		table.ColSeparator = " "
		table.Padding = 4

		var hasPublicFile, hasTag, hasGroup bool

		// Scan for availability of attributes
		for _, file := range resp.Files {
			if !hasPublicFile && file.IsPublic && len(file.PublicName) > 0 {
				hasPublicFile = true
			}

			// Only need to do if requested more details
			if cData.Details > 1 {
				// Has tag
				if !hasTag && len(file.Attributes.Tags) > 0 {
					hasTag = true
				}

				// Has group
				if !hasGroup && len(file.Attributes.Groups) > 0 {
					hasGroup = true
				}
			}
		}

		// Order output
		if len(sOrder) > 0 {
			if order := FileOrderFromString(sOrder); order != nil {
				// Sort
				NewFileSorter(resp.Files).
					Reversed(IsOrderReversed(sOrder)).
					SortBy(*order)
			} else {
				fmtError(fmt.Sprintf("sort by '%s' not supporded", sOrder))
				return
			}
		} else {
			// By default sort by creation desc
			NewFileSorter(resp.Files).Reversed(true).SortBy(CreatedOrder)
		}

		header := []interface{}{
			headingColor.Sprint("ID"), headingColor.Sprint("Name"), headingColor.Sprint("Size"),
		}

		// Add public name
		if hasPublicFile {
			header = append(header, headingColor.Sprint("Public name"))
		}

		// Add created
		header = append(header, headingColor.Sprint("Created"))

		// Show namespace on -dd
		if cData.Details > 2 || cData.AllNamespaces {
			header = append(header, headingColor.Sprintf("Namespace"))
		}

		// Show groups and tags on -d
		if cData.Details > 1 {
			if hasGroup {
				header = append(header, headingColor.Sprintf("Groups"))
			}

			if hasTag {
				header = append(header, headingColor.Sprintf("Tags"))
			}
		}

		table.AddRow(header...)

		for _, file := range resp.Files {
			// Colorize private pubNames if not public
			pubname := file.PublicName
			if len(pubname) > 0 && !file.IsPublic {
				pubname = color.HiMagentaString(pubname)
			}

			// Add items
			rowItems := []interface{}{
				file.ID,
				formatFilename(&file, cData.NameLen, &cData),
				units.BinarySuffix(float64(file.Size)),
			}

			// Append public file
			if hasPublicFile {
				rowItems = append(rowItems, pubname)
			}

			// Append time
			rowItems = append(rowItems, humanTime.Difference(time.Now(), file.CreationDate))

			// Show namespace on -dd
			if cData.Details > 2 || cData.AllNamespaces {
				rowItems = append(rowItems, file.Attributes.Namespace)
			}

			// Show groups and tags on -d
			if cData.Details > 1 {
				if hasGroup {
					rowItems = append(rowItems, strings.Join(file.Attributes.Groups, ", "))
				}

				if hasTag {
					rowItems = append(rowItems, strings.Join(file.Attributes.Tags, ", "))
				}
			}

			table.AddRow(rowItems...)
		}

		fmt.Println(table.String())
	}
}

// PublishFile publishes a file
func PublishFile(cData CommandData, name string, id uint, publicName string) {
	// Convert input
	name, id = getFileCommandData(name, id)

	resp, err := cData.LibDM.PublishFile(name, id, publicName, cData.All, cData.FileAttributes)
	if err != nil || resp == nil {
		printResponseError(err, "publishing file")
		return
	}

	// Output
	if cData.OutputJSON {
		fmt.Println(toJSON(resp))
	} else {
		if cData.All {
			rs := (resp).(libdm.BulkPublishResponse)

			fmt.Printf("Published %d files\n", len(rs.Files))
			for _, file := range rs.Files {
				fmt.Printf("File %s with ID %d Public name: %s\n", file.Filename, file.FileID, file.PublicFilename)
			}
		} else {
			pubName := (resp.(libdm.PublishResponse)).PublicFilename
			fmt.Printf(cData.Config.GetPreviewURL(pubName))
		}
	}
}

// UpdateFile updates a file on the server
func UpdateFile(cData CommandData, name string, id uint, newName string, newNamespace string, addTags []string, removeTags []string, addGroups []string, removeGroups []string, setPublic, setPrivate bool) {
	// Process params: make t1,t2 -> [t1 t2]
	ProcesStrSliceParams(&addTags, &addGroups, &removeTags, &removeGroups)

	// Convert input
	name, id = getFileCommandData(name, id)

	// Can't use both
	if setPrivate && setPublic {
		fmtError("Illegal flag combination")
		return
	}

	response, err := cData.LibDM.UpdateFile(name, id, cData.Namespace, cData.All, libdm.FileChanges{
		NewName:      newName,
		NewNamespace: newNamespace,
		AddTags:      addTags,
		AddGroups:    addGroups,
		RemoveTags:   removeTags,
		RemoveGroups: removeGroups,
		SetPublic:    setPublic,
		SetPrivate:   setPrivate,
	})

	if err != nil {
		printResponseError(err, "updating file")
		return
	}

	if response.Count > 1 {
		fmt.Printf("Updated %d files %s\n", response.Count, color.HiGreenString("successfully"))
	} else {
		fmt.Printf("The file has been %s\n", color.HiGreenString("successfully updated"))
	}
}

// GetFile requests the file from the server and displays or saves it
func GetFile(cData CommandData, fileName string, id uint, savePath string, displayOutput, noPreview, preview bool, args ...bool) (success bool, encryption, serverFileName string) {
	// Convert input
	fileName, id = getFileCommandData(fileName, id)

	shouldPreview := cData.Config.Client.AutoFilePreview || preview
	if noPreview {
		shouldPreview = false
	}

	// a very complicated 'showBar' AI algorythm DON'T TOUCH
	showBar :=
		// Not quiet and preview (using default app)
		(!cData.Quiet && ((displayOutput && shouldPreview) ||
			// OR not printed to stdout
			(!displayOutput && len(savePath) > 0))) ||
			// OR forced to display
			(len(args) > 0 && args[0])

	// Errorhandling 100
	if noPreview && preview {
		fmt.Print("rlly?")
		return
	}

	// Get the file
	resp, serverFileName, checksum, err := cData.LibDM.GetFile(fileName, id, cData.Namespace)
	if err != nil {
		printResponseError(err, "downloading file")
		return
	}

	respData := resp.Body

	if !cData.NoDecrypt {
		encryption = resp.Header.Get(libdm.HeaderEncryption)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Create and setup bar
	var bar *pb.ProgressBar
	if showBar {
		bar = pb.New64(libdm.GetFilesizeFromDownloadRequest(resp)).SetMaxWidth(100)
		respData = bar.NewProxyReader(respData)
	}

	// Display or save file
	if displayOutput && len(savePath) == 0 {
		// Only write to tmpfile if a gui-like preview needed
		if shouldPreview {
			// Save, decrypt and preview file
			file := guiPreview(&cData, serverFileName, encryption, checksum, resp, respData, bar)
			if file != "" {
				// Shredder/Delete file
				ShredderFile(file, -1)
			}
		} else {
			// Write file to os.Stdout
			// Decrypts stream if necessary
			errCh := make(chan error, 1)
			chSum := writeFileToWriter(os.Stdout, encryption, determineDecryptionKey(&cData, resp), respData, errCh, nil)

			select {
			case err := <-errCh:
				if err != nil {
					printError("downloading", err.Error())
				} else {
					fmt.Println("An unexpected error occured")
				}
			case chsum := <-chSum:
				verifyChecksum(&cData, chsum, checksum)
			}

		}
	} else if len(savePath) > 0 {
		// Use server filename if a wildcard was used
		if strings.HasSuffix(fileName, "%") || strings.HasPrefix(fileName, "%") || len(fileName) == 0 {
			fileName = serverFileName
		}

		// Determine output file/path
		outFile := savePath
		if strings.HasSuffix(savePath, "/") {
			outFile = filepath.Join(savePath, fileName)
		} else {
			stat, err := os.Stat(outFile)
			if err == nil && stat.IsDir() {
				outFile = filepath.Join(outFile, fileName)
			} else if stat != nil && stat.Mode().IsRegular() && !cData.Force {
				fmt.Println("File already exists. Use -f to overwrite it")
				return
			}
		}

		// channel if filewriting is done
		errChan := make(chan error)
		doneChan := saveFileToFile(outFile, encryption, determineDecryptionKey(&cData, resp), respData, errChan, bar)
		var chsum string

		// Wait for download to be finished
		// or an error to occur
		select {
		case err := <-errChan:
			if err = <-errChan; err != nil {
				fmt.Println(err)
				return
			}
		case chsum = <-doneChan:
		}

		if !verifyChecksum(&cData, chsum, checksum) {
			return
		}

		if bar != nil {
			bar.Finish()
		}

		// Preview
		if displayOutput {
			previewFile(savePath)
		}

		if !cData.Quiet {
			// Print success message
			fmt.Printf("Saved file into %s\n", outFile)
		}
	} else if !displayOutput && len(savePath) == 0 {
		fmtError("Can't save file if you don't specify a path.")
		return
	}

	// Close body
	resp.Body.Close()

	success = true
	return
}

// EditFile edits a file
func EditFile(cData CommandData, id uint) {
	// Generate temp-filePath
	filePath := GetTempFile(gaw.RandString(10))

	// Download File
	success, encryption, serverName := GetFile(cData, "", id, filePath, false, true, false, !cData.Quiet)
	if !success {
		return
	}

	// Delete temp file
	defer func() {
		ShredderFile(filePath, -1)
	}()

	// Generate md5 of original file
	fileOldMd5 := fileMd5(filePath)

	// Edit file. Return on error
	if !editFile(filePath) {
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
	if len(encryption) != 0 {
		cData.Encryption = encryption
	}

	// Replace file on server with new file
	UploadFile(cData, filePath, serverName, "", false, false, id, false)
}

func editFile(file string) bool {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "/usr/bin/nano"
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
