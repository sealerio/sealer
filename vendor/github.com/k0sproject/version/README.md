# version

A go-language package for managing [k0s](https://github.com/k0sproject/k0s) version numbers. It is based on [hashicorp/go-version](https://github.com/hashicorp/go-version) but adds sorting and comparison capabilities for the k0s version numbering scheme which requires additional sorting by the build tag.

## Usage

### Basic comparison

```go
import (
  "fmt"
  "github.com/k0sproject/version"
)

func main() {
  a := version.NewVersion("1.23.3+k0s.1")
  b := version.NewVersion("1.23.3+k0s.2")
  fmt.Println("a is greater than b: %t", a.GreaterThan(b))
  fmt.Println("a is less than b: %t", a.LessThan(b))
  fmt.Println("a is equal to b: %t", a.Equal(b))
}
```

Outputs:

```text
a is greater than b: false
a is less than b: true
a is equal to b: false
```

### Check online for latest version

```go
import (
  "fmt"
  "github.com/k0sproject/version"
)

func main() {
  latest, err := version.Latest()
  if err != nil {
    panic(err)
  }
  fmt.Println("Latest k0s version is: %s", latest)
}
```
