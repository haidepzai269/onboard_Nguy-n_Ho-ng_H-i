package win32

import (
	"unsafe"
)

// Key là một phím người dùng vừa nhấn trên terminal, đọc trực tiếp bằng
// kernel32.dll!ReadConsoleInputW (không cần nhấn Enter).
type Key struct {
	VirtualKey uint16
	Rune       rune
	Resize     bool
}

// Console giữ handle và mode gốc của terminal để khôi phục khi thoát.
type Console struct {
	in      Handle
	out     Handle
	inMode  uint32
	outMode uint32

	// ANSI cho biết terminal đã bật ENABLE_VIRTUAL_TERMINAL_PROCESSING hay chưa.
	// Nếu chưa, tầng UI sẽ vẽ không màu thay vì in ra ký tự điều khiển thô.
	ANSI bool
}

// NewConsole chuyển terminal sang chế độ thô: tắt line input và echo để nhận
// từng phím ngay lập tức, bật xử lý ANSI cho stdout và đặt codepage UTF-8.
func NewConsole() (*Console, error) {
	in, _, e := procGetStdHandle.Call(STD_INPUT_HANDLE)
	if Handle(in) == 0 || Handle(in) == INVALID_HANDLE {
		return nil, callErr("kernel32.dll!GetStdHandle(STD_INPUT_HANDLE)", e)
	}
	out, _, e := procGetStdHandle.Call(STD_OUTPUT_HANDLE)
	if Handle(out) == 0 || Handle(out) == INVALID_HANDLE {
		return nil, callErr("kernel32.dll!GetStdHandle(STD_OUTPUT_HANDLE)", e)
	}

	c := &Console{in: Handle(in), out: Handle(out)}
	if r, _, e := procGetConsoleMode.Call(in, uintptr(unsafe.Pointer(&c.inMode))); r == 0 {
		return nil, callErr("kernel32.dll!GetConsoleMode(STD_INPUT_HANDLE)", e)
	}
	rawMode := (c.inMode &^ (ENABLE_LINE_INPUT | ENABLE_ECHO_INPUT)) | ENABLE_WINDOW_INPUT
	if r, _, e := procSetConsoleMode.Call(in, uintptr(rawMode)); r == 0 {
		return nil, callErr("kernel32.dll!SetConsoleMode(STD_INPUT_HANDLE)", e)
	}

	if r, _, _ := procGetConsoleMode.Call(out, uintptr(unsafe.Pointer(&c.outMode))); r != 0 {
		if r, _, _ := procSetConsoleMode.Call(out, uintptr(c.outMode|ENABLE_VIRTUAL_TERMINAL_PROCESSING)); r != 0 {
			c.ANSI = true
		}
	}
	// Codepage UTF-8 để tiếng Việt và emoji hiển thị đúng trên Windows Terminal.
	procSetConsoleOutputCP.Call(CP_UTF8)

	return c, nil
}

// Restore trả terminal về mode ban đầu.
func (c *Console) Restore() {
	if c == nil {
		return
	}
	procSetConsoleMode.Call(uintptr(c.in), uintptr(c.inMode))
	procSetConsoleMode.Call(uintptr(c.out), uintptr(c.outMode))
}

// Size trả về số cột và số dòng khả dụng của cửa sổ terminal hiện tại.
func (c *Console) Size() (width, height int) {
	var info CONSOLE_SCREEN_BUFFER_INFO
	r, _, _ := procGetConsoleScreenBufferInfo.Call(uintptr(c.out), uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 100, 30
	}
	width = int(info.Window.Right-info.Window.Left) + 1
	height = int(info.Window.Bottom-info.Window.Top) + 1
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}
	return width, height
}

// ReadKey chờ đúng một phím được nhấn xuống. Sự kiện thay đổi kích thước cửa sổ
// cũng được trả về (Resize = true) để UI vẽ lại; sự kiện chuột bị bỏ qua.
func (c *Console) ReadKey() (Key, error) {
	for {
		var rec INPUT_RECORD
		var read uint32
		r, _, e := procReadConsoleInputW.Call(
			uintptr(c.in),
			uintptr(unsafe.Pointer(&rec)),
			1,
			uintptr(unsafe.Pointer(&read)),
		)
		if r == 0 {
			return Key{}, callErr("kernel32.dll!ReadConsoleInputW", e)
		}
		if read == 0 {
			continue
		}
		switch rec.EventType {
		case WINDOW_BUFFER_SIZE_EVENT:
			return Key{Resize: true}, nil
		case KEY_EVENT:
			ke := (*KEY_EVENT_RECORD)(unsafe.Pointer(&rec.Event[0]))
			if ke.KeyDown == 0 {
				continue
			}
			return Key{VirtualKey: ke.VirtualKeyCode, Rune: rune(ke.UnicodeChar)}, nil
		}
	}
}
