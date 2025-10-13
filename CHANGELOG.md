# Changelog

## [v0.0.4] - 2025-10-11

### 修复
- **Windows 文件占用问题**: 修复了在 Windows 系统上日志轮转时出现的 "The process cannot access the file because it is being used by another process" 错误
  - 使用 Windows API 的文件共享模式（FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE）
  - 为文件打开和重命名操作添加了重试机制
  - 文件打开失败时最多重试 3 次，每次间隔 10ms
  - 文件重命名失败时最多重试 5 次，每次间隔 20ms

### 新增
- 新增 `open_windows.go`: Windows 平台特定的文件操作实现
- 新增 `open_unix.go`: Unix/Linux/macOS 平台的文件操作实现
- 新增 `windows_test.go`: Windows 平台的专项测试
- 新增 `WINDOWS_FILE_LOCK_FIX.md`: Windows 文件占用问题修复说明文档

### 技术细节
- 在 Windows 平台上，使用 `syscall.CreateFile` 替代 `os.OpenFile`，设置适当的共享模式
- 在非 Windows 平台上，保持使用标准库函数，零性能影响
- 通过 Go build tags 实现平台特定的编译

### 影响范围
- 单文件模式和多文件模式均受益于此修复
- 高并发写入场景下的稳定性提升
- 快速日志轮转场景下的可靠性提升

---

## [v0.0.3] - 之前版本

### 修复
- 修复了 goroutine 泄露问题
- 添加了优雅关闭机制

详见 `GOROUTINE_LEAK_FIX.md`

