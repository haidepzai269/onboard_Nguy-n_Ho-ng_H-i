package win32

import (
	"os"
	"strings"
	"testing"
	"unsafe"
)

// TestTakeProcessSnapshotGoiThatAPI chứng minh ảnh chụp bảng tiến trình là dữ
// liệu thật do Kernel trả về, không phải dữ liệu mô phỏng.
func TestTakeProcessSnapshotGoiThatAPI(t *testing.T) {
	snap, err := TakeProcessSnapshot()
	if err != nil {
		t.Fatalf("CreateToolhelp32Snapshot thất bại: %v", err)
	}
	defer snap.Close()

	if len(snap.Procs) < 5 {
		t.Fatalf("bảng tiến trình thật phải có nhiều hơn 5 dòng, nhận được %d", len(snap.Procs))
	}
	self, ok := snap.Procs[uint32(os.Getpid())]
	if !ok {
		t.Fatalf("không tìm thấy chính tiến trình test (PID %d) trong snapshot", os.Getpid())
	}
	if self.Name == "" {
		t.Fatal("Process32FirstW/NextW trả về tên tiến trình rỗng")
	}
	if self.Threads == 0 {
		t.Fatal("cntThreads của tiến trình test phải lớn hơn 0")
	}
}

// TestProbeSelfDocDuLieuThat kiểm tra OpenProcess +
// QueryFullProcessImageNameW + GetProcessMemoryInfo trên chính tiến trình test.
func TestProbeSelfDocDuLieuThat(t *testing.T) {
	p := ProbeProcess(uint32(os.Getpid()))
	if p.OpenErr != nil {
		t.Fatalf("OpenProcess thất bại: %v", p.OpenErr)
	}
	if p.ImageErr != nil {
		t.Fatalf("QueryFullProcessImageNameW thất bại: %v", p.ImageErr)
	}
	if !strings.HasSuffix(strings.ToLower(p.ImagePath), ".exe") {
		t.Fatalf("đường dẫn ảnh thực thi không hợp lệ: %q", p.ImagePath)
	}
	if p.MemoryErr != nil {
		t.Fatalf("GetProcessMemoryInfo thất bại: %v", p.MemoryErr)
	}
	if p.Memory.WorkingSetSize == 0 {
		t.Fatal("WorkingSetSize phải là số byte thật lớn hơn 0")
	}
	if p.Memory.PagefileUsage == 0 {
		t.Fatal("PagefileUsage phải là số byte thật lớn hơn 0")
	}
	if p.ThreadsErr != nil {
		t.Fatalf("CountThreads thất bại: %v", p.ThreadsErr)
	}
	if p.Threads == 0 {
		t.Fatal("số luồng thật của tiến trình test phải lớn hơn 0")
	}
}

// TestForegroundWindowTraVeHandleThat kiểm tra user32 trả về cửa sổ tiền cảnh
// thật của phiên đăng nhập hiện tại.
func TestForegroundWindowTraVeHandleThat(t *testing.T) {
	info, err := QueryForegroundWindow()
	if err != nil {
		t.Skipf("phiên hiện tại không có cửa sổ tiền cảnh (headless/lock screen): %v", err)
	}
	if info.HWND == 0 {
		t.Fatal("HWND không được bằng 0 khi GetForegroundWindow thành công")
	}
	if info.PID == 0 {
		t.Fatalf("GetWindowThreadProcessId phải trả về PID thật, nhận được 0 (HWND %s)", FormatHandle(Handle(info.HWND)))
	}
}

// TestCauTrucCoDungKichThuocC đảm bảo layout bộ nhớ khớp với struct bên C,
// nếu không các lời gọi Win32 sẽ ghi rác vào bộ nhớ Go.
func TestCauTrucCoDungKichThuocC(t *testing.T) {
	var pe ProcessEntry32
	if off := unsafe.Offsetof(pe.ExeFile); off != 44 {
		t.Fatalf("PROCESSENTRY32W.szExeFile phải ở offset 44, nhận được %d", off)
	}
	if size := unsafe.Sizeof(pe); size != sizeOfProcessEntry32 {
		t.Fatalf("PROCESSENTRY32W phải là %d byte, nhận được %d", sizeOfProcessEntry32, size)
	}

	var mc PROCESS_MEMORY_COUNTERS
	if off := unsafe.Offsetof(mc.WorkingSetSize); off != 16 {
		t.Fatalf("PROCESS_MEMORY_COUNTERS.WorkingSetSize phải ở offset 16, nhận được %d", off)
	}
	if off := unsafe.Offsetof(mc.PagefileUsage); off != 56 {
		t.Fatalf("PROCESS_MEMORY_COUNTERS.PagefileUsage phải ở offset 56, nhận được %d", off)
	}
	if size := unsafe.Sizeof(mc); size != sizeOfMemoryCounters {
		t.Fatalf("PROCESS_MEMORY_COUNTERS phải là %d byte, nhận được %d", sizeOfMemoryCounters, size)
	}

	var te ThreadEntry32
	if off := unsafe.Offsetof(te.OwnerProcessID); off != 12 {
		t.Fatalf("THREADENTRY32.th32OwnerProcessID phải ở offset 12, nhận được %d", off)
	}
	if size := unsafe.Sizeof(te); size != sizeOfThreadEntry32 {
		t.Fatalf("THREADENTRY32 phải là %d byte, nhận được %d", sizeOfThreadEntry32, size)
	}

	var rec INPUT_RECORD
	if size := unsafe.Sizeof(rec); size != 20 {
		t.Fatalf("INPUT_RECORD phải là 20 byte, nhận được %d", size)
	}
	var ke KEY_EVENT_RECORD
	if size := unsafe.Sizeof(ke); size != 16 {
		t.Fatalf("KEY_EVENT_RECORD phải là 16 byte, nhận được %d", size)
	}
	if off := unsafe.Offsetof(ke.ControlKeyState); off != 12 {
		t.Fatalf("KEY_EVENT_RECORD.dwControlKeyState phải ở offset 12, nhận được %d", off)
	}
}
