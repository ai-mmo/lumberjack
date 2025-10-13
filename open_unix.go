//go:build !windows
// +build !windows

package lumberjack

import (
	"os"
)

// openFile 在非 Windows 平台上打开文件，直接使用标准库的 os.OpenFile
func openFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// renameFile 在非 Windows 平台上重命名文件，直接使用标准库的 os.Rename
func renameFile(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
