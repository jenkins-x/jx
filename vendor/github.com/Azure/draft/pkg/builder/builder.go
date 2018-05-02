package builder

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Azure/draft/pkg/draft/manifest"
	"github.com/Azure/draft/pkg/draft/pack"
	"github.com/Azure/draft/pkg/local"
	"github.com/Azure/draft/pkg/osutil"
	"github.com/Azure/draft/pkg/storage"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/strvals"
)

const (
	// PullSecretName is the name of the docker pull secret draft will create in the desired destination namespace
	PullSecretName = "draft-pullsecret"
	// DefaultServiceAccountName is the name of the default service account draft will modify with the imagepullsecret
	DefaultServiceAccountName = "default"
)

// Builder contains information about the build environment
type Builder struct {
	ID               string
	ContainerBuilder ContainerBuilder
	Helm             helm.Interface
	Kube             k8s.Interface
	Storage          storage.Store
	LogsDir          string
}

// ContainerBuilder defines how a container is built and pushed to a container registry using the supplied app context.
type ContainerBuilder interface {
	Build(ctx context.Context, app *AppContext, out chan<- *Summary) error
	Push(ctx context.Context, app *AppContext, out chan<- *Summary) error
	AuthToken(ctx context.Context, app *AppContext) (string, error)
}

// Logs returns the path to the build logs.
//
// Set after Up is called (otherwise "").
func (b *Builder) Logs(appName string) string {
	return filepath.Join(b.LogsDir, appName, b.ID)
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
	Obj       *storage.Object
	Bldr      *Builder
	Ctx       *Context
	Buf       *bytes.Buffer
	MainImage string
	Images    []string
	Log       io.WriteCloser
	ID        string
	Vals      chartutil.Values
}

// New creates a new Builder.
func New() *Builder {
	return &Builder{
		ID: getulid(),
	}
}

// newAppContext prepares state carried across the various draft stage boundaries.
func newAppContext(b *Builder, buildCtx *Context) (*AppContext, error) {
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

	images := []string{image}
	for _, tag := range buildCtx.Env.CustomTags {
		images = append(images, fmt.Sprintf("%s:%s", imageRepository, tag))
	}

	// inject certain values into the chart such as the registry location,
	// the application name, buildID and the application version.
	tplstr := "image.repository=%s,image.tag=%s,%s=%s,%s=%s"
	inject := fmt.Sprintf(tplstr, imageRepository, imgtag, local.DraftLabelKey, buildCtx.Env.Name, local.BuildIDKey, b.ID)

	vals, err := chartutil.ReadValues([]byte(buildCtx.Values.Raw))
	if err != nil {
		return nil, err
	}
	if err := strvals.ParseInto(inject, vals); err != nil {
		return nil, err
	}

	err = osutil.EnsureDirectory(filepath.Dir(b.Logs(buildCtx.Env.Name)))
	if err != nil {
		return nil, err
	}

	logf, err := os.OpenFile(b.Logs(buildCtx.Env.Name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	state := &storage.Object{
		BuildID:     b.ID,
		ContextID:   ctxtID,
		LogsFileRef: b.Logs(buildCtx.Env.Name),
	}
	return &AppContext{
		Obj:       state,
		ID:        b.ID,
		Bldr:      b,
		Ctx:       buildCtx,
		Buf:       buf,
		Images:    images,
		MainImage: image,
		Log:       logf,
		Vals:      vals,
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

	// if a chart was specified in manifest, use it
	if ctx.Env.Chart != "" {
		ctx.Chart, err = chartutil.Load(filepath.Join(ctx.AppDir, pack.ChartsDir, ctx.Env.Chart))
		if err != nil {
			return err
		}
	} else {
		// otherwise, find the first directory in chart/ and assume that is the chart we want to deploy.
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
				break
			}
		}
		if !found {
			return ErrChartNotExist
		}
	}

	return nil
}

func loadValues(ctx *Context) error {
	var vals = make(chartutil.Values)
	for _, val := range ctx.Env.Values {
		if err := strvals.ParseInto(val, vals); err != nil {
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
	contextDir, relDockerfile, err := build.GetContextFromLocalDir(ctx.AppDir, ctx.Env.Dockerfile)
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
			app *AppContext
			err error
		)
		defer func() {
			b.saveState(app)
			wg.Done()
		}()
		if app, err = newAppContext(b, bctx); err != nil {
			log.Printf("error creating app context: %v\n", err)
			return
		}
		log.SetOutput(app.Log)
		if err := b.ContainerBuilder.Build(ctx, app, ch); err != nil {
			log.Printf("error while building: %v\n", err)
			return
		}
		if err := b.ContainerBuilder.Push(ctx, app, ch); err != nil {
			log.Printf("error while pushing: %v\n", err)
			return
		}
		if err := b.release(ctx, app, ch); err != nil {
			log.Printf("error while releasing: %v\n", err)
			return
		}
	}()
	go func() {
		wg.Wait()
		close(ch)
	}()
	return ch
}

// saveState saves information collected from a draft build.
func (b *Builder) saveState(app *AppContext) {
	if err := b.Storage.UpdateBuild(context.Background(), app.Ctx.Env.Name, app.Obj); err != nil {
		log.Printf("complete: failed to store build object for app %q: %v\n", app.Ctx.Env.Name, err)
		return
	}
	if app.Log != nil {
		app.Log.Close()
	}
}

// release installs or updates the application deployment.
func (b *Builder) release(ctx context.Context, app *AppContext, out chan<- *Summary) (err error) {
	const stageDesc = "Releasing Application"

	defer Complete(app.ID, stageDesc, out, &err)
	summary := Summarize(app.ID, stageDesc, out)

	// notify that particular stage has started.
	summary("started", SummaryStarted)

	// inject a registry secret only if a registry was configured
	if app.Ctx.Env.Registry != "" {
		if err := b.prepareReleaseEnvironment(ctx, app); err != nil {
			return err
		}
	}

	// If a release does not exist, install it. If another error occurs during the check,
	// ignore the error and continue with the upgrade.
	//
	// The returned error is a gSummaryhat wraps the message from the original error.
	// So we're stuck doing string matching against the wrapped error, which is nested inside
	// of the gSummaryessage.
	_, err = b.Helm.ReleaseContent(app.Ctx.Env.Name, helm.ContentReleaseVersion(1))
	if err != nil && strings.Contains(err.Error(), "not found") {
		msg := fmt.Sprintf("Release %q does not exist. Installing it now.", app.Ctx.Env.Name)
		summary(msg, SummaryLogging)

		vals, err := app.Vals.YAML()
		if err != nil {
			return err
		}

		opts := []helm.InstallOption{
			helm.ReleaseName(app.Ctx.Env.Name),
			helm.ValueOverrides([]byte(vals)),
			helm.InstallWait(app.Ctx.Env.Wait),
		}
		rls, err := b.Helm.InstallReleaseFromChart(app.Ctx.Chart, app.Ctx.Env.Namespace, opts...)
		if err != nil {
			return fmt.Errorf("could not install release: %v", err)
		}
		app.Obj.Release = rls.Release.Name
		formatReleaseStatus(app, rls.Release, summary)

	} else {
		msg := fmt.Sprintf("Upgrading %s.", app.Ctx.Env.Name)
		summary(msg, SummaryLogging)

		vals, err := app.Vals.YAML()
		if err != nil {
			return err
		}

		opts := []helm.UpdateOption{
			helm.UpdateValueOverrides([]byte(vals)),
			helm.UpgradeWait(app.Ctx.Env.Wait),
		}
		rls, err := b.Helm.UpdateReleaseFromChart(app.Ctx.Env.Name, app.Ctx.Chart, opts...)
		if err != nil {
			return fmt.Errorf("could not upgrade release: %v", err)
		}
		app.Obj.Release = rls.Release.Name
		formatReleaseStatus(app, rls.Release, summary)
	}
	return nil
}

func (b *Builder) prepareReleaseEnvironment(ctx context.Context, app *AppContext) error {
	// determine if the destination namespace exists, create it if not.
	if _, err := b.Kube.CoreV1().Namespaces().Get(app.Ctx.Env.Namespace, metav1.GetOptions{}); err != nil {
		if !apiErrors.IsNotFound(err) {
			return err
		}
		_, err = b.Kube.CoreV1().Namespaces().Create(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: app.Ctx.Env.Namespace},
		})
		if err != nil {
			return fmt.Errorf("could not create namespace %q: %v", app.Ctx.Env.Namespace, err)
		}
	}

	authToken, err := b.ContainerBuilder.AuthToken(ctx, app)
	if err != nil {
		return fmt.Errorf("failed to retrieve auth token for image %s: %v", app.MainImage, err)
	}

	// we need to translate the auth token Docker gives us into a Kubernetes registry auth secret token.
	regAuth, err := FromAuthConfigToken(authToken)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to a kubernetes registry auth secret token: %v", authToken, err)
	}

	// create a new json string with the full dockerauth, including the registry URL.
	js, err := json.Marshal(map[string]*DockerConfigEntryWithAuth{app.Ctx.Env.Registry: regAuth})
	if err != nil {
		return fmt.Errorf("could not json encode docker authentication string: %v", err)
	}

	// determine if the registry pull secret exists in the desired namespace, create it if not.
	var secret *v1.Secret
	if secret, err = b.Kube.CoreV1().Secrets(app.Ctx.Env.Namespace).Get(PullSecretName, metav1.GetOptions{}); err != nil {
		if !apiErrors.IsNotFound(err) {
			return err
		}
		_, err = b.Kube.CoreV1().Secrets(app.Ctx.Env.Namespace).Create(
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PullSecretName,
					Namespace: app.Ctx.Env.Namespace,
				},
				Type: v1.SecretTypeDockercfg,
				StringData: map[string]string{
					".dockercfg": string(js),
				},
			},
		)
		if err != nil {
			return fmt.Errorf("could not create registry pull secret: %v", err)
		}
	} else {
		// the registry pull secret exists, check if it needs to be updated.
		if data, ok := secret.StringData[".dockercfg"]; ok && data != string(js) {
			secret.StringData[".dockercfg"] = string(js)
			_, err = b.Kube.CoreV1().Secrets(app.Ctx.Env.Namespace).Update(secret)
			if err != nil {
				return fmt.Errorf("could not update registry pull secret: %v", err)
			}
		}
	}

	// determine if the default service account in the desired namespace has the correct
	// imagePullSecret. If not, add it.
	svcAcct, err := b.Kube.CoreV1().ServiceAccounts(app.Ctx.Env.Namespace).Get(DefaultServiceAccountName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("could not load default service account: %v", err)
	}
	found := false
	for _, ps := range svcAcct.ImagePullSecrets {
		if ps.Name == PullSecretName {
			found = true
			break
		}
	}
	if !found {
		svcAcct.ImagePullSecrets = append(svcAcct.ImagePullSecrets, v1.LocalObjectReference{
			Name: PullSecretName,
		})
		_, err := b.Kube.CoreV1().ServiceAccounts(app.Ctx.Env.Namespace).Update(svcAcct)
		if err != nil {
			return fmt.Errorf("could not modify default service account with registry pull secret: %v", err)
		}
	}

	return nil
}

func formatReleaseStatus(app *AppContext, rls *release.Release, summary func(string, SummaryStatusCode)) {
	status := fmt.Sprintf("%s %v", app.Ctx.Env.Name, rls.Info.Status.Code)
	summary(status, SummaryLogging)
	if rls.Info.Status.Notes != "" {
		notes := fmt.Sprintf("notes: %v", rls.Info.Status.Notes)
		summary(notes, SummaryLogging)
	}
}

// Summarize returns a function closure that wraps writing SummaryStatusCode.
func Summarize(id, desc string, out chan<- *Summary) func(string, SummaryStatusCode) {
	return func(info string, code SummaryStatusCode) {
		out <- &Summary{StageDesc: desc, StatusText: info, StatusCode: code, BuildID: id}
	}
}

// Complete marks the end of a draft build stage.
func Complete(id, desc string, out chan<- *Summary, err *error) {
	switch fn := Summarize(id, desc, out); {
	case *err != nil:
		fn(fmt.Sprintf("failure: %v", *err), SummaryFailure)
	default:
		fn("success", SummarySuccess)
	}
}
