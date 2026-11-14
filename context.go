package ncl

/*
#cgo CFLAGS: -I./include

#include <nickel_lang.h>
#include <stdlib.h>

extern uintptr_t traceCallback(void*, uint8_t*, uintptr_t);

uintptr_t traceCallbackTrampoline(void* context, const uint8_t* buf, uintptr_t len);
*/
import "C"
import (
	"io"
	"runtime"
	"sync"
	"unsafe"
)

var (
	// A map from `nickel_context*` to the configured trace callback for that context.
	// The finalizer for `Context` both deallocates the `nickel_context*` and removes
	// the trace callback from this map.
	contextTracer      = map[unsafe.Pointer]io.Writer{}
	contextTracerMutex sync.RWMutex
)

// Context is the main entry point.
//
// It allows you to customize various aspects of the Nickel interpreter, such
// as the path used to search for imported files.
type Context struct {
	ptr *C.nickel_context
}

// NewContext creates a new Context for storing global Nickel settings.
func NewContext() *Context {
	ctx := &Context{
		ptr: C.nickel_context_alloc(),
	}

	runtime.SetFinalizer(ctx, func(ctx *Context) {
		C.nickel_context_free(ctx.ptr)
		delete(contextTracer, unsafe.Pointer(ctx.ptr))
	})

	return ctx
}

//export traceCallback
func traceCallback(data unsafe.Pointer, buf *C.uint8_t, len C.uintptr_t) C.uintptr_t {
	// This copies the bytes, which is a little unfortunate. Most io.Writers
	// are probably ok with just an unsafe.Slice, but we can't be sure...
	bytes := C.GoBytes(unsafe.Pointer(buf), C.int(len))

	contextTracerMutex.RLock()
	w := contextTracer[data]
	contextTracerMutex.RUnlock()

	// Swallow the error if the write callback fails, since it's just for tracing.
	n, _ := w.Write(bytes)
	return C.uintptr_t(n)
}

// SetTraceWriter provides a "trace" callback to the Nickel evaluator.
//
// When evaluating Nickel code that calls the `std.trace` function, the
// resulting trace outputs will be written to the writer w.
func (ctx *Context) SetTraceWriter(w io.Writer) {
	contextTracerMutex.Lock()
	contextTracer[unsafe.Pointer(ctx.ptr)] = w
	contextTracerMutex.Unlock()
	C.nickel_context_set_trace_callback(ctx.ptr, C.nickel_write_callback(C.traceCallbackTrampoline), nil, unsafe.Pointer(ctx.ptr))
}

// EvalDeep evaluates a Nickel program deeply.
//
// "Deeply" means that we recursively evaluate records and arrays. For
// an alternative, see EvalShallow.
func (ctx *Context) EvalDeep(src string) (*Expr, error) {
	// This is a little silly, because eventually the Rust library converts
	// the null-terminated C string into a length-delimited Rust string.
	// We could avoid some extra copying by having the C API work with
	// length-delimited strings, but then it's a weird API for C users...
	csrc := C.CString(src)
	out_expr := new_expr(ctx)
	out_err := new_err()
	result := C.nickel_context_eval_deep(ctx.ptr, csrc, out_expr.ptr, out_err.ptr)
	C.free(unsafe.Pointer(csrc))

	if result == C.NICKEL_RESULT_OK {
		return out_expr, nil
	} else {
		return nil, out_err
	}
}

// Evaluate a Nickel program shallowly.
//
// The result of this evaluation is a null, bool, number, string,
// enum, record, or array. In case it's a record, array, or enum
// variant, the payload (record values, array elements, or enum
// payloads) will be left unevaluated.
func (ctx *Context) EvalShallow(src string) (*Expr, error) {
	csrc := C.CString(src)
	out_expr := new_expr(ctx)
	out_err := new_err()
	result := C.nickel_context_eval_shallow(ctx.ptr, csrc, out_expr.ptr, out_err.ptr)
	C.free(unsafe.Pointer(csrc))

	if result == C.NICKEL_RESULT_OK {
		return out_expr, nil
	} else {
		return nil, out_err
	}
}
