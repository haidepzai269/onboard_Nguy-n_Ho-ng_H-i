package win32

import "testing"

// BenchmarkTakeProcessSnapshot đo chi phí một vòng quét bảng tiến trình, dùng để
// đối chiếu ràng buộc "polling 250ms không làm CPU tăng quá 1%".
func BenchmarkTakeProcessSnapshot(b *testing.B) {
	for i := 0; i < b.N; i++ {
		snap, err := TakeProcessSnapshot()
		if err != nil {
			b.Fatalf("snapshot thất bại: %v", err)
		}
		snap.Close()
	}
}

// BenchmarkQueryForegroundWindow đo chi phí một vòng quét cửa sổ tiền cảnh.
func BenchmarkQueryForegroundWindow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := QueryForegroundWindow(); err != nil {
			b.Skipf("không có cửa sổ tiền cảnh: %v", err)
		}
	}
}
