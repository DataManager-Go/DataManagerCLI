package commands

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"
	"github.com/fatih/color"
)

// AppYes - whether the program was initialized with --yes
var AppYes bool

// Config - program's config
var config *models.Config

//GetMD5Hash return hash of input
func GetMD5Hash(text []byte) string {
	hash := md5.Sum(text)
	return hex.EncodeToString(hash[:])
}

func printResponseError(response *server.RestRequestResponse, add ...string) {
	sadd := ""
	if len(add) > 0 {
		sadd = add[0]
	}
	printError(sadd + ": " + response.Message)
}

func printError(message interface{}) {
	fmt.Printf("%s %s\n", color.HiRedString("Error"), message)
}
