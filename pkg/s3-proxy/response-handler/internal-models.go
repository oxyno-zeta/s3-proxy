package responsehandler

// bucketListingData Bucket listing data for templating.
type bucketListingData struct {
	Entries    []*Entry
	BucketName string
	Name       string
	Path       string
}
