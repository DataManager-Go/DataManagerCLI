package commands

import (
	"fmt"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/fatih/color"
)

// UpdateAttribute update an attribute
func UpdateAttribute(cData *CommandData, attribute libdm.Attribute, name, newName string) {
	_, err := cData.LibDM.UpdateAttribute(attribute, cData.FileAttributes.Namespace, name, newName)
	if err != nil {
		printResponseError(err, "updating attribute")
		return
	}

	fmt.Printf("The attribute has been %s\n", color.HiGreenString("successfully updated"))
}

// DeleteAttribute update an attribute
func DeleteAttribute(cData *CommandData, attribute libdm.Attribute, name string) {
	_, err := cData.LibDM.DeleteAttribute(attribute, cData.FileAttributes.Namespace, name)
	if err != nil {
		printResponseError(err, "deleting attribute")
		return
	}

	fmt.Printf("The attribute has been %s\n", color.HiGreenString("successfully deleted"))
}
