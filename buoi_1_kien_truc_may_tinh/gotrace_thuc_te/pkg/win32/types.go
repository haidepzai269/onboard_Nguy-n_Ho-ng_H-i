package win32

// Handle là một handle nhân Windows (HANDLE / HWND / hSnapshot).
type Handle uintptr

// Hằng số Win32 dùng trong dự án.
const (
	MAX_PATH = 260

	TH32CS_SNAPPROCESS = 0x00000002
	TH32CS_SNAPTHREAD  = 0x00000004

	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

	STD_INPUT_HANDLE  = 0xFFFFFFF6 // (DWORD)-10
	STD_OUTPUT_HANDLE = 0xFFFFFFF5 // (DWORD)-11
	INVALID_HANDLE    = ^Handle(0)

	ENABLE_PROCESSED_INPUT = 0x0001
	ENABLE_LINE_INPUT      = 0x0002
	ENABLE_ECHO_INPUT      = 0x0004
	ENABLE_WINDOW_INPUT    = 0x0008

	ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	CP_UTF8                            = 65001

	KEY_EVENT                = 0x0001
	MOUSE_EVENT              = 0x0002
	WINDOW_BUFFER_SIZE_EVENT = 0x0004

	VK_RETURN = 0x0D
	VK_ESCAPE = 0x1B
	VK_SPACE  = 0x20
	VK_PRIOR  = 0x21 // Page Up
	VK_NEXT   = 0x22 // Page Down
	VK_UP     = 0x26
	VK_DOWN   = 0x28
	VK_C      = 0x43
	VK_Q      = 0x51

	// Kích thước thật của các struct bên C (sizeof, đã gồm padding căn lề cuối
	// struct). PROCESSENTRY32W: 44 byte tiêu đề + 260 WCHAR = 564, làm tròn
	// theo alignment 8 thành 568. Kernel từ chối dwSize sai bằng ERROR_BAD_LENGTH.
	sizeOfProcessEntry32 = 568
	sizeOfThreadEntry32  = 28
	sizeOfMemoryCounters = 72
)

// PROCESSENTRY32W - bản ghi một tiến trình trong bảng tiến trình của Kernel.
type ProcessEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [MAX_PATH]uint16
}

// THREADENTRY32 - bản ghi một luồng trong bảng luồng của Kernel.
type ThreadEntry32 struct {
	Size           uint32
	Usage          uint32
	ThreadID       uint32
	OwnerProcessID uint32
	BasePri        int32
	DeltaPri       int32
	Flags          uint32
}

// PROCESS_MEMORY_COUNTERS - thống kê bộ nhớ thật của tiến trình do psapi trả về.
type PROCESS_MEMORY_COUNTERS struct {
	Cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

// INPUT_RECORD - một bản ghi đầu vào console. Union 16 byte, căn lề 2+2.
type INPUT_RECORD struct {
	EventType uint16
	_         uint16
	Event     [16]byte
}

// KEY_EVENT_RECORD - phần union của INPUT_RECORD khi EventType == KEY_EVENT.
type KEY_EVENT_RECORD struct {
	KeyDown         int32
	RepeatCount     uint16
	VirtualKeyCode  uint16
	VirtualScanCode uint16
	UnicodeChar     uint16
	ControlKeyState uint32
}

// COORD / SMALL_RECT / CONSOLE_SCREEN_BUFFER_INFO dùng để lấy kích thước cửa sổ terminal.
type COORD struct {
	X int16
	Y int16
}

type SMALL_RECT struct {
	Left   int16
	Top    int16
	Right  int16
	Bottom int16
}

type CONSOLE_SCREEN_BUFFER_INFO struct {
	Size              COORD
	CursorPosition    COORD
	Attributes        uint16
	Window            SMALL_RECT
	MaximumWindowSize COORD
}
