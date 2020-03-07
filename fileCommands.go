package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"
)

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(path string, namespace string, groups []string, tags []string) {
	_, fileName := filepath.Split(path)

	fileBytes, err := ioutil.ReadFile(path)

	if err != nil {
		println("Error processing your file. Please check your input.")
	}

	response, err := server.NewRequest(server.EPFileUpload, &server.UploadStruct{
		Data: fileBytes,
		Name: fileName,
		Attributes: models.FileAttributes{
			Namespace: namespace,
			Groups:    groups,
			Tags:      tags,
		},
	}, config).Do(nil)

	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	if response.Status == server.ResponseError {
		fmt.Println("Error uploading your file..\n" + response.Message)
		return
	}

	fmt.Println("Successfully uploaded your file!")
}

// DeleteFile deletes the desired file(s)
func DeleteFile(name string, namespace string, groups []string, tags []string, id int) {
	response, err := server.NewRequest(server.EPFileUpload, &server.FileRequest{
		FileID: id,
		Attributes: models.FileAttributes{
			Namespace: namespace,
			Groups:    groups,
			Tags:      tags,
		},
	}, config).Do(nil)

	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	if response.Status == server.ResponseError {
		fmt.Println("Error trying to delete your file.\n" + response.Message)
		return
	}

	fmt.Println("The file has been successfully deleted.")
}

// ListFiles lists the files corresponding to the args
func ListFiles(name string, namespace string, groups []string, tags []string, id int) {
	var filesResponse server.FileListResponse
	response, err := server.NewRequest(server.EPFileList, &server.FileRequest{
		FileID: id,
		Name:   name,
		Attributes: models.FileAttributes{
			Groups:    groups,
			Namespace: namespace,
			Tags:      tags,
		},
	}, config).Do(&filesResponse)

	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	if response.Status == server.ResponseError {
		fmt.Println("Error listing files:", response.Message)
		return
	}

	// Output
	fmt.Printf("There were %d files found\n", len(filesResponse.Files))

	printFiles := true
	if len(filesResponse.Files) > 10 {
		fmt.Println("Do you want to print them? (y/n)")

		if !strings.HasPrefix(readInput(), "s") {
			printFiles = false
		}
	}

	if printFiles {
		// Print files
		for i := 0; i < len(filesResponse.Files); i++ {
			fmt.Printf("%d: %s\n", filesResponse.Files[i].ID, filesResponse.Files[i].Name)
		}
	}
}

// DownloadFile requests the file from the server
func DownloadFile(name *string, namespace *string, groups *[]string, tags *[]string, id *int, savePath *string) {

	/*if err != nil || response.Status == server.ResponseError {
		println("File was not downloaded:\n" + response.Message)
		return
	}

	// TODO Make it pretty and fix obvious issues here
	ioutil.WriteFile(*savePath+"/"+foundFile.FileName, foundFile.FileData, 6400)
	*/
}

func readInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.Replace(input, "\n", "", -1)

	return input
}
