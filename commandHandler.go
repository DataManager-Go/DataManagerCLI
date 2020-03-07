package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"./server"
)

type uploadStruct struct {
	data      []byte
	namespace string
	group     string
	tag       string
}

type handleStruct struct {
	name      string
	namespace string
	group     string
	tag       string
}

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(path *string, namespace *string, group *string, tag *string) {
	fileBytes, err := ioutil.ReadFile(*path)

	if err != nil {
		println("Error processing your file. Please check your input.")
	}

	response, err := server.NewRequest(server.UploadFile, &uploadStruct{
		data:      fileBytes,
		namespace: *namespace,
		group:     *group,
		tag:       *tag,
	}, config).Do(nil)

	if err != nil || response.Status == server.ResponseError {
		println("Error uploading your file..\n" + response.Message)
		return
	}

	println("Successfully uploaded your file!")
}

// DeleteFile deletes the desired file(s)
func DeleteFile(name *string, namespace *string, group *string, tag *string) {

	type returnInfo struct {
		fileDeleted bool
	}
	var success returnInfo

	response, err := server.NewRequest(server.UploadFile, &handleStruct{
		name:      *name,
		namespace: *namespace,
		group:     *group,
		tag:       *tag,
	}, config).Do(&success)

	if err != nil || !success.fileDeleted {
		println("Error trying to delete your file.\n" + response.Message)
		return
	}

	println("The file has been successfully deleted.")

}

// ListFiles lists the files corresponding to the args
func ListFiles(name *string, namespace *string, group *string, tag *string) {

	type returnInfo struct {
		filesFound []string
	}
	var listedFiles returnInfo

	response, err := server.NewRequest(server.UploadFile, &handleStruct{
		name:      *name,
		namespace: *namespace,
		group:     *group,
		tag:       *tag,
	}, config).Do(&listedFiles)

	if err != nil {
		println("Error trying to delete your file.\n" + response.Message)
		return
	}

	fmt.Printf("There were %d files found\n", len(listedFiles.filesFound))

	printFiles := true
	if len(listedFiles.filesFound) > 10 {
		println("Do you want to print them? (y/n)")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.Replace(input, "\n", "", -1)

		if !strings.HasPrefix(input, "s") {
			printFiles = false
		}
	}

	if printFiles {
		for i := 0; i < len(listedFiles.filesFound); i++ {
			println(listedFiles.filesFound[i])
		}
	}
}
