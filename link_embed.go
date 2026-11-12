//go:build nickel_embed

package nickel

// With the nickel_embed build tag, the package links the C library of
// nickel-embed (github.com/bahiamovementcenter/rust-nickel, crate
// nickel-embed-capi) instead of the vendored one. Its header, which
// include/nickel_lang.h is a copy of, is a superset of Nickel's, and the extra
// functions back the APIs in files with the same tag, such as PackageMap.
//
// Build the library and point the linker at it:
//
//	cargo build --release -p nickel-embed-capi
//	CGO_LDFLAGS="-L$PWD/target/release" go build -tags nickel_embed
//
// The linker picks the shared library if both are there; delete it (or pass
// the archive's path) for a static executable. On Windows, add the system
// libraries that `cargo rustc -p nickel-embed-capi --release --crate-type
// staticlib -- --print native-static-libs` lists.

/*
#cgo LDFLAGS: -lnickel_lang -lm
#cgo linux LDFLAGS: -lpthread -ldl
*/
import "C"
