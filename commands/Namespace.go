package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/DataManager-Go/libdatamanager"
	"github.com/fatih/color"
	"github.com/gosuri/uiprogress"
)

//Colorized strings
var (
	GreenSuccessfully = color.HiGreenString("Successfully")
	RedError          = color.HiRedString("Error")
)

// CreateNamespace creates a namespace
func CreateNamespace(cData *CommandData, name string, customNS bool) {
	createResponse, err := cData.LibDM.CreateNamespace(name)
	if err != nil {
		printResponseError(err, "creating namespace")
		return
	}

	fmt.Printf("%s created namespace '%s'\n", GreenSuccessfully, createResponse.String)
}

// UpdateNamespace update a namespace
func UpdateNamespace(cData *CommandData, name, newName string, customNS bool) {
	updateResponse, err := cData.LibDM.UpdateNamespace(name, newName)
	if err != nil {
		printResponseError(err, "updating namespace")
		return
	}

	fmt.Printf("%s updated namespace '%s'\n", GreenSuccessfully, updateResponse.String)
}

// DeleteNamespace update a namespace
func DeleteNamespace(cData *CommandData, name string) {
	deleteResponse, err := cData.LibDM.DeleteNamespace(name)
	if err != nil {
		printResponseError(err, "deleting namespace")
		return
	}

	fmt.Printf("%s deleted namespace '%s'\n", GreenSuccessfully, deleteResponse.String)
}

// ListNamespace lists your namespace
func ListNamespace(cData *CommandData) {
	getNamespaceResponse, err := cData.LibDM.GetNamespaces()
	if err != nil {
		printResponseError(err, "listing namespaces")
		return
	}

	if cData.OutputJSON {
		fmt.Println(toJSON(getNamespaceResponse))
	} else {
		fmt.Println("Your namespaces:")
		sort.Strings(getNamespaceResponse.Slice)
		for _, namespace := range getNamespaceResponse.Slice {
			fmt.Println(namespace)
		}
	}
}

// DownloadNamespace download files from  namespace
func (cData *CommandData) DownloadNamespace(exGroups, exTags []string, parallelism uint, outDir string) {
	// Prevent user stupidity
	if parallelism == 0 {
		parallelism = 1
	}

	// Get files in namespace from server
	files, err := cData.LibDM.ListFiles("", 0, false, libdatamanager.FileAttributes{
		Namespace: cData.FileAttributes.Namespace,
	}, 2)

	if err != nil {
		printResponseError(err, "retrieving files")
		return
	}

	// Files with are not excluded
	var toDownloadFiles []libdatamanager.FileResponseItem

	// Filter files by tags and groups
a:
	for i := range files.Files {
		for j := range exGroups {
			if fileHasGroup(&files.Files[i], exGroups[j]) {
				continue a
			}
		}

		for j := range exTags {
			if fileHasTag(&files.Files[i], exTags[j]) {
				continue a
			}
		}

		toDownloadFiles = append(toDownloadFiles, files.Files[i])
	}

	if len(toDownloadFiles) == 0 {
		fmt.Println("No files found")
		return
	}

	// Reduce threads if files are less than threads
	if uint(len(toDownloadFiles)) < parallelism {
		parallelism = uint(len(toDownloadFiles))
	}

	// Waitgroup to wait for all "threads" to be done
	wg := sync.WaitGroup{}
	// Channel for managing amount of parallel upload processes
	c := make(chan uint, 1)

	c <- parallelism
	var pos int

	totalfiles := len(toDownloadFiles)

	// Create and start progress
	progress := uiprogress.New()
	progress.Start()

	// Use first files namespace as destination dir
	if len(outDir) == 0 {
		outDir = toDownloadFiles[0].Attributes.Namespace
	}

	rootDir := filepath.Clean(filepath.Join("./", outDir))

	// Overwrite files
	cData.Force = true

	// Start Uploader pool
	for pos < totalfiles {
		read := <-c
		for i := 0; i < int(read) && pos < totalfiles; i++ {
			wg.Add(1)

			go func(file libdatamanager.FileResponseItem) {
				// Build dest group dir name
				dir := "no_group"
				if len(file.Attributes.Groups) > 0 {
					dir = file.Attributes.Groups[0]
				}

				// Create dir if not exists
				path := filepath.Clean(filepath.Join(rootDir, dir))
				if _, err := os.Stat(path); err != nil {
					err := os.MkdirAll(path, 0750)
					if err != nil {
						printError("Creating dir", err.Error())
						os.Exit(1)
					}
				}

				// Download file
				err := cData.DownloadFile(&DownloadData{
					FileName:  file.Name,
					FileID:    file.ID,
					LocalPath: filepath.Join(rootDir, dir),
				}, progress)

				if err != nil {
					os.Exit(1)
				}

				wg.Done()
				c <- 1
			}(toDownloadFiles[pos])

			pos++
		}
	}

	// Wait for all threads
	// to be done
	wg.Wait()
}
