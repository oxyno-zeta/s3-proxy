package s3client

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

// NewS3Context New S3 Context
func NewS3Context(tgt *config.Target, logger *logrus.FieldLogger) (S3ContextInterface, error) {
	sessionConfig := &aws.Config{
		Region: aws.String(tgt.Bucket.Region),
	}
	// Load credentials if they exists
	if tgt.Bucket.Credentials != nil && tgt.Bucket.Credentials.AccessKey != nil && tgt.Bucket.Credentials.SecretKey != nil {
		sessionConfig.Credentials = credentials.NewStaticCredentials(tgt.Bucket.Credentials.AccessKey.Value, tgt.Bucket.Credentials.SecretKey.Value, "")
	}
	// Load custom endpoint if it exists
	if tgt.Bucket.S3Endpoint != "" {
		sessionConfig.Endpoint = aws.String(tgt.Bucket.S3Endpoint)
		sessionConfig.S3ForcePathStyle = aws.Bool(true)
	}
	sess, err := session.NewSession(sessionConfig)
	if err != nil {
		return nil, err
	}
	svcClient := s3.New(sess)
	return &s3Context{svcClient: svcClient, logger: logger, Target: tgt}, nil
}
