package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"./server"
)

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(path *string, namespace *string, group *string, tag *string) {
	fileBytes, err := ioutil.ReadFile(*path)

	if err != nil {
		println("Error processing your file. Please check your input.")
	}

	response, err := server.NewRequest(server.UploadFile, &server.UploadStruct{
		Data:      fileBytes,
		Namespace: *namespace,
		Group:     *group,
		Tag:       *tag,
	}, config).Do(nil)

	if err != nil || response.Status == server.ResponseError {
		println("Error uploading your file..\n" + response.Message)
		return
	}

	println("Successfully uploaded your file!")
}

// DeleteFile deletes the desired file(s)
func DeleteFile(name *string, namespace *string, group *string, tag *string) {

	response, err := server.NewRequest(server.UploadFile, &server.HandleStruct{
		Name:      *name,
		Namespace: *namespace,
		Group:     *group,
		Tag:       *tag,
		Task:      "Delete",
	}, config).Do(nil)

	if err != nil || response.Status == server.ResponseError {
		println("Error trying to delete your file.\n" + response.Message)
		return
	}

	println("The file has been successfully deleted.")
}

// ListFiles lists the files corresponding to the args
func ListFiles(name *string, namespace *string, group *string, tag *string) {

	// The answer should look like this
	type returnInfo struct {
		filesFound []struct {
			id       int
			fileName string
		}
	}
	var listedFiles returnInfo

	// Send a Request to the server
	response, err := server.NewRequest(server.UploadFile, &server.HandleStruct{
		Name:      *name,
		Namespace: *namespace,
		Group:     *group,
		Tag:       *tag,
		Task:      "List",
	}, config).Do(&listedFiles)

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
			fmt.Printf("%d: %s", listedFiles.filesFound[i].id, listedFiles.filesFound[i].fileName)
		}

		// Download?
		println(`If you want to download a file, write it's index, otherwise type "q"`)
		input := readInput()

		// Try to download the file
		if !strings.HasPrefix(input, "q") {
			id, err := strconv.ParseInt(input, 10, 64)

			if err != nil {
				println("Input error") // TODO Try again
				return
			}

			idMatched := false
			for i := 0; i < len(listedFiles.filesFound); i++ {
				if listedFiles.filesFound[i].id == int(id) {
					idMatched = true
					break
				}
			}

			if !idMatched {
				println("Input error") // TODO Try again
				return
			}

			println("Alright. Your download will be initiated..")
			DownloadFilebyID(int(id))
		}
	}
}

// DownloadFilebyID requests the file from the server
func DownloadFilebyID(id int) {
	// TODO
}

func readInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.Replace(input, "\n", "", -1)

	return input
}
