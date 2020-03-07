package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	gaw "github.com/JojiiOfficial/GoAw"
	"github.com/JojiiOfficial/configService"
	"github.com/Yukaru-san/DataManager_Client/models"
	"github.com/Yukaru-san/DataManager_Client/server"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

//LoginCommand login into the server
func LoginCommand(config *models.Config, usernameArg string, args ...bool) {
	if config.IsLoggedIn() && !*appYes && len(args) == 0 {
		i, _ := gaw.ConfirmInput("You are already logged in. Overwrite session? [y/n]> ", bufio.NewReader(os.Stdin))
		if !i {
			return
		}
	}

	username, pass := credentials(usernameArg, false, 0)

	login := server.CredentialsRequest{
		Password: pass,
		Username: username,
	}

	var response server.LoginResponse

	resp, err := server.NewRequest(server.EPLogin, login, config).Do(&response)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if resp.Status == server.ResponseError && resp.HTTPCode == 403 {
		fmt.Println(color.HiRedString("Failure"))
	} else if resp.Status == server.ResponseSuccess && len(response.Token) > 0 {
		config.User = struct {
			Username     string
			SessionToken string
		}{
			Username:     username,
			SessionToken: response.Token,
		}
		err := configService.Save(config, config.File)
		if err != nil {
			fmt.Println("Error saving config:", err.Error())
			return
		}
		fmt.Println(color.HiGreenString("Success!"), "\nLogged in as", username)
	} else {
		printResponseError(resp)
	}
}

//RegisterCommand create a new account
func RegisterCommand(config *models.Config) {
	username, pass := credentials("", true, 0)
	if len(username) == 0 || len(pass) == 0 {
		return
	}

	req := server.CredentialsRequest{
		Username: username,
		Password: pass,
	}

	resp, err := server.NewRequest(server.EPRegister, req, config).Do(nil)
	if err != nil {
		fmt.Println("Err", err.Error())
		return
	}

	if resp.Status == server.ResponseSuccess {
		fmt.Printf("User '%s' created %s!\n", username, color.HiGreenString("successfully"))
		y, _ := gaw.ConfirmInput("Do you want to login to this account? [y/n]> ", bufio.NewReader(os.Stdin))
		if y {
			LoginCommand(config, username, true)
		}
	} else {
		printResponseError(resp)
	}
}

func credentials(bUser string, repeat bool, index uint8) (string, string) {
	if index >= 3 {
		return "", ""
	}

	reader := bufio.NewReader(os.Stdin)
	var username string
	if len(bUser) > 0 {
		username = bUser
	} else {
		fmt.Print("Enter Username: ")
		username, _ = reader.ReadString('\n')
	}

	if len(username) > 30 {
		fmt.Println("Username too long!")
		return "", ""
	}

	var pass string
	enterPassMsg := "Enter Password: "
	count := 1

	if repeat {
		count = 2
	}

	for i := 0; i < count; i++ {
		fmt.Print(enterPassMsg)
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalln("Error:", err.Error())
			return "", ""
		}
		fmt.Println()
		lPass := strings.TrimSpace(string(bytePassword))

		if len(lPass) > 80 {
			fmt.Println("Your password is too long!")
			return credentials(username, repeat, index+1)
		}
		if len(lPass) < 7 {
			fmt.Println("Your password must have at least 7 characters!")
			return credentials(username, repeat, index+1)
		}

		if repeat && i == 1 && pass != lPass {
			fmt.Println("Passwords don't match!")
			return credentials(username, repeat, index+1)
		}

		pass = lPass
		enterPassMsg = "Enter Password again: "
	}

	return strings.TrimSpace(username), pass
}
