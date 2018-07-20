package kube

func ClusterRoleYaml(user string) string {
	return `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    jx.liggitt.net/version: v0.5.0
  labels:
    jx.liggitt.net/generated: "true"
    jx.liggitt.net/user: ` + user + `
  name: jx:` + user + `
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  - limitranges
  - namespaces
  - persistentvolumeclaims
  - persistentvolumes
  - podtemplates
  - replicationcontrollers
  - resourcequotas
  - services
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - endpoints
  - serviceaccounts
  verbs:
  - create
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
  - get
  - patch
  - update
- apiGroups:
  - ""
  resourceNames:
  - ` + user + `
  resources:
  - nodes
  - nodes/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ""
  resources:
  - nodes
  - pods
  - secrets
  verbs:
  - create
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - persistentvolumes/status
  - pods/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ""
  resources:
  - pods/binding
  verbs:
  - create
- apiGroups:
  - admissionregistration.k8s.io
  resources:
  - initializerconfigurations
  - mutatingwebhookconfigurations
  - validatingwebhookconfigurations
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - apps
  resources:
  - controllerrevisions
  - daemonsets
  - deployments
  - replicasets
  - statefulsets
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - autoscaling
  resources:
  - horizontalpodautoscalers
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - batch
  resources:
  - cronjobs
  - jobs
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - batch
  resourceNames:
  - expose
  resources:
  - jobs/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - certificates.k8s.io
  resources:
  - certificatesigningrequests
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - events.k8s.io
  resources:
  - events
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - extensions
  resources:
  - daemonsets
  - deployments
  - ingresses
  - networkpolicies
  - podsecuritypolicies
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - extensions
  resources:
  - deployments/status
  - replicasets/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - extensions
  resources:
  - replicasets
  verbs:
  - create
  - get
  - list
  - watch
- apiGroups:
  - jenkins.io
  resources:
  - environments
  - gitservices
  - pipelineactivities
  - pipelines
  - releases
  - runs
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - networking.k8s.io
  resources:
  - networkpolicies
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - policy
  resources:
  - poddisruptionbudgets
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - rbac.authorization.k8s.io
  resources:
  - clusterrolebindings
  - clusterroles
  - rolebindings
  - roles
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - scheduling.k8s.io
  resources:
  - priorityclasses
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - settings.k8s.io
  resources:
  - podpresets
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - storage.k8s.io
  resources:
  - storageclasses
  verbs:
  - create
  - get
  - list
  - watch
- apiGroups:
  - storage.k8s.io
  resourceNames:
  - standard
  resources:
  - storageclasses
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - storage.k8s.io
  resources:
  - volumeattachments
  verbs:
  - get
  - list
  - watch`
}

func RoleKubeSystemYaml(user string) string {
	return `apiVersion: rbac.authorization.k8s.io/v1
	  kind: Role
	  metadata:
	    annotations:
	      jx.liggitt.net/version: v0.5.0
	    labels:
	      jx.liggitt.net/generated: "true"
	      jx.liggitt.net/user: ` + user + `
	    name: jx:` + user + `
	    namespace: kube-system
	  rules:
	  - apiGroups:
	    - apps
	    resources:
	    - configmaps
	    verbs:
	    - create
	  - apiGroups:
	    - apps
	    resourceNames:
	    - nginx-load-balancer-conf
	    resources:
	    - configmaps
	    verbs:
	    - get
	    - patch
	    - update
	  - apiGroups:
	    - apps
	    resourceNames:
	    - kube-system
	    resources:
	    - namespaces
	    verbs:
	    - get
	    - patch
	    - update
	  - apiGroups:
	    - ""
	    resources:
	    - replicationcontrollers
	    - services
	    verbs:
	    - create
	    - get
	    - patch
	    - update
	  - apiGroups:
	    - ""
	    resources:
	    - replicationcontrollers/status
	    verbs:
	    - get
	    - patch
	    - update
	  - apiGroups:
	    - apps
	    resources:
	    - deployments
	    verbs:
	    - create
	  - apiGroups:
	    - apps
	    resourceNames:
	    - kubernetes-dashboard
	    resources:
	    - deployments
	    verbs:
	    - get
	    - patch
	    - update
	  - apiGroups:
	    - extensions
	    resources:
	    - deployments
	    verbs:
	    - create
	  - apiGroups:
	    - extensions
	    resourceNames:
	    - kube-dns
	    resources:
	    - deployments
	    verbs:
	    - get
	    - patch
	    - update`
}

func RoleBindingKubeSystemYaml(user string) string {
	return `apiVersion: rbac.authorization.k8s.io/v1
	kind: RoleBinding
	metadata:
	  annotations:
	    jx.liggitt.net/version: v0.5.0
	  labels:
	    jx.liggitt.net/generated: "true"
	    jx.liggitt.net/user: ` + user + `
	  name: jx:` + user + `
	  namespace: kube-system
	roleRef:
	  apiGroup: rbac.authorization.k8s.io
	  kind: Role
	  name: jx:` + user + `
	subjects:
	- apiGroup: rbac.authorization.k8s.io
	  kind: User
	  name: ` + user
}

func ClusterRoleBindingYaml(user string) string {
	return `apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRoleBinding
		metadata:
		  annotations:
		    jx.liggitt.net/version: v0.5.0
		  labels:
		    jx.liggitt.net/generated: "true"
		    jx.liggitt.net/user: ` + user + `
		  name: jx:` + user + `
		roleRef:
		  apiGroup: rbac.authorization.k8s.io
		  kind: ClusterRole
		  name: jx:` + user + `
		subjects:
		- apiGroup: rbac.authorization.k8s.io
		  kind: User
		  name: ` + user + ``
}
