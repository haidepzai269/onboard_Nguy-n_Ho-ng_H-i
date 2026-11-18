# ĐẶC TẢ KỸ THUẬT (SPECIFICATION): DỰ ÁN GOTRACE (CLI)

## Real-Time Interactive Deep-Trace Tool (Terminal TUI với Log Mở Rộng / Dropdown Bóc Tách API & CPU)

> [!IMPORTANT]
> **QUY TẮC BẮT BUỘC THEO DÕI TIẾN ĐỘ THỰC HIỆN:**
>
> - Mỗi khi hoàn thành xong 1 Task: Đánh dấu `[x]` vào Task đó.
> - Mỗi khi hoàn thành toàn bộ Task của 1 Story: Đánh dấu `[x]` vào Story đó.
> - Theo dõi và cập nhật trực tiếp tại: [STORIES_VA_TASKS_GOTRACE.md](file:///c:/onboard/buoi_1_kien_truc_may_tinh/SPEC&STORY&TASK/STORIES_VA_TASKS_GOTRACE.md)

- **Tên dự án:** `GoTrace` (Interactive CLI / TUI)
- **Vị trí tài liệu:** `c:\onboard\buoi_1_kien_truc_may_tinh\SPEC&STORY&TASK\SPEC_DU_AN_1_SYSINSPECT.md`
- **Thư mục mã nguồn:** `c:\onboard\gotrace`
- **Ngôn ngữ triển khai:** **Go (Golang 1.25.5 x64)**
- **Giao diện:** **Terminal tương tác (TUI / Interactive CLI)** — Hỗ trợ mở/đóng menu thả xuống (Expand/Collapse Dropdown Tree) cho từng dòng log bằng bàn phím (Enter / Space / Mũi tên).
- **Trạng thái:** Cập nhật yêu cầu Log Chi Tiết Sâu & Nguồn Gốc API (Pending Final Review).

---

## 1. BỐI CẢNH & TRIẾT LÝ CỐT LÕI (WHY)

Người dùng không chỉ muốn nhìn thấy một dòng log thông báo kết quả chung chung.
Mục tiêu của `GoTrace` là trở thành một **"Kính hiển vi mổ xẻ Hệ điều hành"**:

- Bất kỳ log nào xuất hiện trên màn hình đều phải trả lời được 3 câu hỏi cốt tử:
  1. **TẠI SAO (Why):** Bối cảnh kích hoạt là gì? Do đâu mà có log này?
  2. **TỪ ĐÂU (Where):** Dữ liệu này được lấy từ DLL, Subsystem hay bảng quản lý nào của Windows?
  3. **GỌI HÀM NÀO (How/API Calls):** Tool đã gọi chính xác Win32 API / Syscall nào của Hệ điều hành, và CPU / Bộ nhớ dưới tầng thấp đã chuyển trạng thái ra sao?

---

## 2. GIAO DIỆN TERMINAL TƯƠNG TÁC (INTERACTIVE TUI / LOG DROPDOWN)

### 2.1. Cơ chế hiển thị 2 trạng thái:

1. **Trạng thái Thu gọn (Collapsed - Dạng danh sách thời gian thực):**
   - Màn hình terminal hiển thị dòng thời gian các sự kiện ngắn gọn, rõ ràng.
   - Mỗi dòng có icon chỉ dẫn trạng thái: `▶ [Nhấn Enter để mở bung chi tiết]`.
2. **Trạng thái Mở rộng (Expanded / Dropdown Tree - Bóc tách toàn diện):**
   - Khi người dùng chọn một log và nhấn `Enter` (hoặc `Phím cách`), ngay bên dưới dòng log sẽ bung ra menu phân cấp chi tiết (dạng cây Tree-view).

### 2.2. Minh họa giao diện thực tế trên Terminal:

```text
========================================================================================
  GOTRACE: HỆ THỐNG TRUY VẾT HỆ ĐIỀU HÀNH & THAO TÁC NGƯỜI DÙNG THEO THỜI GIAN THỰC
  Phím tắt: [↑/↓] Chọn log | [Enter/Space] Mở/Đóng menu chi tiết | [C] Xóa màn hình | [Q] Thoát
========================================================================================

▶ [11:25:01.050] 🖱️ [USER ACTION] Chuyển tiêu điểm cửa sổ sang "Visual Studio Code" (PID: 14280)
▼ [11:25:05.120] ⚙️ [OS REACTION] Kernel tạo Process mới: "chrome.exe" [PID: 19504]
  │
  ├── 🎯 1. BỐI CẢNH & NGUYÊN DO (WHY):
  │   └── Người dùng double-click shortcut hoặc app cha gửi lệnh nạp tiến trình con vào hệ thống.
  │
  ├── 🏛️ 2. NGUỒN GỐC DỮ LIỆU & SUBSYSTEM (WHERE):
  │   ├── Windows Subsystem: Windows Kernel Process Manager (ntoskrnl.exe)
  │   └── Bảng quản lý nguồn: Process Active Table & EPROCESS Block trong RAM Kernel.
  │
  ├── ⚡ 3. HÀM WIN32 API / SYSCALL ĐÃ THỰC THI (HOW):
  │   ├── [Call 1] kernel32.dll!CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
  │   │   ↳ Trả về: Snapshot Handle 0x000000C8 chứa danh sách tiến trình toàn máy.
  │   ├── [Call 2] kernel32.dll!Process32NextW(hSnapshot, &pe32)
  │   │   ↳ Phát hiện PID mới 19504 chưa có trong bảng theo dõi của GoTrace.
  │   │   ↳ Parent Process ID (PPID): 18420 ("explorer.exe")
  │   ├── [Call 3] kernel32.dll!OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, FALSE, 19504)
  │   │   ↳ Trả về: Process Handle 0x000001A4 (quyền truy vấn thông tin tiến trình).
  │   ├── [Call 4] kernel32.dll!QueryFullProcessImageNameW(hProcess, 0, lpExeName, &dwSize)
  │   │   ↳ Đường dẫn SSD: "C:\Program Files\Google\Chrome\Application\chrome.exe"
  │   └── [Call 5] psapi.dll!GetProcessMemoryInfo(hProcess, &pmc, sizeof(pmc))
  │       ↳ Đọc trực tiếp cấu trúc PROCESS_MEMORY_COUNTERS từ Kernel.
  │
  └── 🧠 4. PHẢN ỨNG CỦA CPU & BỘ NHỚ LIÊN QUAN:
      ├── Trục Không Gian (Storage -> RAM): OS Loader phân trang từ SSD NVMe đưa vào DRAM.
      ├── Trục Thực Thi (Process Memory):
      │   ├── RAM vật lý cấp ban đầu (Working Set / RSS): 38.45 MB
      │   ├── Bộ nhớ ảo đã cam kết (Commit Size / VMS): 52.10 MB
      │   └── Số lượng luồng ban đầu (Threads): 12 threads đang được nạp vào CPU cores.
      └── Trục Tương Tác (CPU Rings):
          └── Chuyển ngữ cảnh từ User Mode (Ring 3) -> Kernel Mode (Ring 0) để khởi tạo PCB (Process Control Block).

▶ [11:25:12.800] 🖱️ [USER ACTION] Click chuột chuyển sang "Google Chrome" [PID: 19504]
▶ [11:25:20.400] ❌ [OS REACTION] Kernel thu hồi tiến trình: "notepad.exe" [PID: 8812]
```

---

## 3. CẤU TRÚC DỮ LIỆU ĐẰNG SAU MỖI DÒNG LOG

Mỗi event được lưu dưới một cấu trúc Go Struct chuẩn mực:

```go
type LogEntry struct {
    Timestamp   time.Time
    EventType   string // "USER_ACTION" | "OS_REACTION"
    Summary     string // Dòng tóm tắt thu gọn
    IsExpanded  bool   // Trạng thái mở/đóng dropdown

    // THÔNG TIN BÓC TÁCH MỞ RỘNG (KHI EXPAND):
    Why         string   // Tại sao có log này (Bối cảnh kích hoạt)
    Subsystem   string   // Lấy từ subsystem / dll nào của Windows
    APICalls    []string // Danh sách các hàm Win32 API cụ thể kèm tham số & giá trị trả về
    HardwareContext struct {
        CPUDetails     string // CPU Ring, context switch, threads
        MemoryDetails  string // RAM Working Set, Virtual Memory, Storage path
    }
}
```

---

## 4. BỘ HÀM API CHÍNH XÁC MÀ GOTRACE GỌI CHO TỪNG LOẠI SỰ KIỆN

### A. Sự kiện User Action (Click / Chuyển cửa sổ):

- **Tại sao có log:** Người dùng nhấn chuột hoặc ấn phím chuyển đổi ứng dụng làm việc.
- **Hàm API thực thi:**
  1. `user32.dll!GetForegroundWindow()`: Lấy định danh cửa sổ `HWND` mà người dùng đang thao tác.
  2. `user32.dll!GetWindowTextW(HWND, buffer, maxCount)`: Đọc tiêu đề cửa sổ UTF-16 hiển thị trên thanh taskbar/title bar.
  3. `user32.dll!GetWindowThreadProcessId(HWND, &pid)`: Tìm ra PID và ThreadID của phần mềm sở hữu cửa sổ đó.
  4. `user32.dll!GetClassNameW(HWND, buffer, maxCount)`: Lấy loại lớp cửa sổ giao diện.

### B. Sự kiện OS Reaction (Tiến trình sinh ra / Cấp phát bộ nhớ):

- **Tại sao có log:** Hệ điều hành tạo một Process mới theo lệnh từ người dùng hoặc app khác.
- **Hàm API thực thi:**
  1. `kernel32.dll!CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)`: Lấy ảnh chụp trạng thái bảng tiến trình từ nhân Kernel.
  2. `kernel32.dll!Process32FirstW` & `Process32NextW`: Quét từng bản ghi `PROCESSENTRY32W`.
  3. `kernel32.dll!OpenProcess`: Mở handle hợp pháp để đọc bộ nhớ tiến trình.
  4. `kernel32.dll!QueryFullProcessImageNameW`: Truy vấn đường dẫn file thực thi trên ổ đĩa SSD.
  5. `psapi.dll!GetProcessMemoryInfo`: Đọc cấu trúc `PROCESS_MEMORY_COUNTERS` để lấy chính xác byte RAM vật lý và Pagefile.

### C. Sự kiện OS Reaction (Tiến trình kết thúc):

- **Tại sao có log:** Ứng dụng bị người dùng đóng hoặc crash, Kernel kích hoạt cơ chế dọn dẹp và thu hồi PID.
- **Hàm API thực thi:**
  1. Đối chiếu danh sách PID hiện tại với danh sách snapshot trước đó.
  2. Ghi nhận thời điểm PID biến mất khỏi Process Table của Kernel.

---

## 5. RÀNG BUỘC KỸ THUẬT & TRẢI NGHIỆM

1. **Điều khiển bàn phím mượt mà:**
   - Dùng phím mũi tên `↑` / `↓` để duyệt qua từng sự kiện log.
   - Nhấn `Enter` hoặc `Space` để bung dropdown / thu gọn ngay lập tức.
   - Tự động cuộn theo log mới nếu đang ở dòng cuối cùng (Auto-scroll toggle).
2. **100% Real API Call:** Tuyệt đối không hardcode mô phỏng; từng giá trị HWND, PID, WorkingSetBytes, FilePath đều là kết quả thực từ các lệnh gọi Win32 API.
3. **Hiệu năng:** Chu kỳ quét (Polling) tối ưu 150ms-250ms, đảm bảo CPU máy chủ không tăng quá 1%.

---

## 6. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [x] **AC-1:** Giao diện TUI chạy trực tiếp trên terminal PowerShell/Windows Terminal, hỗ trợ phím điều hướng `↑`/`↓` và `Enter`.
- [x] **AC-2:** Mỗi log mặc định hiển thị 1 dòng thu gọn sạch đẹp.
- [x] **AC-3:** Khi ấn `Enter` vào một dòng log, bung ra menu dropdown phân nhánh hình cây (Tree view) đầy đủ 4 phần: _Bối cảnh (Why), Nguồn gốc (Where), Chi tiết hàm API đã gọi (How/APIs), và Phản ứng CPU/Bộ nhớ_.
- [x] **AC-4:** Khi click mở Notepad hay Chrome, log bung ra chỉ rõ hàm `CreateToolhelp32Snapshot`, `GetProcessMemoryInfo` và `QueryFullProcessImageNameW`.
- [x] **AC-5:** Thoát mượt mà bằng phím `Q` hoặc `Ctrl + C`.

> **Ghi chú nghiệm thu (18/11/2026):** AC-2, AC-3, AC-4 đã xác minh tự động bằng test gọi Win32 API thật (`go test ./...`). AC-1 và AC-5 đã xác minh ở tầng logic phím (`pkg/ui/model_test.go`); cần một lượt bấm tay trên terminal thật để xác nhận đường nhận phím vật lý và Ctrl+C. Nhật ký build/test đầy đủ: [STORIES_VA_TASKS_GOTRACE.md](STORIES_VA_TASKS_GOTRACE.md).
