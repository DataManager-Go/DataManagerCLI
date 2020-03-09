package models

import (
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/JojiiOfficial/configService"
	"github.com/JojiiOfficial/gaw"
	"github.com/Yukaru-san/DataManager_Client/constants"
	"gopkg.in/yaml.v2"
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
	URL        string `required:"true"`
	IgnoreCert bool
}

type clientConfig struct {
	MinFilesToDisplay uint16 `required:"true"`
	AutoFilePreview   bool
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
			MinFilesToDisplay: 10,
			AutoFilePreview:   true,
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

//GetDefaultConfig return path of default config
func GetDefaultConfig() string {
	return path.Join(getDataPath(), constants.DefaultConfigFile)
}

func getDataPath() string {
	path := path.Join(gaw.GetHome(), constants.DataDir)
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
