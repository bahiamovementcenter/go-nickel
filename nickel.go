package ncl

/*
#cgo CFLAGS: -I${SRCDIR}/include

#include <nickel_lang.h>
*/
import "C"

import (
	"encoding/json"
	"runtime"
	"unsafe"
)

// Expr is a Nickel expression.
//
// Since Nickel is lazy, it may not yet have been evaluated (see Context.EvalShallow for
// more on lazy evaluation). If it has been evaluated, it could be null, a
// boolean, a number, a string, an enum, a record, or an array.
type Expr struct {
	ptr *C.nickel_expr
	// An Expr keeps a reference to the context that created it. This is a departure
	// from the C API, which makes you keep them both around, but it makes the
	// JSON conversion API and lazy eval APIs a bit nicer. (For example, without
	// keeping the context around, Expr can't implement the MarshalJSON interface on
	// its own.) The cost of this is that the context will stay alive longer than
	// strictly needed. But it isn't too big.
	ctx *Context
}

// Error is a Nickel error message.
type Error struct {
	ptr *C.nickel_error
}

// Implement the Error interface for our Error type.
func (e *Error) Error() string {
	s := C.nickel_string_alloc()
	defer C.nickel_string_free(s)

	result := C.nickel_error_format_as_string(e.ptr, s, C.NICKEL_ERROR_FORMAT_TEXT)
	if result == C.NICKEL_RESULT_ERR {
		return "error formatting error"
	} else {
		var len C.uintptr_t
		var bytes *C.char
		C.nickel_string_data(s, &bytes, &len)
		return C.GoStringN(bytes, C.int(len))
	}
}

func new_expr(ctx *Context) *Expr {
	expr := &Expr{
		ptr: C.nickel_expr_alloc(),
		ctx: ctx,
	}

	runtime.SetFinalizer(expr, func(expr *Expr) {
		C.nickel_expr_free(expr.ptr)
	})

	return expr
}

func new_err() *Error {
	err := &Error{
		ptr: C.nickel_error_alloc(),
	}

	runtime.SetFinalizer(err, func(err *Error) {
		C.nickel_error_free(err.ptr)
	})

	return err
}

// EvalShallow evaluates an unevaluated expression a little bit more.
//
// This has no effect if the expression is already evaluated.
//
// The result of this evaluation is a null, bool, number, string,
// enum, record, or array. In case it's a record, array, or enum
// variant, the payload (record values, array elements, or enum
// payloads) will be left unevaluated.
func (expr *Expr) EvalShallow() (*Expr, error) {
	out_expr := new_expr(expr.ctx)
	out_err := new_err()

	result := C.nickel_context_eval_expr_shallow(expr.ctx.ptr, expr.ptr, out_expr.ptr, out_err.ptr)
	if result == C.NICKEL_RESULT_OK {
		return out_expr, nil
	} else {
		return nil, out_err
	}
}

// ToRecord converts an Expr to a native Go map, if the expression represented a Nickel record.
//
// If the record was the result of lazy evaluation, it may have undefined
// fields. In that case, the returned map will have keys whose values are nil.
func (expr *Expr) ToRecord() (map[string]*Expr, bool) {
	if C.nickel_expr_is_record(expr.ptr) != 0 {
		ptr := C.nickel_expr_as_record(expr.ptr)
		len := C.nickel_record_len(ptr)
		ret := make(map[string]*Expr)

		for i := range len {
			var key *C.char
			var key_len C.uintptr_t
			value := new_expr(expr.ctx)

			has_value := C.nickel_record_key_value_by_index(ptr, C.uintptr_t(i), &key, &key_len, value.ptr)
			if has_value == 0 {
				value = nil
			}

			key_string := C.GoStringN(key, C.int(key_len))
			ret[key_string] = value
		}

		return ret, true
	} else {
		return nil, false
	}
}

// ToArray converts an Expr to a native Go array, if the expression represented a Nickel array.
//
// If the expression was shallowly evaluated, some of the elements of the returned array may
// not have been evaluated yet.
func (expr *Expr) ToArray() ([]*Expr, bool) {
	if C.nickel_expr_is_array(expr.ptr) != 0 {
		ptr := C.nickel_expr_as_array(expr.ptr)
		len := C.nickel_array_len(ptr)
		ret := make([]*Expr, len)

		for i := range len {
			value := new_expr(expr.ctx)
			C.nickel_array_get(ptr, i, value.ptr)
			ret[i] = value
		}
		return ret, true
	} else {
		return nil, false
	}
}

// ToBool converts an Expr into a bool, if the expression represented a Nickel bool.
func (expr *Expr) ToBool() (bool, bool) {
	if C.nickel_expr_is_bool(expr.ptr) != 0 {
		b := C.nickel_expr_as_bool(expr.ptr) != 0
		return b, true
	} else {
		return false, false
	}
}

// ToFloat64 converts an Expr into a float64, if the expression represented a Nickel number.
//
// The conversion from Nickel number to a float64 may involve rounding.
func (expr *Expr) ToFloat64() (float64, bool) {
	if C.nickel_expr_is_number(expr.ptr) != 0 {
		num := C.nickel_expr_as_number(expr.ptr)
		x := C.nickel_number_as_f64(num)
		return float64(x), true
	} else {
		return 0.0, false
	}
}

// ToInt64 converts an Expr into an int64, if the expression represented a Nickel number
// that fits in an int64.
//
// This conversion will fail if the expression is a Nickel number that doesn't fit in
// an int64, either because it is too large or not an integer.
func (expr *Expr) ToInt64() (int64, bool) {
	if C.nickel_expr_is_number(expr.ptr) != 0 {
		num := C.nickel_expr_as_number(expr.ptr)
		if C.nickel_number_is_i64(num) != 0 {
			x := C.nickel_number_as_i64(num)
			return int64(x), true
		}
	}
	return 0, false
}

// ToString converts an Expr into a string, if the expression represented a Nickel string.
func (expr *Expr) ToString() (string, bool) {
	if C.nickel_expr_is_str(expr.ptr) != 0 {
		var ptr *C.char
		len := C.nickel_expr_as_str(expr.ptr, &ptr)
		return C.GoStringN(ptr, (C.int)(len)), true
	} else {
		return "", false
	}
}

// ToEnumTag converts an Expr into a string, if the expression represented a Nickel enum tag.
func (expr *Expr) ToEnumTag() (string, bool) {
	if C.nickel_expr_is_enum_tag(expr.ptr) != 0 {
		var ptr *C.char
		len := C.nickel_expr_as_enum_tag(expr.ptr, &ptr)
		return C.GoStringN(ptr, (C.int)(len)), true
	} else {
		return "", false
	}
}

// ToEnumVariant converts an Expr into a tag and a payload, if the expression represented
// a Nickel enum variant.
//
// If the expression was shallowly evaluated, the payload may
// not have been evaluated yet.
func (expr *Expr) ToEnumVariant() (string, *Expr, bool) {
	if C.nickel_expr_is_enum_variant(expr.ptr) != 0 {
		var ptr *C.char
		out_expr := new_expr(expr.ctx)
		len := C.nickel_expr_as_enum_variant(expr.ptr, &ptr, out_expr.ptr)
		tag := C.GoStringN(ptr, (C.int)(len))
		return tag, out_expr, true
	} else {
		return "", nil, false
	}
}

func (expr *Expr) IsRecord() bool {
	return C.nickel_expr_is_record(expr.ptr) != 0
}

func (expr *Expr) IsArray() bool {
	return C.nickel_expr_is_array(expr.ptr) != 0
}

func (expr *Expr) IsBool() bool {
	return C.nickel_expr_is_bool(expr.ptr) != 0
}

func (expr *Expr) IsNumber() bool {
	return C.nickel_expr_is_number(expr.ptr) != 0
}

func (expr *Expr) IsString() bool {
	return C.nickel_expr_is_str(expr.ptr) != 0
}

func (expr *Expr) IsEnumTag() bool {
	return C.nickel_expr_is_enum_tag(expr.ptr) != 0
}

func (expr *Expr) IsEnumVariant() bool {
	return C.nickel_expr_is_enum_variant(expr.ptr) != 0
}

func (expr *Expr) IsValue() bool {
	return C.nickel_expr_is_value(expr.ptr) != 0
}

func (expr *Expr) IsNull() bool {
	return C.nickel_expr_is_null(expr.ptr) != 0
}

// MarshalJSON implements the json.Marshaler interface for Expr.
func (expr *Expr) MarshalJSON() ([]byte, error) {
	out_err := new_err()
	out_string := C.nickel_string_alloc()
	defer C.nickel_string_free(out_string)

	result := C.nickel_context_expr_to_json(expr.ctx.ptr, expr.ptr, out_string, out_err.ptr)
	if result == C.NICKEL_RESULT_ERR {
		return nil, out_err
	} else {
		var len C.uintptr_t
		var bytes *C.char
		C.nickel_string_data(out_string, &bytes, &len)

		// Copy the string data over, because it's owned on the Rust side
		// and will be freed with `out_string`.
		borrowedSlice := unsafe.Slice((*byte)(unsafe.Pointer(bytes)), int(len))
		slice := make([]byte, int(len))
		copy(slice, borrowedSlice)

		return slice, nil
	}
}

// ConvertTo converts an Expr to anything that can be unmarshaled from JSON.
func (expr *Expr) ConvertTo(target any) error {
	data, err := expr.MarshalJSON()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}
