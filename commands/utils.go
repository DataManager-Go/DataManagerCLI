package commands

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Yukaru-san/DataManager_Client/server"
	"github.com/fatih/color"
)

//GetMD5Hash return hash of input
func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
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

//ProcesStrSliceParam divides args by ,
func ProcesStrSliceParam(slice *[]string) {
	var newSlice []string

	for _, itm := range *slice {
		newSlice = append(newSlice, strings.Split(itm, ",")...)
	}

	*slice = newSlice
}

//ProcesStrSliceParams divides args by ,
func ProcesStrSliceParams(slices ...*[]string) {
	for i := range slices {
		ProcesStrSliceParam(slices[i])
	}
}
