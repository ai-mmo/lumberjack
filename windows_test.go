//go:build windows
// +build windows

package lumberjack

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWindowsFileRotation 测试 Windows 平台上的文件轮转
// 这个测试会快速触发多次日志轮转，验证文件占用问题是否已解决
func TestWindowsFileRotation(t *testing.T) {
	// 创建临时目录
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.log")

	// 创建 logger，设置很小的 MaxSize 以快速触发轮转
	l := &Logger{
		Filename:   filename,
		MaxSize:    1, // 1MB，快速触发轮转
		MaxBackups: 3,
		MaxAge:     0,
		Compress:   false,
	}
	defer l.Close()

	// 写入大量数据，触发多次轮转
	data := make([]byte, 1024*100) // 100KB per write
	for i := 0; i < len(data); i++ {
		data[i] = 'A'
	}

	// 执行 20 次写入，应该会触发多次轮转
	for i := 0; i < 20; i++ {
		n, err := l.Write(data)
		if err != nil {
			t.Fatalf("写入失败 (iteration %d): %v", i, err)
		}
		if n != len(data) {
			t.Fatalf("写入字节数不匹配: got %d, want %d", n, len(data))
		}
	}

	// 验证文件已创建
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("日志文件未创建")
	}

	// 验证备份文件已创建
	files, err := filepath.Glob(filepath.Join(dir, "test-*.log"))
	if err != nil {
		t.Fatalf("查找备份文件失败: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("没有创建备份文件，日志轮转可能未触发")
	}

	t.Logf("成功创建 %d 个备份文件", len(files))
}

// TestWindowsConcurrentWrites 测试 Windows 平台上的并发写入
// 验证多个 goroutine 同时写入时不会出现文件占用问题
func TestWindowsConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "concurrent.log")

	l := &Logger{
		Filename:   filename,
		MaxSize:    1, // 1MB
		MaxBackups: 5,
		MaxAge:     0,
		Compress:   false,
	}
	defer l.Close()

	// 启动多个 goroutine 并发写入
	numGoroutines := 10
	writesPerGoroutine := 50
	done := make(chan bool, numGoroutines)

	data := []byte("This is a test log message that will be written concurrently\n")

	for g := 0; g < numGoroutines; g++ {
		go func(id int) {
			for i := 0; i < writesPerGoroutine; i++ {
				msg := fmt.Sprintf("[Goroutine %d] %s", id, data)
				_, err := l.Write([]byte(msg))
				if err != nil {
					t.Errorf("Goroutine %d 写入失败: %v", id, err)
				}
				// 短暂休眠，模拟真实场景
				time.Sleep(time.Millisecond)
			}
			done <- true
		}(g)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 验证文件存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("日志文件未创建")
	}

	t.Log("并发写入测试成功完成")
}

// TestWindowsRapidRotation 测试快速连续的日志轮转
// 这是最容易触发文件占用问题的场景
func TestWindowsRapidRotation(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "rapid.log")

	l := &Logger{
		Filename:   filename,
		MaxSize:    1, // 1MB
		MaxBackups: 10,
		MaxAge:     0,
		Compress:   false,
	}
	defer l.Close()

	// 写入大块数据，每次都接近或超过 MaxSize
	largeData := make([]byte, 1024*1024) // 1MB
	for i := 0; i < len(largeData); i++ {
		largeData[i] = byte('A' + (i % 26))
	}

	// 连续写入 15 次，每次都会触发轮转
	for i := 0; i < 15; i++ {
		_, err := l.Write(largeData)
		if err != nil {
			t.Fatalf("快速轮转测试失败 (iteration %d): %v", i, err)
		}
	}

	// 验证创建了多个备份文件
	files, err := filepath.Glob(filepath.Join(dir, "rapid-*.log"))
	if err != nil {
		t.Fatalf("查找备份文件失败: %v", err)
	}

	// 应该有 10 个备份文件（MaxBackups=10）
	if len(files) < 5 {
		t.Fatalf("备份文件数量不足: got %d, want at least 5", len(files))
	}

	t.Logf("快速轮转测试成功，创建了 %d 个备份文件", len(files))
}

// TestWindowsFileShareMode 测试文件共享模式是否正确设置
// 验证在文件打开时，其他进程可以读取该文件
func TestWindowsFileShareMode(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "share.log")

	l := &Logger{
		Filename:   filename,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     0,
		Compress:   false,
	}
	defer l.Close()

	// 写入一些数据
	testData := []byte("Test data for file share mode\n")
	_, err := l.Write(testData)
	if err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	// 尝试以只读模式打开同一个文件（模拟其他进程读取）
	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("无法以只读模式打开文件（文件共享模式可能未正确设置）: %v", err)
	}
	defer f.Close()

	// 读取文件内容
	buf := make([]byte, len(testData))
	n, err := f.Read(buf)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if n != len(testData) {
		t.Fatalf("读取字节数不匹配: got %d, want %d", n, len(testData))
	}

	// 在文件仍然被读取时，继续写入日志
	_, err = l.Write(testData)
	if err != nil {
		t.Fatalf("在文件被读取时写入失败: %v", err)
	}

	t.Log("文件共享模式测试成功")
}

