package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	libdm "github.com/DataManager-Go/libdatamanager"
	dmConfig "github.com/DataManager-Go/libdatamanager/config"
	"github.com/JojiiOfficial/gaw"
	"github.com/fatih/color"

	"github.com/DataManager-Go/DataManagerCLI/commands"
	_ "github.com/mattn/go-sqlite3"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	appName = "manager"
	version = "1.0.0"
)

// ...
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

// App commands
var (
	app = kingpin.New(appName, "A DataManager")

	// Global flags
	appYes     = app.Flag("yes", "Skip confirmations").Short('y').Bool()
	appCfgFile = app.Flag("config", "the configuration file for the app").Envar(getEnVar(EnVarConfigFile)).Short('c').String()

	// File related flags
	appTags           = app.Flag("tag", "Specify tags to use").Short('t').Strings()
	appGroups         = app.Flag("group", "Specify groups to use").Short('g').Strings()
	appNamespace      = app.Flag("namespace", "Specify the namespace to use").Default("default").Short('n').HintAction(hintListNamespaces).String()
	appAllNamespaces  = app.Flag("all-namespaces", "Do action for all found files").Bool()
	appAll            = app.Flag("all", "Do action for all found files").Short('a').Bool()
	appVerify         = app.Flag("verify", "Verify a file using a checksum to prevent errors").Bool()
	appNoDecrypt      = app.Flag("no-decrypt", "Don't decrypt files").Bool()
	appForce          = app.Flag("force", "Forces an action").Short('f').Bool()
	appFileEncryption = app.Flag("encryption", "Encrypt/Decrypt the file").Short('e').HintOptions(libdm.EncryptionCiphers...).String()

	// Output related flags
	appDetails     = app.Flag("details", "Print more details of something").Short('d').Counter()
	appQuiet       = app.Flag("quiet", "Less verbose output").Short('q').Bool()
	appOutputJSON  = app.Flag("json", "Print output as json").Bool()
	appNoRedaction = app.Flag("no-redact", "Don't redact secrets").Bool()
	appNoColor     = app.Flag("no-color", "Disable colors").Envar(getEnVar(EnVarNoColor)).Bool()
	appNoEmojis    = app.Flag("no-emojis", "Don't decrypt files").Envar(getEnVar(EnVarNoEmojis)).Bool()

	// Encryptionkey related flags
	appFileEncrRandKey      = app.Flag("gen-key", "Generate Encryption key").Short('r').HintOptions("16", "24", "32").Int()
	appFileEncrKey          = app.Flag("key", "Encryption/Decryption key").Short('k').String()
	appFileEncrPassKey      = app.Flag("read-key", "Read encryption/decryption key as password").Short('p').Bool()
	appFileEncrKeyFromStdin = app.Flag("key-from-stdin", "Read encryption/decryption key from stdin").Bool()
	appFileEncrKeyFile      = app.Flag("keyfile", "File containing a Encryption/Decryption key").HintAction(hintListKeyFiles).String()

	//
	// ---------> Ping --------------------------------------
	appPing = app.Command("ping", "pings the server and checks connectivity").Alias("p")

	//
	// ---------> UserCommands --------------------------------------
	// -- Login
	loginCmd     = app.Command("login", "Login")
	loginCmdUser = loginCmd.Flag("username", "Your username").String()
	// -- Logout
	logoutCmd     = app.Command("logout", "logout from a session")
	logoutCmdUser = logoutCmd.Arg("username", "Delete sessiondata assigned to a different username than the current one").String()
	// -- Register
	registerCmd = app.Command("register", "Create an account").FullCommand()
	// -- Setup
	setupCmd           = app.Command("setup", "Setup your client to get started")
	setupCmdHost       = setupCmd.Arg("host", "The host of the server you want to use").String()
	setupCmdHostFlag   = setupCmd.Flag("host", "The host of the server you want to use").String()
	setupCmdIgnoreCert = setupCmd.Flag("Ignore-cert", "Ignore server certificate (unsafe)").Bool()
	setupCmdServerOnly = setupCmd.Flag("server-only", "Setup the server connection only. No login").Bool()
	setupCmdRegister   = setupCmd.Flag("register", "Register after logging in").Bool()
	setupCmdNoLogin    = setupCmd.Flag("no-login", "Don't login after setting up").Bool()

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
	// ---------> File commands --------------------------------------
	// -- Upload
	appUpload              = app.Command("upload", "Upload the given file").Alias("up").Alias("push")
	fileUploadPaths        = appUpload.Arg("filePath", "Path to the file you want to upload").HintAction(hintListFiles).Strings()
	fileUploadFromStdin    = appUpload.Flag("from-stdin", "Read file from stdin and upload it").Bool()
	fileUploadName         = appUpload.Flag("name", "Specify the name of the file").String()
	fileUploadPublic       = appUpload.Flag("public", "Make uploaded file publci").Bool()
	fileUploadPublicName   = appUpload.Flag("public-name", "Specify the public filename").String()
	fileUploadReplace      = appUpload.Flag("replace-file", "Replace a file").Uint()
	fileUploadParallelism  = appUpload.Flag("parallelism", "Upload n files at the same time").Default("1").Uint()
	fileUploadDeletInvaid  = app.Flag("delete-invaid", "Deletes a file if it's checksum is invalid").Bool()
	fileUploadSetClipboard = app.Flag("set-clip", "Set clipboard to pubilc url").Bool()

	// -- List
	appFileCmd           = app.Command("file", "Do something with a file").Alias("f")
	appFilesCmd          = app.Command("files", "List files").Alias("fs").Alias("ls").Alias("dir")
	appFilesCmdNamespace = appFilesCmd.Arg("namespace", "List files in a specific namespace").String()
	appFilesOrder        = appFilesCmd.Flag("order", "Order the output").Short('o').HintOptions(commands.AvailableOrders...).String()

	// -- Create
	fileCreateCmd     = appFileCmd.Command("create", "create a file").Alias("c").Alias("cr")
	filecreateCmdName = fileCreateCmd.Arg("name", "The name of file to create").String()

	// -- Edit
	fileEditCmd = appFileCmd.Command("edit", "Edit a file").Alias("e")
	fileEditID  = fileEditCmd.Arg("ID", "The fileID").Required().Uint()

	// -- Tree
	appFileTree          = app.Command("tree", "Show your files like the unix file tree")
	appFileTreeOrder     = appFileTree.Flag("order", "Order the output").Short('o').HintOptions(commands.AvailableOrders...).String()
	appFileTreeNamespace = appFileTree.Arg("namespace", "View only a namespace").String()

	// -- Delete file -> rm
	fileRmCmd  = app.Command("rm", "Delete a file")
	fileRmName = fileRmCmd.Arg("fileName", "Name of the file that should be removed").String()
	fileRmID   = fileRmCmd.Arg("fileID", "FileID of file. Only required if mulitple files with same name are available").Uint()
	// -- Delete -> file delete/rm
	fileDeleteCmd  = appFileCmd.Command("delete", "Delete a file").Alias("rm").Alias("del")
	fileDeleteName = fileDeleteCmd.Arg("fileName", "Name of the file that should be removed").String()
	fileDeleteID   = fileDeleteCmd.Arg("fileID", "FileID of file. Only required if mulitple files with same name are available").Uint()
	// -- List
	fileListCmd   = appFileCmd.Command("list", "List files").Alias("ls")
	fileListName  = fileListCmd.Arg("fileName", "Show files with this name").String()
	fileListID    = fileListCmd.Arg("fileID", "The fileID").Uint()
	fileListOrder = fileListCmd.Flag("order", "Order the output").Short('o').HintOptions(commands.AvailableOrders...).String()
	// -- Update
	fileUpdateCmd          = appFileCmd.Command("update", "Update a file").Alias("u")
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
	fileDownloadCmd     = appFileCmd.Command("download", "Download a file from the server").Alias("dl")
	fileDownloadName    = fileDownloadCmd.Arg("fileName", "Download files with this name").String()
	fileDownloadID      = fileDownloadCmd.Arg("fileId", "Specify the fileID").Uint()
	fileDownloadPath    = fileDownloadCmd.Flag("output", "Where to store the file").Default("./").Short('o').String()
	fileDownloadPreview = fileDownloadCmd.Flag("preview", "Whether you want to open the file after downloading it").Bool()
	// -- Publish
	filePublishCmd    = appFileCmd.Command("publish", "publish something").Alias("pub").Alias("p")
	filePublishName   = filePublishCmd.Arg("fileName", "Name of the file that should be published").Required().String()
	filePublishID     = filePublishCmd.Arg("fileID", "FileID of specified file. Only required if mulitple files with same name are available").Uint()
	publishPublicName = filePublishCmd.Flag("public-name", "Specify the public filename").String()
	// -- View
	viewCmd       = appFileCmd.Command("view", "View something").Alias("v")
	viewFileName  = viewCmd.Arg("fileName", "filename of file to view").Required().String()
	viewFileID    = viewCmd.Arg("fileID", "fileID of file to view").Uint()
	viewNoPreview = viewCmd.Flag("no-preview", "Disable preview for command").Bool()
	viewPreview   = viewCmd.Flag("preview", "Show preview for command").Bool()

	//
	// ---------> Tag commands --------------------------------------
	tagCmd = app.Command("tag", "Do something with tags").Alias("t")

	// -- List
	tagListCmd = app.Command("tags", "List existing tags").Alias("ts")
	// -- Delete
	tagDeleteCmd  = tagCmd.Command("delete", "Delete a tag").Alias("rm").Alias("del")
	tagDeleteName = tagDeleteCmd.Arg("tagName", "Name of tag to delete").Required().String()
	// -- Update
	tagUpdateCmd     = tagCmd.Command("update", "Update a tag").Alias("u")
	tagUpdateName    = tagUpdateCmd.Arg("tagname", "Name of the tag that should be updated").Required().String()
	tagUpdateNewName = tagUpdateCmd.Flag("new-name", "New name of a tag").String()

	//
	// ---------> Group commands --------------------------------------
	groupCmd = app.Command("group", "Do something with groups").Alias("g").Alias("gr")

	// -- List
	groupListCmd = app.Command("groups", "List existing groups").Alias("gs")
	// -- Delete
	groupDeleteCmd  = groupCmd.Command("delete", "Delete a group").Alias("rm").Alias("del")
	groupDeleteName = groupDeleteCmd.Arg("groupName", "Name of group to delete").Required().String()
	// -- Update
	groupUpdateCmd     = groupCmd.Command("update", "Update a group").Alias("u")
	groupUpdateName    = groupUpdateCmd.Arg("groupName", "Name of the group that should be updated").Required().String()
	groupUpdateNewName = groupUpdateCmd.Flag("new-name", "Rename a group").String()

	//
	// ---------> Namespace commands --------------------------------------
	namespaceCmd  = app.Command("namespace", "Do something with namespaces").Alias("ns").Alias("n")
	namespacesCmd = app.Command("namespaces", "List your namespaces").Alias("nss")

	// -- Create
	namespaceCreateCmd    = namespaceCmd.Command("create", "Create a namespace").Alias("c").Alias("cr")
	namespaceCreateName   = namespaceCreateCmd.Arg("namespaceName", "Name of namespace").Required().String()
	namespaceCreateCustom = namespaceCreateCmd.Flag("custom", "Create a custom namespace (no username prefix)").Bool()
	// -- Update
	namespaceUpdateCmd     = namespaceCmd.Command("update", "Update a namespace").Alias("u")
	namespaceUpdateName    = namespaceUpdateCmd.Arg("namespaceName", "Name of the namespace that should be updated").Required().String()
	namespaceUpdateNewName = namespaceUpdateCmd.Flag("new-name", "Rename a namespace").String()
	// -- Delete
	namespaceDeleteCmd  = namespaceCmd.Command("delete", "Delete a namespace").Alias("rm").Alias("del")
	namespaceDeleteName = namespaceDeleteCmd.Arg("namespaceName", "Name of namespace to delete").Required().String()
	// -- List
	namespaceListCmd = namespaceCmd.Command("list", "List your namespaces").Alias("ls")
	// -- Download
	namespaceDownloadCmd           = namespaceCmd.Command("download", "Download all files in a namespace").Alias("dl")
	namespaceDownloadNs            = namespaceDownloadCmd.Arg("namespace", "The namespace to download the files from").HintAction(hintListNamespaces).Required().String()
	namespaceDownloadExcludeGroups = namespaceDownloadCmd.Flag("exclude-groups", "Exclude files in specified group(s) from getting downloaded").Strings()
	namespaceDownloadExcludeTags   = namespaceDownloadCmd.Flag("exclude-tags", "Exclude files having specified tags(s) from getting downloaded").Strings()
	namespaceDownloadExcludeFiles  = namespaceDownloadCmd.Flag("exclude-files", "Exclude files by ID").Strings()
	namespaceDownloadParallelism   = namespaceDownloadCmd.Flag("parallelism", "Download multiple files at the same time").Default("1").Uint()
	namespaceDownloadOutputDir     = namespaceDownloadCmd.Flag("output", "Save namespace in a custom directory than the namespacename").Short('o').Default("./").String()

	//
	// ---------> Keystore commands --------------------------------------
	keystoreCmd = app.Command("keystore", "Save keys to assigned to files and store them into a specific directory").Alias("ks")

	// -- Create
	keystoreCreateCmd          = keystoreCmd.Command("create", "Create a keystore").Alias("c").Alias("cr")
	keystoreCreateCmdPath      = keystoreCreateCmd.Arg("path", "The path to store the keys in").Required().String()
	keystoreCreateCmdOverwrite = keystoreCreateCmd.Flag("overwrite", "Overwrite an existing keystore setting").Short('o').Bool()
	// -- Info
	keystoreInfoCmd = keystoreCmd.Command("info", "Show information to your keystore")
	// -- Delete
	keystoreDeleteCmd           = keystoreCmd.Command("delete", "Delete a keystore")
	keystoreDeleteCmdShredCount = keystoreDeleteCmd.Flag("shredder", "Overwrite your keys").Default("6").Uint()
	// -- CleanUp
	keystoreCleanupCmd           = keystoreCmd.Command("cleanup", "Cleans up unassigned keys").Alias("c").Alias("clean")
	keystoreCleanupCmdShredCount = keystoreCleanupCmd.Flag("shredder", "Overwrite your keys").Default("6").Uint()
	// -- AddKey
	keystoreAddKeyCmd       = keystoreCmd.Command("add", "Adds a Key to a file to the keystore").Alias("a")
	keystoreAddKeyCmdFileID = keystoreAddKeyCmd.Arg("fileID", "The file id where the key should be assigned to").Required().Uint()
	keystoreAddKeyCmdKey    = keystoreAddKeyCmd.Arg("keyfile", "The filename of the keyfile. Must be located in the keystore path").HintAction(hintListKeyFiles).Required().String()
	// -- RemoveKey
	keystoreRemoveKeyCmd   = keystoreCmd.Command("remove", "Removes a key from keystore by it's assigned fileid").Alias("rm")
	keystoreRemoveKeyCmdID = keystoreRemoveKeyCmd.Arg("fileID", "The fileID to delete the key from ").Required().Uint()
)

var (
	config       *dmConfig.Config
	appTrimName  int
	unmodifiedNS string
)

func main() {
	app.HelpFlag.Short('h')
	app.Version(version)

	// Init random seed from gaw
	gaw.Init()

	// Prase cli flags
	parsed := kingpin.MustParse(app.Parse(os.Args[1:]))

	// Init config
	if !initConfig(parsed) {
		return
	}

	unmodifiedNS = *appNamespace

	// Process params: make t1,t2 -> [t1 t2]
	commands.ProcesStrSliceParams(appTags, appGroups)

	initDefaults()

	if *appNoColor {
		color.NoColor = true
	}

	// Bulid commandData
	commandData := buildCData(parsed, appTrimName)
	if commandData == nil {
		return
	}
	defer commandData.CloseKeystore()

	// Run desired command
	runCommand(parsed, commandData)
}

// Load and init config. Return false on error
func initConfig(parsed string) bool {
	// Init config
	var err error
	config, err = dmConfig.InitConfig(dmConfig.GetDefaultConfigFile(), *appCfgFile)
	if err != nil {
		log.Fatalln(err)
	}

	if config == nil {
		fmt.Println("New config created")
		if parsed != setupCmd.FullCommand() {
			return false
		}
	}

	return true
}

// Init default values from config
func initDefaults() {
	// Config is nil if a new configfile
	// was created and setup command is running
	if config != nil {
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
			*appDetails = config.Client.Defaults.DefaultDetails
		}
		if len(*appFilesOrder) == 0 {
			*appFilesOrder = config.GetDefaultOrder()
		}
		if !*appVerify && config.User.ForceVerify {
			*appVerify = true
		}
		if !*fileUploadDeletInvaid && config.User.DeleteInvaildFiles {
			*fileUploadDeletInvaid = true
		}
		appTrimName = config.Client.TrimNameAfter
	}
}

// ---- CLI Hint funcs ------

// Returns a slice containing all files in current folder
func hintListFiles() []string {
	fileInfos, err := ioutil.ReadDir(".")
	if err != nil {
		return []string{err.Error()}
	}

	var files []string
	for _, fi := range fileInfos {
		if !fi.IsDir() {
			files = append(files, fi.Name())
		}
	}

	return files
}

// Returns a slice containing all files in current folder
// except the keystore DB file
func hintListKeyFiles() []string {
	files := hintListFiles()
	retFiles := []string{}
	for i := range files {
		if files[i] != libdm.KeystoreDBFile {
			retFiles = append(retFiles, files[i])
		}
	}

	return retFiles
}

// Return a slice containing all available namespaces
func hintListNamespaces() []string {
	if !initConfig("") {
		return []string{}
	}

	config, err := config.ToRequestConfig()
	if err != nil {
		return []string{}
	}

	libDM := libdm.NewLibDM(config)
	namespaces, err := libDM.GetNamespaces()
	if err != nil {
		return []string{}
	}

	return namespaces.Slice
}
