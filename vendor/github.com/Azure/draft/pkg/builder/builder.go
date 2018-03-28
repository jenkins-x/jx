package builder

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Azure/draft/pkg/draft/local"
	"github.com/Azure/draft/pkg/draft/manifest"
	"github.com/Azure/draft/pkg/draft/pack"
	"github.com/Azure/draft/pkg/storage"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/builder/dockerignore"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/strvals"
)

// Builder contains information about the build environment
type Builder struct {
	DockerClient command.Cli
	Helm         helm.Interface
	Kube         k8s.Interface
	Storage      storage.Store
	LogsDir      string
}

// Context contains information about the application
type Context struct {
	Env     *manifest.Environment
	EnvName string
	AppDir  string
	Chart   *chart.Chart
	Values  *chart.Config
	SrcName string
	Archive []byte
}

// AppContext contains state information carried across the various draft stage boundaries.
type AppContext struct {
	obj  *storage.Object
	bldr *Builder
	ctx  *Context
	buf  *bytes.Buffer
	tag  string
	img  string
	out  io.Writer
	id   string
	vals chartutil.Values
}

// newAppContext prepares state carried across the various draft stage boundaries.
func newAppContext(b *Builder, buildCtx *Context, out io.Writer) (*AppContext, error) {
	raw := bytes.NewBuffer(buildCtx.Archive)
	// write build context to a buffer so we can also write to the sha256 hash.
	buf := new(bytes.Buffer)
	h := sha256.New()
	w := io.MultiWriter(buf, h)
	if _, err := io.Copy(w, raw); err != nil {
		return nil, err
	}
	// truncate checksum to the first 40 characters (20 bytes) this is the
	// equivalent of `shasum build.tar.gz | awk '{print $1}'`.
	ctxtID := h.Sum(nil)
	imgtag := fmt.Sprintf("%.20x", ctxtID)
	// if registry == "", then we just assume the image name is the app name and strip out the leading /
	imageRepository := strings.TrimLeft(fmt.Sprintf("%s/%s", buildCtx.Env.Registry, buildCtx.Env.Name), "/")
	image := fmt.Sprintf("%s:%s", imageRepository, imgtag)

	buildID := getulid()

	// inject certain values into the chart such as the registry location,
	// the application name, buildID and the application version.
	tplstr := "image.repository=%s,image.tag=%s,%s=%s,%s=%s"
	inject := fmt.Sprintf(tplstr, imageRepository, imgtag, local.DraftLabelKey, buildCtx.Env.Name, local.BuildIDKey, buildID)

	vals, err := chartutil.ReadValues([]byte(buildCtx.Values.Raw))
	if err != nil {
		return nil, err
	}
	if err := strvals.ParseInto(inject, vals); err != nil {
		return nil, err
	}
	return &AppContext{
		obj:  &storage.Object{BuildID: buildID, ContextID: ctxtID},
		id:   buildID,
		bldr: b,
		ctx:  buildCtx,
		buf:  buf,
		tag:  imgtag,
		img:  image,
		out:  out,
		vals: vals,
	}, nil
}

// LoadWithEnv takes the directory of the application and the environment the application
//  will be pushed to and returns a Context object with a merge of environment and app
//  information
func LoadWithEnv(appdir, whichenv string) (*Context, error) {
	ctx := &Context{AppDir: appdir, EnvName: whichenv}
	// read draft.toml from appdir.
	mfst, err := manifest.Load(filepath.Join(appdir, "draft.toml"))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal draft.toml from %q: %v", appdir, err)
	}
	// if environment does not exist return error.
	var ok bool
	if ctx.Env, ok = mfst.Environments[whichenv]; !ok {
		return nil, fmt.Errorf("no environment named %q in draft.toml", whichenv)
	}
	// load the chart and the build archive; if a chart directory is present
	// this will be given priority over the chart archive specified by the
	// `chart-tar` field in the draft.toml. If this is the case, then build-tar
	// is built from scratch. If no chart directory exists but a chart-tar and
	// build-tar exist, then these will be used for values extraction.
	if err := loadArchive(ctx); err != nil {
		return nil, fmt.Errorf("failed to load chart: %v", err)
	}
	// load values from chart and merge with env.Values.
	if err := loadValues(ctx); err != nil {
		return nil, fmt.Errorf("failed to parse chart values: %v", err)
	}
	return ctx, nil
}

// loadArchive loads the chart package and build archive.
// Precedence is given to the `build-tar` and `chart-tar`
// indicated in the `draft.toml` if present. Otherwise,
// loadArchive loads the chart directory and archives the
// app directory to send to the draft server.
func loadArchive(ctx *Context) (err error) {
	if ctx.Env.BuildTarPath != "" && ctx.Env.ChartTarPath != "" {
		b, err := ioutil.ReadFile(ctx.Env.BuildTarPath)
		if err != nil {
			return fmt.Errorf("failed to load build archive %q: %v", ctx.Env.BuildTarPath, err)
		}
		ctx.SrcName = filepath.Base(ctx.Env.BuildTarPath)
		ctx.Archive = b

		ar, err := os.Open(ctx.Env.ChartTarPath)
		if err != nil {
			return err
		}
		if ctx.Chart, err = chartutil.LoadArchive(ar); err != nil {
			return fmt.Errorf("failed to load chart archive %q: %v", ctx.Env.ChartTarPath, err)
		}
		return nil
	}
	if err = archiveSrc(ctx); err != nil {
		return err
	}
	// find the first directory in chart/ and assume that is the chart we want to deploy.
	chartDir := filepath.Join(ctx.AppDir, pack.ChartsDir)
	files, err := ioutil.ReadDir(chartDir)
	if err != nil {
		return err
	}
	var found bool
	for _, file := range files {
		if file.IsDir() {
			found = true
			if ctx.Chart, err = chartutil.Load(filepath.Join(chartDir, file.Name())); err != nil {
				return err
			}
		}
	}
	if !found {
		return ErrChartNotExist
	}
	return nil
}

func loadValues(ctx *Context) error {
	var vals = make(chartutil.Values)
	for _, val := range ctx.Env.Values {
		if err := strvals.ParseInto(val, vals.AsMap()); err != nil {
			return fmt.Errorf("failed to parse %q from draft.toml: %v", val, err)
		}
	}
	s, err := vals.YAML()
	if err != nil {
		return fmt.Errorf("failed to encode values: %v", err)
	}
	ctx.Values = &chart.Config{Raw: s}
	return nil
}

func archiveSrc(ctx *Context) error {
	contextDir, relDockerfile, err := build.GetContextFromLocalDir(ctx.AppDir, "")
	if err != nil {
		return fmt.Errorf("unable to prepare docker context: %s", err)
	}
	// canonicalize dockerfile name to a platform-independent one
	relDockerfile, err = archive.CanonicalTarNameForPath(relDockerfile)
	if err != nil {
		return fmt.Errorf("cannot canonicalize dockerfile path %s: %v", relDockerfile, err)
	}
	f, err := os.Open(filepath.Join(contextDir, ".dockerignore"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer f.Close()

	var excludes []string
	if err == nil {
		excludes, err = dockerignore.ReadAll(f)
		if err != nil {
			return err
		}
	}

	// do not include the chart directory. That will be packaged separately.
	excludes = append(excludes, filepath.Join(contextDir, "chart"))
	if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
		return fmt.Errorf("error checking docker context: '%s'", err)
	}

	// If .dockerignore mentions .dockerignore or the Dockerfile
	// then make sure we send both files over to the daemon
	// because Dockerfile is, obviously, needed no matter what, and
	// .dockerignore is needed to know if either one needs to be
	// removed. The daemon will remove them for us, if needed, after it
	// parses the Dockerfile. Ignore errors here, as they will have been
	// caught by validateContextDirectory above.
	var includes = []string{"."}
	keepThem1, _ := fileutils.Matches(".dockerignore", excludes)
	keepThem2, _ := fileutils.Matches(relDockerfile, excludes)
	if keepThem1 || keepThem2 {
		includes = append(includes, ".dockerignore", relDockerfile)
	}

	logrus.Debugf("INCLUDES: %v", includes)
	logrus.Debugf("EXCLUDES: %v", excludes)
	rc, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		Compression:     archive.Gzip,
		ExcludePatterns: excludes,
		IncludeFiles:    includes,
	})
	if err != nil {
		return err
	}
	defer rc.Close()

	var b bytes.Buffer
	if _, err := io.Copy(&b, rc); err != nil {
		return err
	}
	ctx.SrcName = "build.tar.gz"
	ctx.Archive = b.Bytes()
	return nil
}

// Up handles incoming draft up requests and returns a stream of summaries or error.
func (b *Builder) Up(ctx context.Context, bctx *Context) <-chan *Summary {
	ch := make(chan *Summary, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		var (
			buf = new(bytes.Buffer)
			app *AppContext
			err error
		)
		defer func() {
			b.finish(app, buf)
			wg.Done()
		}()
		if app, err = newAppContext(b, bctx, buf); err != nil {
			fmt.Printf("buildApp: error creating app context: %v\n", err)
			return
		}
		if err := b.buildImg(ctx, app, ch); err != nil {
			fmt.Printf("buildApp: buildImg error: %v\n", err)
			return
		}
		if app.ctx.Env.Registry != "" {
			if err := b.pushImg(ctx, app, ch); err != nil {
				fmt.Printf("buildApp: pushImg error: %v\n", err)
				return
			}
		}
		if err := b.release(ctx, app, ch); err != nil {
			fmt.Printf("buildApp: release error: %v\n", err)
			return
		}
	}()
	go func() {
		wg.Wait()
		close(ch)
	}()
	return ch
}

// finish updates storage with the information collected during the stages of a draft build and
// writes the aggregated logs to a tempoarary file.
func (b *Builder) finish(app *AppContext, buf *bytes.Buffer) {
	logsFile := filepath.Join(b.LogsDir, app.id)
	app.obj.LogsFileRef = logsFile
	if err := b.Storage.UpdateBuild(context.Background(), app.ctx.Env.Name, app.obj); err != nil {
		fmt.Printf("complete: failed to store build object for app %q: %v\n", app.ctx.Env.Name, err)
		return
	}

	if err := ioutil.WriteFile(logsFile, buf.Bytes(), 0666); err != nil {
		fmt.Printf("complete: failed to write logs to file for build %q: %v\n", app.id, err)
		return
	}
	// TODO: add a debug logger
	// fmt.Printf("complete: wrote logs to %s\n", logsFile)
}

// buildImg builds the docker image.
func (b *Builder) buildImg(ctx context.Context, app *AppContext, out chan<- *Summary) (err error) {
	const stageDesc = "Building Docker Image"

	defer complete(app.id, stageDesc, out, &err)
	summary := summarize(app.id, stageDesc, out)

	// notify that particular stage has started.
	summary("started", SummaryStarted)

	msgc := make(chan string)
	errc := make(chan error)
	go func() {
		buildopts := types.ImageBuildOptions{Tags: []string{app.img}}
		resp, err := b.DockerClient.Client().ImageBuild(ctx, app.buf, buildopts)
		if err != nil {
			errc <- err
			return
		}
		defer func() {
			resp.Body.Close()
			close(msgc)
			close(errc)
		}()
		outFd, isTerm := term.GetFdInfo(app.out)
		if err := jsonmessage.DisplayJSONMessagesStream(resp.Body, app.out, outFd, isTerm, nil); err != nil {
			errc <- err
			return
		}
		if _, _, err = b.DockerClient.Client().ImageInspectWithRaw(ctx, app.img); err != nil {
			if dockerclient.IsErrNotFound(err) {
				errc <- fmt.Errorf("Could not locate image for %s: %v", app.ctx.Env.Name, err)
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
			summary(msg, SummaryLogging)
		case err, ok := <-errc:
			if !ok {
				errc = nil
				continue
			}
			return err
		default:
			summary("ongoing", SummaryOngoing)
			time.Sleep(time.Second)
		}
	}
	return nil
}

// pushImg pushes the results of buildImg to the image repository.
func (b *Builder) pushImg(ctx context.Context, app *AppContext, out chan<- *Summary) (err error) {
	const stageDesc = "Pushing Docker Image"

	defer complete(app.id, stageDesc, out, &err)
	summary := summarize(app.id, stageDesc, out)

	// notify that particular stage has started.
	summary("started", SummaryStarted)

	msgc := make(chan string, 1)
	errc := make(chan error, 1)
	go func() {
		registryAuth, err := command.RetrieveAuthTokenFromImage(ctx, b.DockerClient, app.img)
		if err != nil {
			errc <- err
			return
		}
		resp, err := b.DockerClient.Client().ImagePush(ctx, app.img, types.ImagePushOptions{RegistryAuth: registryAuth})
		if err != nil {
			errc <- err
			return
		}
		defer func() {
			resp.Close()
			close(errc)
			close(msgc)
		}()
		outFd, isTerm := term.GetFdInfo(app.out)
		if err := jsonmessage.DisplayJSONMessagesStream(resp, app.out, outFd, isTerm, nil); err != nil {
			errc <- err
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
			summary(msg, SummaryLogging)
		case err, ok := <-errc:
			if !ok {
				errc = nil
				continue
			}
			return err
		default:
			summary("ongoing", SummaryOngoing)
			time.Sleep(time.Second)
		}
	}
	return nil
}

// release installs or updates the application deployment.
func (b *Builder) release(ctx context.Context, app *AppContext, out chan<- *Summary) (err error) {
	const stageDesc = "Releasing Application"

	defer complete(app.id, stageDesc, out, &err)
	summary := summarize(app.id, stageDesc, out)

	// notify that particular stage has started.
	summary("started", SummaryStarted)

	// If a release does not exist, install it. If another error occurs during the check,
	// ignore the error and continue with the upgrade.
	//
	// The returned error is a gSummaryhat wraps the message from the original error.
	// So we're stuck doing string matching against the wrapped error, which is nested inside
	// of the gSummaryessage.
	_, err = b.Helm.ReleaseContent(app.ctx.Env.Name, helm.ContentReleaseVersion(1))
	if err != nil && strings.Contains(err.Error(), "not found") {
		msg := fmt.Sprintf("Release %q does not exist. Installing it now.", app.ctx.Env.Name)
		summary(msg, SummaryLogging)

		vals, err := app.vals.YAML()
		if err != nil {
			return err
		}

		opts := []helm.InstallOption{
			helm.ReleaseName(app.ctx.Env.Name),
			helm.ValueOverrides([]byte(vals)),
			helm.InstallWait(app.ctx.Env.Wait),
		}
		rls, err := b.Helm.InstallReleaseFromChart(app.ctx.Chart, app.ctx.Env.Namespace, opts...)
		if err != nil {
			return fmt.Errorf("could not install release: %v", err)
		}
		app.obj.Release = rls.Release.Name
		formatReleaseStatus(app, rls.Release, summary)

	} else {
		msg := fmt.Sprintf("Upgrading %s.", app.ctx.Env.Name)
		summary(msg, SummaryLogging)

		vals, err := app.vals.YAML()
		if err != nil {
			return err
		}

		opts := []helm.UpdateOption{
			helm.UpdateValueOverrides([]byte(vals)),
			helm.UpgradeWait(app.ctx.Env.Wait),
		}
		rls, err := b.Helm.UpdateReleaseFromChart(app.ctx.Env.Name, app.ctx.Chart, opts...)
		if err != nil {
			return fmt.Errorf("could not upgrade release: %v", err)
		}
		app.obj.Release = rls.Release.Name
		formatReleaseStatus(app, rls.Release, summary)
	}
	return nil
}

func formatReleaseStatus(app *AppContext, rls *release.Release, summary func(string, SummaryStatusCode)) {
	status := fmt.Sprintf("%s %v", app.ctx.Env.Name, rls.Info.Status.Code)
	summary(status, SummaryLogging)
	if rls.Info.Status.Notes != "" {
		notes := fmt.Sprintf("notes: %v", rls.Info.Status.Notes)
		summary(notes, SummaryLogging)
	}
}

// summarize returns a function closure that wraps writing SummaryStatusCode.
func summarize(id, desc string, out chan<- *Summary) func(string, SummaryStatusCode) {
	return func(info string, code SummaryStatusCode) {
		out <- &Summary{StageDesc: desc, StatusText: info, StatusCode: code, BuildID: id}
	}
}

// complete marks the end of a draft build stage.
func complete(id, desc string, out chan<- *Summary, err *error) {
	switch fn := summarize(id, desc, out); {
	case *err != nil:
		fn(fmt.Sprintf("failure: %v", *err), SummaryFailure)
	default:
		fn("success", SummarySuccess)
	}
}
