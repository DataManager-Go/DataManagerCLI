package server

import "github.com/Yukaru-san/DataManager_Client/models"

//ResponseStatus the status of response
type ResponseStatus uint8

const (
	//ResponseError if there was an error
	ResponseError ResponseStatus = 0
	//ResponseSuccess if the response is successful
	ResponseSuccess ResponseStatus = 1
)

const (
	//HeaderStatus headername for status in response
	HeaderStatus string = "X-Response-Status"
	//HeaderStatusMessage headername for status in response
	HeaderStatusMessage string = "X-Response-Message"
)

//RestRequestResponse the response of a rest call
type RestRequestResponse struct {
	HTTPCode int
	Status   ResponseStatus
	Message  string
}

//StringResponse response containing only one string
type StringResponse struct {
	String string `json:"content"`
}

//FileListResponse response for listing files
type FileListResponse struct {
	Files []models.File
}

//UploadResponse response for uploading file
type UploadResponse struct {
	FileID uint
}
