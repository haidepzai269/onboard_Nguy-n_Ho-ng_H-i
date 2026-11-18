package monitor

import (
	"testing"

	"gotrace/pkg/win32"
)

func TestDiffProcessesPhatHienSinhVaDiet(t *testing.T) {
	prev := map[uint32]win32.ProcessEntry{
		100: {PID: 100, Name: "explorer.exe"},
		200: {PID: 200, Name: "old.exe"},
	}
	cur := map[uint32]win32.ProcessEntry{
		100: {PID: 100, Name: "explorer.exe"},
		300: {PID: 300, Name: "chrome.exe"},
		400: {PID: 400, Name: "notepad.exe"},
	}

	added, removed := diffProcesses(prev, cur)

	if len(added) != 2 {
		t.Fatalf("phải phát hiện 2 PID mới, nhận được %d", len(added))
	}
	if added[0].PID != 300 || added[1].PID != 400 {
		t.Fatalf("danh sách PID mới phải sắp xếp tăng dần, nhận được %+v", added)
	}
	if len(removed) != 1 || removed[0].PID != 200 {
		t.Fatalf("phải phát hiện PID 200 biến mất, nhận được %+v", removed)
	}
}

func TestDiffProcessesKhongDoiThiKhongCoSuKien(t *testing.T) {
	prev := map[uint32]win32.ProcessEntry{100: {PID: 100, Name: "a.exe"}}
	cur := map[uint32]win32.ProcessEntry{100: {PID: 100, Name: "a.exe"}}

	added, removed := diffProcesses(prev, cur)
	if len(added) != 0 || len(removed) != 0 {
		t.Fatalf("không đổi thì không được sinh log: added=%+v removed=%+v", added, removed)
	}
}
