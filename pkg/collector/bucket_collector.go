package collector

import (
	"context"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
	"io/ioutil"
	"path/filepath"
	"time"
)

// BucketCollector stores the state for the git collector
type BucketCollector struct {
	Timeout time.Duration

	bucketURL  string
	bucket     *blob.Bucket
	classifier string
}

// NewBucketCollector creates a new git based collector
func NewBucketCollector(bucketURL string, bucket *blob.Bucket, classifier string) (Collector, error) {
	return &BucketCollector{
		Timeout:    time.Second * 20,
		bucketURL:  bucketURL,
		bucket:     bucket,
		classifier: classifier,
	}, nil
}

// CollectFiles collects files and returns the URLs
func (c *BucketCollector) CollectFiles(patterns []string, outputPath string, basedir string) ([]string, error) {
	urls := []string{}
	bucket := c.bucket

	ctx := c.createContext()
	for _, p := range patterns {
		names, err := filepath.Glob(p)
		if err != nil {
			return urls, errors.Wrapf(err, "failed to evaluate glob pattern '%s'", p)
		}
		for _, name := range names {
			toName := name
			if basedir != "" {
				toName, err = filepath.Rel(basedir, name)
				if err != nil {
					return urls, errors.Wrapf(err, "failed to remove basedir %s from %s", basedir, name)
				}
			}
			data, err := ioutil.ReadFile(name)
			if err != nil {
				return urls, errors.Wrapf(err, "failed to read file %s", name)
			}
			opts := &blob.WriterOptions{
				ContentType: util.ContentTypeForFileName(name),
				Metadata: map[string]string{
					"classification": c.classifier,
				},
			}
			err = bucket.WriteAll(ctx, toName, data, opts)
			if err != nil {
				return urls, errors.Wrapf(err, "failed to write to bucket %s", toName)
			}

			u := util.UrlJoin(c.bucketURL, toName)
			urls = append(urls, u)
		}
	}
	return urls, nil
}

// CollectData collects the data storing it at the given output path and returning the URL
// to access it
func (c *BucketCollector) CollectData(data []byte, outputName string) (string, error) {
	opts := &blob.WriterOptions{
		ContentType: util.ContentTypeForFileName(outputName),
		Metadata: map[string]string{
			"classification": c.classifier,
		},
	}
	u := ""
	ctx := c.createContext()
	err := c.bucket.WriteAll(ctx, outputName, data, opts)
	if err != nil {
		return u, errors.Wrapf(err, "failed to write to bucket %s", outputName)
	}

	u = util.UrlJoin(c.bucketURL, outputName)
	return u, nil
}

func (c *BucketCollector) createContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), c.Timeout)
	return ctx
}
