package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/Yukaru-san/DataManager_Client/models"
)

//Method http request method
type Method string

//Requests
const (
	GET    Method = "GET"
	POST   Method = "POST"
	DELETE Method = "DELETE"
	PUT    Method = "PUT"
)

//ContentType contenttype header of request
type ContentType string

//Content types
const (
	JSONContentType ContentType = "application/json"
)

//PingRequest a ping request content
type PingRequest struct {
	Payload string
}

//Endpoint a remote url-path
type Endpoint string

//Remote endpoints
const (
	//Ping
	EPPing Endpoint = "/ping"

	//User
	EPUser     Endpoint = "/user"
	EPLogin    Endpoint = EPUser + "/login"
	EPRegister Endpoint = EPUser + "/register"

	//Files
	EPFile Endpoint = "/file"

	EPFileList    Endpoint = EPFile + "s"
	EPFileUpdate  Endpoint = EPFile + "/update"
	EPFileDelete  Endpoint = EPFile + "/delete"
	EPFileGet     Endpoint = EPFile + "/get"
	EPFilePublish Endpoint = EPFile + "/publish"

	//Upload
	EPFileUpload Endpoint = "/upload" + EPFile

	//Tags
	EPTag Endpoint = "/tag"

	//Update tags
	EPTagUpdate Endpoint = EPTag + "/update"
)

//Request a rest server request
type Request struct {
	Endpoint      Endpoint
	Payload       interface{}
	Config        *models.Config
	Method        Method
	ContentType   ContentType
	Authorization *Authorization
}

// FileListRequest contains file info (and a file)
type FileListRequest struct {
	FileID         uint                     `json:"fid"`
	Name           string                   `json:"name"`
	OptionalParams OptionalRequetsParameter `json:"opt"`
	Attributes     models.FileAttributes    `json:"attributes"`
}

//OptionalRequetsParameter optional request parameter
type OptionalRequetsParameter struct {
	Verbose uint8 `json:"verb"`
}

// FileRequest contains data to update a file
type FileRequest struct {
	FileID     uint                  `json:"fid"`
	Name       string                `json:"name,omitempty"`
	PublicName string                `json:"pubname,omitempty"`
	Updates    models.FileUpdateItem `json:"updates,omitempty"`
	Attributes models.FileAttributes `json:"attributes"`
}

// TagUpdateRequest contains data to update a tag
type TagUpdateRequest struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	NewName   string `json:"newname,omitempty"`
}

//CredentialsRequest request containing credentials
type CredentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"pass"`
}

// UploadRequest contains file info (and a file)
type UploadRequest struct {
	UploadType UploadType            `json:"type"`
	Data       string                `json:"data"`
	URL        string                `json:"url"`
	Sum        string                `json:"sum"`
	Name       string                `json:"name"`
	Public     bool                  `json:"public"`
	PublicName string                `json:"pbname"`
	FileType   string                `json:"ftype"`
	Attributes models.FileAttributes `json:"attributes"`
}

//UploadType type of upload
type UploadType uint8

//Available upload types
const (
	FileUploadType UploadType = iota
	URLUploadType
)

//NewRequest creates a new post request
func NewRequest(endpoint Endpoint, payload interface{}, config *models.Config) *Request {
	return &Request{
		Endpoint:    endpoint,
		Payload:     payload,
		Config:      config,
		Method:      POST,
		ContentType: JSONContentType,
	}
}

//WithAuth with authorization
func (request *Request) WithAuth(a Authorization) *Request {
	request.Authorization = &a
	return request
}

//DoHTTPRequest do plain http request
func (request *Request) DoHTTPRequest() (*http.Response, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: request.Config.Server.IgnoreCert,
			},
		},
	}

	//Build url
	u, err := url.Parse(request.Config.Server.URL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, string(request.Endpoint))

	//Encode data
	bda, err := json.Marshal(request.Payload)
	if err != nil {
		return nil, err
	}

	//bulid request
	req, _ := http.NewRequest("POST", u.String(), bytes.NewBuffer(bda))
	req.Header.Set("Content-Type", string(request.ContentType))

	//Set Authorization header
	if request.Authorization != nil {
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", string(request.Authorization.Type), request.Authorization.Palyoad))
	}

	return client.Do(req)
}

//Do a better request method
func (request Request) Do(retVar interface{}) (*RestRequestResponse, error) {
	resp, err := request.DoHTTPRequest()
	if err != nil {
		return nil, err
	}

	var response *RestRequestResponse

	if resp != nil {
		response = &RestRequestResponse{
			HTTPCode: resp.StatusCode,
			Headers:  &resp.Header,
		}
	}

	//Read and validate headers
	statusStr := resp.Header.Get(HeaderStatus)
	statusMessage := resp.Header.Get(HeaderStatusMessage)

	if len(statusStr) == 0 {
		return response, ErrInvalidResponseHeaders
	}
	statusInt, err := strconv.Atoi(statusStr)
	if err != nil || (statusInt > 1 || statusInt < 0) {
		return response, ErrInvalidResponseHeaders
	}

	response.Status = (ResponseStatus)(uint8(statusInt))
	response.Message = statusMessage

	//Only fill retVar if response was successful
	if response.Status == ResponseSuccess && retVar != nil {
		//Read response
		d, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		//Parse response into retVar
		err = json.Unmarshal(d, &retVar)
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}
