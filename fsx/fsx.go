// Package fsx implements extensions for the standard [io/fs] package.
package fsx

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FS is an extended [fs.FS].
type FS interface {
	fs.FS
	Remove(name string) error
	RemoveAll(name string) error
	Symlink(name, link string) error
	Readlink(name string) (string, error)
}

type dirFS struct {
	fs.FS
	Dir string
}

// DirFS is an extended [os.DirFS].
func DirFS(path ...string) FS {
	dir := filepath.Join(path...)
	return dirFS{os.DirFS(dir), dir}
}

func (d dirFS) Remove(name string) error             { return os.Remove(d.join(name)) }
func (d dirFS) RemoveAll(name string) error          { return os.RemoveAll(d.join(name)) }
func (d dirFS) Symlink(name, link string) error      { return os.Symlink(d.join(name), d.join(link)) }
func (d dirFS) Readlink(name string) (string, error) { return os.Readlink(d.join(name)) }
func (d dirFS) join(name string) string              { return filepath.Join(d.Dir, name) }
