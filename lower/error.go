package lower

import "github.com/pkg/errors"

// Errorf formats according to a format specifier and returns the string as a
// value that satisfies error.
func (gen *Generator) Errorf(format string, a ...interface{}) error {
	err := errors.Errorf(format, a...)
	gen.eh(err)
	return err
}
