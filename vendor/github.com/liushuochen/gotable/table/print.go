// Package table define all table types methods.
// print.go used to control table printing.
package table

import (
	"fmt"
	"github.com/liushuochen/gotable/cell"
	"sync"
)

// This method print part of table data in STDOUT. It will be called twice in *table.PrintTable method.
// Arguments:
//   group: 		A map that storage column as key, data as value. Data is either "-" or row, if the value of data is
//                  "-", the printGroup method will print the border of the table.
//   columnMaxLen:  A map that storage column as key, max length of cell of column as value.
func (tb *Table) printGroup(group []map[string]cell.Cell, columnMaxLen map[string]int) string {
	result := ""
	for _, item := range group {
		for index, column := range tb.Columns.base {
			itemLen := columnMaxLen[column.Original()]
			if tb.border {
				itemLen += 2
			}
			s := ""
			icon := "|"
			if item[column.String()].String() == "-" {
				if tb.border {
					s, _ = center(item[column.String()], itemLen, "-")
					icon = "+"
				} else {
					icon = " "
				}
			} else {
				switch column.Align() {
				case R:
					s, _ = right(item[column.String()], itemLen, " ")
				case L:
					s, _ = left(item[column.String()], itemLen, " ")
				default:
					s, _ = center(item[column.String()], itemLen, " ")
				}
			}

			if index == 0 {
				s = icon + s + icon
			} else {
				s = "" + s + icon
			}
			result += s
		}
		result += "\n"
	}
	return result
}

func (st *SafeTable) printGroup(group []*sync.Map, columnMaxLen *sync.Map) string {
	result := ""
	for _, item := range group {
		for index, column := range st.Columns.base {
			value, _ := columnMaxLen.Load(column.Original())
			itemLen := value.(int)
			if st.border {
				itemLen += 2
			}
			s := ""
			icon := "|"
			value, _ = item.Load(column.String())
			if value.(cell.Cell).String() == "-" {
				if st.border {
					s, _ = center(value.(cell.Cell), itemLen, "-")
					icon = "+"
				} else {
					icon = " "
				}
			} else {
				switch column.Align() {
				case R:
					s, _ = right(value.(cell.Cell), itemLen, " ")
				case L:
					s, _ = left(value.(cell.Cell), itemLen, " ")
				default:
					s, _ = center(value.(cell.Cell), itemLen, " ")
				}
			}

			if index == 0 {
				s = icon + s + icon
			} else {
				s = "" + s + icon
			}
			result += s
		}
		result += "\n"
	}
	return result
}

func max(x, y int) int {
	if x >= y {
		return x
	}
	return y
}

func center(c cell.Cell, length int, fillchar string) (string, error) {
	if len(fillchar) != 1 {
		err := fmt.Errorf("the fill character must be exactly one" +
			" character long")
		return "", err
	}

	if c.Length() >= length {
		return c.String(), nil
	}

	result := ""
	if isEvenNumber(length - c.Length()) {
		front := ""
		for i := 0; i < ((length - c.Length()) / 2); i++ {
			front = front + fillchar
		}

		result = front + c.String() + front
	} else {
		front := ""
		for i := 0; i < ((length - c.Length() - 1) / 2); i++ {
			front = front + fillchar
		}

		behind := front + fillchar
		result = front + c.String() + behind
	}
	return result, nil
}

func left(c cell.Cell, length int, fillchar string) (string, error) {
	if len(fillchar) != 1 {
		err := fmt.Errorf("the fill character must be exactly one" +
			" character long")
		return "", err
	}

	result := c.String() + block(length-c.Length())
	return result, nil
}

func right(c cell.Cell, length int, fillchar string) (string, error) {
	if len(fillchar) != 1 {
		err := fmt.Errorf("the fill character must be exactly one" +
			" character long")
		return "", err
	}

	result := block(length-c.Length()) + c.String()
	return result, nil
}

func block(length int) string {
	result := ""
	for i := 0; i < length; i++ {
		result += " "
	}
	return result
}

func isEvenNumber(number int) bool {
	if number%2 == 0 {
		return true
	}
	return false
}

func toRow(value map[string]string) map[string]cell.Cell {
	row := make(map[string]cell.Cell)
	for k, v := range value {
		row[k] = cell.CreateData(v)
	}
	return row
}

func toSafeRow(value map[string]string) sync.Map {
	var row sync.Map
	for k, v := range value {
		row.Store(k, cell.CreateData(v))
	}
	return row
}
