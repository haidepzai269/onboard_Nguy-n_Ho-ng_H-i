package ui

import (
	"strings"
	"testing"
	"time"

	"gotrace/pkg/models"
)

func TestComposeFrameDuDauDuoi(t *testing.T) {
	entries := []models.LogEntry{sampleEntry(), sampleEntry()}
	entries[1].ID = 8
	entries[1].Timestamp = time.Date(2026, 11, 18, 11, 25, 12, 800_000_000, time.Local)

	frame, scroll := composeFrame(frameInput{
		entries:    entries,
		selectedID: 8,
		follow:     true,
		width:      100,
		height:     24,
		ansi:       true,
		pollUser:   150 * time.Millisecond,
		pollKernel: 250 * time.Millisecond,
	})

	if !strings.Contains(frame, "GOTRACE: HỆ THỐNG TRUY VẾT") {
		t.Fatal("khung hình thiếu dòng tiêu đề")
	}
	if !strings.Contains(frame, "[↑/↓] Chọn log") || !strings.Contains(frame, "[P] Tạm dừng") {
		t.Fatal("khung hình thiếu dòng phím tắt")
	}
	if !strings.Contains(frame, "Log: 2") || !strings.Contains(frame, "Tự động cuộn: BẬT") {
		t.Fatal("khung hình thiếu footer trạng thái")
	}
	if !strings.Contains(frame, "\x1b[7m") {
		t.Fatal("dòng đang chọn phải được tô sáng (inverse video)")
	}
	// 4 dòng header + 19 dòng nội dung + 1 dòng footer = 24 dòng.
	if got := strings.Count(frame, "\r\n"); got != 24 {
		t.Fatalf("khung hình phải đúng 24 dòng, nhận được %d", got)
	}
	if scroll != 0 {
		t.Fatalf("với 2 log ngắn, vị trí cuộn phải là 0, nhận được %d", scroll)
	}
}

func TestComposeFrameCoGhiChuPollingVaETW(t *testing.T) {
	frame, _ := composeFrame(frameInput{width: 100, height: 22, ansi: true, follow: true})
	for _, want := range []string{
		"Vì sao POLLING",
		"ETW realtime cần Administrator",
		"sự kiện nhiễu (tasklist, conhost) bị LỌC",
		"[P] Tạm dừng",
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("khung hình thiếu ghi chú %q", want)
		}
	}
}

func TestHeaderLineCountKhopVoiHeaderLines(t *testing.T) {
	if got := len(headerLines(100)); got != headerLineCount {
		t.Fatalf("headerLineCount = %d nhưng headerLines trả về %d dòng", headerLineCount, got)
	}
}

func TestGhiChuKhongBiCatOManHinh88Cot(t *testing.T) {
	for _, line := range headerLines(88) {
		if w := displayWidth(line); w > 87 {
			t.Fatalf("dòng header dài %d cột, sẽ bị cắt ở màn hình 88 cột: %q", w, line)
		}
	}
}

func TestComposeFrameTamDungCoBangRon(t *testing.T) {
	frame, _ := composeFrame(frameInput{
		entries:  []models.LogEntry{sampleEntry()},
		width:    100,
		height:   24,
		ansi:     true,
		follow:   true,
		paused:   true,
		filtered: 12,
	})
	if !strings.Contains(frame, "TẠM DỪNG POLLING") {
		t.Fatal("khi tạm dừng phải có băng-rôn thông báo")
	}
	if !strings.Contains(frame, "Polling: TẠM DỪNG") || !strings.Contains(frame, "[P] Chạy tiếp") {
		t.Fatal("footer phải cho biết polling đang tạm dừng và cách chạy tiếp")
	}
	if !strings.Contains(frame, "Đã lọc: 12") {
		t.Fatal("footer phải hiển thị số sự kiện đã bị lọc")
	}
	if got := strings.Count(frame, "\r\n"); got != 24 {
		t.Fatalf("khung hình phải đúng 24 dòng kể cả băng-rôn, nhận được %d", got)
	}
}

func TestComposeFrameKhongANSIKhongCoMaDieuKhien(t *testing.T) {
	frame, _ := composeFrame(frameInput{
		entries: []models.LogEntry{sampleEntry()},
		width:   100,
		height:  20,
		ansi:    false,
	})
	if strings.Contains(frame, "\x1b[") {
		t.Fatal("khi terminal không hỗ trợ ANSI, khung hình không được chứa mã điều khiển")
	}
}

func TestComposeFrameTuDongCuonBamLogMoiNhat(t *testing.T) {
	var entries []models.LogEntry
	for i := 0; i < 40; i++ {
		e := sampleEntry()
		e.ID = int64(i + 1)
		e.Summary = "sự kiện số " + itoa64(int64(i+1))
		entries = append(entries, e)
	}

	frame, scroll := composeFrame(frameInput{
		entries:    entries,
		selectedID: 0,
		follow:     true,
		width:      100,
		height:     24,
		ansi:       true,
	})

	if scroll <= 0 {
		t.Fatalf("với 40 log trong màn hình 24 dòng, vị trí cuộn phải > 0, nhận được %d", scroll)
	}
	if !strings.Contains(frame, "sự kiện số 40") {
		t.Fatal("chế độ tự động cuộn phải hiển thị log mới nhất")
	}
	if strings.Contains(frame, "sự kiện số 1\n") {
		t.Fatal("log cũ nhất phải nằm ngoài khung nhìn")
	}
}

// Log nằm gần đáy khi mở rộng phải được cuộn để thấy trọn cây chi tiết, không
// chỉ dòng tiêu đề.
func TestMoLogGanDayVanThayCaCay(t *testing.T) {
	var entries []models.LogEntry
	for i := 0; i < 10; i++ {
		e := sampleEntry()
		e.ID = int64(i + 1)
		e.Summary = "sự kiện số " + itoa64(int64(i+1))
		entries = append(entries, e)
	}
	entries[7].IsExpanded = true

	frame, scroll := composeFrame(frameInput{
		entries:    entries,
		selectedID: 8,
		follow:     false,
		width:      100,
		height:     32,
		ansi:       true,
	})
	if !strings.Contains(frame, "Trục Tương Tác (CPU Rings)") {
		t.Fatalf("phần cuối của cây mở rộng bị cắt khỏi khung nhìn:\n%s", frame)
	}
	if !strings.Contains(frame, "▼ [11:25:05.120]") {
		t.Fatalf("dòng tiêu đề của log đang mở vẫn phải hiện:\n%s", frame)
	}
	if scroll == 0 {
		t.Fatal("khung nhìn phải cuộn xuống để chứa khối log đang mở")
	}
}

// Khi người dùng tự cuộn bằng PgUp/PgDn, khung hình phải giữ đúng vị trí đã
// cuộn thay vì kéo ngược lại dòng đang chọn.
func TestPinnedGiuNguyenViTriCuon(t *testing.T) {
	var entries []models.LogEntry
	for i := 0; i < 20; i++ {
		e := sampleEntry()
		e.ID = int64(i + 1)
		e.IsExpanded = true
		entries = append(entries, e)
	}

	_, scroll := composeFrame(frameInput{
		entries:    entries,
		selectedID: 3,
		follow:     false,
		pinned:     true,
		scroll:     12,
		width:      100,
		height:     24,
	})
	if scroll != 12 {
		t.Fatalf("vị trí cuộn tay phải được giữ nguyên, nhận được %d", scroll)
	}
}

func TestComposeFrameRongCoHuongDan(t *testing.T) {
	frame, _ := composeFrame(frameInput{width: 100, height: 20, ansi: true, follow: true})
	if !strings.Contains(frame, "chưa có sự kiện nào") {
		t.Fatal("màn hình rỗng phải có hướng dẫn thao tác")
	}
}
