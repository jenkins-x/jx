package cve

import "github.com/jenkins-x/jx/pkg/jx/cmd/table"

type CVEQuery struct {
	ImageName   string
	ImageID     string
	Vesion      string
	Environment string
}
type CVEProvider interface {
	GetImageVulnerabilityTable(table *table.Table, query CVEQuery) error
}
