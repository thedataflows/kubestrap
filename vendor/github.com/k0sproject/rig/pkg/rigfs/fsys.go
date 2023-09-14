// Package rigfs provides fs.FS implementations for remote filesystems.
package rigfs

import "github.com/k0sproject/rig/exec"

// NewFsys returns a fs.FS implementation for a remote filesystem
func NewFsys(c connection, opts ...exec.Option) Fsys {
	if c.IsWindows() {
		return NewWindowsFsys(c, opts...)
	}
	return NewPosixFsys(c, opts...)
}
