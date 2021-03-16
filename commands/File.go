package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/fatih/color"
	humanTime "github.com/sbani/go-humanizer/time"
	"github.com/sbani/go-humanizer/units"
	clitable "gopkg.in/benweidig/cli-table.v2"
)

// DeleteFile deletes the desired file(s)
func DeleteFile(cData *CommandData, name string, id uint) {
	// Convert input
	name, id = GetFileCommandData(name, id)

	if len(strings.TrimSpace(name)) == 0 && id <= 0 {
		fmtError("Missing a valid parameter. Provide fileID or Filename")
		return
	}

	// Confirm 'delete everything'
	if strings.TrimSpace(name) == "%" &&
		!cData.Yes &&
		cData.All &&
		len(cData.FileAttributes.Tags) == 0 &&
		len(cData.FileAttributes.Groups) == 0 {

		if i, _ := gaw.ConfirmInput("Do you really want to delete all files in "+cData.Namespace+"? (y/n)> ", bufio.NewReader(os.Stdin)); !i {
			return
		}
	}

	// Do delete request
	resp, err := cData.LibDM.DeleteFile(name, id, cData.All, cData.FileAttributes)
	if err != nil {
		printResponseError(err, "deleting file")
		return
	}

	// Print correct success message
	if len(resp.IDs) > 1 {
		fmt.Printf("Deleted %d files %s\n", len(resp.IDs), color.HiGreenString("successfully"))
	} else {
		fmt.Printf("The file has been %s\n", color.HiGreenString("successfully deleted"))
	}

	// rm keys from keystore
	if cData.HasKeystoreSupport() {
		keystore, _ := cData.GetKeystore()
		rmFilesFromkeystore(keystore, resp.IDs)
	}
}

// ListFiles lists the files corresponding to the args
func ListFiles(cData *CommandData, name string, id uint, sOrder string) {
	// Convert input
	name, id = GetFileCommandData(name, id)

	// Do ListFile request
	resp, err := cData.LibDM.ListFiles(name, id, cData.All, cData.FileAttributes, cData.Details)
	if err != nil {
		printResponseError(err, "listing files")
		return
	}

	// Request user confirmation if files are too much
	if uint16(len(resp.Files)) > cData.Config.Client.MinFilesToDisplay && !cData.Yes {
		if y, _ := gaw.ConfirmInput("Do you want to view all? (y/n) > ", bufio.NewReader(os.Stdin)); !y {
			return
		}
	}

	// Print as json if desired
	if cData.OutputJSON {
		fmt.Println(toJSON(resp.Files))
	} else {
		if len(resp.Files) == 0 {
			if cData.isFilterUsed() {
				fmt.Println("No files found using given filter")
			} else {
				fmt.Printf("No files in namespace %s\n", cData.FileAttributes.Namespace)
			}

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

		refFiles := fileSliceToRef(resp.Files)

		if !sortFiles(sOrder, refFiles) {
			return
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
		if cData.Details > 2 || cData.All {
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

		// don't add the head row
		// on quiet mode
		if !cData.Quiet {
			table.AddRow(header...)
		}

		for i, file := range refFiles {
			// Toggle between two colors with each new line to make it easier to read
			bgColor := color.New(color.FgHiWhite).Sprintf
			if i%2 != 0 {
				//bgColor = color.BgBlue
				bgColor = color.New(color.FgHiBlack).Sprintf
			}

			// Colorize private pubNames if not public
			pubname := file.PublicName
			if len(pubname) > 0 && !file.IsPublic {
				pubname = color.HiMagentaString(pubname)
			} else {
				pubname = bgColor(pubname)
			}

			// Add items
			rowItems := []interface{}{
				bgColor("%d", file.ID),
				bgColor("%s", formatFilename(file, cData.NameLen, cData)),
				bgColor("%s", units.BinarySuffix(float64(file.Size))),
			}

			// Append public file
			if hasPublicFile {
				rowItems = append(rowItems, pubname)
			}

			// Append time
			rowItems = append(rowItems, bgColor("%s", humanTime.Difference(time.Now(), file.CreationDate)))

			// Show namespace on -dd
			if cData.Details > 2 || cData.All {
				rowItems = append(rowItems, bgColor("%s", file.Attributes.Namespace))
			}

			// Show groups and tags on -d
			if cData.Details > 1 {
				if hasGroup {
					rowItems = append(rowItems, bgColor("%s", strings.Join(file.Attributes.Groups, ", ")))
				}

				if hasTag {
					rowItems = append(rowItems, bgColor("%s", strings.Join(file.Attributes.Tags, ", ")))
				}
			}

			table.AddRow(rowItems...)
		}

		fmt.Println(table)
	}
}

// PublishFile publishes a file
func PublishFile(cData *CommandData, name string, id uint, publicName string, setClip bool) {
	// Convert input
	name, id = GetFileCommandData(name, id)

	if cData.All && len(publicName) > 0 && len(name) > 0 {
		fmt.Println("You can't set the public name of multiple files")
		return
	}

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
			rs := (resp).(libdm.BulkPublishResponse)
			fmt.Printf(cData.Config.GetPreviewURL(rs.Files[0].PublicFilename))

			if setClip {
				cData.setClipboard(rs.Files[0].PublicFilename)
			}
		}
	}
}

// UnPublishFile makes a public file private
func UnPublishFile(cData *CommandData, name string, id uint) {
	// Convert input
	name, id = GetFileCommandData(name, id)
	UpdateFile(cData, name, id, "", "", []string{}, []string{}, []string{}, []string{}, false, true)
}

// UpdateFile updates a file on the server
func UpdateFile(cData *CommandData, name string, id uint, newName string, newNamespace string, addTags []string, removeTags []string, addGroups []string, removeGroups []string, setPublic, setPrivate bool) {
	// Process params: make t1,t2 -> [t1 t2]
	ProcesStrSliceParams(&addTags, &addGroups, &removeTags, &removeGroups)

	// Convert input
	name, id = GetFileCommandData(name, id)

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

	count := len(response.IDs)
	if count > 1 {
		fmt.Printf("Updated %d files %s\n", count, color.HiGreenString("successfully"))
	} else {
		fmt.Printf("The file has been %s\n", color.HiGreenString("successfully updated"))
	}
}

// CreateFile create a file and upload it
func (cData *CommandData) CreateFile(name string) {
	// Create tempfile
	file := createTempFile(&name)
	if len(file) == 0 {
		return
	}

	var success bool
	fmt.Printf("File %s created\n", file)

	// Shredder file at the end
	defer func() {
		if !success {
			if !cData.Yes {
				if y, _ := gaw.ConfirmInput("Upload was unsuccessful. Do you want to delete the local file? (y/n)> ", bufio.NewReader(os.Stdin)); !y {
					return
				}
			}

			defer func() {
				fmt.Printf("%s Deleted", file)
			}()
		}

		ShredderFile(file, -1)
	}()

	// Open file for user "editing"
	if !editFile(file, "") {
		return
	}

	// Open temp file
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		printError("open tempfile", err.Error())
		return
	}

	// Get fileinfo
	stat, err := f.Stat()
	if err != nil {
		printError("open tempfile", err.Error())
		return
	}

	// Return if file is empty
	if stat.Size() == 0 {
		success = true
		return
	}

	// Upload file
	chDone := make(chan string, 1)
	request := cData.LibDM.NewUploadRequest(name, cData.FileAttributes)
	resp, err := request.UploadFile(f, chDone, nil)
	if err != nil {
		printResponseError(err, "uploading")
		return
	}

	sum := <-chDone
	localchecksum := fileCrc32(file)

	if len(sum) == 0 || sum != localchecksum {
		fmt.Println(cData.getChecksumError(localchecksum, sum))
		return
	}

	success = true
	cData.printUploadResponse(resp, &UploadData{
		Name: name,
	}, cData.Quiet, nil)
}

// FileTree shows a unix tree like view of files
func (cData *CommandData) FileTree(sOrder, namespace string) {
	// Get requested namespace. If no ns was set, show all files
	cData.FileAttributes.Namespace = cData.getRealNamespace()
	if len(cData.FileAttributes.Namespace) == 0 && len(namespace) > 0 {
		cData.FileAttributes.Namespace = namespace
	}
	cData.All = len(cData.FileAttributes.Namespace) == 0

	// Do file list request
	resp, err := cData.LibDM.ListFiles("", 0, cData.All, cData.FileAttributes, 3)
	if err != nil {
		printResponseError(err, "getting files")
		return
	}

	if len(resp.Files) == 0 {
		fmt.Println("No files found")
		return
	}

	cData.renderTree(resp.Files, sOrder)
}
