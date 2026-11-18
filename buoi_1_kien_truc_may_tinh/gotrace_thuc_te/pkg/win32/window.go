package win32

import (
	"syscall"
	"unsafe"
)

// WindowInfo là kết quả thật của 4 lời gọi user32.dll cho cửa sổ tiền cảnh.
type WindowInfo struct {
	HWND      HWND
	Title     string
	ClassName string
	PID       uint32
	ThreadID  uint32
}

// HWND là handle cửa sổ.
type HWND Handle

// GetForegroundWindow gọi user32.dll!GetForegroundWindow.
func GetForegroundWindow() (HWND, error) {
	r1, _, e := procGetForegroundWindow.Call()
	if r1 == 0 {
		return 0, callErr("user32.dll!GetForegroundWindow", e)
	}
	return HWND(r1), nil
}

// GetWindowText gọi user32.dll!GetWindowTextW để đọc tiêu đề cửa sổ (UTF-16).
func GetWindowText(h HWND) (string, error) {
	buf := make([]uint16, MAX_PATH)
	r1, _, e := procGetWindowTextW.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r1 == 0 {
		return "", callErr("user32.dll!GetWindowTextW", e)
	}
	return syscall.UTF16ToString(buf[:r1]), nil
}

// GetClassName gọi user32.dll!GetClassNameW để đọc lớp cửa sổ.
func GetClassName(h HWND) (string, error) {
	buf := make([]uint16, 256)
	r1, _, e := procGetClassNameW.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r1 == 0 {
		return "", callErr("user32.dll!GetClassNameW", e)
	}
	return syscall.UTF16ToString(buf[:r1]), nil
}

// GetWindowThreadProcessID gọi user32.dll!GetWindowThreadProcessId.
func GetWindowThreadProcessID(h HWND) (pid, tid uint32, err error) {
	var processID uint32
	r1, _, e := procGetWindowThreadProcessID.Call(uintptr(h), uintptr(unsafe.Pointer(&processID)))
	if r1 == 0 {
		return 0, 0, callErr("user32.dll!GetWindowThreadProcessId", e)
	}
	return processID, uint32(r1), nil
}

// QueryForegroundWindow gom toàn bộ dữ liệu thật của cửa sổ đang ở tiền cảnh.
func QueryForegroundWindow() (WindowInfo, error) {
	hwnd, err := GetForegroundWindow()
	if err != nil {
		return WindowInfo{}, err
	}
	info := WindowInfo{HWND: hwnd}
	info.Title, _ = GetWindowText(hwnd)
	info.ClassName, _ = GetClassName(hwnd)
	info.PID, info.ThreadID, _ = GetWindowThreadProcessID(hwnd)
	return info, nil
}
