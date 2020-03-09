package commands

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func toJSON(in interface{}) string {
	b, err := json.Marshal(in)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

//SaveToTempFile saves a stream to a temporary file
func SaveToTempFile(reader io.ReadCloser, fileName string) (string, error) {
	filePath := filepath.Join(os.TempDir(), fileName)
	//Create temp file
	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}

	//Write from reader
	_, err = io.Copy(f, reader)
	if err != nil {
		return "", err
	}

	//Close streams
	reader.Close()
	f.Close()

	return filePath, nil
}

// PreviewFile opens a locally stored file
func PreviewFile(filepath string) {
	// Windows
	if runtime.GOOS == "windows" {
		fmt.Println("Filepath: " + filepath)
		cmd := exec.Command("cmd", "/C "+filepath)
		output, _ := cmd.Output()

		if len(output) > 0 {
			fmt.Println("Error: Your system hasn't set up a default application for this datatype.")
		}

		// Linux
	} else if runtime.GOOS == "linux" {
		cmd := exec.Command("xdg-open", filepath)

		var errCatcher bytes.Buffer
		cmd.Stderr = &errCatcher

		cmd.Run()

		if errCatcher.Len() > 0 {
			fmt.Println("Error:\n", string(errCatcher.Bytes()))
		}
	}
}
