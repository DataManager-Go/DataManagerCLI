package server

//Authorization the authorization params for a server request
type Authorization struct {
	Type    AuthorizationType
	Palyoad string
}

//AuthorizationType authorization type
type AuthorizationType string

//Authorizanion types
const (
	Bearer AuthorizationType = "Bearer"
	Basic  AuthorizationType = "Basic"
)
