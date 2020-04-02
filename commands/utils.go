package commands

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"
	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
	"github.com/kyokomi/emoji"
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

// ProcesStrSliceParam divides args by ,
func ProcesStrSliceParam(slice *[]string) {
	var newSlice []string

	for _, itm := range *slice {
		newSlice = append(newSlice, strings.Split(itm, ",")...)
	}

	*slice = newSlice
}

// ProcesStrSliceParams divides args by ,
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

// GetTempFile returns tempfile from fileName
func GetTempFile(fileName string) string {
	return filepath.Join(os.TempDir(), fileName)
}

// SaveToTempFile saves a stream to a temporary file
func SaveToTempFile(reader io.Reader, fileName string) (string, error) {
	filePath := GetTempFile(fileName)
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

// Parse file to bytes.Buffer for http multipart request
func fileToBodypart(filename string, cData *CommandData) (*bytes.Buffer, string, error) {
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

	reader, err := getFileEncrypter(filename, fh, cData)
	if err != nil {
		return nil, "", err
	}

	// copy to filewriter
	_, err = io.Copy(fileWriter, *reader)
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
	// Check if name is a fileID
	siID, err := strconv.ParseUint(n, 10, 32)
	if err == nil {
		id = uint(siID)
		return
	}

	name = n
	id = fid

	// otherwise return input
	return
}

func formatFilename(file *models.FileResponseItem, nameLen int, cData *CommandData) string {
	name := file.Name

	if nameLen > 0 && len(name) > cData.NameLen {
		end := nameLen
		if len(name) < nameLen {
			end = len(name)
		}
		name = name[:end] + "..."
	}

	// Add emojis
	if !cData.NoEmojis {
		return filenameAddEmojis(name, file)
	}

	return name
}

func filenameAddEmojis(filename string, file *models.FileResponseItem) string {
	added := false

	// Public globe
	if len(file.PublicName) != 0 && file.IsPublic {
		filename = addEmoji(filename, "globe_with_meridians", !added)
		added = true
	}

	// Encryption lock
	if len(file.Encryption) != 0 {
		filename = addEmoji(filename, "lock", !added)
		added = true
	}

	return filename
}

func addEmoji(name, emojiStr string, addSpace bool) string {
	format := "%s:%s:"
	if addSpace {
		format = "%s :%s:"
	}

	return emoji.Sprintf(fmt.Sprintf(format, name, emojiStr))
}

func encodeBase64(b []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(b))
}

func decodeBase64(b []byte) []byte {
	data, err := base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		fmt.Println("Error: Bad Key!")
		os.Exit(1)
	}
	return data
}

// Return byte slice with base64 encoded file content
func fileToBase64(filename string, fh *os.File) ([]byte, error) {
	s, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	src := make([]byte, s.Size())
	_, err = fh.Read(src)
	if err != nil {
		return nil, err
	}

	return encodeBase64(src), nil
}

func hashFileMd5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil

}

func fileMd5(file string) string {
	md5, err := hashFileMd5(file)
	if err != nil {
		log.Fatal(err)
	}

	return md5
}

const boundary = "MachliJalKiRaniHaiJeevanUskaPaaniHai"

func uploadFile(path string, showBar bool) (r *io.PipeReader, contentType string, size int64) {
	// Open file
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve fileSize
	fi, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}
	size = fi.Size()

	// Create progressbar
	bar := pb.New64(fi.Size()).SetMaxWidth(100)
	if showBar {
		bar.Start()
	}

	r, w := io.Pipe()
	mpw := multipart.NewWriter(w)
	mpw.SetBoundary(boundary)

	contentType = mpw.FormDataContentType()

	go func() {
		part, err := mpw.CreateFormFile("file", fi.Name())
		if err != nil {
			log.Fatal(err)
		}

		if showBar {
			part = bar.NewProxyWriter(part)
		}

		buf := make([]byte, 512)

		for {
			n, err := f.Read(buf)
			if err != nil {
				break
			}
			part.Write(buf[:n])
		}

		bar.Finish()
		w.Close()
		f.Close()
		mpw.Close()
	}()

	return
}
