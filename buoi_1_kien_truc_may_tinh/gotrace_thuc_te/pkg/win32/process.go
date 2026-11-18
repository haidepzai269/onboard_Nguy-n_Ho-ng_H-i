package win32

import (
	"syscall"
	"unsafe"
)

// ProcessEntry là một dòng của bảng tiến trình Kernel, đã giải mã sang Go.
type ProcessEntry struct {
	PID       uint32
	ParentPID uint32
	Name      string
	Threads   uint32
}

// Snapshot là ảnh chụp bảng tiến trình kèm handle thật mà Kernel trả về.
type Snapshot struct {
	Handle Handle
	Procs  map[uint32]ProcessEntry
}

// Close giải phóng handle snapshot.
func (s *Snapshot) Close() {
	CloseHandle(s.Handle)
}

// TakeProcessSnapshot gọi kernel32.dll!CreateToolhelp32Snapshot +
// Process32FirstW/Process32NextW để đọc toàn bộ tiến trình đang sống.
func TakeProcessSnapshot() (*Snapshot, error) {
	h, _, e := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if Handle(h) == INVALID_HANDLE {
		return nil, callErr("kernel32.dll!CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)", e)
	}
	snap := &Snapshot{Handle: Handle(h), Procs: make(map[uint32]ProcessEntry)}

	var entry ProcessEntry32
	entry.Size = sizeOfProcessEntry32
	r1, _, e := procProcess32FirstW.Call(uintptr(snap.Handle), uintptr(unsafe.Pointer(&entry)))
	if r1 == 0 {
		snap.Close()
		return nil, callErr("kernel32.dll!Process32FirstW", e)
	}
	for {
		snap.Procs[entry.ProcessID] = ProcessEntry{
			PID:       entry.ProcessID,
			ParentPID: entry.ParentProcessID,
			Name:      syscall.UTF16ToString(entry.ExeFile[:]),
			Threads:   entry.Threads,
		}
		entry.Size = sizeOfProcessEntry32
		r1, _, e = procProcess32NextW.Call(uintptr(snap.Handle), uintptr(unsafe.Pointer(&entry)))
		if r1 == 0 {
			// ERROR_NO_MORE_FILES là điểm kết thúc bình thường của vòng quét.
			if errno, ok := e.(syscall.Errno); ok && errno != 0 && errno != 18 {
				snap.Close()
				return nil, callErr("kernel32.dll!Process32NextW", e)
			}
			break
		}
	}
	return snap, nil
}

// OpenProcess gọi kernel32.dll!OpenProcess với quyền truy vấn thông tin.
func OpenProcess(pid uint32) (Handle, error) {
	h, _, e := procOpenProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if h == 0 {
		return 0, callErr("kernel32.dll!OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, FALSE, "+itoa(pid)+")", e)
	}
	return Handle(h), nil
}

// QueryFullProcessImageName gọi kernel32.dll!QueryFullProcessImageNameW để lấy
// đường dẫn file thực thi thật trên ổ đĩa.
func QueryFullProcessImageName(h Handle) (string, error) {
	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	r1, _, e := procQueryFullProcessImageNameW.Call(
		uintptr(h), 0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r1 == 0 {
		return "", callErr("kernel32.dll!QueryFullProcessImageNameW", e)
	}
	return syscall.UTF16ToString(buf[:size]), nil
}

// MemoryCounters là bản sao PROCESS_MEMORY_COUNTERS đọc từ psapi.
type MemoryCounters struct {
	WorkingSetSize uintptr // RAM vật lý (RSS)
	PagefileUsage  uintptr // Commit / VMS
	PeakWorkingSet uintptr
}

// GetProcessMemoryInfo gọi psapi.dll!GetProcessMemoryInfo.
func GetProcessMemoryInfo(h Handle) (MemoryCounters, error) {
	var mc PROCESS_MEMORY_COUNTERS
	mc.Cb = sizeOfMemoryCounters
	r1, _, e := procGetProcessMemoryInfo.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&mc)),
		uintptr(sizeOfMemoryCounters),
	)
	if r1 == 0 {
		return MemoryCounters{}, callErr("psapi.dll!GetProcessMemoryInfo", e)
	}
	return MemoryCounters{
		WorkingSetSize: mc.WorkingSetSize,
		PagefileUsage:  mc.PagefileUsage,
		PeakWorkingSet: mc.PeakWorkingSetSize,
	}, nil
}

// CountThreads đếm số luồng thật của một tiến trình bằng
// CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD) + Thread32First/Thread32Next.
func CountThreads(pid uint32) (uint32, error) {
	h, _, e := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPTHREAD, 0)
	if Handle(h) == INVALID_HANDLE {
		return 0, callErr("kernel32.dll!CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD, 0)", e)
	}
	defer CloseHandle(Handle(h))

	var entry ThreadEntry32
	entry.Size = sizeOfThreadEntry32
	r1, _, e := procThread32First.Call(uintptr(h), uintptr(unsafe.Pointer(&entry)))
	if r1 == 0 {
		return 0, callErr("kernel32.dll!Thread32First", e)
	}
	var count uint32
	for {
		if entry.OwnerProcessID == pid {
			count++
		}
		entry.Size = sizeOfThreadEntry32
		r1, _, e = procThread32Next.Call(uintptr(h), uintptr(unsafe.Pointer(&entry)))
		if r1 == 0 {
			if errno, ok := e.(syscall.Errno); ok && errno != 0 && errno != 18 {
				return count, callErr("kernel32.dll!Thread32Next", e)
			}
			break
		}
	}
	return count, nil
}

// Probe là kết quả dò một tiến trình. Mỗi trường giữ nguyên lỗi thật của lời gọi
// tương ứng để tầng log có thể in ra "bị từ chối truy cập" thay vì bịa số liệu.
type Probe struct {
	PID        uint32
	Handle     Handle
	OpenErr    error
	ImagePath  string
	ImageErr   error
	Memory     MemoryCounters
	MemoryErr  error
	Threads    uint32
	ThreadsErr error
}

// ProbeProcess mở handle và đọc đường dẫn, bộ nhớ, số luồng của tiến trình.
func ProbeProcess(pid uint32) Probe {
	p := Probe{PID: pid}
	p.Handle, p.OpenErr = OpenProcess(pid)
	if p.OpenErr != nil {
		return p
	}
	defer CloseHandle(p.Handle)
	p.ImagePath, p.ImageErr = QueryFullProcessImageName(p.Handle)
	p.Memory, p.MemoryErr = GetProcessMemoryInfo(p.Handle)
	p.Threads, p.ThreadsErr = CountThreads(pid)
	return p
}

func itoa(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
