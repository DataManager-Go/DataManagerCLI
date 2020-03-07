package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"./models"

	log "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	appName = "managerclient"
	version = "1.0.0"

	//EnVarPrefix prefix for env vars
	EnVarPrefix = "MANAGER"

	//Datapath the default path for data files
	Datapath = "./data"
	//DefaultConfig the default config file
	DefaultConfig = "config.yaml"
)

var (
	//DefaultConfigPath default config path
	DefaultConfigPath = path.Join(Datapath, DefaultConfig)
)

//App commands
var (
	app = kingpin.New(appName, "A DataManager")

	// Global flags
	appNoColor = app.Flag("no-color", "Disable colors").Envar(getEnVar(EnVarNoColor)).Bool()
	appCfgFile = app.
			Flag("config", "the configuration file for the app").
			Envar(getEnVar(EnVarConfigFile)).
			Short('c').String()

	// File commands
	fileCMD = app.Command("file", "Commands for handling files")
	// -- File child commands
	fileUpload = fileCMD.Command("upload", "Upload the given file")
	fileDelete = fileCMD.Command("delete", "Delete a file stored on the server")
	fileList   = fileCMD.Command("list", "List files stored on the server")
	// -- -- Upload specifier
	fileUploadPath      = fileUpload.Arg("filePath", "Path to the file you want to upload").Required().String()
	fileUploadNamespace = fileUpload.Flag("namespace", "Set the namespace the file should belong to").Short('n').String()
	fileUploadGroup     = fileUpload.Flag("group", "Set the group the file should belong to").Short('g').String()
	fileUploadTag       = fileUpload.Flag("tag", "Set the tag the file should belong to").Short('t').String()
	// -- -- Delete specifier
	fileDeleteName      = fileUpload.Arg("fileName", "Name of the file that should be removed").String()
	fileDeleteNamespace = fileUpload.Flag("namespace", "The namespace the file belongs to").Short('n').String()
	fileDeleteGroup     = fileUpload.Flag("group", "The group the file belongs to").Short('g').String()
	fileDeleteTag       = fileUpload.Flag("tag", "The tag the file belongs to").Short('t').String()
	// -- -- List specifier
	fileListName      = fileUpload.Arg("fileName", "Show files with this name").String()
	fileListNamespace = fileUpload.Flag("namespace", "Show files within this namespace").Short('n').String()
	fileListGroup     = fileUpload.Flag("group", "Show files within this group").Short('g').String()
	fileListTag       = fileUpload.Flag("tag", "Show files with this tag").Short('t').String()
)

var (
	config  *models.Config
	isDebug = false
)

func main() {
	app.HelpFlag.Short('h')
	app.Version(version)

	//parsing the args
	parsed := kingpin.MustParse(app.Parse(os.Args[1:]))

	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  time.Stamp,
		FullTimestamp:    true,
		ForceColors:      !*appNoColor,
		DisableColors:    *appNoColor,
	})

	//Init config
	var err error
	config, err = models.InitConfig(DefaultConfigPath, *appCfgFile)
	if err != nil {
		log.Error(err)
		return
	}

	if config == nil {
		log.Info("New config cerated")
		return
	}

	// Execute the desired command
	switch parsed {
	case fileUpload.FullCommand():
		UploadFile(fileUploadPath, fileUploadNamespace, fileUploadGroup, fileUploadTag)

	case fileDelete.FullCommand():
		DeleteFile(fileUploadPath, fileUploadNamespace, fileUploadGroup, fileUploadTag)

	case fileList.FullCommand():
		ListFiles(fileUploadPath, fileUploadNamespace, fileUploadGroup, fileUploadTag)

	}
}

//Env vars
const (
	//EnVarPrefix prefix of all used env vars
	EnVarLogLevel   = "LOG_LEVEL"
	EnVarNoColor    = "NO_COLOR"
	EnVarConfigFile = "CONFIG"
)

//Return the variable using the server prefix
func getEnVar(name string) string {
	return fmt.Sprintf("%s_%s", EnVarPrefix, name)
}
