package commands

import (
	"fmt"
	"strings"

	gaw "github.com/JojiiOfficial/GoAw"
	"github.com/JojiiOfficial/configService"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/fatih/color"
)

//UseTargets targets for config use
var UseTargets = []string{"namespace", "tags", "groups"}

//ConfigUse command for config use
func ConfigUse(config *models.Config, target string, values []string) {
	//Return if target not found
	if !gaw.IsInStringArray(target, UseTargets) {
		fmt.Println("Target not found")
		return
	}

	//Removing target
	if len(values) == 0 && target != "namespace" {
		//Remove desired target
		switch target {
		case UseTargets[1]:
			config.Default.Tags = []string{}
		case UseTargets[2]:
			config.Default.Groups = []string{}
		}
	} else {
		switch target {
		//Use namespace
		case UseTargets[0]:
			{
				if len(values) == 0 {
					values = []string{"default"}
				}
				fmt.Printf("Using namespace '%s'\n", values[0])
				config.Default.Namespace = values[0]
			}
		//Use tags
		case UseTargets[1]:
			{
				fmt.Printf("Using tags '%s'\n", strings.Join(values, ", "))
				config.Default.Tags = values
			}
		//Use Groups
		case UseTargets[2]:
			{
				fmt.Printf("Using groups '%s'\n", strings.Join(values, ", "))
				config.Default.Groups = values
			}
		default:
			fmt.Printf("Target not found")
			return
		}
	}

	//Save config
	err := configService.Save(config, config.File)
	if err != nil {
		fmt.Println("Error saving config:", err.Error())
	} else {
		fmt.Printf("Config saved %s\n", color.HiGreenString("successfully"))
	}
	return
}
