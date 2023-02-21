# Go-wildcard

[![Go Report Card](https://goreportcard.com/badge/github.com/IGLOU-EU/go-wildcard)](https://goreportcard.com/report/github.com/IGLOU-EU/go-wildcard)
[![Go Reference](https://img.shields.io/badge/api-reference-blue)](https://pkg.go.dev/github.com/IGLOU-EU/go-wildcard)
[![Go coverage](https://gocover.io/_badge/github.com/IGLOU-EU/go-wildcard)](https://gocover.io/github.com/IGLOU-EU/go-wildcard)
[![Apache V2 License](https://img.shields.io/badge/license-Apache%202-blue)](https://opensource.org/licenses/MIT)

>Go-wildcard is forked from [Minio project](https://github.com/minio/minio)   
>https://github.com/minio/minio/tree/master/pkg/wildcard

## Why
This part of Minio project, a very cool, fast and light wildcard pattern matching.    

Originally the purpose of this fork is to give access to this "lib" under Apache license, without importing the entire Minio project ...

Two function are available `MatchSimple` and `Match`   
- `MatchSimple` only covert `*` usage (which is faster than `Match`)
- `Match` support full wildcard matching, `*` and `?`

I know Regex, but they are much more complex, and slower (even prepared regex) ...   
I know Glob, but most of the time, I only need simple wildcard matching.   

This library remains under Apache License Version 2.0, but MinIO project is 
migrated to GNU Affero General Public License 3.0 or later from 
https://github.com/minio/minio/commit/069432566fcfac1f1053677cc925ddafd750730a

## How to
>⚠️ WARNING: Unlike the GNU "libc", this library have no equivalent to "FNM_FILE_NAME". To do this you can use "path/filepath" https://pkg.go.dev/path/filepath#Glob

Using this fork
```sh
go get github.com/IGLOU-EU/go-wildcard@latest
```

Using Official Minio (GNU Affero General Public License 3.0 or later)
>From https://github.com/minio/minio/commit/81d5688d5684bd4d93e7bb691af8cf555a20c28c the minio pkg are moved to https://github.com/minio/pkg     
```sh
go get github.com/minio/pkg/wildcard@latest
```

## Quick Example

This example shows a Go file with pattern matching ...  
```go
package main

import (
	"fmt"

	wildcard "github.com/IGLOU-EU/go-wildcard"
)

func main() {
    str := "daaadabadmanda"
    
    pattern := "da*da*da*"
    result := wildcard.MatchSimple(pattern, str)
	fmt.Println(str, pattern, result)

    pattern = "?a*da*d?*"
    result = wildcard.Match(pattern, str)
	fmt.Println(str, pattern, result)
}
```
