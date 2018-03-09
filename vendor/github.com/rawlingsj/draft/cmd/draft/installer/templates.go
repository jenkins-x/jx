package installer

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
        - name: DOCKER_HOST
          value: tcp://localhost:2375
        livenessProbe:
          httpGet:
            path: /ping
            port: 8080
        readinessProbe:
          httpGet:
            path: /ping
            port: 8080
      - name: dind
        image: docker:17.05.0-ce-dind
        args:
        - --insecure-registry=10.96.0.0/12
        - --insecure-registry=10.0.0.0/24
        env:
        - name: DOCKER_DRIVER
          value: overlay
        securityContext:
            privileged: true
        volumeMounts:
        - mountPath: /var/lib/docker
          name: docker-graph-storage
        {{- if (or .Values.tls.enable .Values.tls.verify) }}
        - mountPath: "/etc/certs"
          name: draftd-certs
          readOnly: true
        {{- end }}
      volumes:
      - name: docker-graph-storage
        emptyDir: {}
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
