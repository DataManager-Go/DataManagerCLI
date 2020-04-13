package commands

import (
	"fmt"

	"github.com/fatih/color"
)

//Colorized strings
var (
	GreenSuccessfully = color.HiGreenString("Successfully")
	RedError          = color.HiRedString("Error")
)

// CreateNamespace creates a namespace
func CreateNamespace(cData *CommandData, name string, customNS bool) {
	createResponse, err := cData.LibDM.CreateNamespace(name, customNS)
	if err != nil {
		printResponseError(err, "creating namespace")
		return
	}

	fmt.Printf("%s created namespace '%s'\n", GreenSuccessfully, createResponse.String)
}

// UpdateNamespace update a namespace
func UpdateNamespace(cData *CommandData, name, newName string, customNS bool) {
	updateResponse, err := cData.LibDM.UpdateNamespace(name, newName, customNS)
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
		for _, namespace := range getNamespaceResponse.Slice {
			fmt.Println(namespace)
		}
	}
}
