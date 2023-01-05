// Package table define all table types methods.
// base.go contains basic methods of table types.
package table

import (
	"fmt"
	"strings"
)

// base struct contains common attributes to the table.
// Columns: Table columns
// border: Control the table border display(true: print table border).
// tableType: Use to record table types
// End: Used to set the ending. The default is newline "\n".
type base struct {
	Columns   *Set
	border    bool
	tableType string
	End       string
}

func createTableBase(columns *Set, tableType string, border bool) *base {
	b := new(base)
	b.Columns = columns
	b.tableType = tableType
	b.border = border
	b.End = "\n"
	return b
}

// Type method returns a table type string.
func (b *base) Type() string {
	return b.tableType
}

// SetDefault method used to set default value for a given column name.
func (b *base) SetDefault(column string, defaultValue string) {
	for _, head := range b.Columns.base {
		if head.Original() == column {
			head.SetDefault(defaultValue)
			break
		}
	}
}

// IsSimpleTable method returns a bool value indicate the table type is simpleTableType.
func (b *base) IsSimpleTable() bool {
	return b.tableType == simpleTableType
}

// IsSafeTable method returns a bool value indicate the table type is safeTableType.
func (b *base) IsSafeTable() bool {
	return b.tableType == safeTableType
}

// GetDefault method returns default value with a designated column name.
func (b *base) GetDefault(column string) string {
	for _, col := range b.Columns.base {
		if col.Original() == column {
			return col.Default()
		}
	}
	return ""
}

// DropDefault method used to delete default value for designated column.
func (b *base) DropDefault(column string) {
	b.SetDefault(column, "")
}

// GetDefaults method return a map that contains all default value of each column.
// * map[column name] = default value
func (b *base) GetDefaults() map[string]string {
	defaults := make(map[string]string)
	for _, column := range b.Columns.base {
		defaults[column.Original()] = column.Default()
	}
	return defaults
}

func (b *base) end(content string) string {
	content = content[:len(content)-1]
	content += b.End
	return content
}

// GetColumns method return a list of string that contains all column names.
func (b *base) GetColumns() []string {
	columns := make([]string, 0)
	for _, col := range b.Columns.base {
		columns = append(columns, col.Original())
	}
	return columns
}

func (b *base) header() []string {
	resultList := make([]string, 0)
	resultList = append(resultList, fmt.Sprintf("TableType:%s", b.tableType))
	resultList = append(resultList, fmt.Sprintf("Border:%v", b.border))

	columns := make([]string, 0)
	for _, column := range b.Columns.base {
		columns = append(columns, column.Original())
	}
	resultList = append(resultList, fmt.Sprintf("Column:[%s]", strings.Join(columns, ",")))
	return resultList
}
