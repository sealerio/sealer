package table

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/liushuochen/gotable/cell"
	"github.com/liushuochen/gotable/exception"
	"github.com/liushuochen/gotable/util"
	"os"
	"strings"
)

const (
	C       = cell.AlignCenter
	L       = cell.AlignLeft
	R       = cell.AlignRight
	Default = "__DEFAULT__"
)

// Table struct:
// - Columns: Save the table columns.
// - Row: Save the list of column and value mapping.
// - border: A flag indicates whether to print table border. If the value is `true`, table will show its border.
//           Default is true(By CreateTable function).
// - tableType: Type of table.
type Table struct {
	*base
	Row []map[string]cell.Cell
}

// CreateTable function returns a pointer of Table.
func CreateTable(set *Set) *Table {
	return &Table{
		base: createTableBase(set, simpleTableType, true),
		Row:  make([]map[string]cell.Cell, 0),
	}
}

// Clear the table. The table is cleared of all data.
func (tb *Table) Clear() {
	tb.Columns.Clear()
	tb.Row = make([]map[string]cell.Cell, 0)
}

// AddColumn method used to add a new column for table. It returns an error when column has been existed.
func (tb *Table) AddColumn(column string) error {
	err := tb.Columns.Add(column)
	if err != nil {
		return err
	}

	// Modify exist value, add new column.
	for _, row := range tb.Row {
		row[column] = cell.CreateEmptyData()
	}
	return nil
}

// AddRow method support Map and Slice argument.
// For Map argument, you must put the data from each row into a Map and use column-data as key-value pairs. If the Map
//   does not contain a column, the table sets it to the default value. If the Map contains a column that does not
//   exist, the AddRow method returns an error.
// For Slice argument, you must ensure that the slice length is equal to the column length. Method will automatically
//   map values in Slice and columns. The default value cannot be omitted and must use gotable.Default constant.
// Return error types:
//   - *exception.UnsupportedRowTypeError: It returned when the type of the argument is not supported.
//   - *exception.RowLengthNotEqualColumnsError: It returned if the argument is type of the Slice but the length is
//       different from the length of column.
//   - *exception.ColumnDoNotExistError: It returned if the argument is type of the Map but contains a nonexistent
//       column as a key.
func (tb *Table) AddRow(row interface{}) error {
	switch v := row.(type) {
	case []string:
		return tb.addRowFromSlice(v)
	case map[string]string:
		return tb.addRowFromMap(v)
	default:
		return exception.UnsupportedRowType(v)
	}
}

func (tb *Table) addRowFromSlice(row []string) error {
	rowLength := len(row)
	if rowLength != tb.Columns.Len() {
		return exception.RowLengthNotEqualColumns(rowLength, tb.Columns.Len())
	}

	rowMap := make(map[string]string, 0)
	for i := 0; i < rowLength; i++ {
		if row[i] == Default {
			rowMap[tb.Columns.base[i].Original()] = tb.Columns.base[i].Default()
		} else {
			rowMap[tb.Columns.base[i].Original()] = row[i]
		}
	}

	tb.Row = append(tb.Row, toRow(rowMap))
	return nil
}

func (tb *Table) addRowFromMap(row map[string]string) error {
	for key := range row {
		if !tb.Columns.Exist(key) {
			return exception.ColumnDoNotExist(key)
		}

		// add row by const `DEFAULT`
		if row[key] == Default {
			row[key] = tb.Columns.Get(key).Default()
		}
	}

	// Add default value
	for _, col := range tb.Columns.base {
		_, ok := row[col.Original()]
		if !ok {
			row[col.Original()] = col.Default()
		}
	}

	tb.Row = append(tb.Row, toRow(row))
	return nil
}

// AddRows used to add a slice of rows maps. It returns a slice of map which add failed.
func (tb *Table) AddRows(rows []map[string]string) []map[string]string {
	failure := make([]map[string]string, 0)
	for _, row := range rows {
		err := tb.AddRow(row)
		if err != nil {
			failure = append(failure, row)
		}
	}
	return failure
}

// String method used to implement fmt.Stringer.
func (tb *Table) String() string {
	columnMaxLength := make(map[string]int)
	tag := make(map[string]cell.Cell)
	taga := make([]map[string]cell.Cell, 0)
	for _, h := range tb.Columns.base {
		columnMaxLength[h.Original()] = h.Length()
		tag[h.String()] = cell.CreateData("-")
	}

	for _, data := range tb.Row {
		for _, h := range tb.Columns.base {
			maxLength := max(h.Length(), data[h.Original()].Length())
			maxLength = max(maxLength, columnMaxLength[h.Original()])
			columnMaxLength[h.Original()] = maxLength
		}
	}

	content := ""
	icon := " "
	// Print first line.
	taga = append(taga, tag)
	if tb.border {
		content += tb.printGroup(taga, columnMaxLength)
		icon = "|"
	}

	// Print table column.
	for index, column := range tb.Columns.base {
		itemLen := columnMaxLength[column.Original()]
		if tb.border {
			itemLen += 2
		}
		s := ""
		switch column.Align() {
		case R:
			s, _ = right(column, itemLen, " ")
		case L:
			s, _ = left(column, itemLen, " ")
		default:
			s, _ = center(column, itemLen, " ")
		}
		if index == 0 {
			s = icon + s + icon
		} else {
			s = "" + s + icon
		}

		content += s
	}

	if tb.border {
		content += "\n"
	}

	tableValue := taga
	if !tb.Empty() {
		for _, row := range tb.Row {
			value := make(map[string]cell.Cell)
			for key := range row {
				col := tb.Columns.Get(key)
				value[col.String()] = row[key]
			}
			tableValue = append(tableValue, value)
		}
		tableValue = append(tableValue, tag)
	}

	content += tb.printGroup(tableValue, columnMaxLength)
	return tb.end(content)
}

// Empty method is used to determine whether the table is empty.
func (tb *Table) Empty() bool {
	return tb.Length() == 0
}

// Length method returns an integer indicates the length of the table row.
func (tb *Table) Length() int {
	return len(tb.Row)
}

func (tb *Table) GetValues() []map[string]string {
	values := make([]map[string]string, 0)
	for _, value := range tb.Row {
		ms := make(map[string]string)
		for k, v := range value {
			ms[k] = v.String()
		}
		values = append(values, ms)
	}
	return values
}

func (tb *Table) Exist(value map[string]string) bool {
	for _, row := range tb.Row {
		exist := true
		for key := range value {
			v, ok := row[key]
			if !ok || v.String() != value[key] {
				exist = false
				break
			}
		}
		if exist {
			return exist
		}
	}
	return false
}

func (tb *Table) json(indent int) ([]byte, error) {
	data := make([]map[string]string, 0)
	for _, row := range tb.Row {
		element := make(map[string]string)
		for col, value := range row {
			element[col] = value.String()
		}
		data = append(data, element)
	}

	if indent < 0 {
		indent = 0
	}
	elems := make([]string, 0)
	for i := 0; i < indent; i++ {
		elems = append(elems, " ")
	}

	return json.MarshalIndent(data, "", strings.Join(elems, " "))
}

// The JSON method returns the JSON string corresponding to the gotable. The indent argument represents the indent
// value. If index is less than zero, the JSON method treats it as zero.
func (tb *Table) JSON(indent int) (string, error) {
	bytes, err := tb.json(indent)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// The XML method returns the XML format string corresponding to the gotable. The indent argument represents the indent
// value. If index is less than zero, the XML method treats it as zero.
func (tb *Table) XML(indent int) string {
	if indent < 0 {
		indent = 0
	}
	indents := make([]string, indent)
	for i := 0; i < indent; i++ {
		indents[i] = " "
	}
	indentString := strings.Join(indents, "")

	contents := []string{"<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>", "<table>"}
	for _, row := range tb.Row {
		contents = append(contents, indentString+"<row>")
		for name := range row {
			line := indentString + indentString + fmt.Sprintf("<%s>%s</%s>", name, row[name], name)
			contents = append(contents, line)
		}
		contents = append(contents, indentString+"</row>")
	}
	contents = append(contents, "</table>")
	content := strings.Join(contents, "\n")
	return content
}

func (tb *Table) CloseBorder() {
	tb.border = false
}

func (tb *Table) OpenBorder() {
	tb.border = true
}

func (tb *Table) Align(column string, mode int) {
	for _, h := range tb.Columns.base {
		if h.Original() == column {
			h.SetAlign(mode)
			return
		}
	}
}

func (tb *Table) ToJsonFile(path string, indent int) error {
	if !util.IsJsonFile(path) {
		return fmt.Errorf("%s: not a regular json file", path)
	}

	bytes, err := tb.json(indent)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	_, err = file.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func (tb *Table) ToCSVFile(path string) error {
	if !util.IsCSVFile(path) {
		return exception.NotARegularCSVFile(path)
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	writer := csv.NewWriter(file)

	contents := make([][]string, 0)
	columns := tb.GetColumns()
	contents = append(contents, columns)
	for _, value := range tb.GetValues() {
		content := make([]string, 0)
		for _, col := range columns {
			content = append(content, value[col])
		}
		contents = append(contents, content)
	}

	err = writer.WriteAll(contents)
	if err != nil {
		return err
	}
	writer.Flush()
	err = writer.Error()
	if err != nil {
		return err
	}
	return nil
}

func (tb *Table) HasColumn(column string) bool {
	for index := range tb.Columns.base {
		if tb.Columns.base[index].Original() == column {
			return true
		}
	}
	return false
}

func (tb *Table) EqualColumns(other *Table) bool {
	return tb.Columns.Equal(other.Columns)
}

func (tb *Table) SetColumnColor(columnName string, display, fount, background int) {
	background += 10
	for _, col := range tb.Columns.base {
		if col.Original() == columnName {
			col.SetColor(display, fount, background)
			break
		}
	}
}

// GoString method used to implement fmt.GoStringer.
func (tb *Table) GoString() string {
	resultList := tb.header()
	values := make([]string, 0)
	for _, row := range tb.Row {
		value := make([]string, 0)
		for _, column := range tb.Columns.base {
			v, ok := row[column.Original()]
			if !ok {
				value = append(value, column.Default())
			} else {
				value = append(value, v.Original())
			}
		}
		values = append(values, strings.Join(value, ","))
	}

	resultList = append(resultList, fmt.Sprintf("Row:[%v]", strings.Join(values, "; ")))
	return fmt.Sprintf("{%s}", strings.Join(resultList, "; "))
}
