### Rig

A golang package for adding multi-protocol connectivity and multi-os operation functionality to your application's Host objects.

#### Design goals

Rig's intention is to be easy to use and extend.

It should be easy to add support for new operating systems and to add new commands to the multi-os support mechanism without breaking go's type checking.

All of the relevant structs have YAML tags and default values to make unmarshaling from YAML configurations as easy as possible.

#### Protocols

Currently rig comes with the most common ways to connect to hosts:
- SSH for connecting to hosts that accept SSH connections
- WinRM as an alternative to SSH for windows hosts (SSH works too)
- Local for treating the localhost as it was one of the remote hosts

#### Usage

The intended way to use rig is to embed the `rig.Connection` struct into your own.

Example:

```go
package main

import "github.com/k0sproject/rig"

type host struct {
  rig.Connection
}

func main() {
  h := host{
    connection: rig.Connection{
      SSH: &rig.SSH{
        Address: 10.0.0.1
      }
    }
  }

  if err := h.Connect(); err != nil {
    panic(err)
  }

  output, err := h.ExecOutput("ls -al")
  if err != nil {
    panic(err)
  }
  println(output)
}
```

But of course you can use it directly on its own too:

```go
package main

import "github.com/k0sproject/rig"

func main() {
  h := rig.Connection{
    SSH: &rig.SSH{
      Address: 10.0.0.1
    }
  }

  if err := h.Connect(); err != nil {
    panic(err)
  }
}
```

See more usage examples in the [examples/](examples/) directory.
