package ui

import (
	"strings"
	"testing"
	"time"

	"gotrace/pkg/models"
)

func sampleEntry() models.LogEntry {
	return models.LogEntry{
		ID:        7,
		Timestamp: time.Date(2026, 11, 18, 11, 25, 5, 120_000_000, time.Local),
		EventType: models.EventOSReaction,
		Icon:      "⚙️",
		Summary:   "Kernel tạo Process mới: \"chrome.exe\" [PID: 19504]",
		Why:       "Người dùng double-click shortcut của Chrome.",
		Subsystem: "Windows Subsystem: Windows Kernel Process Manager (ntoskrnl.exe)\n" +
			"Bảng quản lý nguồn: Process Active Table & EPROCESS Block trong RAM Kernel.",
		APICalls: []string{
			"[Call 1] kernel32.dll!CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)\n↳ Trả về: Snapshot Handle 0x000000C8",
			"[Call 5] psapi.dll!GetProcessMemoryInfo(hProcess, &pmc, sizeof(pmc))\n↳ WorkingSetSize: 38.45 MB",
		},
		HardwareContext: models.HardwareContext{
			SpaceAxis:     "OS Loader phân trang ảnh thực thi từ SSD vào DRAM.",
			MemoryDetails: "RAM vật lý cấp ban đầu (Working Set / RSS): 38.45 MB",
			CPUDetails:    "Số lượng luồng ban đầu: 12 threads đang được nạp vào CPU cores.",
		},
	}
}

func blockText(lines []bodyLine) string {
	parts := make([]string, len(lines))
	for i, l := range lines {
		parts[i] = l.text
	}
	return strings.Join(parts, "\n")
}

func TestThuGonChiMotDong(t *testing.T) {
	e := sampleEntry()
	lines := entryBlock(e)
	if len(lines) != 1 {
		t.Fatalf("log thu gọn phải đúng 1 dòng, nhận được %d", len(lines))
	}
	if !strings.HasPrefix(lines[0].text, "▶ [11:25:05.120] ⚙️ [OS REACTION] ") {
		t.Fatalf("dòng thu gọn sai định dạng: %q", lines[0].text)
	}
}

func TestMoRongCoDuBonNhanhCay(t *testing.T) {
	e := sampleEntry()
	e.IsExpanded = true
	lines := entryBlock(e)
	text := blockText(lines)

	if !strings.HasPrefix(lines[0].text, "▼ [11:25:05.120]") {
		t.Fatalf("log mở rộng phải bắt đầu bằng ▼: %q", lines[0].text)
	}
	want := []string{
		"1. BỐI CẢNH & NGUYÊN DO (WHY)",
		"2. NGUỒN GỐC DỮ LIỆU & SUBSYSTEM (WHERE)",
		"3. HÀM WIN32 API / SYSCALL ĐÃ THỰC THI (HOW/APIs)",
		"4. PHẢN ỨNG CỦA CPU & BỘ NHỚ LIÊN QUAN",
		"Trục Không Gian (Storage -> RAM)",
		"Trục Thực Thi (Process Memory)",
		"Trục Tương Tác (CPU Rings)",
		"kernel32.dll!CreateToolhelp32Snapshot",
		"psapi.dll!GetProcessMemoryInfo",
		"↳ Trả về: Snapshot Handle 0x000000C8",
		"Người dùng double-click shortcut của Chrome.",
	}
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Fatalf("cây mở rộng thiếu %q:\n%s", w, text)
		}
	}
	if strings.Contains(text, "Cửa sổ trước đó") {
		t.Fatal("nội dung không thuộc entry này lại xuất hiện trong cây")
	}
	for _, l := range lines {
		if l.isHeader && l.id != e.ID {
			t.Fatalf("dòng %q gắn sai ID log", l.text)
		}
	}
	// Mọi dòng của cây phải thuộc cùng một log.
	for _, l := range lines[1:] {
		if l.id != e.ID {
			t.Fatalf("dòng %q phải thuộc log ID %d, nhận được %d", l.text, e.ID, l.id)
		}
	}
}

func TestFitCatTheoSoOCoHienThi(t *testing.T) {
	if got := displayWidth("🎯 1. WHY"); got != 9 {
		t.Fatalf("emoji phải tính 2 ô: %q có width %d", "🎯 1. WHY", got)
	}
	if got := fit("abcdef", 4); got != "abc…" {
		t.Fatalf("fit phải cắt còn 4 ô: %q", got)
	}
	if got := fit("abc", 10); got != "abc" {
		t.Fatalf("dòng ngắn không được cắt: %q", got)
	}
}

func TestBuildBodyGhepNhieuLog(t *testing.T) {
	a := sampleEntry()
	b := sampleEntry()
	b.ID = 8
	b.Timestamp = time.Date(2026, 11, 18, 11, 25, 12, 800_000_000, time.Local)
	b.EventType = models.EventUserAction
	b.Icon = "🖱️"
	b.IsExpanded = true

	lines := buildBody([]models.LogEntry{a, b})
	text := blockText(lines)
	if !strings.Contains(text, "[USER ACTION]") || !strings.Contains(text, "[OS REACTION]") {
		t.Fatalf("body phải chứa cả hai loại sự kiện:\n%s", text)
	}
	if lines[0].id != a.ID {
		t.Fatal("body phải giữ đúng thứ tự log")
	}
}
