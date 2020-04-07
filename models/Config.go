package models

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/configService"
	"github.com/JojiiOfficial/gaw"
	"github.com/denisbrodbeck/machineid"
	"github.com/fatih/color"
	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v2"
)

// ...
const (
	// File constants
	DataDir           = ".dmanager"
	DefaultConfigFile = "config.yaml"

	// Keyring constants
	DefaultKeyring     = "login"
	KeyringServiceName = "DataManagerCLI"
)

// Config Configuration structure
type Config struct {
	File      string
	MachineID string
	User      userConfig

	Server  serverConfig
	Client  clientConfig
	Default defaultConfig
}

type userConfig struct {
	Username       string
	SessionToken   string
	DisableKeyring bool
	Keyring        string
}

type serverConfig struct {
	URL            string `required:"true"`
	AlternativeURL string
	IgnoreCert     bool
}

type clientConfig struct {
	MinFilesToDisplay uint16 `required:"true"`
	AutoFilePreview   bool
	TrimNameAfter     int
	Defaults          clientDefaults
}

type clientDefaults struct {
	DefaultOrder   string
	DefaultDetails int
}

type defaultConfig struct {
	Namespace string `default:"default"`
	Tags      []string
	Groups    []string
}

// GetDefaultConfigFile return path of default config
func GetDefaultConfigFile() string {
	return filepath.Join(getDataPath(), DefaultConfigFile)
}

func getDefaultConfig() Config {
	return Config{
		MachineID: GenMachineID(),
		Server: serverConfig{
			URL:        "http://localhost:9999",
			IgnoreCert: false,
		},
		User: userConfig{
			DisableKeyring: false,
			Keyring:        DefaultKeyring,
		},
		Client: clientConfig{
			MinFilesToDisplay: 100,
			AutoFilePreview:   true,
			Defaults: clientDefaults{
				DefaultDetails: 0,
				DefaultOrder:   "created/r",
			},
			TrimNameAfter: 20,
		},
		Default: defaultConfig{
			Namespace: "default",
		},
	}
}

// InitConfig inits the configfile
func InitConfig(defaultFile, file string) (*Config, error) {
	var needCreate bool
	var config Config

	if len(file) == 0 {
		file = defaultFile
		needCreate = true
	}

	// Check if config already exists
	_, err := os.Stat(file)
	needCreate = err != nil

	if needCreate {
		// Autocreate folder
		path, _ := filepath.Split(file)
		_, err := os.Stat(path)
		if err != nil {
			err = os.MkdirAll(path, 0700)
			if err != nil {
				return nil, err
			}
		}

		// Set config to default config
		config = getDefaultConfig()
		config.File = file
	}

	// Create config file if not exists and fill it with the default values
	isDefault, err := configService.SetupConfig(&config, file, configService.NoChange)
	if err != nil {
		return nil, err
	}

	// Return if created but further steps are required
	if isDefault {
		if needCreate {
			return nil, nil
		}
	}

	// Load configuration
	if err = configService.Load(&config, file); err != nil {
		return nil, err
	}

	config.File = file
	config.SetMachineID()

	return &config, nil
}

// SetMachineID sets machineID if empty
func (config *Config) SetMachineID() {
	if len(config.MachineID) == 0 {
		config.MachineID = GenMachineID()
		configService.Save(config, config.File)
	}
}

// Validate check the config
func (config *Config) Validate() error {
	// Put in your validation logic here
	return nil
}

// GetMachineID returns the machineID
func (config *Config) GetMachineID() string {
	// Gen new MachineID if empty
	if len(config.MachineID) == 0 {
		config.SetMachineID()
	}

	// Check length of machineID
	if len(config.MachineID) > 100 {
		fmt.Println("Warning: MachineID too big")
		return ""
	}

	return config.MachineID
}

// IsLoggedIn return true if sessiondata is available
func (config *Config) IsLoggedIn() bool {
	if len(config.User.Username) == 0 {
		return false
	}

	token, err := keyring.Get(KeyringServiceName, config.User.Username)

	// If no keyring was found, use unencrypted token
	if config.User.DisableKeyring || err != nil {
		token = config.User.SessionToken
	}

	return IsTokenValid(token)
}

// IsTokenValid return true if given token is
// a vaild session token
func IsTokenValid(token string) bool {
	return len(token) == 64
}

// GetKeyring returns the keyring to use
func (config *Config) GetKeyring() string {
	if len(config.User.Keyring) == 0 {
		return DefaultKeyring
	}

	return config.User.Keyring
}

// GetDefaultOrder returns the default order.
// If empty returns the default order
func (config *Config) GetDefaultOrder() string {
	if len(config.Client.Defaults.DefaultOrder) > 0 {
		return config.Client.Defaults.DefaultOrder
	}

	// Return default order
	return getDefaultConfig().Client.Defaults.DefaultOrder
}

// GetPreviewURL gets preview URL
func (config *Config) GetPreviewURL(file string) string {
	// Use alternative url if available
	if len(config.Server.AlternativeURL) != 0 {
		//Parse URL
		u, err := url.Parse(config.Server.AlternativeURL)
		if err != nil {
			fmt.Println("Server alternative URL is not valid: ", err)
			return ""
		}
		//Set new path
		u.Path = path.Join(u.Path, file)
		return u.String()
	}

	// Parse URL
	u, err := url.Parse(config.Server.URL)
	if err != nil {
		log.Fatalln("Server URL is not valid: ", err)
		return ""
	}

	// otherwise use default url and 'preview' folder
	u.Path = path.Join(u.Path, "preview", file)
	return u.String()
}

// View view config
func (config Config) View(redactSecrets bool) string {
	// React secrets if desired
	if redactSecrets {
		config.User.SessionToken = "<redacted>"
	}

	// Create yaml
	ymlB, err := yaml.Marshal(config)
	if err != nil {
		return err.Error()
	}

	return string(ymlB)
}

// InsertUser insert a new user
func (config *Config) InsertUser(user, token string) {
	config.User.Username = user
	config.MustSetToken(token)
}

// SetToken sets token for client
// Tries to save token in a keyring, if not supported
// save it unencrypted
func (config *Config) SetToken(token string) error {
	// Save to keyring. Exit return on success
	if err := keyring.Set(KeyringServiceName, config.User.Username, token); err == nil {
		return nil
	}

	fmt.Printf("Your platform doesn't have support for a keyring. Refer to https://github.com/DataManager-Go/DataManagerCLI#keyring\n--> !!! Your token will be saved %s !!! <--\n", color.HiRedString("UNENCRYPTED"))

	// Save sessiontoken in config unencrypted
	config.User.SessionToken = token
	return configService.Save(config, config.File)
}

// MustSetToken fatals on error
func (config *Config) MustSetToken(token string) {
	if err := config.SetToken(token); err != nil {
		log.Fatal(err)
	}
}

// GetToken returns user token
func (config *Config) GetToken() string {
	token, err := keyring.Get(KeyringServiceName, config.User.Username)

	if config.User.DisableKeyring || err != nil {
		return config.User.SessionToken
	}

	return token
}

// IsDefault returns true if config is equal to the default config
func (config Config) IsDefault() bool {
	defaultConfig := getDefaultConfig()
	return config.Client == defaultConfig.Client &&
		config.User == defaultConfig.User &&
		config.Server.IgnoreCert == defaultConfig.Server.IgnoreCert &&
		config.Server.AlternativeURL == config.Server.AlternativeURL
}

// ToRequestConfig create a libdm requestconfig from given cli client config
func (config Config) ToRequestConfig() *libdm.RequestConfig {
	return &libdm.RequestConfig{
		MachineID:    config.GetMachineID(),
		URL:          config.Server.URL,
		IgnoreCert:   config.Server.IgnoreCert,
		SessionToken: config.GetToken(),
		Username:     config.User.Username,
	}
}

// GenMachineID detect the machineID.
// If not detected return random string
func GenMachineID() string {
	username := getPseudoUsername()

	// Protect with username to allow multiple user
	// on a system using the same manager username
	id, err := machineid.ProtectedID(username)
	if err == nil {
		return id
	}

	// If not detected reaturn random string
	return gaw.RandString(60)
}

func getPseudoUsername() string {
	var username string
	user, err := user.Current()
	if err != nil {
		username = gaw.RandString(10)
	} else {
		username = user.Username
	}

	return username
}

func getDataPath() string {
	path := filepath.Join(gaw.GetHome(), DataDir)
	s, err := os.Stat(path)
	if err != nil {
		err = os.Mkdir(path, 0700)
		if err != nil {
			log.Fatalln(err.Error())
		}
	} else if s != nil && !s.IsDir() {
		log.Fatalln("DataPath-name already taken by a file!")
	}
	return path
}
