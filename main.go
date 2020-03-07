package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/Yukaru-san/DataManager_Client/models"

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
	fileUpload   = fileCMD.Command("upload", "Upload the given file")
	fileDelete   = fileCMD.Command("delete", "Delete a file stored on the server")
	fileList     = fileCMD.Command("list", "List files stored on the server")
	fileDownload = fileCMD.Command("download", "Download a file from the server")
	// -- -- Upload specifier
	fileUploadPath      = fileUpload.Arg("filePath", "Path to the file you want to upload").Required().String()
	fileUploadNamespace = fileUpload.Flag("namespace", "Set the namespace the file should belong to").Short('n').String()
	fileUploadGroups    = fileUpload.Flag("group", "Set the group the file should belong to").Short('g').Strings()
	fileUploadTags      = fileUpload.Flag("tag", "Set the tag the file should belong to").Short('t').Strings()
	// -- -- Delete specifier
	fileDeleteName      = fileUpload.Arg("fileName", "Name of the file that should be removed").String()
	fileDeleteNamespace = fileUpload.Flag("namespace", "The namespace the file belongs to").Short('n').String()
	fileDeleteGroups    = fileUpload.Flag("group", "The group the file belongs to").Short('g').Strings()
	fileDeleteID        = fileDownload.Flag("id", "Delete by ID").Int()
	fileDeleteTags      = fileUpload.Flag("tag", "The tag the file belongs to").Short('t').Strings()
	// -- -- List specifier
	fileListName      = fileUpload.Arg("fileName", "Show files with this name").String()
	fileListNamespace = fileUpload.Flag("namespace", "Show files within this namespace").Short('n').String()
	fileListGroups    = fileUpload.Flag("group", "Show files within this group").Short('g').String()
	fileListID        = fileDownload.Flag("id", "Find by ID").Int()
	fileListTags      = fileUpload.Flag("tag", "Show files with this tag").Short('t').String()
	// -- -- Download specifier
	fileDownloadName      = fileDownload.Arg("fileName", "Download files with this name").String()
	fileDownloadNamespace = fileDownload.Flag("namespace", "Download files in this namespace").Short('n').String()
	fileDownloadGroups    = fileDownload.Flag("group", "Download files in this group").Short('g').Strings()
	fileDownloadTags      = fileDownload.Flag("tag", "Download files with this tag").Short('t').Strings()
	fileDownloadID        = fileDownload.Flag("id", "Download by ID").Int()
	fileDownloadPath      = fileDownload.Flag("path", "Where to store the file").Short('p').Required().String()
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
	case fileDownload.FullCommand():
		DownloadFile(fileDownloadName, fileDownloadNamespace, fileDownloadGroups, fileDownloadTags, fileDownloadID, fileDownloadPath)

	case fileUpload.FullCommand():
		UploadFile(fileUploadPath, fileUploadNamespace, fileUploadGroups, fileUploadTags)

	case fileDelete.FullCommand():
		DeleteFile(fileUploadPath, fileUploadNamespace, fileUploadGroups, fileUploadTags, fileDeleteID)

	case fileList.FullCommand():
		ListFiles(fileUploadPath, fileUploadNamespace, fileUploadGroups, fileUploadTags, fileListID)

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
