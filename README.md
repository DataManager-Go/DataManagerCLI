# DataManagerClient
This is the client for the [DataManagerServer](https://github.com/JojiiOfficial/DataManagerServer). It supports uploading, downloading, editing, deleting, moving and en/decrypting files.

# Screenshots
## Listview
![Listview](https://files.jojii.de/preview/raw/PZtqdgIdpNGJkE3v8FgQU0rZp)

## Treeview
![Treeview](https://files.jojii.de/preview/raw/4uZP3dmTlKXihymZ5tTPjlneS)

# Installation

### Arch linux
You can find the datamanager cli client in the AUR: `datamanager-cli-git`.

### Compiling

```go
go build -o main && sudo mv main /usr/local/bin/manager
```
(Go 1.11+ is required)

# Setup
Run `manager setup <host>` to create a configuration file and login.<br>
Alternatively you can use `manager setup <host> --register` to create an account instead of loggin in.<br>
If you want to create a config and don't want to login/register at all, run `manager setup <host> --no-login`<br>
If Your server has no valid SSL certificate, but you want to use it anyway (not recommended), use the `--Ignore-cert` flag.

# Register/Login
Use `manager register` to create an account. The `allowregistration` must be set to true in the server config.<br>
Use `manager login` to login into your account

# Config

#### Client
`autofilepreview` Preview files using the default application. If you turn it off you will see the file content in the terminal
`defaultorder` The default order for listing files. (id, name, size, pubname, created, namespace). Add '/r' at the end to reverse the order<br>
`defaultdetails` The depth of details if no --details flag was set<br>
`trimnameafter` Trims filename after n chars and append a `...` to the end of the filename

#### Default
`namespace` The default namespace to use<br>
`tags` Specify tags to use as default for uploading filetags<br>
`groups` Specify groups to use as default for uploading filegroups<br>

# Usage
```bash
manager [<flags>] <command> [<args> ...]
```

Tipp: Run `manager --help-man | man -l -` to view the manpage of manager<br>

### Autocompletion
#### Bash
```bash
eval "$(manager --completion-script-bash)"
```
#### Zsh
```zsh
eval "$(manager --completion-script-zsh)"
```

### Keyring
A keyring is a secure storage for passwords, keys and token. This app can and should use one. [This](https://github.com/zalando/go-keyring#dependencies) is required in order to use a keyring.


### Keystore
The keystore is a local folder containing all of your keys and a sqlite database with the keys assigned to the files. You can use a 
custom directory to store them secure (eg using an encrypted vault). Have in mind, that all of those keys are stored unencrypted, so
watch for it's access permissions.<br>
To use it run "manager keystore create <path>". Your keys will be saved in this directory automatically
`manager keystore --help` shows you a list with available commands.

### Examples

#### User
- Setup `manager setup <serverURL>` // create a new config and login. Use --register if you want to create an account instead of logging in
- Register `manager register`
- Login `manager login`

#### Files
- Upload and share your .bashrc `manager upload -t dotfile -g myLinuxGroup --public ~/.bashrc`
- Upload and encrypt your .bashrc `manager upload ~/.bashrc --encrypt aes -r 32/24/16`
- Upload and your home directory compressed `manager upload ~/ --compress`
- List files `manager ls`
- List files having the a tag called 'dotfile' `manager ls -t dotfile`
- Delete file by ID `manager file rm 123`
- Delete file by Name  `manager file rm aUniqueName.go`
- Delete all files in namespace `manager file rm % -ay`
- Edit a file `manager file edit 123`
- Add tags to a file `manager file update --add-tags t1,t2`
- Publish a file `manager publish <fileID>`
- UnPublish a file `manager unpublish <fileID>`

#### Namespace
- List all your namespaces `manager namespaces`
- Create a namespace `manager namespace create <name>`
- Delete a namespace `manager namespace delete <name>`
- Download all files insisde a namespace `manager namespace download <name>`

### Nice to have
Here is a list with useful facts abouth this system:
- All file mods (encryption/decryption, compression, archiving) are hooked while streaming, so there is no extra time waiting for them
- Filenames can be wildcarded using `%`
- You can upload all files in a directory without archiving using `--no-archive`
- If you didn't install the client from a repository, you can view the manpage using `manager --help-man | /usr/bin/man -l -`
- Many subcommands have aliases. For instance `file -> f`, `download -> dl`, `edit -> e`, `update -> u`
- Use `--set-clip` to copy the URL of a published file directly into your clipboard
