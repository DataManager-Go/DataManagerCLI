package models

//NamespaceType type of namespace
type NamespaceType uint8

//Namespace types
const (
	UserNamespaceType NamespaceType = iota
	CustomNamespaceType
)
