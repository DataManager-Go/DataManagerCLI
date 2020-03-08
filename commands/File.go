package commands

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
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
func UploadFile(config *models.Config, path string, attributes models.FileAttributes) {
	_, fileName := filepath.Split(path)

	fileBytes, err := ioutil.ReadFile(path)

	if err != nil {
		printError("processing your file. Please check your input")
	}

	var resStruct server.UploadResponse
	response, err := server.NewRequest(server.EPFileUpload, &server.UploadStruct{
		Data:       fileBytes,
		Name:       fileName,
		Sum:        GetMD5Hash(fileBytes),
		Attributes: attributes,
	}, config).WithAuth(server.Authorization{
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
	fmt.Printf("There were %s found\n", color.HiGreenString(strconv.Itoa(len(filesResponse.Files))+" files"))

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
