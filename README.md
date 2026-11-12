# Go bindings for Nickel

With these golang bindings, you can use the [Nickel](https://nickel-lang.org) configuration
language directly from go as part of your single static executable.

# Getting started

Add Nickel bindings to your project using `go get`:

```
go get github.com/nickel-lang/go-nickel@main
```

(You need to use `main` because we haven't had any releases yet.)

Then you can evaluate Nickel strings from within your go code:

```go
import (
  "fmt"
  "github.com/nickel-lang/go-nickel"
)

func main() {
  ctx := nickel.NewContext()
	expr, _ := ctx.EvalDeep(`
  {
    port = 80,
    name = "myserver"
  } | { port | Number, name | String }
  `)
  record, _ := expr.ToRecord()
  name, _ := record["name"].ToString()

  fmt.Printf("The server name is `%s`", name)
}
```

# Unmarshalling support

These bindings have support for go's native unmarshalling:

```go
import (
  "fmt"
  "github.com/nickel-lang/go-nickel"
)

type Config struct {
	Port int    `json:"port"`
	Name string `json:"name"`
}

func main() {
  ctx := nickel.NewContext()
	expr, _ := ctx.EvalDeep(`
  {
    port = 80,
    name = "myserver"
  } | { port | Number, name | String }
  `)
  var config Config
  _ = expr.ConvertTo(&config)
  fmt.Printf("The config is `%v`", config)
}
```

# Lazy (shallow) evaluation

Lazy evaluation is a key feature of Nickel, as it allows you to evaluate
only a small part of a large configuration. These go bindings support shallow
evaluation. Evaluating containers (record and arrays) shallowly means that you
delay evaluating the elements.

```go
import (
  "fmt"
  "github.com/nickel-lang/go-nickel"
)

func main() {
  ctx := nickel.NewContext()
	expr, _ := ctx.EvalShallow(`{
    port = 79 + 1,
    name = "my" ++ "server"
  }`)

  // The top-level expression is a record.
  record, _ := expr.ToRecord()

  // The record has "port" and "name" fields.
  port := record["port"]

  // The port isn't a number yet, it's the (unevaluated) expression 79 + 1
  if port.IsNumber() {
    fmt.Print("it shouldn't be a number yet!")
    return
  }

  portVal, _ := port.EvalShallow()
  portNum, _ := portVal.ToInt64()
  fmt.Printf("now it's a number: %d\n", portNum)
}
```

# Packages (`nickel_embed` builds)

Built with the `nickel_embed` tag, these bindings link the C library of
[nickel-embed](https://github.com/bahiamovementcenter/rust-nickel) instead of
the vendored one. Its header, which `include/nickel_lang.h` is a copy of, is a
superset of Nickel's, and the extra functions add package imports
(`import foo`):

```go
// The packages of the manifest nearest the current directory, fetched if
// needed. Nickel's package manager is experimental.
path, _ := nickel.FindManifest(".")
packages, err := nickel.LoadManifest(path, "") // "" = the default cache directory
if err != nil {
  return err
}
ctx := nickel.NewContext()
ctx.SetPackageMap(packages)
expr, err := ctx.EvalDeep(`(import my_lib).settings`)
```

A map can also be built by hand from directories you already have (vendored,
fetched by Nix): `NewPackageMap`, then `AddPackage(name, dir)` and
`AddDependency(parent, name, dir)`.

Build the library, then point cgo at it:

```
cargo build --release -p nickel-embed-capi   # in rust-nickel
CGO_LDFLAGS="-L/path/to/rust-nickel/target/release" go build -tags nickel_embed
```
