package models

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/JojiiOfficial/configService"
	"github.com/JojiiOfficial/gaw"
	"gopkg.in/yaml.v2"
)

// ...
const (
	DataDir           = ".dmanager"
	DefaultConfigFile = "config.yaml"
)

//Config Configuration structure
type Config struct {
	File string
	User struct {
		Username     string
		SessionToken string
	}

	Server  serverConfig
	Client  clientConfig
	Default defaultConfig
}

type serverConfig struct {
	URL            string `required:"true"`
	AlternativeURL string
	IgnoreCert     bool
}

type clientConfig struct {
	MinFilesToDisplay uint16 `required:"true"`
	AutoFilePreview   bool
	DefaultOrder      string
	DefaultDetails    int
	TrimNameAfter     int
}

type defaultConfig struct {
	Namespace string `default:"default"`
	Tags      []string
	Groups    []string
}

func getDefaultConfig() Config {
	return Config{
		Server: serverConfig{
			URL:        "http://localhost:9999",
			IgnoreCert: false,
		},
		Client: clientConfig{
			MinFilesToDisplay: 100,
			AutoFilePreview:   true,
			DefaultDetails:    0,
			DefaultOrder:      "created/r",
			TrimNameAfter:     20,
		},
		Default: defaultConfig{
			Namespace: "default",
		},
	}
}

//InitConfig inits the configfile
func InitConfig(defaultFile, file string) (*Config, error) {
	var needCreate bool
	var config Config

	if len(file) == 0 {
		file = defaultFile
		needCreate = true
	}

	//Check if config already exists
	_, err := os.Stat(file)
	needCreate = err != nil

	if needCreate {
		//Autocreate folder
		path, _ := filepath.Split(file)
		_, err := os.Stat(path)
		if err != nil {
			err = os.MkdirAll(path, 0770)
			if err != nil {
				return nil, err
			}
		}

		//Set config to default config
		config = getDefaultConfig()
	}

	//Create config file if not exists and fill it with the default values
	isDefault, err := configService.SetupConfig(&config, file, configService.NoChange)
	if err != nil {
		return nil, err
	}
	//Return if created but further steps are required
	if isDefault {
		if needCreate {
			return nil, nil
		}
	}

	//Load configuration
	if err = configService.Load(&config, file); err != nil {
		return nil, err
	}

	config.File = file
	return &config, nil
}

//Validate check the config
func (config *Config) Validate() error {
	//Put in your validation logic here
	return nil
}

//IsLoggedIn return true if sessiondata is available
func (config *Config) IsLoggedIn() bool {
	return len(config.User.Username) > 0 && len(config.User.SessionToken) == 64
}

// GetDefaultOrder returns the default order. If empty returns the default order
func (config *Config) GetDefaultOrder() string {
	if len(config.Client.DefaultOrder) > 0 {
		return config.Client.DefaultOrder
	}

	// Return default order
	return getDefaultConfig().Client.DefaultOrder
}

//GetPreviewURL gets preview URL
func (config *Config) GetPreviewURL(file string) string {
	//Use alternative url if available
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

	//Parse URL
	u, err := url.Parse(config.Server.URL)
	if err != nil {
		log.Fatalln("Server URL is not valid: ", err)
		return ""
	}

	//otherwise use default url and 'preview' folder
	u.Path = path.Join(u.Path, "preview", file)
	return u.String()
}

//GetDefaultConfigFile return path of default config
func GetDefaultConfigFile() string {
	return filepath.Join(getDataPath(), DefaultConfigFile)
}

func getDataPath() string {
	path := filepath.Join(gaw.GetHome(), DataDir)
	s, err := os.Stat(path)
	if err != nil {
		err = os.Mkdir(path, 0770)
		if err != nil {
			log.Fatalln(err.Error())
		}
	} else if s != nil && !s.IsDir() {
		log.Fatalln("DataPath-name already taken by a file!")
	}
	return path
}

//View view config
func (config Config) View(redactSecrets bool) string {
	//Redact secrets if desired
	if redactSecrets {
		config.User.SessionToken = "<redacted>"
	}

	//Create yaml
	ymlB, err := yaml.Marshal(config)
	if err != nil {
		return err.Error()
	}

	return string(ymlB)
}
