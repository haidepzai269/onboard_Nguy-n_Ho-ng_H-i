package monitor

import (
	"context"
	"sort"
	"strings"
	"time"

	"gotrace/pkg/control"
	"gotrace/pkg/models"
	"gotrace/pkg/store"
	"gotrace/pkg/win32"
)

// filterReportTicks là số vòng quét giữa hai lần báo cáo "đã lọc bao nhiêu sự kiện".
const filterReportTicks = 20 // 20 × 250ms = 5 giây

// OSReactionWatcher phát hiện tiến trình sinh ra / biến mất bằng cách so sánh
// hai ảnh chụp bảng tiến trình của Kernel.
type OSReactionWatcher struct {
	Store    *store.Store
	Interval time.Duration
	// Control giữ trạng thái tạm dừng và danh sách tiến trình bị lọc. Nil = luôn chạy, không lọc.
	Control *control.Control
}

// Run giữ baseline ngay khi khởi động rồi lặp vô hạn cho tới khi context bị huỷ.
func (w *OSReactionWatcher) Run(ctx context.Context) {
	interval := w.Interval
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}
	var prev map[uint32]win32.ProcessEntry
	if snap, err := win32.TakeProcessSnapshot(); err == nil {
		prev = snap.Procs
		snap.Close()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	ticks := 0
	wasPaused := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if w.paused() {
				wasPaused = true
				continue // tạm dừng: không gọi API nào
			}
			if wasPaused {
				// Vừa chạy tiếp: chốt lại baseline để những thay đổi xảy ra
				// trong lúc tạm dừng không bị báo thành một loạt sự kiện ảo.
				wasPaused = false
				if snap, err := win32.TakeProcessSnapshot(); err == nil {
					prev = snap.Procs
					snap.Close()
				}
				continue
			}
			prev = w.tick(prev)
			ticks++
			if ticks%filterReportTicks == 0 {
				w.reportFiltered()
			}
		}
	}
}

func (w *OSReactionWatcher) paused() bool {
	return w.Control != nil && w.Control.Paused()
}

// muted cho biết tiến trình có bị lọc khỏi log không (đồng thời đếm để báo cáo).
func (w *OSReactionWatcher) muted(name string) bool {
	return w.Control.Muted(name)
}

// reportFiltered ghi một dòng tổng kết thay cho hàng loạt sự kiện nhiễu vừa bị ẩn.
func (w *OSReactionWatcher) reportFiltered() {
	if w.Control == nil {
		return
	}
	total, detail := w.Control.TakeSuppressed()
	if total == 0 {
		return
	}
	w.Store.Add(models.LogEntry{
		Timestamp: time.Now(),
		EventType: models.EventOSReaction,
		Icon:      "🔇",
		Summary:   "Bộ lọc đã ẩn " + itoa(uint32(total)) + " sự kiện nhiễu: " + strings.Join(detail, ", "),
		Why: "Một số công cụ trên máy (trình giám sát, antivirus, terminal, agent) liên tục spawn " +
			"tiến trình sống rất ngắn. Nếu in hết, log sẽ bị nhiễu và che mất sự kiện bạn cần đọc.",
		Subsystem: "GoTrace Filter — bộ lọc cục bộ của GoTrace, không phải API của Windows.\n" +
			"Sự kiện vẫn là dữ liệu thật: Kernel tạo/thu hồi tiến trình thật và Process32NextW vẫn đọc ra chúng.",
		APICalls: []string{
			"[Call 1] kernel32.dll!Process32NextW(hSnapshot, &pe32)\n↳ Đọc được tiến trình có tên trong danh sách lọc.",
			"[Bộ lọc] So khớp tên ảnh thực thi (không phân biệt hoa thường) ngay sau khi so sánh hai ảnh chụp.\n↳ Không gọi thêm API nào; sự kiện bị ẩn khỏi danh sách log và được đếm ở đây.",
		},
		HardwareContext: models.HardwareContext{
			SpaceAxis:     "(không áp dụng cho dòng tổng kết bộ lọc)",
			MemoryDetails: "(không áp dụng) Các tiến trình bị lọc vẫn tiêu tốn RAM thật trong lúc tồn tại.",
			CPUDetails:    "(không áp dụng) Bộ lọc chỉ bỏ qua việc ghi log, không can thiệp vào lập lịch CPU.",
		},
	})
}

func (w *OSReactionWatcher) tick(prev map[uint32]win32.ProcessEntry) map[uint32]win32.ProcessEntry {
	snap, err := win32.TakeProcessSnapshot()
	if err != nil {
		return prev // giữ baseline cũ, thử lại ở vòng sau
	}
	defer snap.Close()

	if prev == nil {
		return snap.Procs // baseline đầu tiên: chưa coi là sự kiện
	}

	added, removed := diffProcesses(prev, snap.Procs)
	for _, p := range added {
		w.emitNew(p, snap, added)
	}
	for _, p := range removed {
		w.emitGone(p, snap)
	}
	return snap.Procs
}

// diffProcesses trả về danh sách PID mới xuất hiện và PID vừa biến mất.
func diffProcesses(prev, cur map[uint32]win32.ProcessEntry) (added, removed []win32.ProcessEntry) {
	for pid, p := range cur {
		if _, ok := prev[pid]; !ok {
			added = append(added, p)
		}
	}
	for pid, p := range prev {
		if _, ok := cur[pid]; !ok {
			removed = append(removed, p)
		}
	}
	sort.Slice(added, func(i, j int) bool { return added[i].PID < added[j].PID })
	sort.Slice(removed, func(i, j int) bool { return removed[i].PID < removed[j].PID })
	return added, removed
}

func (w *OSReactionWatcher) emitNew(p win32.ProcessEntry, snap *win32.Snapshot, batch []win32.ProcessEntry) {
	if w.muted(p.Name) {
		return
	}
	probe := win32.ProbeProcess(p.PID)

	parent := "không còn trong bảng tiến trình"
	if pp, ok := snap.Procs[p.ParentPID]; ok {
		parent = "\"" + pp.Name + "\" [PID: " + itoa(pp.PID) + "]"
	}

	why := "Người dùng double-click shortcut, gõ lệnh, hoặc một tiến trình cha gửi lệnh nạp tiến trình con vào hệ thống.\n" +
		"Tiến trình cha (PPID " + itoa(p.ParentPID) + "): " + parent

	apiCalls := []string{
		"[Call 1] kernel32.dll!CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)\n↳ Trả về: Snapshot Handle " + win32.FormatHandle(snap.Handle) + " chứa " + itoa(uint32(len(batch))) + " PID mới.",
		"[Call 2] kernel32.dll!Process32NextW(hSnapshot, &pe32)\n↳ Phát hiện PID mới " + itoa(p.PID) + " (\"" + p.Name + "\") chưa có trong bảng theo dõi của GoTrace.\n↳ Parent Process ID (PPID): " + itoa(p.ParentPID),
	}
	apiCalls = append(apiCalls, probeAPICalls(probe)...)

	entry := models.LogEntry{
		Timestamp: time.Now(),
		EventType: models.EventOSReaction,
		Icon:      "⚙️",
		Summary:   "Kernel tạo Process mới: \"" + p.Name + "\" [PID: " + itoa(p.PID) + "]",
		Why:       why,
		Subsystem: "Windows Subsystem: Windows Kernel Process Manager (ntoskrnl.exe)\n" +
			"Bảng quản lý nguồn: Process Active Table & EPROCESS Block trong RAM Kernel.",
		APICalls:        apiCalls,
		HardwareContext: newProcessHardware(probe, p),
	}
	w.Store.Add(entry)
}

func (w *OSReactionWatcher) emitGone(p win32.ProcessEntry, snap *win32.Snapshot) {
	if w.muted(p.Name) {
		return
	}
	w.Store.Add(models.LogEntry{
		Timestamp: time.Now(),
		EventType: models.EventOSReaction,
		Icon:      "❌",
		Summary:   "Kernel thu hồi tiến trình: \"" + p.Name + "\" [PID: " + itoa(p.PID) + "]",
		Why: "Ứng dụng bị người dùng đóng hoặc crash; Kernel kích hoạt cơ chế dọn dẹp, " +
			"giải phóng handle và xoá PID khỏi bảng tiến trình.",
		Subsystem: "Windows Subsystem: Windows Kernel Process Manager (ntoskrnl.exe)\n" +
			"Bảng quản lý nguồn: Process Active Table & EPROCESS Block trong RAM Kernel.",
		APICalls: []string{
			"[Call 1] kernel32.dll!CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)\n↳ Trả về: Snapshot Handle " + win32.FormatHandle(snap.Handle),
			"[Call 2] kernel32.dll!Process32NextW(hSnapshot, &pe32)\n↳ Bảng tiến trình mới không còn PID " + itoa(p.PID) + " (\"" + p.Name + "\") — EPROCESS đã bị thu hồi.",
			"[Call 3] kernel32.dll!CloseHandle(hSnapshot)\n↳ Ảnh chụp bảng tiến trình được giải phóng sau khi so sánh.",
		},
		HardwareContext: models.HardwareContext{
			SpaceAxis: "Kernel giải phóng vùng nhớ ảnh thực thi của tiến trình khỏi DRAM khi trang cuối cùng được thu hồi.",
			MemoryDetails: "Working Set và Pagefile của tiến trình đã được Kernel trả lại cho hệ thống.\n" +
				"Không còn đọc được số byte đã giải phóng vì PID đã bị thu hồi (tránh bịa số liệu).",
			CPUDetails: "Các luồng của tiến trình bị xoá khỏi hàng đợi lập lịch; Kernel thu hồi thread object và đóng handle còn treo ở Ring 0.",
		},
	})
}

func newProcessHardware(p win32.Probe, entry win32.ProcessEntry) models.HardwareContext {
	path := p.ImagePath
	if p.ImageErr != nil || path == "" {
		path = "(không đọc được đường dẫn: " + errText(p.ImageErr) + ")"
	}
	memory := ""
	if p.MemoryErr != nil {
		memory = "Không đọc được PROCESS_MEMORY_COUNTERS: " + p.MemoryErr.Error()
	} else {
		memory = "RAM vật lý cấp ban đầu (Working Set / RSS): " + models.FormatMB(p.Memory.WorkingSetSize) +
			"\nBộ nhớ ảo đã cam kết (Commit Size / VMS): " + models.FormatMB(p.Memory.PagefileUsage)
	}
	threads := itoa(entry.Threads)
	if p.ThreadsErr == nil {
		threads = itoa(p.Threads)
	}
	return models.HardwareContext{
		SpaceAxis:     "OS Loader phân trang ảnh thực thi từ SSD vào DRAM: " + path,
		MemoryDetails: memory,
		CPUDetails: "Kernel chuyển ngữ cảnh User Mode (Ring 3) -> Kernel Mode (Ring 0) để khởi tạo PCB (Process Control Block).\n" +
			"Số lượng luồng ban đầu: " + threads + " threads đang được nạp vào CPU cores.",
	}
}

func errText(err error) string {
	if err == nil {
		return "không rõ lỗi"
	}
	return err.Error()
}
