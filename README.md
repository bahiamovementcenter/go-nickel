# go-ncl: Go bindings for Nickel

With these Go bindings, you can use the [Nickel](https://nickel-lang.org) configuration
language directly from Go as part of your single static executable.

They link [nickel-embed](https://github.com/bahiamovementcenter/rust-nickel), the engine every
`*-ncl` binding shares, through its C library, whose header is a superset of Nickel's own
`nickel_lang.h`. The library is prebuilt for Linux (amd64, arm64), macOS (arm64) and Windows
(amd64) and committed in `lib/`, so `go get` is all it takes: no Rust toolchain needed. This
started as a fork of [nickel-lang/go-nickel](https://github.com/nickel-lang/go-nickel).

# Getting started

```
go get github.com/bahiamovementcenter/go-ncl
```

Then you can evaluate Nickel strings from within your go code:

```go
import (
  "fmt"
  ncl "github.com/bahiamovementcenter/go-ncl"
)

func main() {
  ctx := ncl.NewContext()
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
  ncl "github.com/bahiamovementcenter/go-ncl"
)

type Config struct {
	Port int    `json:"port"`
	Name string `json:"name"`
}

func main() {
  ctx := ncl.NewContext()
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
  ncl "github.com/bahiamovementcenter/go-ncl"
)

func main() {
  ctx := ncl.NewContext()
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

# Packages

Package imports (`import foo`) resolve through a package map:

```go
// The packages of the manifest nearest the current directory, fetched if
// needed. Nickel's package manager is experimental.
path, _ := ncl.FindManifest(".")
packages, err := ncl.LoadManifest(path, "") // "" = the default cache directory
if err != nil {
  return err
}
ctx := ncl.NewContext()
ctx.SetPackageMap(packages)
expr, err := ctx.EvalDeep(`(import my_lib).settings`)
```

A map can also be built by hand from directories you already have (vendored,
fetched by Nix): `NewPackageMap`, then `AddPackage(name, dir)` and
`AddDependency(parent, name, dir)`.

This is how [xD](https://github.com/bahiamovementcenter/xD)'s `xd` package is
used: depend on it in your `Nickel-pkg.ncl` (a `'Git` dependency with
`path = "nickel"`), then `(import xd).units.jd.pattern.render [1, 1, 11]` gives
`"11.11"`.

# Development

```
go test .                  # NCL_TEST_OFFLINE=1 skips the test that fetches xD
```

`lib/`, `include/nickel_lang.h` and `link.go` come from the rust-nickel release
named in `RUST_NICKEL_VERSION`. To move to a new release, run the "Update lib"
workflow with its tag (or `scripts/update-lib.sh vX.Y.Z` locally), which
downloads the release's archives, checks their checksums and opens a pull
request. CI checks that the committed files match the named release.

Releases of this module are tags `vX.Y.Z`; its version moves with its API, not
with rust-nickel's.
