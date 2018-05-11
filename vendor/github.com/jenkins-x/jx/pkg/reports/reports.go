package reports

import "strconv"

type BarReport interface {
	AddText(name string, value string)
	AddNumber(name string, value int)

	Render() error
}

func ReportAddNumber(report BarReport, name string, value int) {
	report.AddText(name, strconv.Itoa(value))
}
