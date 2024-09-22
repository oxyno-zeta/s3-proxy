package webhook

const (
	GETAction    = "GET"
	HEADAction   = "HEAD"
	PUTAction    = "PUT"
	DELETEAction = "DELETE"
)

type HookBody struct {
	InputMetadata  interface{}     `json:"inputMetadata,omitempty"`
	OutputMetadata interface{}     `json:"outputMetadata,omitempty"`
	Target         *TargetHookBody `json:"target"`
	Action         string          `json:"action"`
	RequestPath    string          `json:"requestPath"`
}

type PutInputMetadataHookBody struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	ContentSize int64  `json:"contentSize"`
}

type GetInputMetadataHookBody struct {
	IfModifiedSince   string `json:"ifModifiedSince"`
	IfMatch           string `json:"ifMatch"`
	IfNoneMatch       string `json:"ifNoneMatch"`
	IfUnmodifiedSince string `json:"ifUnmodifiedSince"`
	Range             string `json:"range"`
}

type HeadInputMetadataHookBody struct {
	IfModifiedSince   string `json:"ifModifiedSince"`
	IfMatch           string `json:"ifMatch"`
	IfNoneMatch       string `json:"ifNoneMatch"`
	IfUnmodifiedSince string `json:"ifUnmodifiedSince"`
}

type OutputMetadataHookBody struct {
	Bucket     string `json:"bucket"`
	Region     string `json:"region"`
	S3Endpoint string `json:"s3Endpoint,omitempty"`
	Key        string `json:"key"`
}

type TargetHookBody struct {
	Name string `json:"name"`
}

type BucketHookBody struct {
	Name       string `json:"name"`
	Region     string `json:"region"`
	S3Endpoint string `json:"s3Endpoint"`
	Key        string `json:"key"`
}
