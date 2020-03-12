package commands

import "github.com/Yukaru-san/DataManager_Client/models"

//CommandData data for commands
type CommandData struct {
	Config         *models.Config
	FileAttributes models.FileAttributes
	Namespace      string
	Details        uint8
	All            bool
	AllNamespaces  bool
	NoRedaction    bool
	OutputJSON     bool
	Yes            bool
	Force          bool
}
