package commands

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Yukaru-san/DataManager_Client/constants"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"
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

	reader, err := getFileReader(filename, fh, cData)
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

// returns a reader to the correct source of data
func getFileReader(filename string, fh *os.File, cData *CommandData) (*io.Reader, error) {
	var reader io.Reader

	switch cData.Encryption {
	case constants.EncryptionCiphers[0]:
		{
			// AES
			block, err := aes.NewCipher([]byte(cData.EncryptionKey))
			if err != nil {
				return nil, err
			}

			// Get file content
			b, err := fileToBase64(filename, fh)
			if err != nil {
				return nil, err
			}

			// Set Ciphertext 0->16 to Iv
			ciphertext := make([]byte, aes.BlockSize+len(b))
			iv := ciphertext[:aes.BlockSize]
			if _, err := io.ReadFull(rand.Reader, iv); err != nil {
				return nil, err
			}

			// Encrypt file
			cfb := cipher.NewCFBEncrypter(block, iv)
			cfb.XORKeyStream(ciphertext[aes.BlockSize:], b)

			// Set reader to reader from bytes
			reader = bytes.NewReader(ciphertext)
		}
	case "":
		{
			// Set reader to reader of file
			reader = fh
		}
	default:
		{
			// Return error if cipher is not implemented
			return nil, errors.New("cipher not supported")
		}
	}

	return &reader, nil
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

func respToDecrypted(cData *CommandData, resp *http.Response) (io.Reader, error) {
	var reader io.Reader

	key := []byte(cData.EncryptionKey)
	if len(key) == 0 && len(resp.Header.Get(server.HeaderEncryption)) > 0 {
		fmt.Println("Error: file is encrypted but no key was given. To ignore this use --no-decrypt")
		os.Exit(1)
	}

	switch resp.Header.Get(server.HeaderEncryption) {
	case constants.EncryptionCiphers[0]:
		{
			// AES

			// Read response
			text, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			// Create Cipher
			block, err := aes.NewCipher(key)
			if err != nil {
				panic(err)
			}

			// Validate text length
			if len(text) < aes.BlockSize {
				fmt.Printf("Error!\n")
				os.Exit(0)
			}

			iv := text[:aes.BlockSize]
			text = text[aes.BlockSize:]

			// Decrypt
			cfb := cipher.NewCFBDecrypter(block, iv)
			cfb.XORKeyStream(text, text)

			reader = bytes.NewReader(decodeBase64(text))
		}
	/*case constants.EncryptionCiphers[1]:
	{
		// RSA
		fmt.Println("would decrypt rsa here")
		os.Exit(1)
	}*/
	case "":
		{
			reader = resp.Body
		}
	default:
		{
			return nil, errors.New("Cipher not supported")
		}
	}

	return reader, nil
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
