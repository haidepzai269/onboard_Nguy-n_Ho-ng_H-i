// Package monitor chứa hai engine chạy song song: bắt thao tác người dùng và
// bắt phản ứng của Kernel.
package monitor

import (
	"context"
	"time"

	"gotrace/pkg/control"
	"gotrace/pkg/models"
	"gotrace/pkg/store"
	"gotrace/pkg/win32"
)

// UserActionTracker theo dõi cửa sổ tiền cảnh bằng polling.
type UserActionTracker struct {
	Store    *store.Store
	Interval time.Duration
	// Control giữ trạng thái tạm dừng dùng chung với UI. Nil = luôn chạy.
	Control *control.Control

	lastHWND  win32.HWND
	lastTitle string
	lastPID   uint32
	started   bool
}

// Run chạy vòng lặp polling cho tới khi context bị huỷ.
func (t *UserActionTracker) Run(ctx context.Context) {
	interval := t.Interval
	if interval <= 0 {
		interval = 150 * time.Millisecond
	}
	t.poll() // chụp trạng thái ban đầu
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	wasPaused := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if t.Control != nil && t.Control.Paused() {
				wasPaused = true
				continue // tạm dừng: không gọi user32.dll
			}
			if wasPaused {
				// Vừa chạy tiếp: chụp lại trạng thái mà không ghi log, để cửa sổ
				// đã đổi trong lúc tạm dừng không bị báo thành sự kiện mới.
				wasPaused = false
				t.rebaseline()
				continue
			}
			t.poll()
		}
	}
}

// rebaseline cập nhật mốc so sánh hiện tại mà không phát sinh log.
func (t *UserActionTracker) rebaseline() {
	info, err := win32.QueryForegroundWindow()
	if err != nil {
		return
	}
	t.lastHWND, t.lastTitle, t.lastPID = info.HWND, info.Title, info.PID
	t.started = true
}

func (t *UserActionTracker) poll() {
	info, err := win32.QueryForegroundWindow()
	if err != nil {
		return // không có cửa sổ tiền cảnh (màn hình khoá, desktop chuyển tiếp...)
	}
	if t.started && info.HWND == t.lastHWND && info.Title == t.lastTitle && info.PID == t.lastPID {
		return
	}
	initial := !t.started
	prev := t.lastTitle
	t.lastHWND, t.lastTitle, t.lastPID = info.HWND, info.Title, info.PID
	t.started = true
	t.Store.Add(t.buildEntry(info, prev, initial))
}

func (t *UserActionTracker) buildEntry(info win32.WindowInfo, prevTitle string, initial bool) models.LogEntry {
	probe := win32.ProbeProcess(info.PID)

	processName := probe.ImagePath
	if probe.ImageErr != nil || processName == "" {
		processName = "(không đọc được tên tiến trình)"
	}

	title := info.Title
	if title == "" {
		title = "(cửa sổ không có tiêu đề)"
	}

	why := "Người dùng vừa click chuột hoặc dùng Alt+Tab để chuyển tiêu điểm sang cửa sổ khác; USER32 cập nhật cửa sổ tiền cảnh trong Desktop Window Manager."
	if prevTitle != "" {
		why += "\nCửa sổ trước đó: \"" + prevTitle + "\""
	}
	if initial {
		why = "GoTrace vừa khởi động và chụp trạng thái cửa sổ tiền cảnh hiện tại bằng lời gọi API thật (đây không phải sự kiện chuyển cửa sổ)."
	}

	apiCalls := []string{
		"[Call 1] user32.dll!GetForegroundWindow()\n↳ Trả về: HWND " + win32.FormatHandle(win32.Handle(info.HWND)),
		"[Call 2] user32.dll!GetWindowTextW(HWND, buffer, 260)\n↳ Tiêu đề UTF-16: \"" + title + "\"",
		"[Call 3] user32.dll!GetWindowThreadProcessId(HWND, &pid)\n↳ PID: " + itoa(info.PID) + " | ThreadID: " + itoa(info.ThreadID),
		"[Call 4] user32.dll!GetClassNameW(HWND, buffer, 256)\n↳ Lớp cửa sổ: \"" + info.ClassName + "\"",
	}
	apiCalls = append(apiCalls, probeAPICalls(probe)...)

	entry := models.LogEntry{
		Timestamp: time.Now(),
		EventType: models.EventUserAction,
		Icon:      "🖱️",
		Summary:   "Chuyển tiêu điểm cửa sổ sang \"" + title + "\" (PID: " + itoa(info.PID) + ")",
		Why:       why,
		Subsystem: "Windows Subsystem: USER32 Window Manager (win32kfull.sys + user32.dll)\n" +
			"Bảng quản lý nguồn: Desktop Window Manager (DWM) giữ danh sách top-level window của phiên đăng nhập.",
		APICalls:        apiCalls,
		HardwareContext: probeHardware(probe, processName),
	}
	if initial {
		entry.Summary = "Trạng thái khởi tạo: \"" + title + "\" (PID: " + itoa(info.PID) + ")"
	}
	return entry
}

// probeAPICalls dựng các dòng API đọc tiến trình, giữ nguyên kết quả thật hoặc
// lỗi thật của từng lời gọi.
func probeAPICalls(p win32.Probe) []string {
	calls := []string{
		"[Call 5] kernel32.dll!OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, FALSE, " + itoa(p.PID) + ")",
	}
	if p.OpenErr != nil {
		return append(calls, "↳ Thất bại: "+p.OpenErr.Error()+"\n↳ Không thể đọc thêm thông tin tiến trình.")
	}
	calls = append(calls, "↳ Trả về: Process Handle "+win32.FormatHandle(p.Handle))

	calls = append(calls, "[Call 6] kernel32.dll!QueryFullProcessImageNameW(hProcess, 0, lpExeName, &dwSize)")
	if p.ImageErr != nil {
		calls = append(calls, "↳ Thất bại: "+p.ImageErr.Error())
	} else {
		calls = append(calls, "↳ Đường dẫn SSD: \""+p.ImagePath+"\"")
	}

	calls = append(calls, "[Call 7] psapi.dll!GetProcessMemoryInfo(hProcess, &pmc, sizeof(pmc))")
	if p.MemoryErr != nil {
		calls = append(calls, "↳ Thất bại: "+p.MemoryErr.Error())
	} else {
		calls = append(calls, "↳ WorkingSetSize: "+models.FormatMB(p.Memory.WorkingSetSize)+
			" | PagefileUsage: "+models.FormatMB(p.Memory.PagefileUsage))
	}

	calls = append(calls, "[Call 8] kernel32.dll!CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD, 0) + Thread32Next")
	if p.ThreadsErr != nil {
		calls = append(calls, "↳ Thất bại: "+p.ThreadsErr.Error())
	} else {
		calls = append(calls, "↳ Số luồng của PID "+itoa(p.PID)+": "+itoa(p.Threads))
	}
	return calls
}

// probeHardware dựng 3 trục CPU/Bộ nhớ từ dữ liệu probe thật.
func probeHardware(p win32.Probe, processName string) models.HardwareContext {
	space := "Ảnh thực thi \"" + processName + "\" đang được OS Loader ánh xạ từ ổ đĩa vào DRAM của tiến trình."
	memory := ""
	cpu := "Chuyển ngữ cảnh User Mode (Ring 3) -> Kernel Mode (Ring 0) mỗi khi USER32 hỏi trạng thái cửa sổ; Kernel trả kết quả rồi quay lại Ring 3."

	if p.MemoryErr != nil {
		memory = "Không đọc được PROCESS_MEMORY_COUNTERS: " + p.MemoryErr.Error()
	} else {
		memory = "RAM vật lý cấp cho tiến trình (Working Set / RSS): " + models.FormatMB(p.Memory.WorkingSetSize) +
			"\nBộ nhớ ảo đã cam kết (Commit Size / VMS): " + models.FormatMB(p.Memory.PagefileUsage)
	}
	if p.ThreadsErr != nil {
		cpu += "\nSố luồng: không đọc được (" + p.ThreadsErr.Error() + ")"
	} else {
		cpu += "\nSố luồng đang được Kernel xếp lịch trên CPU core: " + itoa(p.Threads) + " threads"
	}
	return models.HardwareContext{CPUDetails: cpu, MemoryDetails: memory, SpaceAxis: space}
}
