package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/jenkins-x/jx/v2/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/v2/pkg/config"
)

var (
	defaultBucketWriteTimeout = 20 * time.Second
)

// AKSBucketProvider the bucket provider for Azure
type AKSBucketProvider struct {
	Requirements *config.RequirementsConfig
}

// CreateNewBucketForCluster is not implemented
func (b *AKSBucketProvider) CreateNewBucketForCluster(_ string, _ string) (string, error) {
	return "", nil
}

// EnsureBucketIsCreated is not implemented
func (b *AKSBucketProvider) EnsureBucketIsCreated(_ string) error {
	return nil
}

func getAccessToken(resource string) (adal.Token, error) {

	msiEndpoint, err := adal.GetMSIEndpoint()
	if err != nil {
		return adal.Token{}, fmt.Errorf("failed to get endpoint for MSI: %v", err)
	}

	spToken, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
	if err != nil {
		return adal.Token{}, fmt.Errorf("failed to get service principal token from MSI: %v", err)
	}

	err = spToken.Refresh()
	if err != nil {
		return adal.Token{}, fmt.Errorf("failed to refresh service principal token, %w", err)
	}

	return spToken.Token(), nil
}

func getContainerURL(bucketURL string) (azblob.ContainerURL, error) {

	token, err := getAccessToken(azure.PublicCloud.ResourceIdentifiers.Storage)
	if err != nil {
		return azblob.ContainerURL{}, fmt.Errorf("failed to refresh service principal token, %w", err)
	}

	tokenCredential := azblob.NewTokenCredential(token.AccessToken, nil)
	u, err := url.Parse(bucketURL)
	if err != nil {
		return azblob.ContainerURL{}, fmt.Errorf("failed to parse container url, %w", err)
	}

	return azblob.NewContainerURL(*u, azblob.NewPipeline(tokenCredential, azblob.PipelineOptions{})), nil
}

// UploadFileToBucket is yet to be implemented for this provider
func (b *AKSBucketProvider) UploadFileToBucket(r io.Reader, outputName string, bucketURL string) (string, error) {

	containerURL, err := getContainerURL(bucketURL)

	if err != nil {
		return "", fmt.Errorf("failed to initialize containerURL, %w", err)
	}

	blobURL := containerURL.NewBlockBlobURL(outputName)

	ctx, _ := context.WithTimeout(context.Background(), defaultBucketWriteTimeout)
	_, err = azblob.UploadStreamToBlockBlob(ctx, r, blobURL, azblob.UploadStreamToBlockBlobOptions{})

	return blobURL.String(), nil
}

// DownloadFileFromBucket is yet to be implemented for this provider
func (b *AKSBucketProvider) DownloadFileFromBucket(_ string) (io.ReadCloser, error) {
	return nil, nil
}

// NewAKSBucketProvider create a new provider for AKS
func NewAKSBucketProvider(requirements *config.RequirementsConfig) buckets.Provider {
	return &AKSBucketProvider{
		Requirements: requirements,
	}
}
