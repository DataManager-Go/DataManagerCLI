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

	"../models"
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
	//User
	EPPing     Endpoint = "/ping"
	File       Endpoint = "/file"
	UploadFile Endpoint = "/file/upload"
	DeleteFile Endpoint = "/file/delete"
	List       Endpoint = "/file/list"
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

// UploadStruct contains file info (and a file)
type UploadStruct struct {
	Data      []byte
	Namespace string
	Group     string
	Tag       string
}

// HandleStruct contains file info
type HandleStruct struct {
	Name      string
	Namespace string
	Group     string
	Tag       string
	Task      string
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

	//Read and validate headers
	statusStr := resp.Header.Get(HeaderStatus)
	statusMessage := resp.Header.Get(HeaderStatusMessage)

	if len(statusStr) == 0 {
		return nil, ErrInvalidResponseHeaders
	}
	statusInt, err := strconv.Atoi(statusStr)
	if err != nil || (statusInt > 1 || statusInt < 0) {
		return nil, ErrInvalidResponseHeaders
	}
	status := (ResponseStatus)(uint8(statusInt))

	response := &RestRequestResponse{
		HTTPCode: resp.StatusCode,
		Message:  statusMessage,
		Status:   status,
	}

	//Only fill retVar if response was successful
	if status == ResponseSuccess && retVar != nil {
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
