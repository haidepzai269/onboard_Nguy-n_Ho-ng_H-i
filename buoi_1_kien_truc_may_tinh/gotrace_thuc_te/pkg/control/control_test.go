package control

import (
	"strings"
	"testing"
)

func TestNewChuanHoaDanhSachLoc(t *testing.T) {
	c := New([]string{" Tasklist.EXE ", "", "TASKLIST.exe", "conhost.exe"})
	got := strings.Join(c.IgnoreList(), ",")
	if got != "conhost.exe,tasklist.exe" {
		t.Fatalf("danh sách lọc phải được chuẩn hoá và khử trùng, nhận được %q", got)
	}
}

func TestMutedDemSuKienBiAn(t *testing.T) {
	c := New([]string{"tasklist.exe"})
	if !c.Muted("TASKLIST.EXE") {
		t.Fatal("tên tiến trình phải được so sánh không phân biệt hoa thường")
	}
	c.Muted("tasklist.exe")
	if c.Muted("ping.exe") {
		t.Fatal("tiến trình không nằm trong danh sách lọc không được coi là bị ẩn")
	}

	total, detail := c.TakeSuppressed()
	if total != 2 {
		t.Fatalf("phải đếm 2 sự kiện bị ẩn, nhận được %d", total)
	}
	if len(detail) != 1 || detail[0] != "tasklist.exe ×2" {
		t.Fatalf("chi tiết báo cáo sai: %v", detail)
	}
	if n, _ := c.TakeSuppressed(); n != 0 {
		t.Fatalf("bộ đếm phải được đặt lại sau khi báo cáo, nhận được %d", n)
	}
	if c.SuppressedTotal() != 2 {
		t.Fatalf("tổng tích luỹ phải giữ nguyên 2, nhận được %d", c.SuppressedTotal())
	}
}

func TestTogglePause(t *testing.T) {
	c := New(nil)
	if c.Paused() {
		t.Fatal("mặc định phải đang chạy")
	}
	if !c.TogglePause() || !c.Paused() {
		t.Fatal("TogglePause lần đầu phải bật tạm dừng")
	}
	if c.TogglePause() || c.Paused() {
		t.Fatal("TogglePause lần hai phải tắt tạm dừng")
	}
	c.Pause()
	if !c.Paused() {
		t.Fatal("Pause phải bật tạm dừng")
	}
	c.Resume()
	if c.Paused() {
		t.Fatal("Resume phải tắt tạm dừng")
	}
}
