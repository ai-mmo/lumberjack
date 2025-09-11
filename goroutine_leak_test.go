package lumberjack

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestGoroutineLeakFix 测试 goroutine 泄露修复效果
func TestGoroutineLeakFix(t *testing.T) {
	// 启用调试日志
	EnableDebugLog(true)
	defer EnableDebugLog(false)

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "lumberjack_leak_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 记录初始 goroutine 数量
	initialGoroutines := runtime.NumGoroutine()
	fmt.Printf("初始 goroutine 数量: %d\n", initialGoroutines)

	// 创建多个 Logger 实例并触发日志轮转
	const numLoggers = 10
	loggers := make([]*Logger, numLoggers)

	for i := 0; i < numLoggers; i++ {
		loggers[i] = &Logger{
			Filename:   filepath.Join(tempDir, fmt.Sprintf("test_%d.log", i)),
			MaxSize:    1, // 1MB，便于触发轮转
			MaxBackups: 2,
			MaxAge:     1,
			Compress:   true,
		}

		// 写入一些数据触发日志轮转和后台处理
		data := make([]byte, 1024*512) // 512KB，不超过 1MB 限制
		for j := range data {
			data[j] = byte('A' + (j % 26))
		}

		_, err := loggers[i].Write(data)
		if err != nil {
			t.Errorf("写入日志失败: %v", err)
		}

		// 再写入一些数据触发轮转
		data2 := make([]byte, 1024*600) // 600KB，总共超过 1MB
		for j := range data2 {
			data2[j] = byte('B' + (j % 26))
		}

		_, err = loggers[i].Write(data2)
		if err != nil {
			t.Errorf("写入第二批数据失败: %v", err)
		}

		// 再写入一次确保触发后台处理
		_, err = loggers[i].Write([]byte("additional data"))
		if err != nil {
			t.Errorf("写入额外数据失败: %v", err)
		}
	}

	// 等待后台处理完成
	time.Sleep(100 * time.Millisecond)

	// 检查 goroutine 数量（应该增加了）
	afterCreateGoroutines := runtime.NumGoroutine()
	fmt.Printf("创建 Logger 后 goroutine 数量: %d\n", afterCreateGoroutines)

	if afterCreateGoroutines <= initialGoroutines {
		t.Logf("警告: 创建 Logger 后 goroutine 数量没有增加，可能后台处理没有启动")
	}

	// 关闭所有 Logger
	for i, logger := range loggers {
		err := logger.Close()
		if err != nil {
			t.Errorf("关闭 Logger %d 失败: %v", i, err)
		}
	}

	// 等待 goroutine 清理完成
	time.Sleep(200 * time.Millisecond)

	// 强制垃圾回收
	runtime.GC()
	runtime.GC()

	// 再次等待确保清理完成
	time.Sleep(100 * time.Millisecond)

	// 检查最终 goroutine 数量
	finalGoroutines := runtime.NumGoroutine()
	fmt.Printf("关闭 Logger 后 goroutine 数量: %d\n", finalGoroutines)

	// 验证 goroutine 数量是否回到初始水平（允许少量差异）
	if finalGoroutines > initialGoroutines+2 {
		t.Errorf("检测到 goroutine 泄露: 初始=%d, 最终=%d, 差异=%d",
			initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)

		// 输出当前所有 goroutine 的堆栈信息用于调试
		buf := make([]byte, 1<<20)
		stackSize := runtime.Stack(buf, true)
		fmt.Printf("当前 goroutine 堆栈信息:\n%s\n", buf[:stackSize])
	} else {
		fmt.Printf("✅ goroutine 泄露修复验证通过: 初始=%d, 最终=%d\n",
			initialGoroutines, finalGoroutines)
	}
}

// TestMultipleClosesSafety 测试多次关闭的安全性
func TestMultipleClosesSafety(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumberjack_close_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := &Logger{
		Filename: filepath.Join(tempDir, "test.log"),
		MaxSize:  1,
	}

	// 写入数据触发后台处理启动
	_, err = logger.Write([]byte("test data"))
	if err != nil {
		t.Fatalf("写入数据失败: %v", err)
	}

	// 多次关闭应该是安全的
	for i := 0; i < 3; i++ {
		err := logger.Close()
		if err != nil {
			t.Errorf("第 %d 次关闭失败: %v", i+1, err)
		}
	}

	// 关闭后写入应该返回错误
	_, err = logger.Write([]byte("should fail"))
	if err == nil {
		t.Error("关闭后写入应该返回错误")
	}
}

// BenchmarkLoggerCreationAndClose 基准测试 Logger 创建和关闭的性能
func BenchmarkLoggerCreationAndClose(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "lumberjack_bench")
	if err != nil {
		b.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger := &Logger{
			Filename: filepath.Join(tempDir, fmt.Sprintf("bench_%d.log", i)),
			MaxSize:  1,
		}

		// 写入数据触发后台处理
		logger.Write([]byte("benchmark data"))

		// 关闭 Logger
		logger.Close()
	}
}
