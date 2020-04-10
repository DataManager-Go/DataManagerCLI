package commands

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/JojiiOfficial/gaw"
	"github.com/atotto/clipboard"
	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
	"github.com/sbani/go-humanizer/units"

	libdm "github.com/DataManager-Go/libdatamanager"
	humanTime "github.com/sbani/go-humanizer/time"
	clitable "gopkg.in/benweidig/cli-table.v2"
)

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(cData CommandData, uri, name, publicName string, public, fromStdin, setClip bool, replaceFile uint, deletInvalid bool) {
	if len(uri) == 0 && !fromStdin {
		fmt.Println("Either specify a path or use --from-stdin to upload from stdin")
		return
	}

	_, fileName := filepath.Split(uri)
	if len(name) != 0 {
		fileName = name
	}

	// Make public if public name was specified
	if len(publicName) > 0 {
		public = true
	}

	// create upload request
	uploadRequest := cData.LibDM.NewUploadRequest(fileName, cData.FileAttributes)
	if len(cData.Encryption) > 0 {
		uploadRequest.Encrypted(cData.Encryption, cData.EncryptionKey)
	}
	if public {
		uploadRequest.MakePublic(publicName)
	}
	uploadRequest.ReplaceFileID = replaceFile
	var uploadResponse *libdm.UploadResponse

	// Check if uri is a filepath or url
	if u, err := url.Parse(uri); err == nil && u.Scheme != "" {
		// Upload URL
		uploadResponse, err = uploadRequest.UploadURL(u)
		if err != nil {
			printError("uploading url", err.Error())
			return
		}

		printSuccess("uploaded URL: %s", uri)
	} else {
		// Upload file/stdin
		uploadResponse = uploadFileCommand(&cData, uploadRequest, uri, fromStdin)
		if uploadResponse == nil {
			return
		}
	}

	// Set clipboard to public file
	if setClip && len(uploadResponse.PublicFilename) > 0 {
		if clipboard.Unsupported {
			fmt.Println("Clipboard not supported on this OS")
		} else {
			err := clipboard.WriteAll(cData.Config.GetPreviewURL(uploadResponse.PublicFilename))
			if err != nil {
				printError("setting clipboard", err.Error())
			}
		}
	}

	// Add key to keystore
	if cData.Keystore != nil && len(cData.Keyfile) > 0 {
		err := cData.Keystore.AddKey(uploadResponse.FileID, cData.Keyfile)
		if err != nil {
			printError("writing keystore", err.Error())
		}
	}

	// Print response as json
	if cData.OutputJSON {
		fmt.Println(toJSON(uploadResponse))
		return
	}

	// Render table with informations
	cData.printUploadResponse(uploadResponse)
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

		fmt.Println(table)
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

	key := determineDecryptionKey(&cData, resp)

	respData := resp.Body
	if !cData.NoDecrypt {
		encryption = resp.Header.Get(libdm.HeaderEncryption)
		if len(key) == 0 && len(encryption) > 0 {
			fmtError("Error: file is encrypted but no key was given. To ignore this use --no-decrypt")
			os.Exit(1)
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
	UploadFile(cData, filePath, serverName, "", false, false, false, id, false)
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
