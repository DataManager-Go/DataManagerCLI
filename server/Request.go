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

	//Files
	EPFile       Endpoint = "/file"
	EPFileList   Endpoint = EPFile + "/list"
	EPFileUpload Endpoint = EPFile + "/upload"

	//Update file
	EPFileUpdate Endpoint = "/file/update"
	EPFileDelete Endpoint = EPFileUpdate + "/delete"
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

// FileRequest contains file info (and a file)
type FileRequest struct {
	FileID     int                   `json:"fid"`
	Name       string                `json:"name"`
	Attributes models.FileAttributes `json:"attributes"`
}

// FileUpdateRequest contains data to update a file
type FileUpdateRequest struct {
	FileID     int                   `json:"fid"`
	Name       string                `json:"name,omitempty"`
	Attributes models.FileAttributes `json:"attributes"`
}

// UploadStruct contains file info (and a file)
type UploadStruct struct {
	Data       []byte                `json:"data"`
	Sum        string                `json:"sum"`
	Name       string                `json:"name"`
	Attributes models.FileAttributes `json:"attributes"`
}

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

//Do a better request method
func (request Request) Do(retVar interface{}) (*RestRequestResponse, error) {
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	var response *RestRequestResponse

	if resp != nil {
		response = &RestRequestResponse{
			HTTPCode: resp.StatusCode,
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
