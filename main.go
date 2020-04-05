package main

import (
	"fmt"
	"os"
	"time"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/DataManagerServer/constants"
	"github.com/JojiiOfficial/gaw"
	"github.com/Yukaru-san/DataManager_Client/commands"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	appName = "manager"
	version = "1.0.0"
)

//App commands
var (
	app = kingpin.New(appName, "A DataManager")

	// Global flags
	appYes     = app.Flag("yes", "Skip confirmations").Short('y').Bool()
	appNoColor = app.Flag("no-color", "Disable colors").Envar(getEnVar(EnVarNoColor)).Bool()
	appCfgFile = app.Flag("config", "the configuration file for the app").Envar(getEnVar(EnVarConfigFile)).Short('c').String()
	appBench   = app.Flag("bench", "Benchmark web calls").Bool()
	appQuiet   = app.Flag("quiet", "Less verbose output").Short('q').Bool()

	appNamespace             = app.Flag("namespace", "Specify the namespace to use").Default("default").Short('n').String()
	appTags                  = app.Flag("tag", "Specify tags to use").Short('t').Strings()
	appGroups                = app.Flag("group", "Specify groups to use").Short('g').Strings()
	appOutputJSON            = app.Flag("json", "Print output as json").Bool()
	appNoRedaction           = app.Flag("no-redact", "Don't redact secrets").Bool()
	appDetails               = app.Flag("details", "Print more details of something").Short('d').Counter()
	appTrimName              = app.Flag("trim-name", "Trim name after n chars").Int()
	appAll                   = app.Flag("all", "Do action for all found files").Short('a').Bool()
	appAllNamespaces         = app.Flag("all-namespaces", "Do action for all found files").Bool()
	appForce                 = app.Flag("force", "Forces an action").Short('f').Bool()
	appNoDecrypt             = app.Flag("no-decrypt", "Don't decrypt files").Bool()
	appNoEmojis              = app.Flag("no-emojis", "Don't decrypt files").Envar(getEnVar(EnVarNoEmojis)).Bool()
	appFileEncryption        = app.Flag("encryption", "Encrypt/Decrypt the file").Short('e').HintOptions(constants.EncryptionCiphers...).String()
	appFileEncryptionKey     = app.Flag("key", "Encryption/Decryption key").Short('k').String()
	appFileEncryptionRandKey = app.Flag("gen-key", "Generate Encryption key").Short('r').HintOptions("16", "24", "32").Int()
	appFileEncryptionPassKey = app.Flag("read-key", "Read encryption/decryption key as password").Short('p').Bool()

	//
	// ---------> Ping --------------------------------------
	appPing = app.Command("ping", "pings the server and checks connectivity")

	//
	// ---------> UserCommands --------------------------------------
	// -- Login
	loginCmd     = app.Command("login", "Login")
	loginCmdUser = loginCmd.Flag("username", "Your username").String()
	// -- Register
	registerCmd = app.Command("register", "Create an account").FullCommand()

	//
	// ---------> Config commands --------------------------------------
	configCMD = app.Command("config", "Commands for working with the config")

	// -- Use
	configUse            = configCMD.Command("use", "Use something")
	configUseTarget      = configUse.Arg("target", "Use different namespace as default").HintOptions(commands.UseTargets...).Required().String()
	configUseTargetValue = configUse.Arg("value", "the value of the new target").HintOptions("default").Strings()
	// -- View
	configView = configCMD.Command("view", "View config")

	//
	// ---------> Universal commands --------------------------------------

	// -- Upload
	appUpload            = app.Command("upload", "Upload the given file")
	fileUploadPath       = appUpload.Arg("filePath", "Path to the file you want to upload").Required().String()
	fileUploadName       = appUpload.Flag("name", "Specify the name of the file").String()
	fileUploadPublic     = appUpload.Flag("public", "Make uploaded file publci").Bool()
	fileUploadPublicName = appUpload.Flag("public-name", "Specify the public filename").String()
	fileUploadReplace    = appUpload.Flag("replace-file", "Replace a file").Uint()

	//
	// ---------> File commands --------------------------------------
	appFileCmd    = app.Command("file", "Do something with a file").Alias("f")
	appFilesCmd   = app.Command("files", "List files").Alias("fs")
	appFilesOrder = appFilesCmd.Flag("order", "Order the output").Short('o').HintOptions(models.AvailableOrders...).String()

	// -- Edit
	fileEditCmd = appFileCmd.Command("edit", "Edit a file")
	fileEditID  = fileEditCmd.Arg("ID", "The fileID").Required().Uint()

	// -- Delete
	fileDeleteCmd  = appFileCmd.Command("delete", "Delete a file").Alias("rm")
	fileDeleteName = fileDeleteCmd.Arg("fileName", "Name of the file that should be removed").String()
	fileDeleteID   = fileDeleteCmd.Arg("fileID", "FileID of file. Only required if mulitple files with same name are available").Uint()
	// -- List
	fileListCmd   = appFileCmd.Command("list", "List files")
	fileListName  = fileListCmd.Arg("fileName", "Show files with this name").String()
	fileListID    = fileListCmd.Arg("fileID", "The fileID").Uint()
	fileListOrder = fileListCmd.Flag("order", "Order the output").Short('o').HintOptions(models.AvailableOrders...).String()
	// -- Update
	fileUpdateCmd          = appFileCmd.Command("update", "Update a file")
	fileUpdateName         = fileUpdateCmd.Arg("fileName", "Name of the file that should be updated").Required().String()
	fileUpdateID           = fileUpdateCmd.Arg("fileID", "FileID of file. Only required if mulitple files with same name are available").Uint()
	fileUpdateSetPublic    = fileUpdateCmd.Flag("set-public", "Sets a file public").Bool()
	fileUpdateSetPrivate   = fileUpdateCmd.Flag("set-private", "Sets a file private").Bool()
	fileUpdateNewName      = fileUpdateCmd.Flag("new-name", "Change the name of a file").String()
	fileUpdateNewNamespace = fileUpdateCmd.Flag("new-namespace", "Change the namespace of a file").String()
	fileUpdateAddTags      = fileUpdateCmd.Flag("add-tags", "Add tags to a file").Strings()
	fileUpdateRemoveTags   = fileUpdateCmd.Flag("remove-tags", "Remove tags from a file").Strings()
	fileUpdateAddGroups    = fileUpdateCmd.Flag("add-groups", "Add groups to a file").Strings()
	fileUpdateRemoveGroups = fileUpdateCmd.Flag("remove-groups", "Remove groups from a file").Strings()
	// -- Download
	fileDownloadCmd     = appFileCmd.Command("download", "Download a file from the server")
	fileDownloadName    = fileDownloadCmd.Arg("fileName", "Download files with this name").String()
	fileDownloadID      = fileDownloadCmd.Arg("fileId", "Specify the fileID").Uint()
	fileDownloadPath    = fileDownloadCmd.Flag("output", "Where to store the file").Default("./").Short('o').String()
	fileDownloadPreview = fileDownloadCmd.Flag("preview", "Whether you want to open the file after downloading it").Bool()
	// -- Publish
	filePublishCmd    = appFileCmd.Command("publish", "publish something")
	filePublishName   = filePublishCmd.Arg("fileName", "Name of the file that should be published").Required().String()
	filePublishID     = filePublishCmd.Arg("fileID", "FileID of specified file. Only required if mulitple files with same name are available").Uint()
	publishPublicName = filePublishCmd.Flag("public-name", "Specify the public filename").String()
	// -- View
	viewCmd       = appFileCmd.Command("view", "View something")
	viewFileName  = viewCmd.Arg("fileName", "filename of file to view").Required().String()
	viewFileID    = viewCmd.Arg("fileID", "fileID of file to view").Uint()
	viewNoPreview = viewCmd.Flag("no-preview", "Disable preview for command").Bool()
	viewPreview   = viewCmd.Flag("preview", "Show preview for command").Bool()

	//
	// ---------> Tag commands --------------------------------------
	tagCmd = app.Command("tag", "Do something with tags")

	// -- Delete
	tagDeleteCmd  = tagCmd.Command("delete", "Delete a tag").Alias("rm").Alias("del")
	tagDeleteName = tagDeleteCmd.Arg("tagName", "Name of tag to delete").Required().String()
	// -- Update
	tagUpdateCmd     = tagCmd.Command("update", "Update a tag")
	tagUpdateName    = tagUpdateCmd.Arg("tagname", "Name of the tag that should be updated").Required().String()
	tagUpdateNewName = tagUpdateCmd.Flag("new-name", "New name of a tag").String()

	//
	// ---------> Group commands --------------------------------------
	groupCmd = app.Command("group", "Do something with groups")

	// -- Delete
	groupDeleteCmd  = groupCmd.Command("delete", "Delete a group").Alias("rm").Alias("del")
	groupDeleteName = groupDeleteCmd.Arg("groupName", "Name of group to delete").Required().String()
	// -- Update
	groupUpdateCmd     = groupCmd.Command("update", "Update a group")
	groupUpdateName    = groupUpdateCmd.Arg("groupName", "Name of the group that should be updated").Required().String()
	groupUpdateNewName = groupUpdateCmd.Flag("new-name", "Rename a group").String()

	//
	// ---------> Namespace commands --------------------------------------
	namespaceCmd  = app.Command("namespace", "Do something with namespaces").Alias("ns")
	namespacesCmd = app.Command("namespaces", "List your namespaces")

	// -- Create
	namespaceCreateCmd    = namespaceCmd.Command("create", "Create a namespace")
	namespaceCreateName   = namespaceCreateCmd.Arg("namespaceName", "Name of namespace").Required().String()
	namespaceCreateCustom = namespaceCreateCmd.Flag("custom", "Create a custom namespace (no username prefix)").Bool()
	// -- Update
	namespaceUpdateCmd     = namespaceCmd.Command("update", "Update a namespace")
	namespaceUpdateName    = namespaceUpdateCmd.Arg("namespaceName", "Name of the namespace that should be updated").Required().String()
	namespaceUpdateNewName = namespaceUpdateCmd.Flag("new-name", "Rename a namespace").String()
	// -- Delete
	namespaceDeleteCmd  = namespaceCmd.Command("delete", "Delete a namespace").Alias("rm").Alias("del")
	namespaceDeleteName = namespaceDeleteCmd.Arg("namespaceName", "Name of namespace to delete").Required().String()
	// -- List
	namespaceListCmd = namespaceCmd.Command("list", "List your namespaces")
)

var (
	config *models.Config
)

func main() {
	app.HelpFlag.Short('h')
	app.Version(version)

	gaw.Init()

	// Parsing the args
	parsed := kingpin.MustParse(app.Parse(os.Args[1:]))

	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  time.Stamp,
		FullTimestamp:    true,
		ForceColors:      !*appNoColor,
		DisableColors:    *appNoColor,
	})

	// Init config
	var err error
	config, err = models.InitConfig(models.GetDefaultConfigFile(), *appCfgFile)
	if err != nil {
		log.Error(err)
		return
	}

	if config == nil {
		log.Info("New config created")
		return
	}

	// Use config default values if not set
	if len(*appNamespace) == 0 || (*appNamespace) == "default" {
		*appNamespace = config.Default.Namespace
	}
	if len(*appTags) == 0 {
		*appTags = config.Default.Tags
	}
	if len(*appGroups) == 0 {
		*appGroups = config.Default.Groups
	}
	if *appDetails == 0 {
		*appDetails = config.Client.DefaultDetails
	}
	if len(*appFilesOrder) == 0 {
		*appFilesOrder = config.GetDefaultOrder()
	}
	if *appTrimName == 0 {
		*appTrimName = config.Client.TrimNameAfter
	}

	// Process params: make t1,t2 -> [t1 t2]
	commands.ProcesStrSliceParams(appTags, appGroups)

	// Generate  file attributes
	fileAttributes := models.FileAttributes{
		Namespace: *appNamespace,
		Groups:    *appGroups,
		Tags:      *appTags,
	}

	// Command data
	commandData := commands.CommandData{
		Command:           parsed,
		Config:            config,
		Details:           uint8(*appDetails),
		FileAttributes:    fileAttributes,
		Namespace:         *appNamespace,
		All:               *appAll,
		AllNamespaces:     *appAllNamespaces,
		NoRedaction:       *appNoRedaction,
		OutputJSON:        *appOutputJSON,
		Yes:               *appYes,
		Force:             *appForce,
		Bench:             *appBench,
		NameLen:           *appTrimName,
		Encryption:        *appFileEncryption,
		EncryptionKey:     *appFileEncryptionKey,
		EncryptionPassKey: *appFileEncryptionPassKey,
		NoDecrypt:         *appNoDecrypt,
		NoEmojis:          *appNoEmojis,
		RandKey:           *appFileEncryptionRandKey,
		Quiet:             *appQuiet,
	}

	if !commandData.Init() {
		return
	}

	startTime := time.Now()
	if *appBench {
		// Create channel
		commandData.BenchDone = make(chan time.Time, 1)
	}

	// Execute the desired command
	switch parsed {
	// -- File commands
	case fileDownloadCmd.FullCommand():
		// Download file
		commands.GetFile(commandData, *fileDownloadName, *fileDownloadID, *fileDownloadPath, *fileDownloadPreview, *viewNoPreview, *viewPreview)

	// iew file
	case viewCmd.FullCommand():
		commands.GetFile(commandData, *viewFileName, *viewFileID, "", true, *viewNoPreview, *viewPreview)

	// Upload
	case appUpload.FullCommand():
		commands.UploadFile(commandData, *fileUploadPath, *fileUploadName, *fileUploadPublicName, *fileUploadPublic, *fileUploadReplace)

	// Delete file
	case fileDeleteCmd.FullCommand():
		commands.DeleteFile(commandData, *fileDeleteName, *fileDeleteID)

	// List files
	case fileListCmd.FullCommand():
		commands.ListFiles(commandData, *fileListName, *fileDownloadID, *fileListOrder)

	// List file(s)
	case appFilesCmd.FullCommand():
		commands.ListFiles(commandData, "", *fileDownloadID, *appFilesOrder)

	// Update File
	case fileUpdateCmd.FullCommand():
		commands.UpdateFile(commandData, *fileUpdateName, *fileUpdateID, *fileUpdateNewName, *fileUpdateNewNamespace, *fileUpdateAddTags, *fileUpdateRemoveTags, *fileUpdateAddGroups, *fileUpdateRemoveGroups, *fileUpdateSetPublic, *fileUpdateSetPrivate)

	// Publish file
	case filePublishCmd.FullCommand():
		commands.PublishFile(commandData, *filePublishName, *filePublishID, *publishPublicName)

	// Edit file
	case fileEditCmd.FullCommand():
		commands.EditFile(commandData, *fileEditID)

	// -- Attributes commands
	// Update tag
	case tagUpdateCmd.FullCommand():
		commands.UpdateAttribute(commandData, libdm.TagAttribute, *tagUpdateName, *tagUpdateNewName)

	// Delete Tag
	case tagDeleteCmd.FullCommand():
		commands.DeleteAttribute(commandData, libdm.TagAttribute, *tagDeleteName)

	// Update group
	case groupUpdateCmd.FullCommand():
		commands.UpdateAttribute(commandData, libdm.GroupAttribute, *groupUpdateName, *groupUpdateNewName)

	// Delete Group
	case groupDeleteCmd.FullCommand():
		commands.DeleteAttribute(commandData, libdm.GroupAttribute, *groupDeleteName)

	// -- Namespace commands
	// Create namespace
	case namespaceCreateCmd.FullCommand():
		commands.CreateNamespace(commandData, *namespaceCreateName, *namespaceCreateCustom)

	// Update namespace
	case namespaceUpdateCmd.FullCommand():
		commands.UpdateNamespace(commandData, *namespaceUpdateName, *namespaceUpdateNewName, *namespaceCreateCustom)

	// Delete namespace
	case namespaceDeleteCmd.FullCommand():
		commands.DeleteNamespace(commandData, *namespaceDeleteName)

	// List namespaces
	case namespaceListCmd.FullCommand(), namespacesCmd.FullCommand():
		commands.ListNamespace(commandData)

	// -- Ping command
	case appPing.FullCommand():
		pingServer(commandData)

	// -- User commands
	case loginCmd.FullCommand():
		commands.LoginCommand(commandData, *loginCmdUser)
	case registerCmd:
		commands.RegisterCommand(commandData)

	// -- Config commands
	case configUse.FullCommand():
		commands.ConfigUse(commandData, *configUseTarget, *configUseTargetValue)
	case configView.FullCommand():
		commands.ConfigView(commandData)
	}

	if *appBench {
		endTime := <-commandData.BenchDone
		fmt.Println()
		logrus.Infof("Bench took %s", endTime.Sub(startTime).String())
	}
}

// Env vars
const (
	// EnVarPrefix prefix for env vars
	EnVarPrefix = "MANAGER"

	// EnVarPrefix prefix of all used env vars
	EnVarLogLevel   = "LOG_LEVEL"
	EnVarNoColor    = "NO_COLOR"
	EnVarNoEmojis   = "NO_EMOJIS"
	EnVarConfigFile = "CONFIG"
)

// Return the variable using the server prefix
func getEnVar(name string) string {
	return fmt.Sprintf("%s_%s", EnVarPrefix, name)
}

func pingServer(cData commands.CommandData) {
	var response server.StringResponse
	authorization := server.Authorization{}

	// Use session if available
	if config.IsLoggedIn() {
		authorization.Type = server.Bearer
		authorization.Palyoad = config.User.SessionToken
	}

	res, err := server.NewRequest(server.EPPing, server.PingRequest{Payload: "ping"}, cData.Config).
		WithAuth(authorization).
		WithBenchCallback(cData.BenchDone).
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
