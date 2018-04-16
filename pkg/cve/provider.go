package cve

import "github.com/jenkins-x/jx/pkg/jx/cmd/table"

type CVEProvider interface {
	GetImageVulnerabilityTable(table *table.Table, image string) error
}
