package s3man

import (
	"io"
	"os"
	"path"
	"strings"

	"github.com/alpacahq/gopaca/env"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Manager struct {
	session    *session.Session
	bucketName string
	namespace  string
}

// New creates a new instance of Manager using AWS environment variables.
func New() *Manager {
	keyID := env.GetVar("AWS_ACCESS_KEY_ID")
	secret := env.GetVar("AWS_SECRET_ACCESS_KEY")
	bucketName := env.GetVar("AWS_S3_BUCKETNAME")
	namespace := env.GetVar("AWS_S3_NAMESPACE")
	region := "us-east-1"

	sess, _ := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(keyID, secret, ""),
		Region:      &region,
	})

	if namespace != "" && strings.HasSuffix(namespace, "/") {
		namespace = namespace[:len(namespace)-1]
	}

	return &Manager{
		session:    sess,
		bucketName: bucketName,
		namespace:  namespace,
	}
}

// Exists returns true if the object exists in the S3 bucket
func (m *Manager) Exists(path string) (bool, error) {
	cli := s3.New(m.session)
	_, err := cli.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(m.bucketName),
		Key:    aws.String(m.namespace + path),
	})

	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Upload an object to S3
func (m *Manager) Upload(file io.ReadSeeker, path string) error {
	uploader := s3manager.NewUploader(m.session)

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(m.bucketName),
		Key:    aws.String(m.namespace + path),
		Body:   file,
	})

	return err
}

// DownloadInMemory downloads an S3 object into memory
func (m *Manager) DownloadInMemory(path string) ([]byte, error) {
	downloader := s3manager.NewDownloader(m.session)

	w := &aws.WriteAtBuffer{}
	_, err := downloader.Download(w, &s3.GetObjectInput{
		Bucket: aws.String(m.bucketName),
		Key:    aws.String(m.namespace + path),
	})

	return w.Bytes(), err
}

// DownloadDirectory downloads a S3 directory and write it to a local directory
func (m *Manager) DownloadDirectory(local, remote string) error {
	return m.getBucketObjects(local, &remote)
}

// DownloadBucket downloads a S3 bucket and write it to a local directory
func (m *Manager) DownloadBucket(local string) error {
	return m.getBucketObjects(local, nil)
}

func (m *Manager) getBucketObjects(local string, prefix *string) error {
	query := &s3.ListObjectsV2Input{
		Bucket: &m.bucketName,
		Prefix: prefix,
	}

	svc := s3.New(m.session)

	// Flag used to check if we need to go further
	truncatedListing := true

	for truncatedListing {
		resp, err := svc.ListObjectsV2(query)

		if err != nil {
			return err
		}

		// Get all files
		if err = m.getObjectsAll(resp, svc, local); err != nil {
			return err
		}

		// Set continuation token
		query.ContinuationToken = resp.NextContinuationToken
		truncatedListing = *resp.IsTruncated
	}

	return nil
}

func (m *Manager) getObjectsAll(bucketObjectsList *s3.ListObjectsV2Output, s3Client *s3.S3, local string) error {
	// Iterate through the files inside the bucket
	for _, key := range bucketObjectsList.Contents {
		destFilename := *key.Key

		if strings.HasSuffix(*key.Key, "/") {
			continue
		}

		if strings.Contains(*key.Key, "/") {
			var dirTree string

			s3FileFullPathList := strings.Split(*key.Key, "/")

			for _, dir := range s3FileFullPathList[:len(s3FileFullPathList)-1] {
				dirTree += "/" + dir
			}
			os.MkdirAll(local+"/"+dirTree, 0775)
		}

		out, err := s3Client.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(m.bucketName),
			Key:    key.Key,
		})

		if err != nil {
			return err
		}

		destFile, err := os.Create(path.Join(local, destFilename))
		if err != nil {
			return err
		}

		if _, err = io.Copy(destFile, out.Body); err != nil {
			return err
		}

		if err = out.Body.Close(); err != nil {
			return err
		}

		if err = destFile.Close(); err != nil {
			return err
		}
	}

	return nil
}
