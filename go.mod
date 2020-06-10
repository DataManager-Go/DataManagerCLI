module github.com/DataManager-Go/DataManagerCLI

go 1.14

replace github.com/gosuri/uiprogress v0.0.1 => github.com/JojiiOfficial/uiprogress v1.0.5
replace github.com/DataManager-Go/libdatamanager => /home/jojii/programming/go/src/libdatamanager

require (
	github.com/DataManager-Go/libdatamanager v1.2.4
	github.com/DataManager-Go/libdatamanager/config v0.0.0-20200421144809-c51b64037a89
	github.com/JojiiOfficial/configService v0.0.0-20200219132202-6e71512e2e28
	github.com/JojiiOfficial/gaw v1.2.5
	github.com/JojiiOfficial/shred v1.2.1
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/atotto/clipboard v0.1.2
	github.com/danieljoos/wincred v1.1.0 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.9.0
	github.com/gosuri/uiprogress v0.0.1
	github.com/kyokomi/emoji v2.2.4+incompatible
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/pkg/errors v0.9.1
	github.com/sbani/go-humanizer v0.3.1
	github.com/zalando/go-keyring v0.0.0-20200121091418-667557018717
	golang.org/x/crypto v0.0.0-20200604202706-70a84ac30bf9
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/benweidig/cli-table.v2 v2.0.0-20180519085552-8b9fa48fb374
	gopkg.in/yaml.v2 v2.3.0 // indirect
)
