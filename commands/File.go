package commands

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/JojiiOfficial/gaw"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"
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

	// Bulid request
	request := server.UploadRequest{
		Name:       fileName,
		Attributes: cData.FileAttributes,
		Public:     public,
		PublicName: publicName,
		Encryption: cData.Encryption,
	}

	if replaceFile != 0 {
		request.ReplaceFile = replaceFile
	}

	var contentType string
	var body io.Reader

	// Check for url/file
	u, err := url.Parse(path)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		request.UploadType = server.URLUploadType
		request.URL = path
		contentType = string(server.JSONContentType)
	} else {
		// Init upload stuff
		body, contentType, request.Size = uploadFile(&cData, path, !cData.Quiet)
	}

	// Make json header content
	rbody, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Invalid Json:", err)
		return
	}
	rBase := base64.StdEncoding.EncodeToString(rbody)

	// Do request
	var resStruct server.UploadResponse
	response, err := server.
		NewRequest(server.EPFileUpload, body, cData.Config).
		WithMethod(server.PUT).
		WithAuth(server.Authorization{
			Type:    server.Bearer,
			Palyoad: cData.Config.User.SessionToken,
		}).WithHeader(server.HeaderRequest, rBase).
		WithRequestType(server.RawRequestType).
		WithContentType(server.ContentType(contentType)).
		WithBenchCallback(cData.BenchDone).
		Do(&resStruct)

	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	// Verifying response status
	if response.Status == server.ResponseError {
		printResponseError(response, "uploading your file")
		return
	}

	// Print output
	if cData.OutputJSON {
		fmt.Println(toJSON(resStruct))
	} else {
		if len(resStruct.PublicFilename) != 0 {
			fmt.Printf("Public name: %s\nName: %s\nID %d\n", cData.Config.GetPreviewURL(resStruct.PublicFilename), fileName, resStruct.FileID)
		} else {
			fmt.Printf("Name: %s\nID: %d\n", fileName, resStruct.FileID)
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

	var response server.CountResponse
	resp, err := server.NewRequest(server.EPFileDelete, &server.FileRequest{
		Name:       name,
		FileID:     id,
		All:        cData.All,
		Attributes: cData.FileAttributes,
	}, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).WithBenchCallback(cData.BenchDone).Do(&response)

	if err != nil {
		if resp != nil {
			fmt.Println("http:", resp.HTTPCode)
			return
		}

		log.Fatalln(err)
	}

	if resp.Status == server.ResponseError {
		printResponseError(resp, "trying to delete your file")
		return
	}

	if response.Count > 1 {
		fmt.Printf("Deleted %d files %s\n", response.Count, color.HiGreenString("successfully"))
	} else {
		fmt.Printf("The file has been %s\n", color.HiGreenString("successfully deleted"))
	}
}

// ListFiles lists the files corresponding to the args
func ListFiles(cData CommandData, name string, id uint, sOrder string) {
	// Convert input
	name, id = getFileCommandData(name, id)

	var filesResponse server.FileListResponse
	response, err := server.NewRequest(server.EPFileList, &server.FileListRequest{
		FileID:        id,
		Name:          name,
		AllNamespaces: cData.AllNamespaces,
		Attributes:    cData.FileAttributes,
		OptionalParams: server.OptionalRequetsParameter{
			Verbose: cData.Details,
		},
	}, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).WithBenchCallback(cData.BenchDone).Do(&filesResponse)

	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	if response.Status == server.ResponseError {
		printResponseError(response, "listing files")
		return
	}

	if uint16(len(filesResponse.Files)) > cData.Config.Client.MinFilesToDisplay && !cData.Yes {
		y, _ := gaw.ConfirmInput("Do you want to view all? (y/n) > ", bufio.NewReader(os.Stdin))
		if !y {
			return
		}
	}

	// Print as json if desired
	if cData.OutputJSON {
		fmt.Println(toJSON(filesResponse.Files))
	} else {
		if len(filesResponse.Files) == 0 {
			fmt.Printf("No files in namespace %s\n", cData.Namespace)
			return
		}

		headingColor := color.New(color.FgHiGreen, color.Underline, color.Bold)

		// Table setup
		table := clitable.New()
		table.ColSeparator = " "
		table.Padding = 4

		var hasPublicFile, hasTag, hasGroup bool

		// scan for availability of attributes
		for _, file := range filesResponse.Files {
			if !hasPublicFile && file.IsPublic && len(file.PublicName) > 0 {
				hasPublicFile = true
			}

			// only need to do if requested more details
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
					NewFileSorter(filesResponse.Files).
					Reversed(models.IsOrderReversed(sOrder)).
					SortBy(*order)
			} else {
				fmt.Printf("Error: Sort by '%s' not supporded", sOrder)
				return
			}
		} else {
			// By default sort by creation desc
			models.NewFileSorter(filesResponse.Files).Reversed(true).SortBy(models.CreatedOrder)
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

		for _, file := range filesResponse.Files {
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

	request := server.NewRequest(server.EPFilePublish, server.FileRequest{
		Name:       name,
		FileID:     id,
		PublicName: publicName,
		All:        cData.All,
		Attributes: cData.FileAttributes,
	}, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).WithBenchCallback(cData.BenchDone)

	var err error
	var response *server.RestRequestResponse
	var resp interface{}

	if cData.All {
		var respData server.BulkPublishResponse
		response, err = request.Do(&respData)
		resp = respData
	} else {
		var respData server.PublishResponse
		response, err = request.Do(&respData)
		resp = respData
	}

	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	// Error handling #2
	if response.Status == server.ResponseError {
		printResponseError(response, "publishing")
		return
	}

	// Output
	if cData.OutputJSON {
		fmt.Println(toJSON(resp))
	} else {
		if cData.All {
			rs := (resp).(server.BulkPublishResponse)

			fmt.Printf("Published %d files\n", len(rs.Files))
			for _, file := range rs.Files {
				fmt.Printf("File %s with ID %d Public name: %s\n", file.Filename, file.FileID, file.PublicFilename)
			}
		} else {
			pubName := (resp.(server.PublishResponse)).PublicFilename
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

	// Set attributes
	attributes := models.FileAttributes{
		Namespace: cData.Namespace,
	}

	// Can't use both
	if setPrivate && setPublic {
		fmt.Println("Illegal flag combination")
		return
	}

	var isPublic string
	if setPublic {
		isPublic = "true"
	}
	if setPrivate {
		isPublic = "false"
	}

	// Set fileUpdates
	fileUpdates := models.FileUpdateItem{
		IsPublic:     isPublic,
		NewName:      newName,
		NewNamespace: newNamespace,
		RemoveTags:   removeTags,
		RemoveGroups: removeGroups,
		AddTags:      addTags,
		AddGroups:    addGroups,
	}

	var response server.CountResponse

	// Combine and send it
	resp, err := server.NewRequest(server.EPFileUpdate, &server.FileRequest{
		Name:       name,
		FileID:     id,
		All:        cData.All,
		Updates:    fileUpdates,
		Attributes: attributes,
	}, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).WithBenchCallback(cData.BenchDone).Do(&response)

	// Error handling #1
	if err != nil {
		if resp != nil {
			fmt.Println("http:", resp.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	// Error handling #2
	if resp.Status == server.ResponseError {
		printResponseError(resp, "trying to update your file")
		return
	}

	if response.Count > 1 {
		fmt.Printf("Updated %d files %s\n", response.Count, color.HiGreenString("successfully"))
	} else {
		fmt.Printf("The file has been %s\n", color.HiGreenString("successfully updated"))
	}
}

// GetFile requests the file from the server and displays or saves it
func GetFile(cData CommandData, fileName string, id uint, savePath string, displayOutput, noPreview, preview bool) (success bool, encryption, serverFileName string) {
	// Convert input
	fileName, id = getFileCommandData(fileName, id)

	shouldPreview := cData.Config.Client.AutoFilePreview || preview
	if noPreview {
		shouldPreview = false
	}

	// Errorhandling 100
	if noPreview && preview {
		fmt.Print("rlly?")
		return
	}

	resp, err := server.NewRequest(server.EPFileGet, &server.FileRequest{
		Name:   fileName,
		FileID: id,
		Attributes: models.FileAttributes{
			Namespace: cData.Namespace,
		},
	}, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).DoHTTPRequest()

	// Check for error
	if err != nil {
		fmt.Println(err)
		return
	}

	// Check response headers
	if resp.Header.Get(server.HeaderStatus) == strconv.Itoa(int(server.ResponseError)) {
		statusMessage := resp.Header.Get(server.HeaderStatusMessage)
		fmt.Println(color.HiRedString("Error: ") + statusMessage)
		return
	}

	// Get filename from response headers
	serverFileName = resp.Header.Get(server.HeaderFileName)

	// Check headers
	if len(serverFileName) == 0 {
		fmt.Println(color.HiRedString("Error:") + " Received corrupted Data from the server")
		return
	}

	var respData io.Reader

	// Set respData to designed source
	if cData.NoDecrypt {
		respData = resp.Body
	} else {
		respData, err = respToDecrypted(&cData, resp)
		encryption = resp.Header.Get(server.HeaderEncryption)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Display or save file
	if displayOutput && len(savePath) == 0 {
		// Only write to tmpfile if preview needed
		if shouldPreview {
			file, err := SaveToTempFile(respData, serverFileName)
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

		// Close file
		f.Close()

		// Preview
		if displayOutput {
			previewFile(savePath)
		}

		// Print success message
		fmt.Printf("Saved file into %s\n", outFile)
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
	success, encryption, serverName := GetFile(cData, "", id, filePath, false, true, false)
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
