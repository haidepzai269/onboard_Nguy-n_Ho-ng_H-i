package ui

import (
	"testing"

	"gotrace/pkg/control"
	"gotrace/pkg/models"
	"gotrace/pkg/store"
	"gotrace/pkg/win32"
)

func newTestModel(t *testing.T, summaries ...string) (*Model, *store.Store) {
	t.Helper()
	st := store.New(100)
	for _, s := range summaries {
		st.Add(models.LogEntry{Summary: s})
	}
	return New(st, &win32.Console{}, control.New(nil)), st
}

func TestHandleKeyDieuHuongMoDongVaThoat(t *testing.T) {
	m, st := newTestModel(t, "một", "hai", "ba")
	if !m.follow {
		t.Fatal("model mới phải ở chế độ tự động cuộn")
	}

	// Enter/Space trên dòng mới nhất khi chưa chọn: chọn dòng mới nhất rồi mở.
	if quit, changed := m.handleKey(win32.Key{VirtualKey: win32.VK_RETURN}); quit || !changed {
		t.Fatalf("Enter phải mở log đang chọn: quit=%v changed=%v", quit, changed)
	}
	if !st.Entries()[2].IsExpanded {
		t.Fatal("Enter phải bật IsExpanded của log mới nhất")
	}
	m.handleKey(win32.Key{VirtualKey: win32.VK_SPACE})
	if st.Entries()[2].IsExpanded {
		t.Fatal("Space phải đóng log đang mở")
	}

	// ↑ rời đáy -> tắt tự động cuộn, chọn dòng phía trên.
	if _, changed := m.handleKey(win32.Key{VirtualKey: win32.VK_UP}); !changed {
		t.Fatal("mũi tên lên phải làm thay đổi trạng thái hiển thị")
	}
	if m.follow {
		t.Fatal("rời khỏi dòng cuối phải tắt tự động cuộn")
	}
	if id := m.currentID(st.Entries()); id != 2 {
		t.Fatalf("↑ phải chọn log thứ hai, nhận được #%d", id)
	}

	// ↓ về đáy -> bật lại tự động cuộn.
	m.handleKey(win32.Key{VirtualKey: win32.VK_DOWN})
	if !m.follow {
		t.Fatal("quay lại dòng cuối phải bật tự động cuộn")
	}
	m.handleKey(win32.Key{VirtualKey: win32.VK_DOWN}) // đã ở đáy, không vượt biên

	if quit, _ := m.handleKey(win32.Key{Rune: 'q'}); !quit {
		t.Fatal("phím Q phải thoát chương trình")
	}
	if quit, _ := m.handleKey(win32.Key{Rune: 'Q'}); !quit {
		t.Fatal("phím Q hoa cũng phải thoát chương trình")
	}
}

func TestHandleKeyPTamDungPolling(t *testing.T) {
	m, _ := newTestModel(t, "một")
	if m.Control.Paused() {
		t.Fatal("model mới không được ở trạng thái tạm dừng")
	}
	if _, changed := m.handleKey(win32.Key{Rune: 'p'}); !changed {
		t.Fatal("phím P phải làm màn hình vẽ lại")
	}
	if !m.Control.Paused() {
		t.Fatal("phím P phải tạm ngắt polling")
	}
	if quit, _ := m.handleKey(win32.Key{Rune: 'P'}); quit {
		t.Fatal("phím P không được thoát chương trình")
	}
	if m.Control.Paused() {
		t.Fatal("phím P lần hai phải cho polling chạy tiếp")
	}
}

func TestHandleKeyPgUpPgDnCuonTay(t *testing.T) {
	m, _ := newTestModel(t, "một", "hai", "ba")
	// Chọn log thứ hai rồi mở rộng để có khối chi tiết dài.
	m.handleKey(win32.Key{VirtualKey: win32.VK_UP})
	m.handleKey(win32.Key{VirtualKey: win32.VK_RETURN})

	if _, changed := m.handleKey(win32.Key{VirtualKey: win32.VK_NEXT}); !changed {
		t.Fatal("PgDn phải làm màn hình vẽ lại")
	}
	if !m.pinned || m.follow {
		t.Fatal("PgDn phải chuyển sang chế độ cuộn tay và tắt tự động cuộn")
	}
	if m.scroll <= 0 {
		t.Fatalf("PgDn phải cuộn xuống, vị trí nhận được %d", m.scroll)
	}

	m.handleKey(win32.Key{VirtualKey: win32.VK_PRIOR})
	if m.scroll != 0 {
		t.Fatalf("PgUp phải cuộn ngược về đầu, vị trí nhận được %d", m.scroll)
	}

	// Chọn dòng khác thì bỏ neo cuộn tay.
	m.handleKey(win32.Key{VirtualKey: win32.VK_UP})
	if m.pinned {
		t.Fatal("đổi dòng đang chọn phải bỏ chế độ cuộn tay")
	}
}

func TestHandleKeyCXoaToanBoLog(t *testing.T) {
	m, st := newTestModel(t, "một", "hai")
	if _, changed := m.handleKey(win32.Key{Rune: 'c'}); !changed {
		t.Fatal("phím C phải làm thay đổi màn hình")
	}
	if st.Len() != 0 {
		t.Fatalf("phím C phải xoá toàn bộ log, còn %d dòng", st.Len())
	}
	if m.selectedID != 0 || !m.follow {
		t.Fatal("sau khi xoá phải trở về trạng thái ban đầu")
	}
}

func TestHandleKeyKhiChuaCoLogKhongPanic(t *testing.T) {
	m, _ := newTestModel(t)
	for _, k := range []win32.Key{
		{VirtualKey: win32.VK_UP},
		{VirtualKey: win32.VK_DOWN},
		{VirtualKey: win32.VK_RETURN},
		{VirtualKey: win32.VK_SPACE},
	} {
		if quit, changed := m.handleKey(k); quit || changed {
			t.Fatalf("phím %+v không được làm gì khi chưa có log", k)
		}
	}
}
