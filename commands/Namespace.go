package commands

import (
	"fmt"

	"github.com/JojiiOfficial/DataManagerServer/constants"
)

// CreateNamespace creates a namespace
func CreateNamespace(cData CommandData, name string, customNS bool) {
	createResponse, err := cData.LibDM.CreateNamespace(name, customNS)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s created namespace '%s'\n", constants.GreenSuccessfully, createResponse.String)
}

// UpdateNamespace update a namespace
func UpdateNamespace(cData CommandData, name, newName string, customNS bool) {
	updateResponse, err := cData.LibDM.UpdateNamespace(name, newName, customNS)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s updated namespace '%s'\n", constants.GreenSuccessfully, updateResponse.String)
}

// DeleteNamespace update a namespace
func DeleteNamespace(cData CommandData, name string) {
	deleteResponse, err := cData.LibDM.DeleteNamespace(name)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s deleted namespace '%s'\n", constants.GreenSuccessfully, deleteResponse.String)
}

// ListNamespace lists your namespace
func ListNamespace(cData CommandData) {
	getNamespaceResponse, err := cData.LibDM.GetNamespaces()
	if err != nil {
		fmt.Println(err)
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
