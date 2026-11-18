package monitor

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"gotrace/pkg/control"
	"gotrace/pkg/models"
	"gotrace/pkg/store"
)

// TestUserActionTrackerGhiLogTuDuLieuThat chứng minh tracker ghi log từ lời gọi
// user32.dll thật (HWND, PID, tên cửa sổ), không có dữ liệu mô phỏng.
func TestUserActionTrackerGhiLogTuDuLieuThat(t *testing.T) {
	logs := store.New(20)
	tracker := &UserActionTracker{Store: logs, Interval: 150 * time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go tracker.Run(ctx)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && logs.Len() == 0 {
		time.Sleep(50 * time.Millisecond)
	}
	if logs.Len() == 0 {
		t.Skip("phiên hiện tại không có cửa sổ tiền cảnh (headless/lock screen)")
	}

	entry := logs.Entries()[0]
	if entry.EventType != models.EventUserAction {
		t.Fatalf("log đầu tiên phải là USER_ACTION, nhận được %s", entry.EventType)
	}
	if !strings.Contains(entry.Summary, "PID: ") {
		t.Fatalf("dòng tóm tắt phải chứa PID thật: %q", entry.Summary)
	}
	joined := strings.Join(entry.APICalls, "\n")
	for _, want := range []string{
		"user32.dll!GetForegroundWindow()",
		"user32.dll!GetWindowTextW",
		"user32.dll!GetWindowThreadProcessId",
		"user32.dll!GetClassNameW",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("log user action thiếu lời gọi %q:\n%s", want, joined)
		}
	}
	if !strings.Contains(joined, "WorkingSetSize") {
		t.Fatalf("log user action phải kèm số liệu RAM thật:\n%s", joined)
	}
	if entry.HardwareContext.MemoryDetails == "" || entry.HardwareContext.CPUDetails == "" {
		t.Fatal("log user action phải có bóc tách CPU/Bộ nhớ")
	}
}

// demLogPing đếm số log nhắc tới ping.exe.
func demLogPing(logs *store.Store) int {
	n := 0
	for _, e := range logs.Entries() {
		if strings.Contains(strings.ToLower(e.Summary), "ping") {
			n++
		}
	}
	return n
}

// TestBoLocAnSuKienNhieu chứng minh bộ lọc chặn đúng tiến trình nhiễu nhưng vẫn
// đếm và báo cáo lại số sự kiện đã ẩn.
func TestBoLocAnSuKienNhieu(t *testing.T) {
	logs := store.New(200)
	ctrl := control.New([]string{"ping.exe"})
	watcher := &OSReactionWatcher{Store: logs, Interval: 150 * time.Millisecond, Control: ctrl}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watcher.Run(ctx)
	time.Sleep(400 * time.Millisecond)

	cmd := exec.Command("ping", "-n", "6", "127.0.0.1")
	if err := cmd.Start(); err != nil {
		t.Skipf("không chạy được ping.exe: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) && ctrl.SuppressedTotal() == 0 {
		time.Sleep(100 * time.Millisecond)
	}
	if ctrl.SuppressedTotal() == 0 {
		t.Fatal("bộ lọc không ghi nhận sự kiện của ping.exe")
	}
	if n := demLogPing(logs); n != 0 {
		t.Fatalf("ping.exe nằm trong danh sách lọc nhưng vẫn có %d dòng log", n)
	}

	watcher.reportFiltered()
	var report *models.LogEntry
	for _, e := range logs.Entries() {
		if e.Icon == "🔇" {
			entry := e
			report = &entry
		}
	}
	if report == nil {
		t.Fatal("không có dòng tổng kết bộ lọc")
	}
	if !strings.Contains(report.Summary, "ping.exe") || !strings.Contains(report.Summary, "×") {
		t.Fatalf("dòng tổng kết phải nêu tên tiến trình và số lần: %q", report.Summary)
	}
}

// TestTamDungPollingRoiChayTiep kiểm tra P thật sự ngắt polling: không có log
// trong lúc tạm dừng, và khi chạy tiếp không bắn loạt sự kiện của quãng đã dừng.
func TestTamDungPollingRoiChayTiep(t *testing.T) {
	logs := store.New(200)
	ctrl := control.New([]string{"tasklist.exe", "conhost.exe"})
	ctrl.Pause()
	watcher := &OSReactionWatcher{Store: logs, Interval: 100 * time.Millisecond, Control: ctrl}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watcher.Run(ctx)
	time.Sleep(300 * time.Millisecond)

	first := exec.Command("ping", "-n", "10", "127.0.0.1")
	if err := first.Start(); err != nil {
		t.Skipf("không chạy được ping.exe: %v", err)
	}
	defer func() {
		_ = first.Process.Kill()
		_ = first.Wait()
	}()
	time.Sleep(700 * time.Millisecond)
	if n := demLogPing(logs); n != 0 {
		t.Fatalf("đang tạm dừng nhưng vẫn ghi %d dòng log", n)
	}

	ctrl.Resume()
	time.Sleep(600 * time.Millisecond) // watcher chốt lại baseline
	if n := demLogPing(logs); n != 0 {
		t.Fatalf("chạy tiếp không được bắn sự kiện của quãng đã tạm dừng, nhận được %d dòng", n)
	}

	second := exec.Command("ping", "-n", "10", "127.0.0.1")
	if err := second.Start(); err != nil {
		t.Fatalf("không chạy được ping.exe lần hai: %v", err)
	}
	defer func() {
		_ = second.Process.Kill()
		_ = second.Wait()
	}()

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) && demLogPing(logs) == 0 {
		time.Sleep(100 * time.Millisecond)
	}
	if demLogPing(logs) == 0 {
		t.Fatal("sau khi chạy tiếp, watcher phải bắt được tiến trình mới")
	}
}

// TestOSReactionWatcherBatDuocTienTrinhMoi mở một tiến trình thật rồi kiểm tra
// watcher bắt được nó kèm đường dẫn SSD và RAM thật.
func TestOSReactionWatcherBatDuocTienTrinhMoi(t *testing.T) {
	logs := store.New(200)
	watcher := &OSReactionWatcher{Store: logs, Interval: 150 * time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watcher.Run(ctx)
	time.Sleep(400 * time.Millisecond) // để watcher chốt baseline

	cmd := exec.Command("ping", "-n", "6", "127.0.0.1")
	if err := cmd.Start(); err != nil {
		t.Skipf("không chạy được ping.exe: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	var found *models.LogEntry
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) && found == nil {
		for _, e := range logs.Entries() {
			if e.EventType == models.EventOSReaction && strings.Contains(strings.ToLower(e.Summary), "ping") {
				entry := e
				found = &entry
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if found == nil {
		t.Fatal("watcher không bắt được tiến trình ping.exe vừa được tạo")
	}

	joined := strings.Join(found.APICalls, "\n")
	for _, want := range []string{
		"kernel32.dll!CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)",
		"kernel32.dll!Process32NextW",
		"kernel32.dll!QueryFullProcessImageNameW",
		"psapi.dll!GetProcessMemoryInfo",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("log OS reaction thiếu lời gọi %q:\n%s", want, joined)
		}
	}
	if !strings.Contains(joined, "WorkingSetSize") {
		t.Fatalf("log OS reaction phải kèm số liệu RAM thật:\n%s", joined)
	}
	if !strings.Contains(strings.ToLower(found.HardwareContext.SpaceAxis), "ping.exe") &&
		!strings.Contains(strings.ToLower(found.HardwareContext.SpaceAxis), ".exe") {
		t.Fatalf("phải ghi nhận đường dẫn ảnh thực thi thật: %q", found.HardwareContext.SpaceAxis)
	}
}
