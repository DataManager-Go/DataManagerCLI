package main

import (
	"fmt"

	"github.com/DataManager-Go/DataManagerCLI/commands"
	libdm "github.com/DataManager-Go/libdatamanager"
)

func runCommand(parsed string, commandData *commands.CommandData) {
	// Execute the desired command
	switch parsed {
	// -- File commands
	// Download file
	case fileDownloadCmd.FullCommand():
		filename, id := commands.GetFileCommandData(*fileDownloadName, *fileDownloadID)
		commandData.DownloadFile(&commands.DownloadData{
			FileName:  filename,
			FileID:    id,
			Preview:   *viewPreview,
			NoPreview: *viewNoPreview,
			LocalPath: *fileDownloadPath,
		}, nil)

	// View file
	case viewCmd.FullCommand():
		filename, id := commands.GetFileCommandData(*viewFileName, *viewFileID)
		commandData.ViewFile(&commands.DownloadData{
			FileName:  filename,
			FileID:    id,
			Preview:   *viewPreview,
			NoPreview: *viewNoPreview,
		}, nil)

	// Upload
	case appUpload.FullCommand():
		commandData.UploadItems(*fileUploadPaths, *fileUploadParallelism, &commands.UploadData{
			Name:          *fileUploadName,
			DeleteInvalid: *fileUploadDeletInvaid,
			FromStdIn:     *fileUploadFromStdin,
			Public:        *fileUploadPublic,
			PublicName:    *fileUploadPublicName,
			ReplaceFile:   *fileUploadReplace,
			SetClip:       *fileUploadSetClipboard,
			NoArchiving:   *fileUploadNoArchiving,
		})

	case fileCreateCmd.FullCommand():
		commandData.CreateFile(*filecreateCmdName)

	// Delete file
	case fileDeleteCmd.FullCommand():
		commands.DeleteFile(commandData, *fileDeleteName, *fileDeleteID)

	// Delete file (rm)
	case fileRmCmd.FullCommand():
		commands.DeleteFile(commandData, *fileRmName, *fileRmID)

	// List files
	case fileListCmd.FullCommand():
		commands.ListFiles(commandData, *fileListName, *fileListID, *fileListOrder)

	// List file(s)
	case appFilesCmd.FullCommand():
		if len(*appFilesCmdNamespace) > 0 {
			commandData.FileAttributes.Namespace = *appFilesCmdNamespace
		}
		commands.ListFiles(commandData, "", *fileListID, *appFilesOrder)

	// File Tree
	case appFileTree.FullCommand():
		commandData.FileTree(*appFileTreeOrder, *appFileTreeNamespace)

	// Update File
	case fileUpdateCmd.FullCommand():
		commands.UpdateFile(commandData, *fileUpdateName, *fileUpdateID, *fileUpdateNewName, *fileUpdateNewNamespace, *fileUpdateAddTags, *fileUpdateRemoveTags, *fileUpdateAddGroups, *fileUpdateRemoveGroups, *fileUpdateSetPublic, *fileUpdateSetPrivate)

	// Publish file
	case filePublishCmd.FullCommand():
		commands.PublishFile(commandData, *filePublishName, *filePublishID, *publishPublicName, *fileUploadSetClipboard)

	// Edit file
	case fileEditCmd.FullCommand():
		commandData.EditFile(*fileEditID)

	// Move file
	case fileMoveCmd.FullCommand():
		commands.UpdateFile(commandData, *fileMoveFile, 0, "", *fileMoveNewNs, nil, nil, nil, nil, false, false)

	// -- Attributes commands
	// List Tags
	case tagListCmd.FullCommand():
		commandData.ListAttributes(libdm.TagAttribute)

	// Update tag
	case tagUpdateCmd.FullCommand():
		commands.UpdateAttribute(commandData, libdm.TagAttribute, *tagUpdateName, *tagUpdateNewName)

	// Delete Tag
	case tagDeleteCmd.FullCommand():
		commands.DeleteAttribute(commandData, libdm.TagAttribute, *tagDeleteName)

	// List Groups
	case groupListCmd.FullCommand():
		commandData.ListAttributes(libdm.GroupAttribute)

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

	// Download files in namespace
	case namespaceDownloadCmd.FullCommand():
		commandData.FileAttributes.Namespace = *namespaceDownloadNs
		commandData.DownloadNamespace(*namespaceDownloadExcludeGroups, *namespaceDownloadExcludeTags, *namespaceDownloadExcludeFiles, *namespaceDownloadParallelism, *namespaceDownloadOutputDir)

	// -- Ping command
	case appPing.FullCommand():
		commands.Ping(commandData)

	// -- User commands
	// Login
	case loginCmd.FullCommand():
		commands.LoginCommand(commandData, *loginCmdUser)

	case logoutCmd.FullCommand():
		commandData.Logout(*logoutCmdUser)

	// Register
	case registerCmd:
		commands.RegisterCommand(commandData)

	// Setup
	case setupCmd.FullCommand():
		{
			host := *setupCmdHostFlag
			if len(host) == 0 {
				host = *setupCmdHost
			}
			if len(host) == 0 {
				fmt.Println("You have to specify a host")
				return
			}

			commands.SetupClient(commandData, host, *appCfgFile, *setupCmdIgnoreCert, *setupCmdServerOnly, *setupCmdRegister, *setupCmdNoLogin, *setupCmdToken, *setupCmdUsername)
		}

	// -- Config commands
	// Config use
	case configUse.FullCommand():
		commands.ConfigUse(commandData, *configUseTarget, *configUseTargetValue)

	// Config view
	case configView.FullCommand():
		commands.ConfigView(commandData, *configViewTokenBase)

	// -- KeystoreCommands
	// Keystore create
	case keystoreCreateCmd.FullCommand():
		commands.CreateKeystore(commandData, *keystoreCreateCmdPath, *keystoreCreateCmdOverwrite)

	// Keystore Info
	case keystoreInfoCmd.FullCommand():
		commands.KeystoreInfo(commandData)

	// Keystore delete
	case keystoreDeleteCmd.FullCommand():
		commands.KeystoreDelete(commandData, *keystoreDeleteCmdShredCount)

	// Keystore cleanup
	case keystoreCleanupCmd.FullCommand():
		commands.KeystoreCleanup(commandData, *keystoreCleanupCmdShredCount)

	// Keystore add key
	case keystoreAddKeyCmd.FullCommand():
		commands.KeystoreAddKey(commandData, *keystoreAddKeyCmdKey, *keystoreAddKeyCmdFileID)

	// Remove key from keystore
	case keystoreRemoveKeyCmd.FullCommand():
		commands.KeystoreRemoveKey(commandData, *keystoreRemoveKeyCmdID)

	}
}
