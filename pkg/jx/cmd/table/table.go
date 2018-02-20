package table

import (
	"fmt"
	"io"
	"math"
	"strings"
)

const (
	ALIGN_LEFT   = 0
	ALIGN_CENTER = 1
	ALIGN_RIGHT  = 2
)

type Table struct {
	Out          io.Writer
	Rows         [][]string
	ColumnWidths []int
	ColumnAlign  []int
}

func CreateTable(out io.Writer) Table {
	return Table{
		Out: out,
	}
}

func (t *Table) AddRow(col ...string) {
	t.Rows = append(t.Rows, col)
}

func (t *Table) Render() {
	// lets figure out the max widths of each column
	for _, row := range t.Rows {
		for ci, col := range row {
			l := len(col)
			t.ColumnWidths = ensureArrayCanContain(t.ColumnWidths, ci)
			if l > t.ColumnWidths[ci] {
				t.ColumnWidths[ci] = l
			}
		}
	}

	out := t.Out
	for _, row := range t.Rows {
		lastColumn := len(row) - 1
		for ci, col := range row {
			if ci > 0 {
				fmt.Fprint(out, " ")
			}
			l := t.ColumnWidths[ci]
			align := t.GetColumnAlign(ci)
			if ci >= lastColumn && align != ALIGN_CENTER && align != ALIGN_RIGHT {
				fmt.Fprint(out, col)
			} else {
				fmt.Fprint(out, pad(col, " ", l, align))
			}
		}
		fmt.Fprint(out, "\n")
	}
}

// SetColumnsAligns sets the alignment of the columns
func (t *Table) SetColumnsAligns(colAligns []int) {
	t.ColumnAlign = colAligns
}

// GetColumnAlign return the column alignment
func (t *Table) GetColumnAlign(i int) int {
	t.ColumnAlign = ensureArrayCanContain(t.ColumnAlign, i)
	return t.ColumnAlign[i]
}

// SetColumnAlign sets the column alignment for the given column index
func (t *Table) SetColumnAlign(i int, align int) {
	t.ColumnAlign = ensureArrayCanContain(t.ColumnAlign, i)
	t.ColumnAlign[i] = align
}

func ensureArrayCanContain(array []int, idx int) []int {
	diff := idx + 1 - len(array)
	for i := 0; i < diff; i++ {
		array = append(array, 0)
	}
	return array
}

func pad(s, pad string, width int, align int) string {
	switch align {
	case ALIGN_CENTER:
		return padCenter(s, pad, width)
	case ALIGN_RIGHT:
		return padLeft(s, pad, width)
	default:
		return padRight(s, pad, width)
	}
}

func padRight(s, pad string, width int) string {
	gap := width - len(s)
	if gap > 0 {
		return s + strings.Repeat(string(pad), gap)
	}
	return s
}

func padLeft(s, pad string, width int) string {
	gap := width - len(s)
	if gap > 0 {
		return strings.Repeat(string(pad), gap) + s
	}
	return s
}

func padCenter(s, pad string, width int) string {
	gap := width - len(s)
	if gap > 0 {
		gapLeft := int(math.Ceil(float64(gap / 2)))
		gapRight := gap - gapLeft
		return strings.Repeat(string(pad), gapLeft) + s + strings.Repeat(string(pad), gapRight)
	}
	return s
}
