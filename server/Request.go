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
	"time"

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

	// Attributes
	EPAttribute Endpoint = "/attribute"
	//Tags
	EPAttributeTag = EPAttribute + "/tag"
	EPTagUpdate    = EPAttributeTag + "/update"
	EPTagDelete    = EPAttributeTag + "/delete"
	//Group
	EPAttributeGroup = EPAttribute + "/group"
	EPGroupUpdate    = EPAttributeGroup + "/update"
	EPGroupDelete    = EPAttributeGroup + "/delete"

	// Namespace
	EPNamespace       Endpoint = "/namespace"
	EPNamespaceCreate          = EPNamespace + "/create"
	EPNamespaceUpdate          = EPNamespace + "/update"
	EPNamespaceDelete          = EPNamespace + "/delete"
	EPNamespaceList            = EPNamespace + "s"
)

//Request a rest server request
type Request struct {
	RequestType   RequestType
	Endpoint      Endpoint
	Payload       interface{}
	Config        *models.Config
	Method        Method
	ContentType   ContentType
	Authorization *Authorization
	Headers       map[string]string
	BenchChan     chan time.Time
}

// FileListRequest contains file info (and a file)
type FileListRequest struct {
	FileID         uint                     `json:"fid"`
	Name           string                   `json:"name"`
	AllNamespaces  bool                     `json:"allns"`
	OptionalParams OptionalRequetsParameter `json:"opt"`
	Order          string                   `json:"order,omitempty"`
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
	All        bool                  `json:"all"`
	Attributes models.FileAttributes `json:"attributes"`
}

// UpdateAttributeRequest contains data to update a tag
type UpdateAttributeRequest struct {
	Name      string `json:"name"`
	NewName   string `json:"newname"`
	Namespace string `json:"namespace"`
}

//CredentialsRequest request containing credentials
type CredentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"pass"`
}

//NamespaceRequest namespace action request
type NamespaceRequest struct {
	Namespace string               `json:"ns"`
	NewName   string               `json:"newName,omitempty"`
	Type      models.NamespaceType `json:"nstype"`
}

// UploadRequest contains file info (and a file)
type UploadRequest struct {
	UploadType UploadType            `json:"type"`
	URL        string                `json:"url"`
	Name       string                `json:"name"`
	Public     bool                  `json:"public"`
	PublicName string                `json:"pbname"`
	Attributes models.FileAttributes `json:"attributes"`
}

//UploadType type of upload
type UploadType uint8

//Available upload types
const (
	FileUploadType UploadType = iota
	URLUploadType
)

//RequestType type of request
type RequestType uint8

//Request types
const (
	JSONRequestType RequestType = iota
	RawRequestType
)

//NewRequest creates a new post request
func NewRequest(endpoint Endpoint, payload interface{}, config *models.Config) *Request {
	return &Request{
		RequestType: JSONRequestType,
		Endpoint:    endpoint,
		Payload:     payload,
		Config:      config,
		Method:      POST,
		ContentType: JSONContentType,
	}
}

//WithRequestType use different request type
func (request *Request) WithRequestType(rType RequestType) *Request {
	request.RequestType = rType
	return request
}

//WithAuth with authorization
func (request *Request) WithAuth(a Authorization) *Request {
	request.Authorization = &a
	return request
}

//WithBenchCallback with bench
func (request *Request) WithBenchCallback(c chan time.Time) *Request {
	request.BenchChan = c
	return request
}

//WithContentType with contenttype
func (request *Request) WithContentType(ct ContentType) *Request {
	request.ContentType = ct
	return request
}

//WithHeader add header to request
func (request *Request) WithHeader(name string, value string) *Request {
	if request.Headers == nil {
		request.Headers = make(map[string]string)
	}

	request.Headers[name] = value
	return request
}

//BuildClient return client
func (request *Request) BuildClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: request.Config.Server.IgnoreCert,
			},
		},
	}
}

//DoHTTPRequest do plain http request
func (request *Request) DoHTTPRequest() (*http.Response, error) {
	client := request.BuildClient()

	//Build url
	u, err := url.Parse(request.Config.Server.URL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, string(request.Endpoint))

	var bytePayload []byte

	//Use correct payload
	if request.RequestType == JSONRequestType {
		//Encode data
		var err error
		bytePayload, err = json.Marshal(request.Payload)
		if err != nil {
			return nil, err
		}
	} else if request.RequestType == RawRequestType {
		bytePayload = (request.Payload).([]byte)
	}

	//bulid request
	req, _ := http.NewRequest("POST", u.String(), bytes.NewBuffer(bytePayload))

	//Set contenttype header
	req.Header.Set("Content-Type", string(request.ContentType))

	for headerKey, headerValue := range request.Headers {
		req.Header.Set(headerKey, headerValue)
	}

	//Set Authorization header
	if request.Authorization != nil {
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", string(request.Authorization.Type), request.Authorization.Palyoad))
	}

	return client.Do(req)
}

//Do a better request method
func (request Request) Do(retVar interface{}) (*RestRequestResponse, error) {
	resp, err := request.DoHTTPRequest()

	//Call bench callbac
	if request.BenchChan != nil {
		request.BenchChan <- time.Now()
	}

	if err != nil {
		return nil, err
	}

	var response *RestRequestResponse

	response = &RestRequestResponse{
		HTTPCode: resp.StatusCode,
		Headers:  &resp.Header,
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
