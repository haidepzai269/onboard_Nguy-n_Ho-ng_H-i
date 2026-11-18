// Package models định nghĩa cấu trúc dữ liệu cho mỗi dòng log của GoTrace.
package models

import (
	"fmt"
	"time"
)

// Loại sự kiện được ghi nhận trong log.
const (
	EventUserAction EventType = "USER_ACTION"
	EventOSReaction EventType = "OS_REACTION"
)

// EventType phân biệt log do thao tác người dùng hay do phản ứng của Kernel.
type EventType string

// Label trả về nhãn hiển thị trên dòng log thu gọn, ví dụ "USER ACTION".
func (t EventType) Label() string {
	if t == EventUserAction {
		return "USER ACTION"
	}
	return "OS REACTION"
}

// HardwareContext giữ phần bóc tách CPU / bộ nhớ của một sự kiện.
//
// Ba trục tương ứng với 3 nhánh cuối của cây dropdown:
//   - SpaceAxis:       Trục Không Gian (Storage -> RAM)
//   - MemoryDetails:   Trục Thực Thi (Working Set, Commit Size, Threads) - nhiều dòng, ngăn bởi '\n'
//   - CPUDetails:      Trục Tương Tác (CPU Rings, context switch) - nhiều dòng, ngăn bởi '\n'
type HardwareContext struct {
	CPUDetails    string
	MemoryDetails string
	SpaceAxis     string
}

// LogEntry là một dòng log cùng toàn bộ thông tin bóc tách khi mở rộng.
type LogEntry struct {
	ID         int64
	Timestamp  time.Time
	EventType  EventType
	Icon       string
	Summary    string
	IsExpanded bool

	// Thông tin chỉ hiển thị khi expand.
	Why       string
	Subsystem string   // nhiều dòng ngăn bởi '\n' sẽ được render thành nhiều nhánh con
	APICalls  []string // mỗi phần tử: dòng đầu là lời gọi, các dòng sau nối bằng '\n' là kết quả trả về

	HardwareContext HardwareContext
}

// TimestampLabel định dạng thời gian tới mili-giây: 11:25:05.120
func (e LogEntry) TimestampLabel() string {
	return e.Timestamp.Format("15:04:05.000")
}

// CollapsedLine dựng đúng một dòng thu gọn cho log này.
func (e LogEntry) CollapsedLine() string {
	arrow := "▶"
	if e.IsExpanded {
		arrow = "▼"
	}
	icon := e.Icon
	if icon == "" {
		icon = "•"
	}
	return fmt.Sprintf("%s [%s] %s [%s] %s", arrow, e.TimestampLabel(), icon, e.EventType.Label(), e.Summary)
}

// FormatMB đổi số byte thật đọc từ Win32 API sang chuỗi MB.
func FormatMB(bytes uintptr) string {
	return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
}
