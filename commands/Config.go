package commands

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	libdm "github.com/DataManager-Go/libdatamanager"
	dmConfig "github.com/DataManager-Go/libdatamanager/config"
	"github.com/JojiiOfficial/configService"
	"github.com/JojiiOfficial/gaw"
	"github.com/fatih/color"
)

// UseTargets targets for config use
var UseTargets = []string{"namespace", "tags", "groups"}

// ConfigUse command for config use
func ConfigUse(cData *CommandData, target string, values []string) {
	// Return if target not found
	if !gaw.IsInStringArray(target, UseTargets) {
		fmtError("Target not found")
		return
	}

	// Removing target
	if len(values) == 0 && target != "namespace" {
		//Remove desired target
		switch target {
		case UseTargets[1]:
			cData.Config.Default.Tags = []string{}
			fmt.Println("Removing tags")
		case UseTargets[2]:
			cData.Config.Default.Groups = []string{}
			fmt.Println("Removing groups")
		}
	} else {
		switch target {
		// Use namespace
		case UseTargets[0]:
			{
				if len(values) == 0 {
					values = []string{"default"}
				}
				fmt.Printf("Using namespace '%s'\n", values[0])
				cData.Config.Default.Namespace = values[0]
			}
		// Use tags
		case UseTargets[1]:
			{
				fmt.Printf("Using tags '%s'\n", strings.Join(values, ", "))
				cData.Config.Default.Tags = values
			}
		// Use Groups
		case UseTargets[2]:
			{
				fmt.Printf("Using groups '%s'\n", strings.Join(values, ", "))
				cData.Config.Default.Groups = values
			}
		default:
			fmt.Printf("Target not found")
			return
		}
	}

	// Save config
	err := configService.Save(cData.Config, cData.Config.File)
	if err != nil {
		fmt.Println("Error saving config:", err.Error())
	} else {
		fmt.Printf("Config saved %s\n", color.HiGreenString("successfully"))
	}
	return
}

// ConfigView view config
func ConfigView(cData *CommandData, sessionBase64 bool) {
	if len(cData.Config.User.SessionToken) == 0 && cData.NoRedaction {
		s, err := cData.Config.GetToken()
		if err == nil {
			if sessionBase64 {
				enc := base64.RawStdEncoding.EncodeToString([]byte(s))
				cData.Config.User.SessionToken = enc
			} else {
				cData.Config.User.SessionToken = s
			}
		}
	}

	if !cData.OutputJSON {
		// Print human output
		fmt.Println(cData.Config.View(!cData.NoRedaction))
	} else {
		// Redact secrets
		if !cData.NoRedaction {
			cData.Config.User.SessionToken = "<redacted>"
		}

		// Make json
		b, err := json.Marshal(cData.Config)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Print output
		fmt.Println(string(b))
	}
}

// SetupClient sets up client config
func SetupClient(cData *CommandData, host, configFile string, ignoreCert, serverOnly, register, noLogin bool, token, username string) {
	if len(token)*len(username) == 0 {
		fmt.Println("Either --user or --token is missing")
		return
	}

	// Confirm creating a config anyway
	if cData.Config != nil && !cData.Config.IsDefault() && !cData.Yes {
		y, _ := gaw.ConfirmInput("There is already a config. Do you want to overwrite it? [y/n]> ", bufio.NewReader(os.Stdin))
		if !y {
			return
		}
	}

	// Load config
	if cData.Config == nil {
		var err error
		cData.Config, err = dmConfig.InitConfig(dmConfig.GetDefaultConfigFile(), configFile)
		if err != nil {
			printError("loading config", err.Error())
			return
		}
	}

	u := bulidURL(host)

	// Check host and verify response
	if err := checkHost(u.String(), ignoreCert); err != nil {
		printError("checking host", err.Error())
		return
	}

	fmt.Printf("%s connected to server\n", color.HiGreenString("Succesfully"))

	// Set new config values
	cData.Config.Server.URL = u.String()
	cData.Config.Server.IgnoreCert = ignoreCert

	err := configService.Save(cData.Config, cData.Config.File)
	if err != nil {
		printError("saving config", err.Error())
		return
	}

	// If severonly mode is requested, stop here
	if serverOnly {
		return
	}

	// Initialize server connection library instance
	// ignore token error since user might not
	// be logged in after setup process
	config, _ := cData.Config.ToRequestConfig()
	cData.LibDM = libdm.NewLibDM(config)

	// Insert user directly if token and user is set
	if len(token) > 0 && len(username) > 0 {
		// Decode token
		dec, err := base64.RawStdEncoding.DecodeString(token)
		if err != nil {
			fmt.Println(err)
			return
		}

		token = string(dec)
		cData.Config.InsertUser(username, token)
		cData.Config.Save()
		return
	}

	// In register mode, don't login
	if register {
		noLogin = true
	}

	// if not noLogin, login
	if !noLogin {
		fmt.Println("Login")
		LoginCommand(cData, "")
		return
	}

	if register {
		fmt.Println("Create an account")
		RegisterCommand(cData)
	}
}

func bulidURL(host string) *url.URL {
	u, err := url.Parse(host)
	if err != nil {
		log.Fatal(err)
	}

	if u.Scheme == "" {
		u.Scheme = "https"
	}

	// Validate scheme
	if !gaw.IsInStringArray(u.Scheme, []string{"http", "https"}) {
		log.Fatalf("Invalid scheme '%s'. Use http or https\n", u.Scheme)
	}

	return u
}

func checkHost(host string, ignoreCert bool) error {
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: ignoreCert,
			},
		},
	}

	resp, err := client.Get(host)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("Invalid responsecode")
	}
	return nil
}
