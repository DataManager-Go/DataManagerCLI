package models

import "time"

//FileAttributes attributes for a file
type FileAttributes struct {
	Tags      []string `json:"tags"`
	Groups    []string `json:"groups"`
	Namespace string   `json:"ns"`
}

//FileResponseItem file item for file response
type FileResponseItem struct {
	ID           uint      `json:"id"`
	Size         int64     `json:"size"`
	CreationDate time.Time `json:"creation"`
	Name         string    `json:"name"`
}
