package main

import (
	"io/fs"
	"os"
	"runtime"
)

// fsx is an extended fs.FS that supports removing files and interacting with symlinks.
type fsx interface {
	fs.FS
	Remove(name string) error
	RemoveAll(name string) error
	Symlink(oldname, newname string) error
	Readlink(name string) (string, error)
}

// dirFSx is an extended version of os.dirFS that implements fsx.
type dirFSx struct {
	fs.FS
	dir string
}

func dirFS(dir string) fsx { return dirFSx{os.DirFS(dir), dir} }

func (dfs dirFSx) Remove(name string) error {
	if !fs.ValidPath(name) || runtime.GOOS == "windows" && containsAny(name, `\:`) {
		return &os.PathError{Op: "remove", Path: name, Err: os.ErrInvalid}
	}
	return os.Remove(dfs.dir + "/" + name)
}

func (dfs dirFSx) RemoveAll(path string) error {
	if !fs.ValidPath(path) || runtime.GOOS == "windows" && containsAny(path, `\:`) {
		return &os.PathError{Op: "removeall", Path: path, Err: os.ErrInvalid}
	}
	return os.RemoveAll(dfs.dir + "/" + path)
}

func (dfs dirFSx) Symlink(oldname, newname string) error {
	if !fs.ValidPath(oldname) || runtime.GOOS == "windows" && containsAny(oldname, `\:`) {
		return &os.PathError{Op: "symlink", Path: oldname, Err: os.ErrInvalid}
	}
	if !fs.ValidPath(newname) || runtime.GOOS == "windows" && containsAny(newname, `\:`) {
		return &os.PathError{Op: "symlink", Path: newname, Err: os.ErrInvalid}
	}
	return os.Symlink(dfs.dir+"/"+oldname, dfs.dir+"/"+newname)
}

func (dfs dirFSx) Readlink(name string) (string, error) {
	if !fs.ValidPath(name) || runtime.GOOS == "windows" && containsAny(name, `\:`) {
		return "", &os.PathError{Op: "readlink", Path: name, Err: os.ErrInvalid}
	}
	return os.Readlink(dfs.dir + "/" + name)
}

func containsAny(s, chars string) bool {
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(chars); j++ {
			if s[i] == chars[j] {
				return true
			}
		}
	}
	return false
}
