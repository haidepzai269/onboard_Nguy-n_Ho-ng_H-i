// Package store giữ danh sách log dùng chung giữa các goroutine monitor và UI.
package store

import (
	"sync"

	"gotrace/pkg/models"
)

// Store là hàng đợi log có giới hạn. Mọi truy cập đều qua mutex vì monitor
// ghi từ goroutine riêng còn UI đọc và đổi trạng thái mở/đóng.
type Store struct {
	mu       sync.RWMutex
	entries  []models.LogEntry
	nextID   int64
	capacity int
	updates  chan struct{}
}

// New tạo store giữ tối đa capacity dòng log gần nhất.
func New(capacity int) *Store {
	if capacity <= 0 {
		capacity = 500
	}
	return &Store{capacity: capacity, updates: make(chan struct{}, 1)}
}

// Add gán ID tăng dần, thêm log mới và báo cho UI biết có thay đổi.
func (s *Store) Add(entry models.LogEntry) {
	s.mu.Lock()
	s.nextID++
	entry.ID = s.nextID
	s.entries = append(s.entries, entry)
	if len(s.entries) > s.capacity {
		s.entries = append([]models.LogEntry(nil), s.entries[len(s.entries)-s.capacity:]...)
	}
	s.mu.Unlock()
	s.notify()
}

// Entries trả về bản sao danh sách log tại thời điểm gọi.
func (s *Store) Entries() []models.LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.LogEntry, len(s.entries))
	copy(out, s.entries)
	return out
}

// Toggle đảo trạng thái mở/đóng của log theo ID và trả về trạng thái mới.
func (s *Store) Toggle(id int64) (expanded bool, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].ID == id {
			s.entries[i].IsExpanded = !s.entries[i].IsExpanded
			return s.entries[i].IsExpanded, true
		}
	}
	return false, false
}

// Clear xoá toàn bộ log đang giữ.
func (s *Store) Clear() {
	s.mu.Lock()
	s.entries = nil
	s.mu.Unlock()
	s.notify()
}

// Len trả về số dòng log hiện có.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// Updates là kênh báo hiệu có log mới (buffer 1, gửi không chặn).
func (s *Store) Updates() <-chan struct{} {
	return s.updates
}

func (s *Store) notify() {
	select {
	case s.updates <- struct{}{}:
	default:
	}
}
