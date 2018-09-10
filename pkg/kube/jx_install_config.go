package kube

// JXInstallConfig is the struct used to create the jx-install-config configmap
type JXInstallConfig struct {
	Server string `structs:"server" yaml:"server" json:"server"`
	CA     []byte `structs:"ca.crt" yaml:"ca.crt" json:"ca.crt"`
}
