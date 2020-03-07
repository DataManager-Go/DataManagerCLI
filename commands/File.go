package commands

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	gaw "github.com/JojiiOfficial/GoAw"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"
	"github.com/fatih/color"
)

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(config *models.Config, path string, namespace string, groups []string, tags []string) {
	_, fileName := filepath.Split(path)

	fileBytes, err := ioutil.ReadFile(path)

	if err != nil {
		printError("processing your file. Please check your input")
	}

	var resStruct server.UploadResponse
	response, err := server.NewRequest(server.EPFileUpload, &server.UploadStruct{
		Data: fileBytes,
		Name: fileName,
		Sum:  GetMD5Hash(fileBytes),
		Attributes: models.FileAttributes{
			Namespace: namespace,
			Groups:    groups,
			Tags:      tags,
		},
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
func DeleteFile(config *models.Config, name string, namespace string, groups []string, tags []string, id int) {
	response, err := server.NewRequest(server.EPFileDelete, &server.FileUpdateRequest{
		Name:   name,
		FileID: id,
		Attributes: models.FileAttributes{
			Namespace: namespace,
		},
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
func ListFiles(config *models.Config, name string, namespace string, groups []string, tags []string, id uint) {
	var filesResponse server.FileListResponse
	response, err := server.NewRequest(server.EPFileList, &server.FileRequest{
		FileID: id,
		Name:   name,
		Attributes: models.FileAttributes{
			Groups:    groups,
			Namespace: namespace,
			Tags:      tags,
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
	for i := 0; i < len(filesResponse.Files); i++ {
		fmt.Printf("%d: %s\n", filesResponse.Files[i].ID, filesResponse.Files[i].Name)
	}
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
