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

package daemonset

import (
	"github.com/heptio/sonobuoy/pkg/templates"
)

var daemonSetTemplate = templates.NewTemplate("daemonTemplate", `
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  annotations:
    sonobuoy-driver: DaemonSet
    sonobuoy-plugin: {{.PluginName}}
    sonobuoy-result-type: {{.ResultType}}
  labels:
    component: sonobuoy
    sonobuoy-run: '{{.SessionID}}'
    tier: analysis
  name: sonobuoy-{{.PluginName}}-daemon-set-{{.SessionID}}
  namespace: '{{.Namespace}}'
spec:
  selector:
    matchLabels:
      sonobuoy-run: '{{.SessionID}}'
  template:
    metadata:
      labels:
        component: sonobuoy
        sonobuoy-run: '{{.SessionID}}'
        tier: analysis
    spec:
      containers:
      - {{.ProducerContainer | indent 8}}
      - command: ["/run_single_node_worker.sh"]
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: RESULTS_DIR
          value: /tmp/results
        - name: MASTER_URL
          value: '{{.MasterAddress}}'
        - name: RESULT_TYPE
          value: {{.ResultType}}
        - name: CA_CERT
          value: |
            {{.CACert | indent 12}}
        - name: CLIENT_CERT
          valueFrom:
            secretKeyRef:
              name: {{.SecretName}}
              key: tls.crt

        - name: CLIENT_KEY
          valueFrom:
            secretKeyRef:
              name: {{.SecretName}}
              key: tls.key
        image: {{.SonobuoyImage}}
        imagePullPolicy: {{.ImagePullPolicy}}
        name: sonobuoy-worker
        volumeMounts:
        - mountPath: /tmp/results
          name: results
          readOnly: false
      dnsPolicy: ClusterFirstWithHostNet
      hostIPC: true
      hostNetwork: true
      hostPID: true
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
        operator: Exists
      - key: CriticalAddonsOnly
        operator: Exists
      volumes:
      - emptyDir: {}
        name: results
      - hostPath:
          path: /
        name: root
			{{- range .ExtraVolumes }}
      - {{ . | indent 8 }}
      {{- end -}}
`)
