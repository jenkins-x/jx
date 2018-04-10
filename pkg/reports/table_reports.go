package reports

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
)

type TableBarReport struct {
	Table table.Table
}

func NewTableBarReport(table table.Table) TableBarReport {
	table.AddRow("Name", "Value")

	return TableBarReport{
		Table: table,
	}
}

func (t *TableBarReport) AddText(name string, value string) {
	t.Table.AddRow(name, value)
}

func (t *TableBarReport) AddNumber(name string, value int) {
	ReportAddNumber(t, name, value)
}

func (t *TableBarReport) Render() {
	t.Table.Render()

}
