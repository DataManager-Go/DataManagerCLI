package main

import (
	"fmt"

	libdm "github.com/DataManager-Go/libdatamanager"

	"github.com/DataManager-Go/DataManagerCLI/commands"
)

func runCommand(parsed string, commandData *commands.CommandData) {
	// Execute the desired command
	switch parsed {
	// -- File commands
	case fileDownloadCmd.FullCommand():
		// Download file
		commands.GetFile(commandData, *fileDownloadName, *fileDownloadID, *fileDownloadPath, *fileDownloadPreview, *viewNoPreview, *viewPreview)

	// View file
	case viewCmd.FullCommand():
		commands.GetFile(commandData, *viewFileName, *viewFileID, "", true, *viewNoPreview, *viewPreview)

	// Upload
	case appUpload.FullCommand():
		commands.UploadFile(commandData, *fileUploadPath, *fileUploadName, *fileUploadPublicName, *fileUploadPublic, *fileUploadFromStdin, *fileUploadSetClipboard, *fileUploadReplace, *fileUploadDeletInvaid)

	// Delete file
	case fileDeleteCmd.FullCommand():
		commands.DeleteFile(commandData, *fileDeleteName, *fileDeleteID)

	// Delete file (rm)
	case fileRmCmd.FullCommand():
		commands.DeleteFile(commandData, *fileRmName, *fileRmID)

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
		commands.Ping(commandData)

	// -- User commands
	// Login
	case loginCmd.FullCommand():
		commands.LoginCommand(commandData, *loginCmdUser)

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

			commands.SetupClient(commandData, host, *appCfgFile, *setupCmdIgnoreCert, *setupCmdServerOnly, *setupCmdRegister, *setupCmdNoLogin)
		}

	// -- Config commands
	// Config use
	case configUse.FullCommand():
		commands.ConfigUse(commandData, *configUseTarget, *configUseTargetValue)

	// Config view
	case configView.FullCommand():
		commands.ConfigView(commandData)

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

	}
}
