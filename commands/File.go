package commands

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
	humanTime "github.com/sbani/go-humanizer/time"
	"github.com/sbani/go-humanizer/units"
	clitable "gopkg.in/benweidig/cli-table.v2"
)

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(cData CommandData, path, name, publicName string, public bool, replaceFile uint) {
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
	wg := sync.WaitGroup{}
	done := make(chan int8, 1)
	c := make(chan int64, 1)

	// Init progressbar and proxy
	proxy := libdm.NoProxyWriter
	if !cData.Quiet {
		bar = pb.New64(0).SetRefreshRate(10 * time.Millisecond).SetMaxWidth(100)
		proxy = func(w io.Writer) io.Writer {
			return bar.NewProxyWriter(w)
		}
	}

	// Start upload
	go (func(wg *sync.WaitGroup, path, fileName string, public bool, replaceFile uint, fileattributes libdm.FileAttributes, proxy func(io.Writer) io.Writer, c chan int64, done chan int8, publicName, encryption, encryptionKey string) {
		wg.Add(1)
		uploadResponse, err = cData.LibDM.UploadFile(path, fileName, public, replaceFile, fileattributes, proxy, c, done, publicName, encryption, encryptionKey)
		wg.Done()
	})(&wg, path, fileName, public, replaceFile, cData.FileAttributes, proxy, c, done, publicName, cData.Encryption, cData.EncryptionKey)

	// Read filesize and set bars total to filesize
	fileSize := <-c
	if bar != nil {
		bar.SetTotal(fileSize)
		bar.Start()
	}

	// Wait for request and
	// upload to be finished
	wg.Wait()
	<-done

	// Stop bar
	if bar != nil {
		bar.Finish()
	}

	if err != nil || uploadResponse == nil {
		printResponseError(err, "uploading file")
		return
	}

	// Print output
	if cData.OutputJSON {
		fmt.Println(toJSON(uploadResponse))
	} else {
		if len(uploadResponse.PublicFilename) != 0 {
			fmt.Printf("Public name: %s\nName: %s\nID %d\n", cData.Config.GetPreviewURL(uploadResponse.PublicFilename), fileName, uploadResponse.FileID)
		} else {
			fmt.Printf("Name: %s\nID: %d\n", fileName, uploadResponse.FileID)
		}
	}
}

// DeleteFile deletes the desired file(s)
func DeleteFile(cData CommandData, name string, id uint) {
	// Convert input
	name, id = getFileCommandData(name, id)

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

	if resp.Count > 1 {
		fmt.Printf("Deleted %d files %s\n", resp.Count, color.HiGreenString("successfully"))
	} else {
		fmt.Printf("The file has been %s\n", color.HiGreenString("successfully deleted"))
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
			if order := models.FileOrderFromString(sOrder); order != nil {
				// Sort
				models.
					NewFileSorter(resp.Files).
					Reversed(models.IsOrderReversed(sOrder)).
					SortBy(*order)
			} else {
				fmt.Printf("Error: Sort by '%s' not supporded", sOrder)
				return
			}
		} else {
			// By default sort by creation desc
			models.NewFileSorter(resp.Files).Reversed(true).SortBy(models.CreatedOrder)
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
		fmt.Println("Illegal flag combination")
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

	// magic
	showBar := (!cData.Quiet && !displayOutput) || (len(args) > 0 && args[0])

	// Errorhandling 100
	if noPreview && preview {
		fmt.Print("rlly?")
		return
	}

	// Do http request
	resp, serverFileName, err := cData.LibDM.GetFile(fileName, id, cData.Namespace)
	if err != nil {
		printResponseError(err, "downloading file")
		return
	}

	var respData io.Reader

	// Set respData to designed source
	if cData.NoDecrypt {
		respData = resp.Body
	} else {
		respData, err = respToDecrypted(&cData, resp)
		encryption = resp.Header.Get(libdm.HeaderEncryption)
		if err != nil {
			log.Fatal(err)
		}
	}

	var bar *pb.ProgressBar

	// Show bar on download
	if showBar {
		// Get filesize header
		var size int64
		if len(resp.Header.Get(libdm.HeaderContentLength)) > 0 {
			s, err := strconv.ParseInt(resp.Header.Get(libdm.HeaderContentLength), 10, 64)
			if err == nil {
				size = s
			}
		}

		// Create and hook bar
		bar = pb.New64(size).SetMaxWidth(100).SetRefreshRate(10 * time.Millisecond).Start()
		respData = bar.NewProxyReader(respData)
	}

	// Display or save file
	if displayOutput && len(savePath) == 0 {
		// Only write to tmpfile if preview needed
		if shouldPreview {
			file, err := SaveToTempFile(respData, serverFileName)
			if bar != nil {
				bar.Finish()
			}

			if err != nil {
				fmt.Printf("%s writing temporary file: %s\n", color.HiRedString("Error:"), err)
				return
			}

			// Preview file
			previewFile(file)

			// Shredder/Delete file
			ShredderFile(file, -1)
		} else {
			// Printf like a boss
			io.Copy(os.Stdout, respData)
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

		// Create or truncate file
		f, err := os.Create(outFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Vrite file
		_, err = io.Copy(f, respData)
		if err != nil {
			fmt.Println(err)
			return
		}

		if bar != nil {
			bar.Finish()
		}

		// Close file
		f.Close()

		// Preview
		if displayOutput {
			previewFile(savePath)
		}

		if !cData.Quiet {
			// Print success message
			fmt.Printf("Saved file into %s\n", outFile)
		}
	} else if !displayOutput && len(savePath) == 0 {
		fmt.Println("Can't save file if you don't specify a path.")
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

	// Delete temp file
	defer func() {
		ShredderFile(filePath, -1)
	}()

	// Download File
	success, encryption, serverName := GetFile(cData, "", id, filePath, false, true, false, !cData.Quiet)
	if !success {
		return
	}

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
	UploadFile(cData, filePath, serverName, "", false, id)
}

func editFile(file string) bool {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "/usr/bin/nano"
	}

	// Check editor
	if _, err := os.Stat(editor); err != nil {
		fmt.Println("Error finding editor. Either install nano or set $EDITOR to your desired editor")
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
