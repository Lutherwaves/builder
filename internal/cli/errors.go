package cli

import (
	"errors"
	"syscall"
	"unicode"
)

// Strerror is the OS reason of an error, as Python's OSError.strerror spells it ("Not a directory").
func Strerror(err error) string {
	var errno syscall.Errno
	msg := err.Error()
	if errors.As(err, &errno) {
		msg = errno.Error()
	}
	r := []rune(msg)
	if len(r) > 0 {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}
