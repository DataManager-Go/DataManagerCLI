package server

import "errors"

var (
	//ErrInvalidResponseHeaders error on missing or malformed response headers
	ErrInvalidResponseHeaders = errors.New("Invalid response headers")
	//ErrInvalidAuthorizationMethod error if authorization method is not implemented
	ErrInvalidAuthorizationMethod = errors.New("Invalid request authorization method")
)
