package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Yukaru-san/DataManager_Client/server"
)

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(path *string, namespace *string, groups *[]string, tags *[]string) {
	fileBytes, err := ioutil.ReadFile(*path)

	if err != nil {
		println("Error processing your file. Please check your input.")
	}

	response, err := server.NewRequest(server.UploadFile, &server.UploadStruct{
		Data:      fileBytes,
		Namespace: *namespace,
		Groups:    *groups,
		Tags:      *tags,
	}, config).Do(nil)

	if err != nil || response.Status == server.ResponseError {
		println("Error uploading your file..\n" + response.Message)
		return
	}

	println("Successfully uploaded your file!")
}

// DeleteFile deletes the desired file(s)
func DeleteFile(name *string, namespace *string, groups *[]string, tags *[]string, id *int) {

	response, err := sendRequest(name, namespace, groups, tags, id, nil, "Delete")

	if err != nil || response.Status == server.ResponseError {
		println("Error trying to delete your file.\n" + response.Message)
		return
	}

	println("The file has been successfully deleted.")
}

// ListFiles lists the files corresponding to the args
func ListFiles(name *string, namespace *string, groups *[]string, tags *[]string, id *int) {

	// The answer should look like this
	type returnInfo struct {
		filesFound []struct {
			ID       int
			FileName string
		}
	}
	var listedFiles returnInfo

	// Send a Request to the server
	response, err := sendRequest(name, namespace, groups, tags, id, &listedFiles, "List")

	if err != nil {
		println("Error trying to delete your file.\n" + response.Message)
		return
	}

	// Output
	fmt.Printf("There were %d files found\n", len(listedFiles.filesFound))

	printFiles := true
	if len(listedFiles.filesFound) > 10 {
		println("Do you want to print them? (y/n)")

		if !strings.HasPrefix(readInput(), "s") {
			printFiles = false
		}
	}

	if printFiles {
		// Print files
		for i := 0; i < len(listedFiles.filesFound); i++ {
			fmt.Printf("%d: %s", listedFiles.filesFound[i].ID, listedFiles.filesFound[i].FileName)
		}
	}
}

// DownloadFile requests the file from the server
func DownloadFile(name *string, namespace *string, groups *[]string, tags *[]string, id *int, savePath *string) {

	// The answer should look like this
	type returnInfo struct {
		FileData []byte
		FileName string
	}
	var foundFile returnInfo

	response, err := sendRequest(name, namespace, groups, tags, id, &foundFile, "Download")

	if err != nil || response.Status == server.ResponseError {
		println("File was not downloaded:\n" + response.Message)
		return
	}

	// TODO Make it pretty and fix obvious issues here
	ioutil.WriteFile(*savePath+"/"+foundFile.FileName, foundFile.FileData, 6400)
}

func sendRequest(name *string, namespace *string, groups *[]string, tags *[]string, id *int, retVar interface{}, task string) (*server.RestRequestResponse, error) {
	response, err := server.NewRequest(server.UploadFile, &server.HandleStruct{
		Name:      *name,
		Namespace: *namespace,
		Groups:    *groups,
		Tags:      *tags,
		ID:        *id,
		Task:      task,
	}, config).Do(&retVar)

	return response, err
}

func readInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.Replace(input, "\n", "", -1)

	return input
}
