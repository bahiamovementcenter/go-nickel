//go:build !nickel_embed

package nickel

// Links the Nickel C library vendored in lib/. Build with the nickel_embed tag
// to link nickel-embed's instead (see link_embed.go).

/*
#cgo linux,amd64 LDFLAGS: ${SRCDIR}/lib/linux_amd64/libnickel_lang.a -lm
#cgo linux,arm64 LDFLAGS: ${SRCDIR}/lib/linux_arm64/libnickel_lang.a -lm
#cgo darwin,arm64 LDFLAGS: ${SRCDIR}/lib/darwin_arm64/libnickel_lang.a -lm
#cgo windows,amd64 LDFLAGS: ${SRCDIR}/lib/windows_amd64/libnickel_lang.a -lm -lntdll
*/
import "C"
