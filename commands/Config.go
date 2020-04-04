package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JojiiOfficial/configService"
	"github.com/JojiiOfficial/gaw"
	"github.com/fatih/color"
)

// UseTargets targets for config use
var UseTargets = []string{"namespace", "tags", "groups"}

// ConfigUse command for config use
func ConfigUse(cData CommandData, target string, values []string) {
	// Return if target not found
	if !gaw.IsInStringArray(target, UseTargets) {
		fmt.Println("Target not found")
		return
	}

	// Removing target
	if len(values) == 0 && target != "namespace" {
		//Remove desired target
		switch target {
		case UseTargets[1]:
			cData.Config.Default.Tags = []string{}
			fmt.Println("Removing tags")
		case UseTargets[2]:
			cData.Config.Default.Groups = []string{}
			fmt.Println("Removing groups")
		}
	} else {
		switch target {
		// Use namespace
		case UseTargets[0]:
			{
				if len(values) == 0 {
					values = []string{"default"}
				}
				fmt.Printf("Using namespace '%s'\n", values[0])
				cData.Config.Default.Namespace = values[0]
			}
		// Use tags
		case UseTargets[1]:
			{
				fmt.Printf("Using tags '%s'\n", strings.Join(values, ", "))
				cData.Config.Default.Tags = values
			}
		// Use Groups
		case UseTargets[2]:
			{
				fmt.Printf("Using groups '%s'\n", strings.Join(values, ", "))
				cData.Config.Default.Groups = values
			}
		default:
			fmt.Printf("Target not found")
			return
		}
	}

	//Save config
	err := configService.Save(cData.Config, cData.Config.File)
	if err != nil {
		fmt.Println("Error saving config:", err.Error())
	} else {
		fmt.Printf("Config saved %s\n", color.HiGreenString("successfully"))
	}
	return
}

//ConfigView view config
func ConfigView(cData CommandData) {
	if !cData.OutputJSON {
		//Print human output
		fmt.Println(cData.Config.View(!cData.NoRedaction))
	} else {
		//Redact secrets
		if !cData.NoRedaction {
			cData.Config.User.SessionToken = "<redacted>"
		}

		//Make json
		b, err := json.Marshal(cData.Config)
		if err != nil {
			fmt.Println(err)
			return
		}

		//Print output
		fmt.Println(string(b))
	}
}
