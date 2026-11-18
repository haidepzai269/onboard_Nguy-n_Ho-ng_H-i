package win32

import (
	"fmt"
	"syscall"
)

// Nạp trực tiếp các DLL hệ thống. Không dùng thư viện ngoài: syscall.NewLazyDLL
// chỉ giải phóng hàm khi thực sự được gọi lần đầu.
var (
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")
	modPsapi    = syscall.NewLazyDLL("psapi.dll")

	procGetForegroundWindow      = modUser32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = modUser32.NewProc("GetWindowTextW")
	procGetClassNameW            = modUser32.NewProc("GetClassNameW")
	procGetWindowThreadProcessID = modUser32.NewProc("GetWindowThreadProcessId")

	procCreateToolhelp32Snapshot   = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW            = modKernel32.NewProc("Process32FirstW")
	procProcess32NextW             = modKernel32.NewProc("Process32NextW")
	procThread32First              = modKernel32.NewProc("Thread32First")
	procThread32Next               = modKernel32.NewProc("Thread32Next")
	procOpenProcess                = modKernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = modKernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                = modKernel32.NewProc("CloseHandle")
	procGetStdHandle               = modKernel32.NewProc("GetStdHandle")
	procGetConsoleMode             = modKernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = modKernel32.NewProc("SetConsoleMode")
	procGetConsoleScreenBufferInfo = modKernel32.NewProc("GetConsoleScreenBufferInfo")
	procReadConsoleInputW          = modKernel32.NewProc("ReadConsoleInputW")
	procSetConsoleOutputCP         = modKernel32.NewProc("SetConsoleOutputCP")

	procGetProcessMemoryInfo = modPsapi.NewProc("GetProcessMemoryInfo")
)

// callErr dựng lỗi có tên hàm Win32. LazyProc.Call luôn trả về một error (kể cả
// khi lời gọi thành công), nên khi r1 == 0 mà errno bằng 0 ta báo lỗi không rõ mã.
func callErr(name string, e error) error {
	if errno, ok := e.(syscall.Errno); ok && errno == 0 {
		return fmt.Errorf("%s thất bại (không có mã lỗi từ Kernel)", name)
	}
	return fmt.Errorf("%s: %w", name, e)
}

// CloseHandle đóng một handle nhân. Bỏ qua lỗi vì không có cách xử lý hữu ích.
func CloseHandle(h Handle) {
	if h != 0 && h != INVALID_HANDLE {
		procCloseHandle.Call(uintptr(h))
	}
}

// FormatHandle in handle theo dạng 0x000000C8 để hiển thị trong log.
func FormatHandle(h Handle) string {
	return fmt.Sprintf("0x%08X", uint64(h))
}
