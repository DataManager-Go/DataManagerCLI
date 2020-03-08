package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/Yukaru-san/DataManager_Client/commands"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"

	log "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	appName = "manager"
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
	app     = kingpin.New(appName, "A DataManager")
	appPing = app.Command("ping", "pings the server and checks connectivity")

	// Global flags
	appYes     = app.Flag("yes", "Skip confirmations").Bool()
	appNoColor = app.Flag("no-color", "Disable colors").Envar(getEnVar(EnVarNoColor)).Bool()
	appCfgFile = app.
			Flag("config", "the configuration file for the app").
			Envar(getEnVar(EnVarConfigFile)).
			Short('c').String()
	appNamespace = app.Flag("namespace", "Specify the namespace to use").Short('n').String()
	appTags      = app.Flag("tag", "Specify tags to use").Short('t').Strings()
	appGroups    = app.Flag("group", "Specify groups to use").Short('g').Strings()

	// UserCommands
	//Login
	loginCmd     = app.Command("login", "Login")
	loginCmdUser = loginCmd.Flag("username", "Your username").String()
	//Register
	registerCmd = app.Command("register", "Create an account").FullCommand()
	// Manager commands
	// File commands
	fileCMD = app.Command("file", "Commands for handling files")
	fileID  = fileDownload.Flag("file-id", "Specify the fileID").Uint()
	//Config commands
	configCMD = app.Command("config", "Commands for working with the config")

	// Child Commands
	// -- File child commands
	fileUpload   = fileCMD.Command("upload", "Upload the given file")
	fileDelete   = fileCMD.Command("delete", "Delete a file stored on the server")
	fileList     = fileCMD.Command("list", "List files stored on the server")
	fileDownload = fileCMD.Command("download", "Download a file from the server")

	// -- Manager child commands
	managerUpdate = app.Command("update", "Update the filesystem")

	// -- -- ManagerUpdate childs
	fileUpdate = managerUpdate.Command("file", "Update a file")
	tagUpdate  = managerUpdate.Command("tag", "Update a tag")

	// -- Config child command
	configUse = configCMD.Command("use", "Use something")

	// Args/Flags
	// -- -- Upload specifier
	fileUploadPath = fileUpload.Arg("filePath", "Path to the file you want to upload").Required().String()
	fileUploadName = fileUpload.Flag("name", "Specify the name of the file").String()
	// -- -- Delete specifier
	fileDeleteName = fileDelete.Arg("fileName", "Name of the file that should be removed").Required().String()
	fileDeleteID   = fileDelete.Arg("fileID", "FileID of file. Only required if mulitple files with same name are available").Int()
	// -- -- List specifier
	fileListName    = fileList.Arg("fileName", "Show files with this name").String()
	fileListDetails = fileList.Flag("details", "Print more details of files").Short('d').Counter()
	// -- -- Download specifier
	fileDownloadName = fileDownload.Arg("fileName", "Download files with this name").String()
	fileDownloadPath = fileDownload.Flag("path", "Where to store the file").Short('p').Required().String()
	// -- -- File-Update specifier
	fileUpdateName         = fileUpdate.Arg("fileName", "Name of the file that should be updated").Required().String()
	fileUpdateID           = fileUpdate.Arg("fileID", "FileID of file. Only required if mulitple files with same name are available").Int()
	fileUpdateTogglePublic = fileUpdate.Flag("isPublic", "Sets a file public or private").String()
	fileUpdateNewName      = fileUpdate.Flag("new-name", "Change the name of a file").String()
	fileUpdateNewNamespace = fileUpdate.Flag("new-namespace", "Change the namespace of a file").String()
	fileUpdateAddTags      = fileUpdate.Flag("add-tags", "Add tags to a file").Strings()
	fileUpdateRemoveTags   = fileUpdate.Flag("remove-tags", "Remove tags from a file").Strings()
	fileUpdateAddGroups    = fileUpdate.Flag("add-groups", "Add groups to a file").Strings()
	fileUpdateRemoveGroups = fileUpdate.Flag("remove-groups", "Remove groups from a file").Strings()

	// -- -- Tag-Update specifier
	tagUpdateName    = tagUpdate.Arg("fileName", "Name of the tag that should be updated").Required().String()
	tagUpdateNewName = tagUpdate.Flag("new-name", "Add a tag to a file").String()
	tagUpdateDelete  = tagUpdate.Flag("delete", "Delete the given tag").Bool()

	// Args/Config
	configUseTarget      = configUse.Arg("target", "Use different namespace as default").HintOptions(commands.UseTargets...).Required().String()
	configUseTargetValue = configUse.Arg("value", "the value of the new target").HintOptions("default").Strings()
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
		log.Info("New config created")
		return
	}

	//Use in config specified values for targets
	if len(*appNamespace) == 0 {
		*appNamespace = config.Default.Namespace
	}
	if len(*appTags) == 0 {
		*appTags = config.Default.Tags
	}
	if len(*appGroups) == 0 {
		*appGroups = config.Default.Groups
	}

	//Generate  file attributes
	fileAttributes := models.FileAttributes{
		Namespace: *appNamespace,
		Groups:    *appGroups,
		Tags:      *appTags,
	}

	// Execute the desired command
	switch parsed {
	// File commands
	case fileDownload.FullCommand():
		commands.DownloadFile(*fileDownloadName, *appNamespace, *appGroups, *appTags, *fileID, *fileDownloadPath)

	case fileUpload.FullCommand():
		commands.UploadFile(config, *fileUploadPath, *fileUploadName, fileAttributes)

	case fileDelete.FullCommand():
		commands.DeleteFile(config, *fileDeleteName, *fileDeleteID, fileAttributes)

	case fileList.FullCommand():
		commands.ListFiles(config, *fileListName, *fileID, fileAttributes, uint8(*fileListDetails)+1)

	case fileUpdate.FullCommand():
		commands.UpdateFile(config, *fileUpdateName, *fileUpdateID, *appNamespace, *fileUpdateTogglePublic, *fileUpdateNewName, *fileUpdateNewNamespace, *fileUpdateAddTags, *fileUpdateRemoveTags, *fileUpdateAddGroups, *fileUpdateRemoveGroups)

	case tagUpdate.FullCommand():
		commands.UpdateTag(config, *tagUpdateName, *appNamespace, *tagUpdateNewName, *tagUpdateDelete)

	// Ping
	case appPing.FullCommand():
		pingServer(config)

	// User
	case loginCmd.FullCommand():
		commands.LoginCommand(config, *loginCmdUser, *appYes)
	case registerCmd:
		commands.RegisterCommand(config)

	//Config
	case configUse.FullCommand():
		commands.ConfigUse(config, *configUseTarget, *configUseTargetValue)
	}
}

// Env vars
const (
	//EnVarPrefix prefix of all used env vars
	EnVarLogLevel   = "LOG_LEVEL"
	EnVarNoColor    = "NO_COLOR"
	EnVarConfigFile = "CONFIG"
)

// Return the variable using the server prefix
func getEnVar(name string) string {
	return fmt.Sprintf("%s_%s", EnVarPrefix, name)
}

func pingServer(config *models.Config) {
	var response server.StringResponse
	authorization := server.Authorization{}

	// Use session if available
	if config.IsLoggedIn() {
		authorization.Type = server.Bearer
		authorization.Palyoad = config.User.SessionToken
	}

	res, err := server.NewRequest(server.EPPing, server.PingRequest{Payload: "ping"}, config).WithAuth(authorization).Do(&response)

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
