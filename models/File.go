package models

//FileAttributes attributes for a file
type FileAttributes struct {
	Tags      []string `json:"tags"`
	Groups    []string `json:"groups"`
	Namespace string   `json:"ns"`
}

//File a file
type File struct {
	ID         int64 `json:"id"`
	Name       string
	Attributes FileAttributes
}
