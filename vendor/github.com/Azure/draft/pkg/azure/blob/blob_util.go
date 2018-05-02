package blob

import (
	"net/url"
	"time"

	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
)

// GetAppendBlobURL returns an AppendBlobURL for the specified logFileURL.
func GetAppendBlobURL(logFileURL string) azblob.AppendBlobURL {
	po := azblob.PipelineOptions{}
	po.Retry = azblob.RetryOptions{
		Policy:   azblob.RetryPolicyExponential,
		MaxTries: 3,

		// Maximum time allowed for any HTTP request
		TryTimeout: time.Second * 10,

		// Retry delay between requests
		RetryDelay: time.Second * 3,

		// Maximum retry delay between requests
		MaxRetryDelay: time.Second * 3,
	}

	p := azblob.NewPipeline(azblob.NewAnonymousCredential(), po)
	u, _ := url.Parse(logFileURL)

	appendBlobURL := azblob.NewAppendBlobURL(*u, p)
	return appendBlobURL
}
