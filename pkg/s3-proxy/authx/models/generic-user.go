package models

// Generic user interface used to know specific type of user
type GenericUser interface {
	// Get type of user
	GetType() string
}
