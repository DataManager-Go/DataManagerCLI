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
	app = kingpin.New(appName, "A DataManager")

	// Global flags
	appYes     = app.Flag("yes", "Skip confirmations").Bool()
	appNoColor = app.Flag("no-color", "Disable colors").Envar(getEnVar(EnVarNoColor)).Bool()
	appCfgFile = app.Flag("config", "the configuration file for the app").Envar(getEnVar(EnVarConfigFile)).Short('c').String()

	appNamespace = app.Flag("namespace", "Specify the namespace to use").Default("default").Short('n').String()
	appTags      = app.Flag("tag", "Specify tags to use").Short('t').Strings()
	appGroups    = app.Flag("group", "Specify groups to use").Short('g').Strings()
	appDetails   = app.Flag("details", "Print more details of something").Short('d').Counter()

	// --- :Commands: -------

	// ---------> Ping
	appPing = app.Command("ping", "pings the server and checks connectivity")

	// ---------> UserCommands
	userCmd = app.Command("user", "Do user related stuff")
	// -- Login
	loginCmd     = userCmd.Command("login", "Login")
	loginCmdUser = loginCmd.Flag("username", "Your username").String()
	// -- Register
	registerCmd = userCmd.Command("register", "Create an account").FullCommand()

	// ---------> Config commands
	configCMD = app.Command("config", "Commands for working with the config")
	//Use
	configUse            = configCMD.Command("use", "Use something")
	configUseTarget      = configUse.Arg("target", "Use different namespace as default").HintOptions(commands.UseTargets...).Required().String()
	configUseTargetValue = configUse.Arg("value", "the value of the new target").HintOptions("default").Strings()

	// ---------> File commands
	// -- Upload
	appUpload      = app.Command("upload", "Upload the given file")
	fileUploadPath = appUpload.Arg("filePath", "Path to the file you want to upload").Required().String()
	fileUploadName = appUpload.Flag("name", "Specify the name of the file").String()

	// -- Delete
	appDelete = app.Command("delete", "Delete something from the server")
	//Delete file
	deleteFileCmd  = appDelete.Command("file", "Delete a file")
	fileDeleteName = deleteFileCmd.Arg("fileName", "Name of the file that should be removed").Required().String()
	fileDeleteID   = deleteFileCmd.Arg("fileID", "FileID of file. Only required if mulitple files with same name are available").Uint()
	//Delete Tag
	deleteTagCmd  = appDelete.Command("tag", "Delete a tag")
	deleteTagName = deleteTagCmd.Arg("tagName", "Name of tag to delete").Required().String()
	//Delete Group
	deleteGroup     = appDelete.Command("group", "Delete a group")
	deleteGroupName = deleteGroup.Arg("groupName", "Name of group to delete").Required().String()

	// -- List
	appList = app.Command("list", "List stuff stored on the server")
	//List File
	listFilesCmd = appList.Command("files", "List files")
	//List files
	listFileCmd  = appList.Command("file", "List files")
	fileListName = listFileCmd.Arg("fileName", "Show files with this name").String()

	// -- Download
	fileDownload     = app.Command("download", "Download a file from the server")
	fileDownloadName = fileDownload.Arg("fileName", "Download files with this name").String()
	fileDownloadPath = fileDownload.Arg("path", "Where to store the file").Default("./").String()
	fileDownloadID   = fileDownload.Flag("file-id", "Specify the fileID").Uint()

	// -- Update
	appUpdate = app.Command("update", "Update the filesystem")
	//Update File
	fileUpdate             = appUpdate.Command("file", "Update a file")
	fileUpdateName         = fileUpdate.Arg("fileName", "Name of the file that should be updated").Required().String()
	fileUpdateID           = fileUpdate.Arg("fileID", "FileID of file. Only required if mulitple files with same name are available").Uint()
	fileUpdateSetPublic    = fileUpdate.Flag("set-public", "Sets a file public").Bool()
	fileUpdateSetPrivate   = fileUpdate.Flag("set-private", "Sets a file private").Bool()
	fileUpdateNewName      = fileUpdate.Flag("new-name", "Change the name of a file").String()
	fileUpdateNewNamespace = fileUpdate.Flag("new-namespace", "Change the namespace of a file").String()
	fileUpdateAddTags      = fileUpdate.Flag("add-tags", "Add tags to a file").Strings()
	fileUpdateRemoveTags   = fileUpdate.Flag("remove-tags", "Remove tags from a file").Strings()
	fileUpdateAddGroups    = fileUpdate.Flag("add-groups", "Add groups to a file").Strings()
	fileUpdateRemoveGroups = fileUpdate.Flag("remove-groups", "Remove groups from a file").Strings()
	//Update Tag
	tagUpdate        = appUpdate.Command("tag", "Update a tag")
	tagUpdateName    = tagUpdate.Arg("fileName", "Name of the tag that should be updated").Required().String()
	tagUpdateNewName = tagUpdate.Flag("new-name", "New name of a tag").String()
	//Update Group
	groupUpdate        = appUpdate.Command("group", "Update a group")
	groupUpdateName    = groupUpdate.Arg("groupName", "Name of the group that should be updated").Required().String()
	groupUpdateNewName = groupUpdate.Flag("new-name", "Rename a group").String()

	// -- View
	viewCmd = app.Command("view", "View something")
	//View file
	viewFileCmd  = viewCmd.Command("file", "view a file")
	viewFileName = viewFileCmd.Arg("fileName", "The filename to view").Required().String()
	viewFileID   = viewFileCmd.Arg("fileID", "The fileID to view").Uint()
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
	if len(*appNamespace) == 0 || (*appNamespace) == "default" {
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
		commands.DownloadFile(*fileDownloadName, *appNamespace, *appGroups, *appTags, *fileDownloadID, *fileDownloadPath)

	case appUpload.FullCommand():
		commands.UploadFile(config, *fileUploadPath, *fileUploadName, fileAttributes)

	case deleteFileCmd.FullCommand():
		commands.DeleteFile(config, *fileDeleteName, *fileDeleteID, fileAttributes)

	case listFileCmd.FullCommand():
		commands.ListFiles(config, *fileListName, *fileDownloadID, fileAttributes, uint8(*appDetails)+1)

	case listFilesCmd.FullCommand():
		commands.ListFiles(config, "", *fileDownloadID, fileAttributes, uint8(*appDetails)+1)

	case fileUpdate.FullCommand():
		commands.UpdateFile(config, *fileUpdateName, *fileUpdateID, *appNamespace, *fileUpdateNewName, *fileUpdateNewNamespace, *fileUpdateAddTags, *fileUpdateRemoveTags, *fileUpdateAddGroups, *fileUpdateRemoveGroups, *fileUpdateSetPublic, *fileUpdateSetPrivate)

	case tagUpdate.FullCommand():
		commands.UpdateTag(config, *tagUpdateName, *appNamespace, *tagUpdateNewName)

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
