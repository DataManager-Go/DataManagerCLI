package commands

import (
	"fmt"
	"log"

	"github.com/Yukaru-san/DataManager_Client/constants"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"
)

func handleNamespaceCommand(cData CommandData, action uint8, name, newName string, customNs bool) {
	//Get correct endpoint
	endpoint := namespaceActionToEndpoint(action)

	nsType := models.UserNamespaceType
	if customNs {
		nsType = models.CustomNamespaceType
	}

	var res server.StringResponse

	response, err := server.NewRequest(endpoint, server.NamespaceRequest{
		Namespace: name,
		NewName:   newName,
		Type:      nsType,
	}, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).Do(&res)

	// Error handling #1
	if err != nil {
		if response != nil {
			fmt.Println("http:", response.HTTPCode)
			return
		}
		log.Fatalln(err)
	}

	// Error handling #2
	if response.Status == server.ResponseError {
		printResponseError(response, namespaceActionToString(action, false, true))
	} else {
		switch action {
		case 2, 0:
			fmt.Printf("%s %s namespace '%s'\n", constants.GreenSuccessfully, namespaceActionToString(action, true, false), name)
		case 1:
			fmt.Printf("%s created namespace '%s'\n", constants.GreenSuccessfully, res.String)
		}
	}
}

//CreateNamespace creates a namespace
func CreateNamespace(cData CommandData, name string, customNS bool) {
	handleNamespaceCommand(cData, 1, name, "", customNS)
}

//UpdateNamespace update a namespace
func UpdateNamespace(cData CommandData, name, newName string, customNS bool) {
	handleNamespaceCommand(cData, 2, name, newName, false)
}

//DeleteNamespace update a namespace
func DeleteNamespace(cData CommandData, name string) {
	handleNamespaceCommand(cData, 0, name, "", false)
}

//ListNamespace lists your namespace
func ListNamespace(cData CommandData, name string) {

}

func namespaceActionToString(action uint8, past, pp bool) (name string) {
	switch action {
	case 0:
		name = "delete"
	case 1:
		name = "create"
	case 2:
		name = "update"
	}

	if pp {
		name += "ing"
		return
	}

	if past {
		name += "d"
		return
	}

	return
}

func namespaceActionToEndpoint(action uint8) (endpoint server.Endpoint) {
	switch action {
	case 0:
		endpoint = server.EPNamespaceDelete
	case 1:
		endpoint = server.EPNamespaceCreate
	case 2:
		endpoint = server.EPNamespaceUpdate
	}

	return
}
