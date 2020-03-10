package commands

import (
	"fmt"
	"log"

	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"
	"github.com/fatih/color"
)

// do an attribute request (update/delete group or tag). action: 0 - delete, 1 - update
func attributeRequest(cData CommandData, attribute models.Attribute, action uint8, name string, newName ...string) {
	var endpoint server.Endpoint

	//Pick right endpoint
	if action == 1 {
		if attribute == models.GroupAttribute {
			endpoint = server.EPGroupUpdate
		} else {
			endpoint = server.EPTagUpdate
		}
	} else if action == 0 {
		if attribute == models.GroupAttribute {
			endpoint = server.EPGroupDelete
		} else {
			endpoint = server.EPTagDelete
		}
	}

	// Build request
	request := server.UpdateAttributeRequest{
		Name:      name,
		Namespace: cData.Namespace,
	}

	// Add new name on update request
	if action == 1 {
		request.NewName = newName[0]
	}

	response, err := server.NewRequest(endpoint, request, cData.Config).WithAuth(server.Authorization{
		Type:    server.Bearer,
		Palyoad: cData.Config.User.SessionToken,
	}).Do(nil)

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
		printResponseError(response)
		return
	}

	// Output
	actionString := "updated"
	if action == 0 {
		actionString = "deleted"
	}

	fmt.Printf("The attribute has been %s\n", color.HiGreenString("successfully "+actionString))
}

// UpdateAttribute update an attribute
func UpdateAttribute(cData CommandData, attribute models.Attribute, name, newName string) {
	attributeRequest(cData, attribute, 1, name, newName)
}

// DeleteAttribute update an attribute
func DeleteAttribute(cData CommandData, attribute models.Attribute, name string) {
	attributeRequest(cData, attribute, 0, name)
}
