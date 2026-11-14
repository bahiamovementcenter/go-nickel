package ncl

import (
	"bytes"
	"strings"
	"testing"
)

func TestRecord(t *testing.T) {
	ctx := NewContext()
	expr, err := ctx.EvalDeep("{ foo = 1, bar = 2 }")

	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	if !expr.IsRecord() {
		t.Fatal("not a record")
	}

	if !expr.IsValue() {
		t.Fatal("not a value")
	}

	record, ok := expr.ToRecord()
	if !ok {
		t.Fatal("not a record")
	}

	if len(record) != 2 {
		t.Fatal("expected 2 elements")
	}
	_, ok = record["foo"]
	if !ok {
		t.Fatal("no foo")
	}
	_, ok = record["bar"]
	if !ok {
		t.Fatal("no bar")
	}
}

func TestArray(t *testing.T) {
	ctx := NewContext()
	expr, err := ctx.EvalDeep("[{ foo = 1, bar = 2 }, 1, 2]")

	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	arr, ok := expr.ToArray()
	if !ok {
		t.Fatal("not an array")
	}

	if len(arr) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr))
	}
	if !arr[0].IsRecord() {
		t.Fatal("first element not a record")
	}
	x, ok := arr[1].ToInt64()
	if !ok || x != 1 {
		t.Fatal("no 1")
	}
	x, ok = arr[2].ToInt64()
	if !ok || x != 2 {
		t.Fatal("no 2")
	}
}

func TestLazyInspection(t *testing.T) {
	ctx := NewContext()
	expr, err := ctx.EvalShallow("{ foo = [1, 2 + 3], bar = \"hi\", baz = 'Tag (1 + 1) }")

	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	record, ok := expr.ToRecord()
	if !ok {
		t.Fatal("not a record")
	}
	if len(record) != 3 {
		t.Fatal("expected 3 elements")
	}

	if record["foo"].IsValue() {
		t.Fatal("expected a lazy foo")
	}
	foo, err := record["foo"].EvalShallow()
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	fooArr, ok := foo.ToArray()
	if !ok {
		t.Fatal("expected an array")
	}
	if len(fooArr) != 2 {
		t.Fatal("expected 2 elements")
	}

	elt, ok := fooArr[0].ToInt64()
	if !ok {
		t.Fatal("expected an int")
	}
	if elt != 1 {
		t.Fatal("expected 1")
	}
	elt, ok = fooArr[1].ToInt64()
	if ok {
		t.Fatal("expected a lazy")
	}
	eltExpr, err := fooArr[1].EvalShallow()
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	elt, ok = eltExpr.ToInt64()
	if !ok {
		t.Fatal("expected an int")
	}
	if elt != 5 {
		t.Fatal("expected 5")
	}
}

func TestEnumVariant(t *testing.T) {
	ctx := NewContext()
	expr, err := ctx.EvalDeep("'Tag (2 + 3)")

	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	tag, payload, ok := expr.ToEnumVariant()
	if !ok {
		t.Fatal("expected enum variant")
	}
	if tag != "Tag" {
		t.Fatal("expected 'Tag")
	}
	x, ok := payload.ToInt64()
	if !ok {
		t.Fatal("expected an int")
	}
	if x != 5 {
		t.Fatal("expected 5")
	}
}

func TestEvalError(t *testing.T) {
	ctx := NewContext()
	_, err := ctx.EvalDeep("{ foo | String = 1, bar = 2 }")

	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "broken by the value of `foo`") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

type FooBar struct {
	Foo int `json:"foo"`
	Bar int `json:"bar"`
}

func TestMarshalJSON(t *testing.T) {
	ctx := NewContext()
	expr, err := ctx.EvalDeep("{ foo | Number = 1, bar = 2 }")

	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	_, err = expr.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var target FooBar
	err = expr.ConvertTo(&target)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}
	if target.Foo != 1 {
		t.Fatalf("expected foo = 1")
	}
	if target.Bar != 2 {
		t.Fatalf("expected bar = 2")
	}
}

func TestTrace(t *testing.T) {
	var buf bytes.Buffer

	ctx := NewContext()
	ctx.SetTraceWriter(&buf)
	_, err := ctx.EvalDeep("std.trace \"hi\" { bye = std.trace \"bye\" 1 }")
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	traceOutput := buf.String()
	if traceOutput != "std.trace: hi\nstd.trace: bye\n" {
		t.Fatalf("unexpected buf contents: `%s`", traceOutput)
	}
}

func TestErrorFormat(t *testing.T) {
	ctx := NewContext()
	_, err := ctx.EvalDeep("{ port | Number = \"80\", name = \"myserver\" }")
	if err == nil {
		t.Fatal("expected an error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "contract broken by the value of `port`") {
		t.Fatalf("unexpected message: %s\n", msg)

	}
}
