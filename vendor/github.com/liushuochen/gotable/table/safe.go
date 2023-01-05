package table

import (
	"github.com/liushuochen/gotable/cell"
	"github.com/liushuochen/gotable/exception"
	"sync"
)

type SafeTable struct {
	*base
	Row []sync.Map
}

// CreateSafeTable returns a pointer of SafeTable.
func CreateSafeTable(set *Set) *SafeTable {
	return &SafeTable{
		base: createTableBase(set, safeTableType, true),
		Row:  make([]sync.Map, 0),
	}
}

// Clear the table. The table is cleared of all data.
func (st *SafeTable) Clear() {
	st.Columns.Clear()
	st.Row = make([]sync.Map, 0)
}

// AddColumn method used to add a new column for table. It returns an error when column has been existed.
func (st *SafeTable) AddColumn(column string) error {
	err := st.Columns.Add(column)
	if err != nil {
		return err
	}

	for index := range st.Row {
		st.Row[index].Store(column, cell.CreateEmptyData())
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
func (st *SafeTable) AddRow(row interface{}) error {
	switch v := row.(type) {
	case []string:
		return st.addRowFromSlice(v)
	case map[string]string:
		return st.addRowFromMap(v)
	default:
		return exception.UnsupportedRowType(v)
	}
}

// AddRows used to add a slice of rows map. It returns a slice of map which add failed.
func (st *SafeTable) AddRows(rows []map[string]string) []map[string]string {
	failure := make([]map[string]string, 0)
	for _, row := range rows {
		err := st.AddRow(row)
		if err != nil {
			failure = append(failure, row)
		}
	}
	return failure
}

func (st *SafeTable) addRowFromMap(row map[string]string) error {
	for key := range row {
		if !st.Columns.Exist(key) {
			return exception.ColumnDoNotExist(key)
		}

		// add row by const `DEFAULT`
		if row[key] == Default {
			row[key] = st.Columns.Get(key).Default()
		}
	}

	// Add default value
	for _, col := range st.Columns.base {
		_, ok := row[col.Original()]
		if !ok {
			row[col.Original()] = col.Default()
		}
	}

	st.Row = append(st.Row, toSafeRow(row))
	return nil
}

func (st *SafeTable) addRowFromSlice(row []string) error {
	rowLength := len(row)
	if rowLength != st.Columns.Len() {
		return exception.RowLengthNotEqualColumns(rowLength, st.Columns.Len())
	}

	rowMap := make(map[string]string, 0)
	for i := 0; i < rowLength; i++ {
		if row[i] == Default {
			rowMap[st.Columns.base[i].Original()] = st.Columns.base[i].Default()
		} else {
			rowMap[st.Columns.base[i].Original()] = row[i]
		}
	}

	st.Row = append(st.Row, toSafeRow(rowMap))
	return nil
}

// Length method returns an integer indicates the length of the table row.
func (st *SafeTable) Length() int {
	return len(st.Row)
}

// Empty method is used to determine whether the table is empty.
func (st *SafeTable) Empty() bool {
	return st.Length() == 0
}

// String method used to implement fmt.Stringer.
func (st *SafeTable) String() string {
	var (
		// Variable columnMaxLength original mode is map[string]int
		columnMaxLength sync.Map

		// Variable tag original mode is map[string]cell.Cell
		tag sync.Map

		// Variable taga original mode is []map[string]cell.Cell
		taga = make([]*sync.Map, 0)
	)

	for _, col := range st.Columns.base {
		columnMaxLength.Store(col.Original(), col.Length())
		tag.Store(col.String(), cell.CreateData("-"))
	}

	for index := range st.Row {
		for _, col := range st.Columns.base {
			value, _ := st.Row[index].Load(col.Original())
			maxLength := max(col.Length(), value.(cell.Cell).Length())

			value, _ = columnMaxLength.Load(col.Original())
			maxLength = max(maxLength, value.(int))
			columnMaxLength.Store(col.Original(), maxLength)
		}
	}

	content := ""
	icon := " "
	// Print first line.
	taga = append(taga, &tag)
	if st.border {
		content += st.printGroup(taga, &columnMaxLength)
		icon = "|"
	}

	// Print table column.
	for index, column := range st.Columns.base {
		value, _ := columnMaxLength.Load(column.Original())
		itemLen := value.(int)
		if st.border {
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

	if st.border {
		content += "\n"
	}

	tableValue := taga
	if !st.Empty() {
		for index := range st.Row {
			// map[string]cell.Cell
			var cellMap sync.Map
			f := func(key, value interface{}) bool {
				column := st.Columns.Get(key.(string))
				cellMap.Store(column.String(), value.(cell.Cell))
				return true
			}
			st.Row[index].Range(f)
			tableValue = append(tableValue, &cellMap)
		}
		tableValue = append(tableValue, &tag)
	}

	content += st.printGroup(tableValue, &columnMaxLength)
	return st.end(content)
}
