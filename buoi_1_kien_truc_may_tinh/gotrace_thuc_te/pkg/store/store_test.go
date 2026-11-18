package store

import (
	"testing"

	"gotrace/pkg/models"
)

func TestAddGanIDVaGiuThuTu(t *testing.T) {
	s := New(10)
	s.Add(models.LogEntry{Summary: "một"})
	s.Add(models.LogEntry{Summary: "hai"})

	entries := s.Entries()
	if len(entries) != 2 {
		t.Fatalf("phải có 2 log, nhận được %d", len(entries))
	}
	if entries[0].ID != 1 || entries[1].ID != 2 {
		t.Fatalf("ID phải tăng dần: %d, %d", entries[0].ID, entries[1].ID)
	}
	if entries[1].Summary != "hai" {
		t.Fatalf("log mới nhất phải ở cuối, nhận được %q", entries[1].Summary)
	}
}

func TestToggleDoiTrangThai(t *testing.T) {
	s := New(10)
	s.Add(models.LogEntry{Summary: "một"})
	if s.Entries()[0].IsExpanded {
		t.Fatal("log mới phải ở trạng thái thu gọn")
	}
	if expanded, ok := s.Toggle(1); !ok || !expanded {
		t.Fatalf("Toggle lần 1 phải mở log: expanded=%v ok=%v", expanded, ok)
	}
	if expanded, ok := s.Toggle(1); !ok || expanded {
		t.Fatalf("Toggle lần 2 phải đóng log: expanded=%v ok=%v", expanded, ok)
	}
	if _, ok := s.Toggle(999); ok {
		t.Fatal("Toggle với ID không tồn tại phải trả ok=false")
	}
}

func TestStoreCatBotLogCuNhat(t *testing.T) {
	s := New(2)
	s.Add(models.LogEntry{Summary: "một"})
	s.Add(models.LogEntry{Summary: "hai"})
	s.Add(models.LogEntry{Summary: "ba"})

	entries := s.Entries()
	if len(entries) != 2 {
		t.Fatalf("store phải giữ tối đa 2 log, nhận được %d", len(entries))
	}
	if entries[0].Summary != "hai" || entries[1].Summary != "ba" {
		t.Fatalf("phải bỏ log cũ nhất, nhận được %q, %q", entries[0].Summary, entries[1].Summary)
	}
}

func TestClearXoaHet(t *testing.T) {
	s := New(10)
	s.Add(models.LogEntry{Summary: "một"})
	s.Clear()
	if s.Len() != 0 {
		t.Fatalf("sau Clear phải rỗng, nhận được %d", s.Len())
	}
}
