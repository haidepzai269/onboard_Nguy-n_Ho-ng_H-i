package ui

import (
	"strings"

	"gotrace/pkg/models"
)

// Mã ANSI dùng cho khung hình.
const (
	ansiReset    = "\x1b[0m"
	ansiBold     = "\x1b[1m"
	ansiDim      = "\x1b[2m"
	ansiCyan     = "\x1b[1;36m"
	ansiYellow   = "\x1b[1;33m"
	ansiGreen    = "\x1b[1;32m"
	ansiRed      = "\x1b[1;31m"
	ansiInverse  = "\x1b[7m"
	ansiHideCur  = "\x1b[?25l"
	ansiShowCur  = "\x1b[?25h"
	ansiHome     = "\x1b[H"
	ansiClearEOL = "\x1b[K"
	ansiClearEnd = "\x1b[0J"
)

// bodyLine là một dòng trong vùng nội dung, gắn với ID của log sở hữu.
type bodyLine struct {
	text     string
	id       int64
	isHeader bool
	style    string
}

// fit cắt dòng theo số cột hiển thị được (emoji và chữ Hán tính 2 ô).
func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if displayWidth(s) <= width {
		return s
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		w := runeWidth(r)
		if used+w > width-1 {
			break
		}
		b.WriteRune(r)
		used += w
	}
	b.WriteString("…")
	return b.String()
}

func displayWidth(s string) int {
	total := 0
	for _, r := range s {
		total += runeWidth(r)
	}
	return total
}

func runeWidth(r rune) int {
	switch {
	case r == 0xFE0F || r == 0x200D: // variation selector / zero-width joiner
		return 0
	case r >= 0x1F300 && r <= 0x1FAFF: // emoji
		return 2
	case r >= 0x2600 && r <= 0x27BF: // ký hiệu emoji (⚙ 🎯 …)
		return 2
	case r >= 0x2E80 && r <= 0xA4CF: // CJK
		return 2
	case r >= 0x1100 && r <= 0x115F:
		return 2
	}
	return 1
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}

// entryBlock dựng các dòng của một log, gồm dòng tiêu đề và (nếu mở) cây 4 nhánh.
func entryBlock(e models.LogEntry) []bodyLine {
	header := bodyLine{text: e.CollapsedLine(), id: e.ID, isHeader: true, style: headerStyle(e)}
	if !e.IsExpanded {
		return []bodyLine{header}
	}

	out := []bodyLine{header, {text: "  │", id: e.ID}}
	add := func(text, style string) {
		out = append(out, bodyLine{text: text, id: e.ID, style: style})
	}

	// 1. WHY
	add("  ├── 🎯 1. BỐI CẢNH & NGUYÊN DO (WHY):", ansiCyan)
	out = append(out, childLines(e.ID, "  │   ", splitLines(e.Why), "")...)
	add("  │", "")

	// 2. WHERE
	add("  ├── 🏛️ 2. NGUỒN GỐC DỮ LIỆU & SUBSYSTEM (WHERE):", ansiCyan)
	out = append(out, childLines(e.ID, "  │   ", splitLines(e.Subsystem), "")...)
	add("  │", "")

	// 3. HOW / APIs
	add("  ├── ⚡ 3. HÀM WIN32 API / SYSCALL ĐÃ THỰC THI (HOW/APIs):", ansiCyan)
	out = append(out, childLines(e.ID, "  │   ", e.APICalls, ansiDim)...)
	add("  │", "")

	// 4. CPU & MEMORY
	out = append(out, hardwareLines(e)...)
	return out
}

func headerStyle(e models.LogEntry) string {
	if e.EventType == models.EventUserAction {
		return ansiGreen
	}
	return ansiYellow
}

// childLines render danh sách mục con dưới một nhánh cây.
// Mỗi mục có thể nhiều dòng (ngăn bởi '\n'); các dòng sau là kết quả trả về.
func childLines(id int64, stem string, items []string, style string) []bodyLine {
	if len(items) == 0 {
		items = []string{"(không có dữ liệu)"}
	}
	out := make([]bodyLine, 0, len(items))
	for i, item := range items {
		branch, pad := "├──", "│   "
		if i == len(items)-1 {
			branch, pad = "└──", "    "
		}
		parts := splitLines(item)
		out = append(out, bodyLine{text: stem + branch + " " + parts[0], id: id, style: style})
		for _, cont := range parts[1:] {
			out = append(out, bodyLine{text: stem + pad + cont, id: id, style: style})
		}
	}
	return out
}

func hardwareLines(e models.LogEntry) []bodyLine {
	hw := e.HardwareContext
	out := []bodyLine{{text: "  └── 🧠 4. PHẢN ỨNG CỦA CPU & BỘ NHỚ LIÊN QUAN:", id: e.ID, style: ansiCyan}}

	space := strings.TrimSpace(hw.SpaceAxis)
	if space == "" {
		space = "(không áp dụng cho sự kiện này)"
	}
	out = append(out, bodyLine{text: "      ├── Trục Không Gian (Storage -> RAM): " + space, id: e.ID})

	out = append(out, bodyLine{text: "      ├── Trục Thực Thi (Process Memory):", id: e.ID})
	mem := splitLines(hw.MemoryDetails)
	for i, line := range mem {
		branch := "├──"
		if i == len(mem)-1 {
			branch = "└──"
		}
		out = append(out, bodyLine{text: "      │   " + branch + " " + line, id: e.ID})
	}

	out = append(out, bodyLine{text: "      └── Trục Tương Tác (CPU Rings):", id: e.ID})
	cpu := splitLines(hw.CPUDetails)
	for i, line := range cpu {
		branch := "├──"
		if i == len(cpu)-1 {
			branch = "└──"
		}
		out = append(out, bodyLine{text: "          " + branch + " " + line, id: e.ID})
	}
	return out
}

// buildBody ghép toàn bộ log thành danh sách dòng nội dung.
func buildBody(entries []models.LogEntry) []bodyLine {
	var out []bodyLine
	for _, e := range entries {
		out = append(out, entryBlock(e)...)
	}
	return out
}
