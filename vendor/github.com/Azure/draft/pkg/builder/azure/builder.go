package azure

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/draft/pkg/azure/blob"
	"github.com/Azure/draft/pkg/azure/containerregistry"
	"github.com/Azure/draft/pkg/builder"
	"github.com/Azure/go-autorest/autorest/adal"
	azurecli "github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

// Builder contains information about the build environment
type Builder struct {
	RegistryClient containerregistry.RegistriesClient
	BuildsClient   containerregistry.BuildsClient
	AdalToken      adal.Token
	Subscription   azurecli.Subscription
}

// Build builds the docker image.
func (b *Builder) Build(ctx context.Context, app *builder.AppContext, out chan<- *builder.Summary) (err error) {
	const stageDesc = "Building Docker Image"

	defer builder.Complete(app.ID, stageDesc, out, &err)
	summary := builder.Summarize(app.ID, stageDesc, out)

	// notify that particular stage has started.
	summary("started", builder.SummaryStarted)

	msgc := make(chan string)
	errc := make(chan error)
	go func() {
		defer func() {
			close(msgc)
			close(errc)
		}()
		// the azure SDK wants only the name of the registry rather than the full registry URL
		registryName := getRegistryName(app.Ctx.Env.Registry)
		// first, upload the tarball to the upload storage URL given to us by acr build
		sourceUploadDefinition, err := b.RegistryClient.GetBuildSourceUploadURL(ctx, app.Ctx.Env.ResourceGroupName, registryName)
		if err != nil {
			errc <- fmt.Errorf("Could not retrieve acr build's upload URL: %v", err)
			return
		}
		u, err := url.Parse(*sourceUploadDefinition.UploadURL)
		if err != nil {
			errc <- fmt.Errorf("Could not parse blob upload URL: %v", err)
			return
		}

		blockBlobService := azblob.NewBlockBlobURL(*u, azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{}))
		// Upload the application tarball to acr build
		_, err = blockBlobService.PutBlob(ctx, bytes.NewReader(app.Ctx.Archive), azblob.BlobHTTPHeaders{ContentType: "application/gzip"}, azblob.Metadata{}, azblob.BlobAccessConditions{})
		if err != nil {
			errc <- fmt.Errorf("Could not upload docker context to acr build: %v", err)
			return
		}

		var imageNames []string
		for i := range app.Images {
			imageNameParts := strings.Split(app.Images[i], ":")
			// get the tag name from the image name
			imageNames = append(imageNames, fmt.Sprintf("%s:%s", app.Ctx.Env.Name, imageNameParts[len(imageNameParts)-1]))
		}

		req := containerregistry.QuickBuildRequest{
			ImageNames:     to.StringSlicePtr(imageNames),
			SourceLocation: sourceUploadDefinition.RelativePath,
			// TODO: make this configurable with https://github.com/Azure/draft/issues/663
			BuildArguments: nil,
			IsPushEnabled:  to.BoolPtr(true),
			Timeout:        to.Int32Ptr(600),
			Platform: &containerregistry.PlatformProperties{
				// TODO: make this configurable once ACR build supports windows containers
				OsType: containerregistry.Linux,
				// NB: CPU isn't required right now, possibly want to make this configurable
				// It'll actually default to 2 from the server
				// CPU: to.Int32Ptr(1),
			},
			// TODO: make this configurable
			DockerFilePath: to.StringPtr("Dockerfile"),
			Type:           containerregistry.TypeQuickBuild,
		}
		bas, ok := req.AsBasicQueueBuildRequest()
		if !ok {
			errc <- errors.New("Failed to create quick build request")
			return
		}
		future, err := b.RegistryClient.QueueBuild(ctx, app.Ctx.Env.ResourceGroupName, registryName, bas)
		if err != nil {
			errc <- fmt.Errorf("Could not while queue acr build: %v", err)
			return
		}

		if err := future.WaitForCompletion(ctx, b.RegistryClient.Client); err != nil {
			errc <- fmt.Errorf("Could not wait for acr build to complete: %v", err)
			return
		}

		fin, err := future.Result(b.RegistryClient)
		if err != nil {
			errc <- fmt.Errorf("Could not retrieve acr build future result: %v", err)
			return
		}

		logResult, err := b.BuildsClient.GetLogLink(ctx, app.Ctx.Env.ResourceGroupName, registryName, *fin.BuildID)
		if err != nil {
			errc <- fmt.Errorf("Could not retrieve acr build logs: %v", err)
			return
		}

		if *logResult.LogLink == "" {
			errc <- errors.New("Unable to create a link to the logs: no link found")
			return
		}

		blobURL := blob.GetAppendBlobURL(*logResult.LogLink)

		// Used for progress reporting to report the total number of bytes being downloaded.
		var contentLength int64
		rs := azblob.NewDownloadStream(ctx,
			// We pass more than "blobUrl.GetBlob" here so we can capture the blob's full
			// content length on the very first internal call to Read.
			func(ctx context.Context, blobRange azblob.BlobRange, ac azblob.BlobAccessConditions, rangeGetContentMD5 bool) (*azblob.GetResponse, error) {
				for {
					properties, err := blobURL.GetPropertiesAndMetadata(ctx, ac)
					if err != nil {
						// retry if the blob doesn't exist yet
						if strings.Contains(err.Error(), "The specified blob does not exist.") {
							continue
						}
						return nil, err
					}
					// retry if the blob hasn't "completed"
					if !blobComplete(properties.NewMetadata()) {
						continue
					}
					break
				}
				resp, err := blobURL.GetBlob(ctx, blobRange, ac, rangeGetContentMD5)
				if err != nil {
					return nil, err
				}
				if contentLength == 0 {
					// If 1st successful Get, record blob's full size for progress reporting
					contentLength = resp.ContentLength()
				}
				return resp, nil
			},
			azblob.DownloadStreamOptions{})
		defer rs.Close()

		_, err = io.Copy(app.Log, rs)
		if err != nil {
			errc <- fmt.Errorf("Could not stream acr build logs: %v", err)
			return
		}

		return

	}()
	for msgc != nil || errc != nil {
		select {
		case msg, ok := <-msgc:
			if !ok {
				msgc = nil
				continue
			}
			summary(msg, builder.SummaryLogging)
		case err, ok := <-errc:
			if !ok {
				errc = nil
				continue
			}
			return err
		default:
			summary("ongoing", builder.SummaryOngoing)
			time.Sleep(time.Second)
		}
	}
	return nil
}

// Push pushes the results of Build to the image repository.
func (b *Builder) Push(ctx context.Context, app *builder.AppContext, out chan<- *builder.Summary) (err error) {
	// no-op: acr build pushes to the registry through the quickbuild request
	const stageDesc = "Building Docker Image"
	builder.Complete(app.ID, stageDesc, out, &err)
	return nil
}

// AuthToken retrieves the auth token for the given image.
func (b *Builder) AuthToken(ctx context.Context, app *builder.AppContext) (string, error) {
	dockerAuth, err := b.getACRDockerEntryFromARMToken(app.Ctx.Env.Registry)
	if err != nil {
		return "", err
	}
	buf, err := json.Marshal(dockerAuth)
	return base64.StdEncoding.EncodeToString(buf), err
}

func getRegistryName(registry string) string {
	return strings.Replace(registry, ".azurecr.io", "", 1)
}

func blobComplete(metadata azblob.Metadata) bool {
	for k := range metadata {
		if strings.ToLower(k) == "complete" {
			return true
		}
	}
	return false
}

func (b *Builder) getACRDockerEntryFromARMToken(loginServer string) (*builder.DockerConfigEntryWithAuth, error) {
	accessToken := b.AdalToken.OAuthToken()

	directive, err := containerregistry.ReceiveChallengeFromLoginServer(loginServer)
	if err != nil {
		return nil, fmt.Errorf("failed to receive challenge: %s", err)
	}

	registryRefreshToken, err := containerregistry.PerformTokenExchange(
		loginServer, directive, b.Subscription.TenantID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to perform token exchange: %s", err)
	}

	glog.V(4).Infof("adding ACR docker config entry for: %s", loginServer)
	return &builder.DockerConfigEntryWithAuth{
		Username: containerregistry.DockerTokenLoginUsernameGUID,
		Password: registryRefreshToken,
	}, nil
}
