package models

import (
	authxmodels "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
)

// folderListingData Folder listing data for templating.
type FolderListingData struct {
	User       authxmodels.GenericUser
	Request    *LightSanitizedRequest
	BucketName string
	Name       string
	Entries    []*Entry
}

// errorData represents the structure used by error templating.
type ErrorData struct {
	Request *LightSanitizedRequest
	User    authxmodels.GenericUser
	Error   error
}

// targetListData represents the structure used by target list templating.
type TargetListData struct {
	Request *LightSanitizedRequest
	User    authxmodels.GenericUser
	Targets map[string]any
}

// streamFileHeaderData represents the structure used by stream file header templating.
type StreamFileHeaderData struct {
	Request    *LightSanitizedRequest
	User       authxmodels.GenericUser
	StreamFile *StreamInput
}

// putData represents the structure used by put templating.
type PutData struct {
	Request *LightSanitizedRequest
	User    authxmodels.GenericUser
	PutData *PutInput
}

// deleteData represents the structure used by delete templating.
type DeleteData struct {
	Request    *LightSanitizedRequest
	User       authxmodels.GenericUser
	DeleteData *DeleteInput
}
