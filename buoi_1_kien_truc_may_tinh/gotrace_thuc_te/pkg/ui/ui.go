// Package ui vẽ giao diện terminal tương tác của GoTrace.
package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gotrace/pkg/control"
	"gotrace/pkg/models"
	"gotrace/pkg/store"
	"gotrace/pkg/win32"
)

// Model giữ trạng thái hiển thị (dòng đang chọn, chế độ tự cuộn, vị trí cuộn).
type Model struct {
	Store   *store.Store
	Console *win32.Console
	// Control giữ trạng thái tạm dừng polling và bộ đếm sự kiện bị lọc.
	Control *control.Control

	PollUser   time.Duration
	PollKernel time.Duration

	selectedID int64
	follow     bool
	scroll     int
	// pinned bật khi người dùng tự cuộn bằng PgUp/PgDn: giữ nguyên vị trí
	// cuộn đó thay vì tự kéo khối log đang chọn vào khung nhìn.
	pinned bool
}

// New tạo model ở trạng thái tự động bám log mới nhất.
func New(st *store.Store, c *win32.Console, ctrl *control.Control) *Model {
	if ctrl == nil {
		ctrl = control.New(nil)
	}
	return &Model{Store: st, Console: c, Control: ctrl, follow: true}
}

// Run vẽ khung hình đầu tiên rồi vào vòng lặp sự kiện cho tới khi người dùng
// nhấn Q hoặc context bị huỷ (Ctrl+C).
func (m *Model) Run(ctx context.Context) error {
	keys := make(chan win32.Key)
	errs := make(chan error, 1)
	go func() {
		for {
			k, err := m.Console.ReadKey()
			if err != nil {
				errs <- err
				return
			}
			keys <- k
		}
	}()

	if m.Console.ANSI {
		fmt.Print("\x1b[2J" + ansiHome + ansiHideCur)
	}
	defer func() {
		if m.Console.ANSI {
			fmt.Print(ansiReset + ansiShowCur + "\r\n")
		}
	}()

	m.render()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.Store.Updates():
			m.render()
		case k := <-keys:
			quit, changed := m.handleKey(k)
			if quit {
				return nil
			}
			if changed {
				m.render()
			}
		case err := <-errs:
			return err
		}
	}
}

func (m *Model) handleKey(k win32.Key) (quit, changed bool) {
	if k.Resize {
		return false, true
	}
	entries := m.Store.Entries()
	switch k.VirtualKey {
	case win32.VK_UP:
		if len(entries) == 0 {
			return false, false
		}
		idx := m.selectedIndex(entries)
		if idx > 0 {
			idx--
		}
		m.selectedID = entries[idx].ID
		m.follow, m.pinned = idx == len(entries)-1, false
		return false, true
	case win32.VK_DOWN:
		if len(entries) == 0 {
			return false, false
		}
		idx := m.selectedIndex(entries)
		if idx < len(entries)-1 {
			idx++
		}
		m.selectedID = entries[idx].ID
		m.follow, m.pinned = idx == len(entries)-1, false
		return false, true
	case win32.VK_PRIOR, win32.VK_NEXT:
		// Cuộn tay: cần cho những log mở rộng cao hơn cả khung nhìn, khi phần
		// cây bên dưới không thể hiện hết một lượt.
		_, height := m.Console.Size()
		page := bodyHeightFor(height, m.Control.Paused())
		if k.VirtualKey == win32.VK_PRIOR {
			m.scroll -= page
		} else {
			m.scroll += page
		}
		if m.scroll < 0 {
			m.scroll = 0
		}
		m.follow, m.pinned = false, true
		return false, true
	case win32.VK_RETURN, win32.VK_SPACE:
		if id := m.currentID(entries); id != 0 {
			m.Store.Toggle(id)
			// Mở log là để đọc chi tiết: neo khung nhìn vào khối log này thay vì
			// bám đáy, nếu không phần cây vừa mở sẽ trôi khỏi màn hình.
			m.follow, m.pinned = false, false
			return false, true
		}
		return false, false
	}
	switch k.Rune {
	case 'q', 'Q':
		return true, false
	case 'p', 'P':
		m.Control.TogglePause()
		return false, true
	case 'c', 'C':
		m.Store.Clear()
		m.selectedID, m.follow, m.scroll, m.pinned = 0, true, 0, false
		return false, true
	}
	return false, false
}

// selectedIndex trả về vị trí của log đang chọn; 0 nghĩa là chưa chọn dòng nào
// nên coi như đang ở dòng mới nhất.
func (m *Model) selectedIndex(entries []models.LogEntry) int {
	if len(entries) == 0 {
		return 0
	}
	if m.selectedID == 0 {
		return len(entries) - 1
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].ID == m.selectedID {
			return i
		}
	}
	return len(entries) - 1
}

func (m *Model) currentID(entries []models.LogEntry) int64 {
	if len(entries) == 0 {
		return 0
	}
	return entries[m.selectedIndex(entries)].ID
}

// frameInput là toàn bộ dữ liệu cần để ghép một khung hình.
type frameInput struct {
	entries    []models.LogEntry
	selectedID int64
	follow     bool
	scroll     int
	pinned     bool
	width      int
	height     int
	ansi       bool
	pollUser   time.Duration
	pollKernel time.Duration
	paused     bool
	filtered   int
}

// composeFrame ghép toàn bộ màn hình thành đúng một chuỗi (một lần ghi duy nhất
// khi vẽ nên không nhấp nháy) và trả về vị trí cuộn mới. Tách khỏi console để
// kiểm thử được mà không cần terminal thật.
func composeFrame(in frameInput) (string, int) {
	header := headerLines(in.width)
	footer := footerLine(in.width, len(in.entries), in.selectedID, in.follow, in.pollUser, in.pollKernel, in.paused, in.filtered)
	bodyHeight := bodyHeightFor(in.height, in.paused)

	body := buildBody(in.entries)
	if len(body) == 0 {
		body = emptyState()
	}

	// Chưa chọn dòng nào thì coi như đang chọn dòng mới nhất.
	selID := in.selectedID
	if selID == 0 && len(in.entries) > 0 {
		selID = in.entries[len(in.entries)-1].ID
	}
	selIdx := -1
	for i, bl := range body {
		if bl.isHeader && bl.id == selID {
			selIdx = i
			break
		}
	}

	// Giữ dòng đang chọn trong khung nhìn; nếu đang tự cuộn thì luôn bám cuối.
	// Khi log đang mở, cả khối (tiêu đề + cây chi tiết) phải nằm trong khung
	// nhìn, nếu không phần cây ở gần đáy sẽ bị footer che mất.
	scroll := in.scroll
	switch {
	case in.follow:
		scroll = len(body) - bodyHeight
	case in.pinned:
		// Người dùng đang tự cuộn: tôn trọng vị trí, chỉ kẹp biên bên dưới.
	case selIdx >= 0:
		if selIdx < scroll {
			scroll = selIdx
		}
		if end := blockEnd(body, selIdx); end >= scroll+bodyHeight {
			scroll = end - bodyHeight + 1
			if scroll > selIdx {
				// Khối cao hơn cả khung nhìn: giữ dòng tiêu đề ở trên cùng để
				// người dùng đọc từ đầu, phần còn lại cuộn bằng PgUp/PgDn.
				scroll = selIdx
			}
		}
	}
	if maxScroll := len(body) - bodyHeight; scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	var b strings.Builder
	if in.ansi {
		b.WriteString(ansiHome)
		b.WriteString(ansiClearEnd)
	}
	for _, line := range header {
		writeLine(&b, line, ansiDim, false, in.width, in.ansi)
	}
	if in.paused {
		writeLine(&b, pauseBanner(), ansiYellow, false, in.width, in.ansi)
	}
	for i := 0; i < bodyHeight; i++ {
		idx := scroll + i
		if idx >= len(body) {
			writeLine(&b, "", "", false, in.width, in.ansi)
			continue
		}
		bl := body[idx]
		writeLine(&b, bl.text, bl.style, bl.isHeader && bl.id == selID, in.width, in.ansi)
	}
	writeLine(&b, footer, ansiDim, false, in.width, in.ansi)
	return b.String(), scroll
}

func (m *Model) render() {
	width, height := m.Console.Size()
	frame, scroll := composeFrame(frameInput{
		entries:    m.Store.Entries(),
		selectedID: m.selectedID,
		follow:     m.follow,
		scroll:     m.scroll,
		pinned:     m.pinned,
		width:      width,
		height:     height,
		ansi:       m.Console.ANSI,
		pollUser:   m.PollUser,
		pollKernel: m.PollKernel,
		paused:     m.Control.Paused(),
		filtered:   m.Control.SuppressedTotal(),
	})
	m.scroll = scroll
	os.Stdout.WriteString(frame)
}

// headerLineCount là số dòng cố định của headerLines().
const headerLineCount = 7

// bodyHeightFor trả về số dòng nội dung của khung hình (đã trừ header, footer
// và băng-rôn tạm dừng).
func bodyHeightFor(height int, paused bool) int {
	h := height - headerLineCount - 1 // 1 dòng footer
	if paused {
		h-- // dòng băng-rôn "đang tạm dừng"
	}
	if h < 3 {
		h = 3
	}
	return h
}

// blockEnd trả về chỉ số dòng cuối cùng của khối log bắt đầu tại start (mọi
// dòng con của một log đều mang cùng ID).
func blockEnd(body []bodyLine, start int) int {
	end := start
	id := body[start].id
	for i := start + 1; i < len(body); i++ {
		if body[i].id != id {
			break
		}
		end = i
	}
	return end
}

func writeLine(b *strings.Builder, text, style string, selected bool, width int, ansi bool) {
	text = fit(text, width-1)
	if ansi {
		if style != "" {
			text = style + text + ansiReset
		}
		if selected {
			text = ansiInverse + text + ansiReset
		}
		b.WriteString(text)
		b.WriteString(ansiClearEOL)
	} else {
		b.WriteString(padTo(text, width))
	}
	b.WriteString("\r\n")
}

func padTo(s string, width int) string {
	w := displayWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// headerLines gồm tiêu đề, phím tắt và ghi chú giải thích vì sao GoTrace dùng
// polling + bộ lọc thay vì kênh sự kiện đẩy của Windows (ETW).
func headerLines(width int) []string {
	bar := strings.Repeat("=", maxInt(1, minInt(width-1, 88)))
	title := "  GOTRACE: HỆ THỐNG TRUY VẾT HỆ ĐIỀU HÀNH & THAO TÁC NGƯỜI DÙNG THEO THỜI GIAN THỰC"
	keys := "  [↑/↓] Chọn log | [Enter] Mở | [PgUp/PgDn] Cuộn | [P] Tạm dừng | [C] Xóa | [Q] Thoát"
	note := []string{
		"  ⓘ Vì sao POLLING: Windows không có kênh đẩy sự kiện process cho user thường;",
		"    ETW realtime cần Administrator + tự parse event, desktop lại không có SSE.",
		"    Polling 250ms process / 150ms cửa sổ; sự kiện nhiễu (tasklist, conhost) bị LỌC.",
	}
	out := make([]string, 0, len(note)+4)
	out = append(out, bar, title, keys)
	out = append(out, note...)
	return append(out, bar)
}

// pauseBanner hiển thị khi người dùng tạm ngắt polling để đọc log.
func pauseBanner() string {
	return "  ⏸  TẠM DỪNG POLLING — log đứng yên để đọc; nhấn P để chạy tiếp."
}

func footerLine(width, total int, selectedID int64, follow bool, userPoll, kernelPoll time.Duration, paused bool, filtered int) string {
	if userPoll <= 0 {
		userPoll = 150 * time.Millisecond
	}
	if kernelPoll <= 0 {
		kernelPoll = 250 * time.Millisecond
	}
	auto := "TẮT"
	if follow {
		auto = "BẬT"
	}
	sel := "mới nhất"
	if selectedID != 0 {
		sel = "#" + itoa64(selectedID)
	}
	poll := fmt.Sprintf("%s user / %s kernel", userPoll, kernelPoll)
	hint := "[P] Tạm dừng"
	if paused {
		poll = "TẠM DỪNG"
		hint = "[P] Chạy tiếp"
	}
	footer := fmt.Sprintf(" Log: %d | Đang chọn: %s | Tự động cuộn: %s | Polling: %s | Đã lọc: %d | %s",
		total, sel, auto, poll, filtered, hint)
	return fit(footer, width)
}

func emptyState() []bodyLine {
	return []bodyLine{
		{text: ""},
		{text: "  (chưa có sự kiện nào) Hãy Alt+Tab sang cửa sổ khác hoặc mở Notepad / Chrome để GoTrace bắt sự kiện đầu tiên.", style: ansiDim},
		{text: ""},
		{text: "  GoTrace đang quét Process Active Table của Kernel và cửa sổ tiền cảnh bằng Win32 API thật.", style: ansiDim},
	}
}

func itoa64(v int64) string {
	return fmt.Sprintf("%d", v)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
