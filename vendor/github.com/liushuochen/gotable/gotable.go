package gotable

import (
	"encoding/csv"
	"encoding/json"
	"github.com/liushuochen/gotable/exception"
	"github.com/liushuochen/gotable/table"
	"github.com/liushuochen/gotable/util"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
)

const (
	Center  = table.C
	Left    = table.L
	Right   = table.R
	Default = table.Default
)

// Colored display control
const (
	TerminalDefault = 0
	Highlight       = 1
	Underline       = 4
	Flash           = 5
)

// Colored control
const (
	Black          = 30
	Red            = 31
	Green          = 32
	Yellow         = 33
	Blue           = 34
	Purple         = 35
	Cyan           = 36
	Write          = 37
	NoneBackground = 0
)

// Create an empty simple table. When duplicate values in columns, table creation fails.
// It will return a table pointer and an error.
// Error:
// - If the length of columns is not greater than 0, an *exception.ColumnsLengthError error is returned.
// - If columns contain duplicate values, an error is returned.
// - Otherwise, the value of error is nil.
func Create(columns ...string) (*table.Table, error) {
	set, err := table.CreateSetFromString(columns...)
	if err != nil {
		return nil, err
	}
	tb := table.CreateTable(set)
	return tb, nil
}

// CreateSafeTable function used to create an empty safe table. When duplicate values in columns, table creation fails.
// It will return a table pointer and an error.
// Error:
// - If the length of columns is not greater than 0, an *exception.ColumnsLengthError error is returned.
// - If columns contain duplicate values, an error is returned.
// - Otherwise, the value of error is nil.
func CreateSafeTable(columns ...string) (*table.SafeTable, error) {
	set, err := table.CreateSetFromString(columns...)
	if err != nil {
		return nil, err
	}
	tb := table.CreateSafeTable(set)
	return tb, nil
}

// CreateByStruct creates an empty table from struct. You can rename a field using struct tag: gotable
// It will return a table pointer and an error.
// Error:
// - If the length of columns is not greater than 0, an *exception.ColumnsLengthError error is returned.
// - If columns contain duplicate values, an error is returned.
// - Otherwise, the value of error is nil.
func CreateByStruct(v interface{}) (*table.Table, error) {
	set := &table.Set{}
	s := reflect.TypeOf(v).Elem()
	numField := s.NumField()
	if numField <= 0 {
		return nil, exception.ColumnsLength()
	}

	for i := 0; i < numField; i++ {
		field := s.Field(i)
		name := field.Tag.Get("gotable")
		if name == "" {
			name = field.Name
		}

		err := set.Add(name)
		if err != nil {
			return nil, err
		}
	}
	tb := table.CreateTable(set)
	return tb, nil
}

// Version
// The version function returns a string representing the version information of the gotable.
// e.g.
//
//	gotable 3.4.0
func Version() string {
	return "gotable " + strings.Join(getVersions(), ".")
}

// Versions returns a list of version numbers.
func Versions() []string { return getVersions() }

// getVersions 5.18.0
func getVersions() []string {
	return []string{"5", "18", "0"}
}

// Read from a csv file to create a *table instance.
// This function is a private function that only called from Read function. It will return a table pointer and an error.
// Error:
// - If the contents of the csv file are empty, an *exception.ColumnsLengthError is returned.
// - If there are duplicate columns in the parse result, an error is returned.
// - Otherwise the value if error is nil.
func readFromCSVFile(file *os.File) (*table.Table, error) {
	reader := csv.NewReader(file)
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(lines) < 1 {
		return Create()
	}

	tb, err := Create(lines[0]...)
	if err != nil {
		return nil, err
	}

	rows := make([]map[string]string, 0)
	for _, line := range lines[1:] {
		row := make(map[string]string)
		for i := range line {
			row[lines[0][i]] = line[i]
		}
		rows = append(rows, row)
	}
	tb.AddRows(rows)
	return tb, nil
}

// Read from a json file to create a *table instance.
// This function is a private function that only called from Read function. It will return a table pointer and an error.
// Error:
//   - If the contents of the json file are not eligible table contents, an *exception.NotGotableJSONFormatError is
//     returned.
//   - If there are duplicate columns in the parse result, an error is returned.
//   - Otherwise the value if error is nil.
func readFromJSONFile(file *os.File) (*table.Table, error) {
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	rows := make([]map[string]string, 0)
	err = json.Unmarshal(byteValue, &rows)
	if err != nil {
		return nil, exception.NotGotableJSONFormat(file.Name())
	}

	columns := make([]string, 0)
	for column := range rows[0] {
		columns = append(columns, column)
	}
	tb, err := Create(columns...)
	if err != nil {
		return nil, err
	}
	tb.AddRows(rows)
	return tb, nil
}

// Read from file to create a *table instance.
// Currently, support csv and json file. It will return a table pointer and an error.
// Error:
//   - If path is not a file, or does not exist, an *exception.FileDoNotExistError is returned.
//   - If path is a JSON file, the contents of the file are not eligible table contents, an
//     *exception.NotGotableJSONFormatError is returned.
//   - If path is a CSV file, and the contents of the file are empty, an *exception.ColumnsLengthError is returned.
//   - If there are duplicate columns in the parse result, an error is returned.
//   - Otherwise the value if error is nil.
func Read(path string) (*table.Table, error) {
	if !util.IsFile(path) {
		return nil, exception.FileDoNotExist(path)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	if util.IsJsonFile(file.Name()) {
		return readFromJSONFile(file)
	}

	if util.IsCSVFile(file.Name()) {
		return readFromCSVFile(file)
	}

	return nil, exception.UnSupportedFileType(file.Name())
}
