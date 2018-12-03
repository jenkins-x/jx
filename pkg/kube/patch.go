package kube

import (
	"encoding/json"
)

// PatchRow used to generate the patch JSON for patching
type PatchRow struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// CreatePatchBytes creates a kubernetes PATCH block
func CreatePatchBytes(op string, path string, value interface{}) ([]byte, error) {
	payload := &PatchRow{
		Op:    op,
		Path:  path,
		Value: value,
	}
	return json.Marshal(payload)
}
