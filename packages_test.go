//go:build nickel_embed

package nickel

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, contents := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func evalString(t *testing.T, ctx *Context, src string) string {
	t.Helper()
	expr, err := ctx.EvalDeep(src)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := expr.ToString()
	if !ok {
		t.Fatalf("%s is not a string", src)
	}
	return s
}

func TestPackageMap(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"app/main.ncl":    "{ inner = (import util).name }",
		"util-a/main.ncl": `{ name = "a" }`,
		"util-b/main.ncl": `{ name = "b" }`,
	})
	app := filepath.Join(dir, "app")

	m := NewPackageMap()
	for _, err := range []error{
		m.AddPackage("app", app),
		m.AddPackage("util", filepath.Join(dir, "util-a")),
		// Inside app, util is another package.
		m.AddDependency(app, "util", filepath.Join(dir, "util-b")),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}

	packages := m.Packages()
	if len(packages) != 2 || packages[0].Name != "app" || packages[1].Name != "util" {
		t.Fatalf("unexpected packages %v", packages)
	}

	ctx := NewContext()
	ctx.SetPackageMap(m)
	if got := evalString(t, ctx, "(import util).name"); got != "a" {
		t.Errorf("util is %q", got)
	}
	if got := evalString(t, ctx, "(import app).inner"); got != "b" {
		t.Errorf("app's util is %q", got)
	}

	// Without a map, package imports fail.
	if _, err := NewContext().EvalDeep("import app"); err == nil {
		t.Error("expected an error")
	}
}

func manifest(name, dependencies string) string {
	return `{
  name = "` + name + `",
  description = "test",
  version = "0.1.0",
  authors = ["go-nickel"],
  minimal_nickel_version = "1.9.0",
  dependencies = { ` + dependencies + ` },
} | std.package.Manifest
`
}

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"app/Nickel-pkg.ncl": manifest("app", `lib = 'Path "../lib"`),
		"lib/Nickel-pkg.ncl": manifest("lib", ""),
		"lib/main.ncl":       `{ greeting = "hello" }`,
	})
	if err := os.MkdirAll(filepath.Join(dir, "app/src"), 0o755); err != nil {
		t.Fatal(err)
	}

	path, ok := FindManifest(filepath.Join(dir, "app/src"))
	if !ok || path != filepath.Join(dir, "app/Nickel-pkg.ncl") {
		t.Fatalf("FindManifest found %q, %v", path, ok)
	}

	m, err := LoadManifest(path, filepath.Join(dir, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app/Nickel-pkg.lock")); err != nil {
		t.Errorf("no lock file: %v", err)
	}
	ctx := NewContext()
	ctx.SetPackageMap(m)
	if got := evalString(t, ctx, "(import lib).greeting"); got != "hello" {
		t.Errorf("greeting is %q", got)
	}
}

func TestLoadBadManifest(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Nickel-pkg.ncl": `{ name = 1 } | std.package.Manifest`,
	})
	if _, err := LoadManifest(filepath.Join(dir, "Nickel-pkg.ncl"), filepath.Join(dir, "cache")); err == nil {
		t.Error("expected an error")
	}
}
