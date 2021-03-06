package commands

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/fatih/color"
)

//Colorized strings
var (
	GreenSuccessfully = color.HiGreenString("Successfully")
	RedError          = color.HiRedString("Error")
)

// CreateNamespace creates a namespace
func CreateNamespace(cData *CommandData, name string, customNS bool) {
	_, err := cData.LibDM.CreateNamespace(name)
	if err != nil {
		printResponseError(err, "creating namespace")
		return
	}

	fmt.Printf("%s created namespace '%s'\n", GreenSuccessfully, name)
}

// UpdateNamespace update a namespace
func UpdateNamespace(cData *CommandData, name, newName string, customNS bool) {
	_, err := cData.LibDM.UpdateNamespace(name, newName)
	if err != nil {
		printResponseError(err, "updating namespace")
		return
	}

	fmt.Printf("%s updated namespace '%s'\n", GreenSuccessfully, name)
}

// DeleteNamespace update a namespace
func DeleteNamespace(cData *CommandData, name string) {
	if !cData.Yes {
		if y, _ := gaw.ConfirmInput("Do you really want to delete this namespace [yn]> ", bufio.NewReader(os.Stdin)); !y {
			return
		}
	}

	_, err := cData.LibDM.DeleteNamespace(name)
	if err != nil {
		printResponseError(err, "deleting namespace")
		return
	}

	fmt.Printf("%s deleted namespace '%s'\n", GreenSuccessfully, name)
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
		fmt.Printf("Namespaces(%d):\n\n", len(getNamespaceResponse.Slice))
		sort.Strings(getNamespaceResponse.Slice)

		for _, namespace := range getNamespaceResponse.Slice {
			// Omit username in NS list
			if strings.HasPrefix(namespace+"_", cData.Config.User.Username) {
				namespace = namespace[len(cData.Config.User.Username)+1:]
				// Make first item uppercase
				namespace = strings.ToUpper(string(namespace[0])) + namespace[1:]
			}

			fmt.Println("- " + namespace)
		}
	}
}

// DownloadNamespace download files from  namespace
func (cData *CommandData) DownloadNamespace(exGroups, exTags, exFiles []string, parallelism int, outDir string) {
	ProcesStrSliceParams(&exTags, &exGroups, &exFiles)

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
		// Exclude FileID
		if gaw.IsInStringArray(strconv.FormatUint(uint64(files.Files[i].ID), 10), exFiles) {
			continue
		}

		// Exclude Groups
		for j := range exGroups {
			if fileHasGroup(&files.Files[i], exGroups[j]) {
				continue a
			}
		}

		// Exclude Tags
		for j := range exTags {
			if fileHasTag(&files.Files[i], exTags[j]) {
				continue a
			}
		}

		toDownloadFiles = append(toDownloadFiles, files.Files[i])
	}

	cData.downloadFiles(toDownloadFiles, outDir, parallelism, func(file libdatamanager.FileResponseItem) string {
		name := "no_group"
		if len(file.Attributes.Groups) > 0 {
			name = file.Attributes.Groups[0]
		}

		return name
	})
}
