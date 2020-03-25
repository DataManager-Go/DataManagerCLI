package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Yukaru-san/DataManager_Client/server"
	"github.com/fatih/color"
)

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
	f.Close()

	return filePath, nil
}

// previewFile opens a locally stored file
func previewFile(filepath string) {
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

//Parse file to bytes.Buffer for http multipart request
func fileToBodypart(filename string) (*bytes.Buffer, string, error) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	// this step is very important
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		return nil, "", err
	}

	// open file handle
	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
		return nil, "", err
	}
	defer fh.Close()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return nil, "", err
	}

	bodyWriter.Close()
	return bodyBuf, bodyWriter.FormDataContentType(), nil
}

func benchCheck(cData CommandData) {
	if cData.Bench {
		fmt.Println("This command doesn't support benchmarks")
		os.Exit(1)
	}
}

func getFileCommandData(n string, fid uint) (name string, id uint) {
	//Check if name is a fileID
	siID, err := strconv.ParseUint(n, 10, 32)
	if err == nil {
		id = uint(siID)
		return
	}

	name = n
	id = fid

	//otherwise return input
	return
}

func formatFilename(name string, nameLen int) string {
	if nameLen > 0 {
		end := nameLen
		if len(name) < nameLen {
			end = len(name)
		}
		return name[:end] + "..."
	}
	return name
}
