package buckets

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	jenkinsv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/v2/pkg/cloud"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
)

// CreateBucketURL creates a go-cloud URL to a bucket
func CreateBucketURL(name string, kind string, settings *jenkinsv1.TeamSettings) (string, error) {
	if kind == "" {
		provider := settings.KubeProvider
		if provider == "" {
			return "", fmt.Errorf("No bucket kind provided nor is a kubernetes provider configured for this team so it could not be defaulted")
		}
		kind = KubeProviderToBucketScheme(provider)
		if kind == "" {
			return "", fmt.Errorf("No bucket kind is associated with kubernetes provider %s", provider)
		}
	}
	return kind + "://" + name, nil
}

// KubeProviderToBucketScheme returns the bucket scheme for the cloud provider
func KubeProviderToBucketScheme(provider string) string {
	switch provider {
	case cloud.AKS:
		return "azblob"
	case cloud.AWS, cloud.EKS:
		return "s3"
	case cloud.GKE:
		return "gs"
	default:
		return ""
	}
}

// ReadURL reads the given URL from either a http/https endpoint or a bucket URL path.
// if specified the httpFn is a function which can append the user/password or token and/or add a header with the token if using a git provider
func ReadURL(urlText string, timeout time.Duration, httpFn func(urlString string) (string, func(*http.Request), error)) (io.ReadCloser, error) {
	u, err := url.Parse(urlText)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse URL %s", urlText)
	}
	var headerFunc func(*http.Request)
	switch u.Scheme {
	case "http", "https":
		if httpFn != nil {
			urlText, headerFunc, err = httpFn(urlText)
			if err != nil {
				return nil, err
			}
		}
		return ReadHTTPURL(urlText, headerFunc, timeout)
	default:
		return ReadBucketURL(u, timeout)
	}
}

// ReadHTTPURL reads the HTTP based URL, modifying the headers as needed, and returns the data or returning an error if a 2xx status is not returned
func ReadHTTPURL(u string, headerFunc func(*http.Request), timeout time.Duration) (io.ReadCloser, error) {
	httpClient := util.GetClientWithTimeout(timeout)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	headerFunc(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to invoke GET on %s", u)
	}
	stream := resp.Body

	if resp.StatusCode >= 400 {
		_ = stream.Close()
		return nil, fmt.Errorf("status %s when performing GET on %s", resp.Status, u)
	}
	return stream, nil
}

// ReadBucketURL reads the content of a bucket URL of the for 's3://bucketName/foo/bar/whatnot.txt?param=123'
// where any of the query arguments are applied to the underlying Bucket URL and the path is extracted and resolved
// within the bucket
func ReadBucketURL(u *url.URL, timeout time.Duration) (io.ReadCloser, error) {
	bucketURL, key := SplitBucketURL(u)

	ctx, _ := context.WithTimeout(context.Background(), timeout)
	bucket, err := blob.Open(ctx, bucketURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open bucket %s", bucketURL)
	}
	data, err := bucket.NewReader(ctx, key, nil)
	if err != nil {
		return data, errors.Wrapf(err, "failed to read key %s in bucket %s", key, bucketURL)
	}
	return data, nil
}

// WriteBucketURL writes the data to a bucket URL of the for 's3://bucketName/foo/bar/whatnot.txt?param=123'
// with the given timeout
func WriteBucketURL(u *url.URL, data io.Reader, timeout time.Duration) error {
	bucketURL, key := SplitBucketURL(u)
	return WriteBucket(bucketURL, key, data, timeout)
}

// WriteBucket writes the data to a bucket URL and key of the for 's3://bucketName' and key 'foo/bar/whatnot.txt'
// with the given timeout
func WriteBucket(bucketURL string, key string, reader io.Reader, timeout time.Duration) (err error) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	bucket, err := blob.Open(ctx, bucketURL)
	if err != nil {
		return errors.Wrapf(err, "failed to open bucket %s", bucketURL)
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return errors.Wrapf(err, "failed to read data for key %s in bucket %s", key, bucketURL)
	}
	err = bucket.WriteAll(ctx, key, data, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to write key %s in bucket %s", key, bucketURL)
	}
	return nil
}

// SplitBucketURL splits the full bucket URL into the URL to open the bucket and the file name to refer to
// within the bucket
func SplitBucketURL(u *url.URL) (string, string) {
	u2 := *u
	u2.Path = ""
	return u2.String(), strings.TrimPrefix(u.Path, "/")
}
