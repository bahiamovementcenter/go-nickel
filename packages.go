//go:build nickel_embed

package nickel

/*
#include <nickel_lang.h>
#include <stdlib.h>
*/
import "C"

import (
	"runtime"
	"unsafe"
)

// PackageMap says where package imports (`import foo`) are found.
//
// Each package is a directory, and importing it evaluates its main.ncl. Names
// are local to the importer: the program's own packages are added with
// AddPackage, and those of a package with AddDependency, so two packages can
// each have their own `foo`. LoadManifest builds a map from a Nickel-pkg.ncl
// manifest instead, the way the nickel command does.
//
// Use a map with Context.SetPackageMap. Package maps need the nickel_embed
// build tag (see link_embed.go).
type PackageMap struct {
	ptr *C.nickel_package_map
}

// Package is one of a PackageMap's packages: its name, and its directory.
type Package struct {
	Name string
	Dir  string
}

// NewPackageMap returns an empty map: every package import fails until
// packages are added.
func NewPackageMap() *PackageMap {
	m := &PackageMap{ptr: C.nickel_package_map_alloc()}
	runtime.SetFinalizer(m, func(m *PackageMap) {
		C.nickel_package_map_free(m.ptr)
	})
	return m
}

// AddPackage makes `import name` in the program load the package in dir,
// replacing any package of that name. A relative dir is taken from the
// current directory; a missing one is reported when the package is imported.
func (m *PackageMap) AddPackage(name, dir string) error {
	defer runtime.KeepAlive(m)
	cname, cdir := C.CString(name), C.CString(dir)
	defer C.free(unsafe.Pointer(cname))
	defer C.free(unsafe.Pointer(cdir))
	err := new_err()
	if C.nickel_package_map_add_package(m.ptr, cname, cdir, err.ptr) != C.NICKEL_RESULT_OK {
		return err
	}
	return nil
}

// AddDependency makes `import name` inside the package in parent load the
// package in dir. parent must be the directory the parent package was added
// with.
func (m *PackageMap) AddDependency(parent, name, dir string) error {
	defer runtime.KeepAlive(m)
	cparent, cname, cdir := C.CString(parent), C.CString(name), C.CString(dir)
	defer C.free(unsafe.Pointer(cparent))
	defer C.free(unsafe.Pointer(cname))
	defer C.free(unsafe.Pointer(cdir))
	err := new_err()
	if C.nickel_package_map_add_dependency(m.ptr, cparent, cname, cdir, err.ptr) != C.NICKEL_RESULT_OK {
		return err
	}
	return nil
}

// LoadManifest builds the map for a package manifest (Nickel-pkg.ncl), like
// the nickel command does before evaluating a package.
//
// It resolves versions, fetches git and index dependencies that aren't cached
// yet, and writes or updates Nickel-pkg.lock next to the manifest. Fetched
// packages are cached in cacheDir, or in the platform's cache directory (such
// as ~/.cache/nickel) if cacheDir is empty. Nickel's package manager is
// experimental: its manifest and lock formats may change.
func LoadManifest(manifest, cacheDir string) (*PackageMap, error) {
	cmanifest := C.CString(manifest)
	defer C.free(unsafe.Pointer(cmanifest))
	var ccache *C.char
	if cacheDir != "" {
		ccache = C.CString(cacheDir)
		defer C.free(unsafe.Pointer(ccache))
	}
	m := NewPackageMap()
	err := new_err()
	if C.nickel_package_map_load_manifest(m.ptr, cmanifest, ccache, err.ptr) != C.NICKEL_RESULT_OK {
		return nil, err
	}
	return m, nil
}

// FindManifest returns the nearest Nickel-pkg.ncl in dir or a parent
// directory, as the nickel command looks for one, and whether there is one.
func FindManifest(dir string) (string, bool) {
	cdir := C.CString(dir)
	defer C.free(unsafe.Pointer(cdir))
	s := C.nickel_string_alloc()
	defer C.nickel_string_free(s)
	if C.nickel_package_map_find_manifest(cdir, s) == 0 {
		return "", false
	}
	return goString(s), true
}

// Packages returns the program's own packages, sorted by name.
func (m *PackageMap) Packages() []Package {
	defer runtime.KeepAlive(m)
	name, dir := C.nickel_string_alloc(), C.nickel_string_alloc()
	defer C.nickel_string_free(name)
	defer C.nickel_string_free(dir)
	count := C.nickel_package_map_package_count(m.ptr)
	packages := make([]Package, 0, int(count))
	for i := range count {
		C.nickel_package_map_package(m.ptr, i, name, dir)
		packages = append(packages, Package{Name: goString(name), Dir: goString(dir)})
	}
	return packages
}

// String describes the whole map, with every package's dependencies, as
// `nickel package debug-resolution` prints it.
func (m *PackageMap) String() string {
	defer runtime.KeepAlive(m)
	s := C.nickel_string_alloc()
	defer C.nickel_string_free(s)
	C.nickel_package_map_to_string(m.ptr, s)
	return goString(s)
}

// SetPackageMap resolves package imports in this context with a copy of m,
// replacing any map set before. Without one, package imports fail.
func (ctx *Context) SetPackageMap(m *PackageMap) {
	C.nickel_context_set_package_map(ctx.ptr, m.ptr)
	runtime.KeepAlive(m)
	runtime.KeepAlive(ctx)
}

func goString(s *C.nickel_string) string {
	var data *C.char
	var len C.uintptr_t
	C.nickel_string_data(s, &data, &len)
	return C.GoStringN(data, C.int(len))
}
