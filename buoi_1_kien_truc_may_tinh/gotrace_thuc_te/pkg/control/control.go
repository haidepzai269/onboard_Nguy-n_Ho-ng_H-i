// Package control giữ hai công tắc dùng chung giữa UI và hai bộ theo dõi:
// tạm dừng polling và danh sách tiến trình bị lọc khỏi log.
package control

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Control an toàn cho nhiều goroutine: UI bật/tắt tạm dừng còn monitor đọc
// trạng thái đó ở mỗi vòng quét.
type Control struct {
	paused atomic.Bool

	mu         sync.Mutex
	ignore     map[string]bool
	suppressed map[string]int // số sự kiện đã ẩn theo tên tiến trình, chờ báo cáo
	pending    int            // tổng chờ báo cáo
	all        int            // tổng đã ẩn từ lúc khởi động (hiển thị ở footer)
}

// New tạo Control với danh sách tên tiến trình cần lọc (không phân biệt hoa thường).
// Danh sách rỗng nghĩa là không lọc gì.
func New(ignore []string) *Control {
	c := &Control{ignore: make(map[string]bool), suppressed: make(map[string]int)}
	for _, name := range ignore {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || c.ignore[name] {
			continue
		}
		c.ignore[name] = true
	}
	return c
}

// Paused cho biết polling có đang tạm dừng không.
func (c *Control) Paused() bool { return c.paused.Load() }

// TogglePause đảo trạng thái tạm dừng và trả về trạng thái mới.
func (c *Control) TogglePause() bool {
	for {
		cur := c.paused.Load()
		if c.paused.CompareAndSwap(cur, !cur) {
			return !cur
		}
	}
}

// Pause ngắt polling: monitor vẫn chạy vòng lặp nhưng không gọi API nào.
func (c *Control) Pause() { c.paused.Store(true) }

// Resume cho polling chạy tiếp.
func (c *Control) Resume() { c.paused.Store(false) }

// Muted cho biết tiến trình này có nằm trong danh sách lọc không, đồng thời
// đếm sự kiện bị ẩn để báo lại cho người dùng.
func (c *Control) Muted(name string) bool {
	if c == nil || len(c.ignore) == 0 {
		return false
	}
	key := strings.ToLower(name)
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ignore[key] {
		return false
	}
	c.suppressed[key]++
	c.pending++
	c.all++
	return true
}

// TakeSuppressed trả về tổng số sự kiện đã ẩn kể từ lần gọi trước kèm chi tiết
// theo tên tiến trình (nhiều nhất trước), rồi đặt lại bộ đếm chờ báo cáo.
func (c *Control) TakeSuppressed() (int, []string) {
	if c == nil {
		return 0, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending == 0 {
		return 0, nil
	}
	type kv struct {
		name string
		n    int
	}
	list := make([]kv, 0, len(c.suppressed))
	for name, n := range c.suppressed {
		list = append(list, kv{name: name, n: n})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].n != list[j].n {
			return list[i].n > list[j].n
		}
		return list[i].name < list[j].name
	})
	detail := make([]string, len(list))
	for i, it := range list {
		detail[i] = fmt.Sprintf("%s ×%d", it.name, it.n)
	}
	total := c.pending
	c.suppressed = make(map[string]int)
	c.pending = 0
	return total, detail
}

// SuppressedTotal là tổng số sự kiện đã bị ẩn từ lúc khởi động.
func (c *Control) SuppressedTotal() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.all
}

// IgnoreList trả về danh sách tên đang bị lọc, đã sắp xếp.
func (c *Control) IgnoreList() []string {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.ignore))
	for name := range c.ignore {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
