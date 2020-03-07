package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"

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
	fileCMD       = app.Command("file", "Commands for handling files")
	fileNamespace = app.Flag("namespace", "Set the namespace the file should belong to").Default("default").Short('n').String()
	fileTags      = app.Flag("tag", "Download files with this tag").Short('t').Strings()
	fileGroups    = app.Flag("group", "Set the group the file should belong to").Short('g').Strings()
	fileID        = fileDownload.Flag("file-id", "Specify the fileID").Int()

	//child Commands
	// -- File child commands
	fileUpload   = fileCMD.Command("upload", "Upload the given file")
	fileDelete   = fileCMD.Command("delete", "Delete a file stored on the server")
	fileList     = fileCMD.Command("list", "List files stored on the server")
	fileDownload = fileCMD.Command("download", "Download a file from the server")

	//Args/Flags
	// -- -- Upload specifier
	fileUploadPath = fileUpload.Arg("filePath", "Path to the file you want to upload").Required().String()
	// -- -- Delete specifier
	fileDeleteName = fileDelete.Arg("fileName", "Name of the file that should be removed").String()
	// -- -- List specifier
	fileListName = fileList.Arg("fileName", "Show files with this name").String()
	// -- -- Download specifier
	fileDownloadName = fileDownload.Arg("fileName", "Download files with this name").String()
	fileDownloadPath = fileDownload.Flag("path", "Where to store the file").Short('p').Required().String()

	appPing = app.Command("ping", "pings the server and checks connectivity")
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
		DownloadFile(fileDownloadName, fileNamespace, fileGroups, fileTags, fileID, fileDownloadPath)

	case fileUpload.FullCommand():
		UploadFile(*fileUploadPath, *fileNamespace, *fileGroups, *fileTags)

	case fileDelete.FullCommand():
		DeleteFile(*fileUploadPath, *fileNamespace, *fileGroups, *fileTags, *fileID)

	case fileList.FullCommand():
		ListFiles(*fileListName, *fileNamespace, *fileGroups, *fileTags, *fileID)

	case appPing.FullCommand():
		pingServer(config)
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

func pingServer(config *models.Config) {
	var response server.StringResponse
	res, err := server.
		NewRequest(server.EPPing, server.PingRequest{Payload: "ping"}, config).
		Do(&response)

	if err != nil {
		log.Error(err.Error())
		return
	}

	if res.Status == server.ResponseSuccess {
		fmt.Println("Ping success:", response.String)
	} else {
		log.Errorf("Error (%d) %s\n", res.HTTPCode, res.Message)
	}
}
