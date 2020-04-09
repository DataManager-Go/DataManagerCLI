package commands

import (
	"bufio"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/JojiiOfficial/shred"
	"github.com/fatih/color"
	"github.com/kyokomi/emoji"
	"golang.org/x/crypto/ssh/terminal"
)

func fmtError(message ...interface{}) {
	fmt.Printf("%s %s\n", color.HiRedString("Error:"), fmt.Sprint(message...))
}

func printError(message interface{}, err string) {
	fmt.Printf("%s %s: %s\n", color.HiRedString("Error"), message, err)
}

func printJSONError(message interface{}) {
	m := make(map[string]interface{}, 1)
	m["error"] = message
	json.NewEncoder(os.Stdout).Encode(m)
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
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s", gaw.RandString(10), fileName))
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()

		if err != nil {
			fmt.Println("Error:\n", err)
		}
	}
}

func getFileCommandData(n string, fid uint) (name string, id uint) {
	// Check if name is a fileID
	siID, err := strconv.ParseUint(n, 10, 32)
	if err == nil {
		id = uint(siID)
		return
	}

	// otherwise return input
	return n, fid
}

func formatFilename(file *libdm.FileResponseItem, nameLen int, cData *CommandData) string {
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

func filenameAddEmojis(filename string, file *libdm.FileResponseItem) string {
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

// ShredderFile shreddres a file
func ShredderFile(localFile string, size int64) {
	shredder := shred.Shredder{}

	var shredConfig *shred.ShredderConf
	if size < 0 {
		s, err := os.Stat(localFile)
		if err != nil {
			fmt.Println("File to shredder not found")
			return
		}
		size = s.Size()
	}

	if size >= 1000000000 {
		// Size >= 1GB
		shredConfig = shred.NewShredderConf(&shredder, shred.WriteZeros, 1, true)
	} else if size >= 1000000000 {
		// Size >= 1GB
		shredConfig = shred.NewShredderConf(&shredder, shred.WriteZeros|shred.WriteRandSecure, 2, true)
	} else if size >= 5000 {
		// Size > 5kb
		shredConfig = shred.NewShredderConf(&shredder, shred.WriteZeros|shred.WriteRandSecure, 3, true)
	} else {
		// Size < 5kb
		shredConfig = shred.NewShredderConf(&shredder, shred.WriteZeros|shred.WriteRandSecure, 6, true)
	}

	// Shredder & Delete local file
	err := shredConfig.ShredFile(localFile)
	if err != nil {
		fmt.Println(err)
		// Delete file if shredder didn't
		err = os.Remove(localFile)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func rmFilesFromkeystore(keystore *libdm.Keystore, files []uint) {
	for _, id := range files {
		// Delete file from keystore
		file, err := keystore.DeleteKey(id)
		if err == nil {
			// Shredder file
			path := keystore.GetKeystoreFile(file.Key)
			ShredderFile(path, -1)
		}
	}
}

// Print an response error for normies
func printResponseError(err error, msg string) {
	if err == nil {
		return
	}

	switch err.(type) {
	case *libdm.ResponseErr:
		lrerr := err.(*libdm.ResponseErr)

		var cause string

		if lrerr.Response != nil {
			cause = lrerr.Response.Message
		} else if lrerr.Err != nil {
			cause = lrerr.Err.Error()
		} else {
			cause = lrerr.Error()
		}

		printError(msg, cause)
	default:
		if err != nil {
			printError(msg, err.Error())
		} else {
			printError(msg, "no error provided")
		}
	}
}

func isValidAESLen(l int) bool {
	switch l {
	case 16, 24, 32:
		return true
	}
	return false
}

// Read from stdin with a timeout of 2s
func readFullStdin(bufferSize int) string {
	c := make(chan []byte, 1)

	// Read in background to allow using a select for a timeout
	go (func() {
		r := bufio.NewReader(os.Stdin)
		buf := make([]byte, bufferSize)

		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		c <- buf[:n]
	})()

	select {
	case b := <-c:
		return string(b)
	// Timeout
	case <-time.After(2 * time.Second):
		fmtError("No input received")
		os.Exit(1)
		return ""
	}
}

// Generates a secure random key
func randKey(l int) []byte {
	b := make([]byte, l)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("error:", err)
		return nil
	}

	return b
}

func saveFile(key []byte, file string) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()

	if err != nil {
		return err
	}

	_, err = f.Write(key)
	if err != nil {
		return err
	}
	return nil
}

// Gen filename for args
func genFileName(path, prefix string) string {
	var name string
	for {
		name = prefix + gaw.RandString(7)

		if len(path) > 0 {
			name = filepath.Join(path, name)
		}

		_, err := os.Stat(name)
		if err != nil {
			break
		}
	}
	return name
}

// Read password/key from stdin
func readPassword(message string) string {
	fmt.Print(message + "> ")

	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalln("Error:", err.Error())
		return ""
	}

	var pass string

	for _, a := range bytePassword {
		if int(a) != 0 && int(a) != 32 {
			pass += string(a)
		}
	}

	return strings.TrimSpace(pass)
}
