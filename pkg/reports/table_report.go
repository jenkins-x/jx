package reports

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
)

type TableBarReport struct {
	Table table.Table
}

func NewTableBarReport(table table.Table, legends ...string) *TableBarReport {
	table.AddRow(legends...)

	return &TableBarReport{
		Table: table,
	}
}

func (t *TableBarReport) AddText(name string, value string) {
	t.Table.AddRow(name, value)
}

func (t *TableBarReport) AddNumber(name string, value int) {
	ReportAddNumber(t, name, value)
}

func (t *TableBarReport) Render() error {
	t.Table.Render()
	return nil
}
