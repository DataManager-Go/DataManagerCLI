package commands

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gaw "github.com/JojiiOfficial/GoAw"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"
	"github.com/fatih/color"
	humanTime "github.com/sbani/go-humanizer/time"
	"github.com/sbani/go-humanizer/units"
	clitable "gopkg.in/benweidig/cli-table.v2"
)

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(config *models.Config, path, name string, attributes models.FileAttributes) {
	_, fileName := filepath.Split(path)
	if len(name) != 0 {
		fileName = name
	}

	request := server.UploadRequest{
		Name:       fileName,
		Attributes: attributes,
	}

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

		request.UploadType = server.FileUploadType
		request.Data = fileBytes
		request.Sum = GetMD5Hash(fileBytes)
	}

	//Do request
	var resStruct server.UploadResponse
	response, err := server.NewRequest(server.EPFileUpload, request, config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: config.User.SessionToken,
	}).Do(&resStruct)

	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	if response.Status == server.ResponseError {
		printResponseError(response, "uploading your file")
		return
	}

	fmt.Printf("Name: %s\nID: %d\n", fileName, resStruct.FileID)
}

// DeleteFile deletes the desired file(s)
func DeleteFile(config *models.Config, name string, id int, attributes models.FileAttributes) {
	response, err := server.NewRequest(server.EPFileDelete, &server.FileUpdateRequest{
		Name:       name,
		FileID:     id,
		Attributes: attributes,
	}, config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: config.User.SessionToken,
	}).Do(nil)

	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	if response.Status == server.ResponseError {
		printResponseError(response, "trying to delete your file")
		return
	}

	fmt.Printf("The file has been %s\n", color.HiGreenString("successfully deleted"))
}

// ListFiles lists the files corresponding to the args
func ListFiles(config *models.Config, name string, id uint, attributes models.FileAttributes, verbosity uint8) {
	var filesResponse server.FileListResponse
	response, err := server.NewRequest(server.EPFileList, &server.FileRequest{
		FileID:     id,
		Name:       name,
		Attributes: attributes,
		OptionalParams: server.OptionalRequetsParameter{
			Verbose: verbosity,
		},
	}, config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: config.User.SessionToken,
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

	// Output
	fmt.Printf("There were %s found in '%s'\n", color.HiGreenString(strconv.Itoa(len(filesResponse.Files))+" files"), attributes.Namespace)

	if uint16(len(filesResponse.Files)) > config.Client.MinFilesToDisplay {
		y, _ := gaw.ConfirmInput("Do you want to view all? (y/n) > ", bufio.NewReader(os.Stdin))
		if !y {
			return
		}
	}

	// Print files
	headingColor := color.New(color.FgHiGreen, color.Underline, color.Bold)

	table := clitable.New()
	table.ColSeparator = " "
	table.Padding = 7

	header := []interface{}{
		headingColor.Sprint("ID"), headingColor.Sprint("Name"), headingColor.Sprint("Size"), headingColor.Sprint("Created"),
	}

	//Show namespace on -dd
	if verbosity > 2 {
		header = append(header, headingColor.Sprintf("Namespace"))
	}
	//Show groups and tags on -d
	if verbosity > 1 {
		header = append(header, headingColor.Sprintf("Groups"))
		header = append(header, headingColor.Sprintf("Tags"))
	}

	table.AddRow(header...)

	for _, file := range filesResponse.Files {
		rowItems := []interface{}{
			file.ID,
			file.Name,
			units.BinarySuffix(float64(file.Size)),
			humanTime.Difference(time.Now(), file.CreationDate),
		}

		//Show namespace on -dd
		if verbosity > 2 {
			rowItems = append(rowItems, file.Attributes.Namespace)
		}

		//Show groups and tags on -d
		if verbosity > 1 {
			rowItems = append(rowItems, strings.Join(file.Attributes.Groups, ", "))
			rowItems = append(rowItems, strings.Join(file.Attributes.Tags, ", "))
		}

		table.AddRow(rowItems...)
	}

	fmt.Println(table.String())
}

// UpdateFile updates a file on the server
func UpdateFile(config *models.Config, name string, id int, namespace string, isPublic string, newName string, newNamespace string, addTags []string, removeTags []string, addGroups []string, removeGroups []string) {

	// Set attributes
	attributes := models.FileAttributes{
		Tags:      addTags,
		Groups:    addGroups,
		Namespace: namespace,
	}

	// Set fileUpdates
	fileUpdates := models.FileUpdateItem{
		IsPublic:     isPublic,
		NewName:      newName,
		NewNamespace: newNamespace,
		RemoveTags:   removeTags,
		RemoveGroups: removeGroups,
	}

	// Combine and send it
	response, err := server.NewRequest(server.EPFileUpdate, &server.FileUpdateRequest{
		Name:       name,
		FileID:     id,
		Updates:    fileUpdates,
		Attributes: attributes,
	}, config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: config.User.SessionToken,
	}).Do(nil)

	// Error handling #1
	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	// Error handling #2
	if response.Status == server.ResponseError {
		printResponseError(response, "trying to update your file")
		return
	}

	// Output
	fmt.Printf("The file has been %s\n", color.HiGreenString("successfully updated"))
}

// UpdateTag updates a given tag
func UpdateTag(config *models.Config, name string, namespace string, newName string, delete bool) {

	// Combine and send it
	response, err := server.NewRequest(server.EPTagUpdate, &server.TagUpdateRequest{
		Name:      name,
		NewName:   newName,
		Namespace: namespace,
		Delete:    delete,
	}, config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: config.User.SessionToken,
	}).Do(nil)

	// Error handling #1
	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	// Error handling #2
	if response.Status == server.ResponseError {
		printResponseError(response, "trying to update your tag")
		return
	}

	// Output
	fmt.Printf("The tag has been %s\n", color.HiGreenString("successfully updated"))
}

// DownloadFile requests the file from the server
func DownloadFile(name string, namespace string, groups []string, tags []string, id uint, savePath string) {

	/*if err != nil || response.Status == server.ResponseError {
		println("File was not downloaded:\n" + response.Message)
		return
	}

	// TODO Make it pretty and fix obvious issues here
	ioutil.WriteFile(*savePath+"/"+foundFile.FileName, foundFile.FileData, 6400)
	*/
}
