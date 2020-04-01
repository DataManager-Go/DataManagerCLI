# DataManager_Client
This is the client for the [DataManagerServer](https://github.com/JojiiOfficial/DataManagerServer). It supports uploading, downloading, editing, deleting, moving and en/decrypting files.

# Installation
Go 1.11+ is required
```go
go mod download && go build -o main && sudo mv main /usr/local/bin/manager
```

# Setup
Run `manager ping` once to create a config file in `~/.dmanager/`. Change the 'server.url' to your DataManager server url.

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
