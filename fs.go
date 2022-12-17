package main

import (
	"io/fs"
	"os"
)

// fsx is an extended fs.FS that supports removing files and creating symlinks.
type fsx interface {
	fs.FS
	Remove(name string) error
	RemoveAll(name string) error
	Symlink(oldname, newname string) error
}

type osFS struct{ fs.FS }

func dirFS(dir string) fsx { return osFS{os.DirFS(dir)} }

func (osFS) Remove(name string) error              { return os.Remove(name) }
func (osFS) RemoveAll(path string) error           { return os.RemoveAll(path) }
func (osFS) Symlink(oldname, newname string) error { return os.Symlink(oldname, newname) }
