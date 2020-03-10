package commands

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/JojiiOfficial/gaw"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"
	"github.com/fatih/color"
	"github.com/h2non/filetype"
	humanTime "github.com/sbani/go-humanizer/time"
	"github.com/sbani/go-humanizer/units"
	clitable "gopkg.in/benweidig/cli-table.v2"
)

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(cData CommandData, path, name, publicName string, public bool) {
	_, fileName := filepath.Split(path)
	if len(name) != 0 {
		fileName = name
	}

	//Make public if public name was specified
	if len(publicName) > 0 {
		public = true
	}

	//bulid request
	request := server.UploadRequest{
		Name:       fileName,
		Attributes: cData.FileAttributes,
		Public:     public,
		PublicName: publicName,
	}

	//Check for url/file
	u, err := url.Parse(path)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		request.UploadType = server.URLUploadType
		request.URL = path
	} else {
		fileBytes, err := ioutil.ReadFile(path)

		if err != nil {
			printError("processing your file. Please check your input")
			return
		}

		//Try to detect filetype
		ft, err := filetype.Get(fileBytes)
		if err == nil {
			//Set filetype if detected successfully
			request.FileType = ft.MIME.Value
		}

		request.UploadType = server.FileUploadType
		request.Data = base64.StdEncoding.EncodeToString(fileBytes)
		request.Sum = gaw.GetMD5Hash(request.Data)
	}

	//Do request
	var resStruct server.UploadResponse
	response, err := server.NewRequest(server.EPFileUpload, request, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).Do(&resStruct)

	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	//Verifying response status
	if response.Status == server.ResponseError {
		printResponseError(response, "uploading your file")
		return
	}

	//print output
	if cData.OutputJSON {
		fmt.Println(toJSON(resStruct))
	} else {
		fmt.Printf("Name: %s\nID: %d\n", fileName, resStruct.FileID)
	}
}

// DeleteFile deletes the desired file(s)
func DeleteFile(cData CommandData, name string, id uint) {
	var response server.CountResponse
	resp, err := server.NewRequest(server.EPFileDelete, &server.FileRequest{
		Name:       name,
		FileID:     id,
		All:        cData.All,
		Attributes: cData.FileAttributes,
	}, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).Do(&response)

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
func ListFiles(cData CommandData, name string, id uint) {
	var filesResponse server.FileListResponse
	response, err := server.NewRequest(server.EPFileList, &server.FileListRequest{
		FileID:     id,
		Name:       name,
		Attributes: cData.FileAttributes,
		OptionalParams: server.OptionalRequetsParameter{
			Verbose: cData.Details,
		},
	}, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).Do(&filesResponse)

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

	//Print as json if desired
	if cData.OutputJSON {
		fmt.Println(toJSON(filesResponse.Files))
	} else {
		fmt.Printf("There were %s found in '%s'\n", color.HiGreenString(strconv.Itoa(len(filesResponse.Files))+" files"), cData.Namespace)
		// Print files
		headingColor := color.New(color.FgHiGreen, color.Underline, color.Bold)

		table := clitable.New()
		table.ColSeparator = " "
		table.Padding = 4

		header := []interface{}{
			headingColor.Sprint("ID"), headingColor.Sprint("Name"), headingColor.Sprint("Size"), headingColor.Sprint("Public name"), headingColor.Sprint("Created"),
		}

		//Show namespace on -dd
		if cData.Details > 2 {
			header = append(header, headingColor.Sprintf("Namespace"))
		}
		//Show groups and tags on -d
		if cData.Details > 1 {
			header = append(header, headingColor.Sprintf("Groups"))
			header = append(header, headingColor.Sprintf("Tags"))
		}

		table.AddRow(header...)

		for _, file := range filesResponse.Files {
			//Colorize private pubNames if not public
			pubname := file.PublicName
			if len(pubname) > 0 && !file.IsPublic {
				pubname = color.HiMagentaString(pubname)
			}

			//Add items
			rowItems := []interface{}{
				file.ID,
				file.Name,
				units.BinarySuffix(float64(file.Size)),
				pubname,
				humanTime.Difference(time.Now(), file.CreationDate),
			}

			//Show namespace on -dd
			if cData.Details > 2 {
				rowItems = append(rowItems, file.Attributes.Namespace)
			}

			//Show groups and tags on -d
			if cData.Details > 1 {
				rowItems = append(rowItems, strings.Join(file.Attributes.Groups, ", "))
				rowItems = append(rowItems, strings.Join(file.Attributes.Tags, ", "))
			}

			table.AddRow(rowItems...)
		}

		fmt.Println(table.String())
	}
}

//PublishFile publishes a file
func PublishFile(cData CommandData, name string, id uint, publicName string) {
	var resData server.PublishResponse
	response, err := server.NewRequest(server.EPFilePublish, server.FileRequest{
		Name:       name,
		FileID:     id,
		PublicName: publicName,
		All:        cData.All,
		Attributes: cData.FileAttributes,
	}, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).Do(&resData)

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
		fmt.Println(toJSON(resData))
	} else {
		fmt.Printf(resData.PublicFilename)
	}
}

// UpdateFile updates a file on the server
func UpdateFile(cData CommandData, name string, id uint, newName string, newNamespace string, addTags []string, removeTags []string, addGroups []string, removeGroups []string, setPublic, setPrivate bool) {
	//Process params: make t1,t2 -> [t1 t2]
	ProcesStrSliceParams(&addTags, &addGroups, &removeTags, &removeGroups)

	// Set attributes
	attributes := models.FileAttributes{
		Namespace: cData.Namespace,
	}

	//Can't use both
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
	}).Do(&response)

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
func GetFile(cData CommandData, fileName string, id uint, savePath string, displayOutput, noPreview, preview bool) {
	shouldPreview := cData.Config.Client.AutoFilePreview || preview
	if noPreview {
		fmt.Println("noPreview")
		shouldPreview = false
	}

	//Errorhandling 100
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

	//Check for error
	if err != nil {
		fmt.Println(err)
		return
	}

	//Check response headers
	if resp.Header.Get(server.HeaderStatus) == strconv.Itoa(int(server.ResponseError)) {
		statusMessage := resp.Header.Get(server.HeaderStatusMessage)
		fmt.Println(color.HiRedString("Error: ") + statusMessage)
		return
	}

	//Get filename from response headers
	serverFileName := resp.Header.Get(server.HeaderFileName)

	//Check headers
	if len(serverFileName) == 0 {
		fmt.Println(color.HiRedString("Error:") + " Received corrupted Data from the server")
		return
	}

	//Display or save file
	if displayOutput && len(savePath) == 0 {
		//Only write to tmpfile if preview needed
		if shouldPreview {
			file, err := SaveToTempFile(resp.Body, serverFileName)
			if err != nil {
				fmt.Printf("%s writing temporary file: %s\n", color.HiRedString("Error:"), err)
				return
			}

			//Preview file
			previewFile(file)
		} else {
			//Printf like a boss
			io.Copy(os.Stdout, resp.Body)
		}
	} else if len(savePath) > 0 {
		//Determine output file/path
		outFile := savePath
		if strings.HasSuffix(savePath, "/") {
			outFile = filepath.Join(savePath, fileName)
		} else {
			stat, err := os.Stat(outFile)
			if err == nil && stat.IsDir() {
				outFile = filepath.Join(outFile, fileName)
			}
		}

		//Create or truncate file
		f, err := os.Create(outFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		//Write file
		_, err = io.Copy(f, resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		//Close file
		f.Close()

		// Preview
		if displayOutput {
			previewFile(savePath)
		}

		//Print success message
		fmt.Printf("Saved file into %s\n", outFile)
	} else if !displayOutput && len(savePath) == 0 {
		fmt.Println("Can't save file if you don't specify a path.")
		return
	}

	//Close body
	resp.Body.Close()
}
