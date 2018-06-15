/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package templates

// Manifest is the template found in examples
var Manifest = NewTemplate("manifest", `
---
apiVersion: v1
kind: Namespace
metadata:
  name: {{.Namespace}}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    component: sonobuoy
  name: sonobuoy-serviceaccount
  namespace: {{.Namespace}}
{{- if .EnableRBAC }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    component: sonobuoy
  name: sonobuoy-serviceaccount-{{.Namespace}}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: sonobuoy-serviceaccount
subjects:
- kind: ServiceAccount
  name: sonobuoy-serviceaccount
  namespace: {{.Namespace}}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    component: sonobuoy
  name: sonobuoy-serviceaccount
rules:
- apiGroups:
  - '*'
  resources:
  - '*'
  verbs:
  - '*'
{{- end }}
---
apiVersion: v1
data:
  config.json: |
    {{.SonobuoyConfig}}
kind: ConfigMap
metadata:
  labels:
    component: sonobuoy
  name: sonobuoy-config-cm
  namespace: {{.Namespace}}
---
apiVersion: v1
data:
  e2e.yaml: |
    sonobuoy-config:
      driver: Job
      plugin-name: e2e
      result-type: e2e
    spec:
      env:
      - name: E2E_FOCUS
        value: '{{.E2EFocus}}'
      - name: E2E_SKIP
        value: '{{.E2ESkip}}'
      - name: E2E_PARALLEL
        value: '{{.E2EParallel}}'
      command: ["/run_e2e.sh"]
      image: {{.KubeConformanceImage}}
      imagePullPolicy: {{.ImagePullPolicy}}
      name: e2e
      volumeMounts:
      - mountPath: /tmp/results
        name: results
        readOnly: false
  systemd-logs.yaml: |
    sonobuoy-config:
      driver: DaemonSet
      plugin-name: systemd-logs
      result-type: systemd_logs
    spec:
      command: ["/bin/sh", "-c", "/get_systemd_logs.sh && sleep 3600"]
      env:
      - name: NODE_NAME
        valueFrom:
          fieldRef:
            fieldPath: spec.nodeName
      - name: RESULTS_DIR
        value: /tmp/results
      - name: CHROOT_DIR
        value: /node
      image: gcr.io/heptio-images/sonobuoy-plugin-systemd-logs:latest
      imagePullPolicy: {{.ImagePullPolicy}}
      name: sonobuoy-systemd-logs-config
      securityContext:
        privileged: true
      volumeMounts:
      - mountPath: /tmp/results
        name: results
        readOnly: false
      - mountPath: /node
        name: root
        readOnly: false
kind: ConfigMap
metadata:
  labels:
    component: sonobuoy
  name: sonobuoy-plugins-cm
  namespace: {{.Namespace}}
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    component: sonobuoy
    run: sonobuoy-master
    tier: analysis
  name: sonobuoy
  namespace: {{.Namespace}}
spec:
  containers:
  - command:
    - /bin/bash
    - -c
    - /sonobuoy master --no-exit=true -v 3 --logtostderr
    env:
    - name: SONOBUOY_ADVERTISE_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    image: {{.SonobuoyImage}}
    imagePullPolicy: {{.ImagePullPolicy}}
    name: kube-sonobuoy
    volumeMounts:
    - mountPath: /etc/sonobuoy
      name: sonobuoy-config-volume
    - mountPath: /plugins.d
      name: sonobuoy-plugins-volume
    - mountPath: /tmp/sonobuoy
      name: output-volume
  restartPolicy: Never
  serviceAccountName: sonobuoy-serviceaccount
  volumes:
  - configMap:
      name: sonobuoy-config-cm
    name: sonobuoy-config-volume
  - configMap:
      name: sonobuoy-plugins-cm
    name: sonobuoy-plugins-volume
  - emptyDir: {}
    name: output-volume
---
apiVersion: v1
kind: Service
metadata:
  labels:
    component: sonobuoy
    run: sonobuoy-master
  name: sonobuoy-master
  namespace: {{.Namespace}}
spec:
  ports:
  - port: 8080
    protocol: TCP
    targetPort: 8080
  selector:
    run: sonobuoy-master
  type: ClusterIP
`)
