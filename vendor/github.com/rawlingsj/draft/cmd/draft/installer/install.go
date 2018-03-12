package installer

import (
	"errors"
	"fmt"
	"path"

	"google.golang.org/grpc"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"

	draftconfig "github.com/Azure/draft/cmd/draft/installer/config"
	"github.com/Azure/draft/pkg/version"
)

// ReleaseName is the name of the release used when installing/uninstalling draft via helm.
const ReleaseName = "draft"

const draftChart = `name: draftd
description: The Draft server
version: %s
apiVersion: v1
`

const draftValues = `# Default values for Draftd.
# This is a YAML-formatted file.
# Declare variables to be passed into your templates.
replicaCount: 1
ingress:
    enabled: false
    basedomain: example.com
image:
  repository: microsoft/draft
  tag: %s
  pullPolicy: Always
imageOverride: ""
debug: false
service:
  http:
    externalPort: 80
    internalPort: 44135
registry:
  url: docker.io
  org: draft
  # This field follows the format of Docker's X-Registry-Auth header.
  #
  # See https://github.com/docker/docker/blob/master/docs/api/v1.22.md#push-an-image-on-the-registry
  #
  # For credential-based logins, use
  #
  # $ echo '{"username":"jdoe","password":"secret","email":"jdoe@acme.com"}' | base64 -w 0
  #
  # For token-based logins, use
  #
  # $ echo '{"registrytoken":"9cbaf023786cd7"}' | base64 -w 0
  authtoken: e30K
tls:
  enable: false
  verify: false
  key: ""
  cert: ""
  cacert: ""
`

const draftIgnore = `# Patterns to ignore when building packages.
# This supports shell glob matching, relative path matching, and
# negation (prefixed with !). Only one pattern per line.
.DS_Store
# Common VCS dirs
.git/
.gitignore
.bzr/
.bzrignore
.hg/
.hgignore
.svn/
# Common backup files
*.swp
*.bak
*.tmp
*~
# Various IDEs
.project
.idea/
*.tmproj
`

const draftDeployment = `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: draftd
  labels:
    chart: "{{ .Chart.Name }}-{{ .Chart.Version }}"
spec:
  replicas: {{ .Values.replicaCount }}
  template:
    metadata:
      labels:
        app: draft
        name: draftd
        draft: draftd
    spec:
      containers:
      - name: draftd
        image: "{{ default (printf "%s:%s" .Values.image.repository .Values.image.tag) .Values.imageOverride }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        args:
        - start
        - --registry-url={{ .Values.registry.url }}
        - --registry-auth={{ .Values.registry.authtoken }}
        - --basedomain={{ .Values.ingress.basedomain }}
        - --ingress-enabled={{.Values.ingress.enabled}}
        - --tls={{.Values.tls.enable}}
        - --tls-verify={{.Values.tls.verify}}
        {{- if .Values.debug }}
        - --debug
        {{- end }}
        ports:
        - containerPort: {{ .Values.service.http.internalPort }}
        env:
        - name: DRAFT_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: DRAFT_TLS_VERIFY
          value: {{ if .Values.tls.verify }}"1"{{ else }}"0"{{ end }}
        - name: DRAFT_TLS_ENABLE
          value: {{ if .Values.tls.enable }}"1"{{ else }}"0"{{ end }}
        - name: DRAFT_TLS_CERTS
          value: "/etc/certs"
        livenessProbe:
          httpGet:
            path: /ping
            port: 8080
        readinessProbe:
          httpGet:
            path: /ping
            port: 8080
        volumeMounts:
        - mountPath: /var/run/docker.sock
          name: docker-socket
        {{- if (or .Values.tls.enable .Values.tls.verify) }}
        - mountPath: "/etc/certs"
          name: draftd-certs
          readOnly: true
        {{- end }}
      volumes:
      - name: docker-socket
        hostPath:
          path: /var/run/docker.sock
      {{- if (or .Values.tls.enable .Values.tls.verify) }}
      - name: draftd-certs
        secret:
          secretName: draftd-secret
      {{- end }}
      nodeSelector:
        beta.kubernetes.io/os: linux
`

const draftService = `apiVersion: v1
kind: Service
metadata:
  name: {{ .Chart.Name }}
spec:
  ports:
    - name: http
      port: {{ .Values.service.http.externalPort }}
      targetPort: {{ .Values.service.http.internalPort }}
  selector:
    app: {{ .Chart.Name }}
`

const draftSecret = `apiVersion: v1
kind: Secret
type: Opaque
data:
  tls.key: {{ .Values.tls.key }}
  tls.crt: {{ .Values.tls.cert }}
  ca.crt: {{ .Values.tls.cacert }}
metadata:
  name: draftd-secret
`

const draftNotes = `Now you can deploy an app using Draft!

  $ cd my-app
  $ draft create
  $ draft up
  --> Building Dockerfile
  --> Pushing my-app:latest
  --> Deploying to Kubernetes
  --> Deployed!

That's it! You're now running your app in a Kubernetes cluster.
`

// this file left intentionally blank.
const draftHelpers = ``

// DefaultChartFiles represent the default chart files relevant to a Draft chart installation
var DefaultChartFiles = []*chartutil.BufferedFile{
	{
		Name: chartutil.ChartfileName,
		Data: []byte(fmt.Sprintf(draftChart, version.Release)),
	},
	{
		Name: chartutil.ValuesfileName,
		Data: []byte(fmt.Sprintf(draftValues, version.Release)),
	},
	{
		Name: chartutil.IgnorefileName,
		Data: []byte(draftIgnore),
	},
	{
		Name: path.Join(chartutil.TemplatesDir, chartutil.DeploymentName),
		Data: []byte(draftDeployment),
	},
	{
		Name: path.Join(chartutil.TemplatesDir, chartutil.ServiceName),
		Data: []byte(draftService),
	},
	{
		Name: path.Join(chartutil.TemplatesDir, chartutil.NotesName),
		Data: []byte(draftNotes),
	},
	{
		Name: path.Join(chartutil.TemplatesDir, chartutil.HelpersName),
		Data: []byte(draftHelpers),
	},
}

// Install uses the helm client to install Draftd with the given config.
//
// Returns an error if the command failed.
func Install(client *helm.Client, namespace string, config *draftconfig.DraftConfig) error {
	if config.WithTLS() {
		DefaultChartFiles = append(DefaultChartFiles, &chartutil.BufferedFile{
			// TODO: Add chartutil.SecretName to k8s.io/helm/pkg/chartutil/create.go
			Name: path.Join(chartutil.TemplatesDir, "secret.yaml"),
			Data: []byte(draftSecret),
		})
	}

	chart, err := chartutil.LoadFiles(DefaultChartFiles)
	if err != nil {
		return err
	}
	_, err = client.InstallReleaseFromChart(
		chart,
		namespace,
		helm.ReleaseName(ReleaseName),
		helm.ValueOverrides([]byte(config.String())),
	)
	return prettyError(err)
}

//
// Upgrade uses the helm client to upgrade Draftd using the given config.
//
// Returns an error if the command failed.
func Upgrade(client *helm.Client, chartConfig *chart.Config) error {
	chart, err := chartutil.LoadFiles(DefaultChartFiles)
	if err != nil {
		return err
	}
	_, err = client.UpdateReleaseFromChart(
		ReleaseName,
		chart,
		helm.UpdateValueOverrides([]byte(chartConfig.Raw)))
	return prettyError(err)
}

// prettyError unwraps grpc error descriptions to make them more user-friendly.
func prettyError(err error) error {
	if err == nil {
		return nil
	}

	return errors.New(grpc.ErrorDesc(err))
}
