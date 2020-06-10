package commands

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/DataManager-Go/libdatamanager"
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

func getError(message interface{}, err string) string {
	return fmt.Sprintf("%s %s: %s\n", color.HiRedString("Error"), message, err)
}

func printError(message interface{}, err string) {
	fmt.Println(getError(message, err))
}

func printWarning(message interface{}, err string) {
	fmt.Printf("%s %s: %s\n", color.YellowString("Warn"), message, err)
}

func printJSONError(message interface{}) {
	m := make(map[string]interface{}, 1)
	m["error"] = message
	json.NewEncoder(os.Stdout).Encode(m)
}

func sPrintSuccess(format string, message ...interface{}) string {
	return fmt.Sprintf("%s %s", color.HiGreenString("Successfully"), fmt.Sprintf(format, message...))
}

func printSuccess(format string, message ...interface{}) {
	fmt.Println(sPrintSuccess(format, message...) + "\n")
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
func (cData *CommandData) previewFile(filepath string) {
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

		start := time.Now().Unix()

		if err := cmd.Run(); err != nil {
			fmt.Println("Error: ", err)
			return
		}

		if time.Now().Unix()-start <= 2 {
			// Wait
			if !cData.Quiet {
				fmt.Println("Application exited too fast. Trying to wait for its PID to exit")
			}

			// Get all PIDs in current terminal
			pids, err := getTerminalPIDs()
			if err != nil {
				fmt.Println(err)
				return
			}

			// Only wait for programs launched AFTER xdg-open
			for _, pid := range pids {
				if pid >= cmd.Process.Pid {
					waitForPID(pid)
				}
			}
		}
	}
}

// Get all PIDs in current terminal
func getTerminalPIDs() ([]int, error) {
	var nums []int

	// Get current terminals processes
	out, err := exec.Command("sh", "-c", "ps --no-headers | grep -vE 'sh$' | grep -vE 'ps|grep' | cut -d ' ' -f2").Output()
	if err != nil {
		return nil, err
	}

	// Go for each line
	for _, snum := range strings.Split(string(out), "\n") {
		num, err := strconv.Atoi(snum)
		if err == nil {
			nums = append(nums, num)
		}
	}

	return nums, nil
}

func waitForPID(pid int) {
	if err := exec.Command("tail", fmt.Sprintf("--pid=%d", pid), "-f", "/dev/null").Run(); err != nil {
		fmt.Println(err)
	}
}

// GetFileCommandData returns id if name is id
func GetFileCommandData(n string, fid uint) (name string, id uint) {
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

// Read password/key from stdin
func readPassword(message string) []byte {
	fmt.Print(message + "> ")

	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalln("Error:", err.Error())
		return nil
	}

	var pass string

	for _, a := range bytePassword {
		if int(a) != 0 && int(a) != 32 {
			pass += string(a)
		}
	}

	return []byte(strings.TrimSpace(pass))
}

func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func awaitOrInterrupt(Ch chan string, onInterrupt func(os.Signal), onChan func(string)) {
	// make channel to listen for kill signals
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt, os.Kill, syscall.SIGTERM)

	select {
	case killsig := <-kill:
		onInterrupt(killsig)
	case data := <-Ch:
		onChan(data)
	}
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

// create a temporary file ending with "name"
func createTempFile(name *string) string {
	if name == nil {
		return ""
	}

	if len(*name) == 0 {
		*name = gaw.RandString(10)
	}

	tmpFile := GetTempFile(*name)

	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		printError("opening tempfile", err.Error())
		return ""
	}
	f.Close()

	return tmpFile
}

func nameFromURL(u *url.URL) string {
	path := strings.ReplaceAll(u.EscapedPath(), string(filepath.Separator), "")
	name := u.Host
	if len(path) > 0 {
		name = filepath.Join(name, path)
		name = strings.ReplaceAll(name, string(filepath.Separator), "-")
	}
	return name
}

func fileHasGroup(file *libdatamanager.FileResponseItem, group string) bool {
	for i := range file.Attributes.Groups {
		if file.Attributes.Groups[i] == group {
			return true
		}
	}

	return false
}

func fileHasTag(file *libdatamanager.FileResponseItem, tag string) bool {
	for i := range file.Attributes.Tags {
		if file.Attributes.Tags[i] == tag {
			return true
		}
	}

	return false
}

// Return amount of figures of nr
func getFigureAmount(nr uint) int {
	return int(math.Log10(float64(nr))) + 1
}
