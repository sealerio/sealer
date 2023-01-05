# gotable 5: Safe
Generate beautiful ASCII tables.

```go
package main

import (
	"fmt"
	"github.com/liushuochen/gotable"
)

func main() {
	table, err := gotable.Create("version", "description")
	if err != nil {
		fmt.Println("Create table failed: ", err.Error())
		return
	}

	table.AddRow([]string{"gotable 5", "Safe: New table type to enhance concurrency security"})
	table.AddRow([]string{"gotable 4", "Colored: Print colored column"})
	table.AddRow([]string{"gotable 3", "Storage: Store the table data as a file"})
	table.AddRow([]string{"gotable 2", "Simple: Use simpler APIs to control table"})
	table.AddRow([]string{"gotable 1", "Gotable: Print a beautiful ASCII table"})

	fmt.Println(table)
}

```

```text
+-----------+------------------------------------------------------+
|  version  |                     description                      |
+-----------+------------------------------------------------------+
| gotable 5 | Safe: New table type to enhance concurrency security |
| gotable 4 |            Colored: Print colored column             |
| gotable 3 |       Storage: Store the table data as a file        |
| gotable 2 |      Simple: Use simpler APIs to control table       |
| gotable 1 |        Gotable: Print a beautiful ASCII table        |
+-----------+------------------------------------------------------+

```


## Reference
Please refer to guide: 
[gotable guide](https://blog.csdn.net/TCatTime/article/details/103068260#%E8%8E%B7%E5%8F%96gotable)


## Supported character set
* ASCII
* Chinese characters


## API
Please refer to '[gotable APIs](doc/api.md)' for more gotable API information.


## Demo
Please refer to [gotable demo page](doc/demo.md) for more demos code.


## Error type
Please refer to this guide '[error type](doc/errors.md)' for more gotable error information.
