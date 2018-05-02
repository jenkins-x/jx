package docker

import (
	"fmt"
	"sync"
	"time"

	"github.com/Azure/draft/pkg/builder"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"golang.org/x/net/context"
)

// Builder contains information about the build environment
type Builder struct {
	DockerClient command.Cli
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
		buildopts := types.ImageBuildOptions{
			Tags:       app.Images,
			Dockerfile: app.Ctx.Env.Dockerfile,
		}

		resp, err := b.DockerClient.Client().ImageBuild(ctx, app.Buf, buildopts)
		if err != nil {
			errc <- err
			return
		}
		defer func() {
			resp.Body.Close()
			close(msgc)
			close(errc)
		}()
		outFd, isTerm := term.GetFdInfo(app.Buf)
		if err := jsonmessage.DisplayJSONMessagesStream(resp.Body, app.Log, outFd, isTerm, nil); err != nil {
			errc <- err
			return
		}
		if _, _, err = b.DockerClient.Client().ImageInspectWithRaw(ctx, app.MainImage); err != nil {
			if dockerclient.IsErrNotFound(err) {
				errc <- fmt.Errorf("Could not locate image for %s: %v", app.Ctx.Env.Name, err)
				return
			}
			errc <- fmt.Errorf("ImageInspectWithRaw error: %v", err)
			return
		}
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
	if app.Ctx.Env.Registry == "" {
		return
	}

	const stageDesc = "Pushing Docker Image"

	defer builder.Complete(app.ID, stageDesc, out, &err)
	summary := builder.Summarize(app.ID, stageDesc, out)

	// notify that particular stage has started.
	summary("started", builder.SummaryStarted)

	msgc := make(chan string, 1)
	errc := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(len(app.Images))

	go func() {
		registryAuth, err := command.RetrieveAuthTokenFromImage(ctx, b.DockerClient, app.MainImage)
		if err != nil {
			errc <- err
			return
		}

		for _, tag := range app.Images {

			go func(tag string) {
				defer wg.Done()

				resp, err := b.DockerClient.Client().ImagePush(ctx, tag, types.ImagePushOptions{RegistryAuth: registryAuth})
				if err != nil {
					errc <- err
					return
				}

				defer resp.Close()
				outFd, isTerm := term.GetFdInfo(app.Log)
				if err := jsonmessage.DisplayJSONMessagesStream(resp, app.Log, outFd, isTerm, nil); err != nil {
					errc <- err
					return
				}
			}(tag)
		}

		defer func() {
			close(errc)
			close(msgc)
		}()

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
	wg.Wait()
	return nil
}

// AuthToken retrieves the auth token for the given image.
func (b *Builder) AuthToken(ctx context.Context, app *builder.AppContext) (string, error) {
	return command.RetrieveAuthTokenFromImage(ctx, b.DockerClient, app.MainImage)
}
