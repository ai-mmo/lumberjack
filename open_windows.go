//go:build windows
// +build windows

package lumberjack

import (
	"os"
	"syscall"
	"time"
)

// openFile 在 Windows 平台上打开文件，使用适当的共享模式避免文件占用冲突
// 允许其他进程读取和删除文件，这样可以避免 "The process cannot access the file" 错误
// 如果遇到文件占用错误，会进行短暂重试
func openFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	// 将 Go 的文件标志转换为 Windows 的访问模式和创建模式
	var access uint32
	switch flag & (os.O_RDONLY | os.O_WRONLY | os.O_RDWR) {
	case os.O_RDONLY:
		access = syscall.GENERIC_READ
	case os.O_WRONLY:
		access = syscall.GENERIC_WRITE
	case os.O_RDWR:
		access = syscall.GENERIC_READ | syscall.GENERIC_WRITE
	}

	// 如果包含 O_APPEND，添加 FILE_APPEND_DATA 权限
	if flag&os.O_APPEND != 0 {
		access &^= syscall.GENERIC_WRITE
		access |= syscall.FILE_APPEND_DATA
	}

	// 创建模式
	var createMode uint32
	switch {
	case flag&os.O_CREATE != 0 && flag&os.O_EXCL != 0:
		createMode = syscall.CREATE_NEW
	case flag&os.O_CREATE != 0 && flag&os.O_TRUNC != 0:
		createMode = syscall.CREATE_ALWAYS
	case flag&os.O_CREATE != 0:
		createMode = syscall.OPEN_ALWAYS
	case flag&os.O_TRUNC != 0:
		createMode = syscall.TRUNCATE_EXISTING
	default:
		createMode = syscall.OPEN_EXISTING
	}

	// 关键：设置共享模式，允许其他进程读取和删除
	// FILE_SHARE_READ: 允许其他进程读取文件
	// FILE_SHARE_WRITE: 允许其他进程写入文件（用于日志轮转时的重命名）
	// FILE_SHARE_DELETE: 允许其他进程删除或重命名文件
	shareMode := uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)

	// 文件属性
	attrs := uint32(syscall.FILE_ATTRIBUTE_NORMAL)

	// 转换路径为 UTF-16
	pathp, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}

	// 重试机制：在 Windows 上，即使使用了共享模式，文件可能仍被短暂占用
	// 最多重试 3 次，每次间隔 10ms
	var handle syscall.Handle
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		handle, err = syscall.CreateFile(
			pathp,
			access,
			shareMode,
			nil,
			createMode,
			attrs,
			0,
		)
		if err == nil {
			break
		}

		// 检查是否是文件占用错误
		if errno, ok := err.(syscall.Errno); ok {
			// ERROR_SHARING_VIOLATION (32) 或 ERROR_LOCK_VIOLATION (33)
			if errno == 32 || errno == 33 {
				if i < maxRetries-1 {
					// 短暂等待后重试
					time.Sleep(10 * time.Millisecond)
					continue
				}
			}
		}
		// 其他错误或重试次数用尽，直接返回
		return nil, err
	}

	// 将 Windows 句柄转换为 Go 的 *os.File
	return os.NewFile(uintptr(handle), name), nil
}

// renameFile 在 Windows 平台上重命名文件，带重试机制
// 在文件轮转时，可能会遇到短暂的文件占用问题
func renameFile(oldpath, newpath string) error {
	// 重试机制：最多重试 5 次，每次间隔 20ms
	maxRetries := 5
	var err error
	for i := 0; i < maxRetries; i++ {
		err = os.Rename(oldpath, newpath)
		if err == nil {
			return nil
		}

		// 检查是否是文件占用错误
		if errno, ok := err.(*os.LinkError); ok {
			if syserr, ok := errno.Err.(syscall.Errno); ok {
				// ERROR_SHARING_VIOLATION (32) 或 ERROR_LOCK_VIOLATION (33) 或 ERROR_ACCESS_DENIED (5)
				if syserr == 32 || syserr == 33 || syserr == 5 {
					if i < maxRetries-1 {
						// 短暂等待后重试
						time.Sleep(20 * time.Millisecond)
						continue
					}
				}
			}
		}
		// 其他错误或重试次数用尽，直接返回
		return err
	}
	return err
}
