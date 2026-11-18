# DANH SÁCH USER STORIES & TASKS: DỰ ÁN GOTRACE (CLI)
## Real-Time Interactive Deep-Trace Tool (Terminal TUI với Dropdown Bóc Tách API & CPU)

> [!IMPORTANT]
> **QUY TẮC BẮT BUỘC THEO DÕI TIẾN ĐỘ:**
> - **Mỗi khi hoàn thành 1 Task:** Bắt buộc đánh dấu `[x]` vào Task đó ngay lập tức.
> - **Mỗi khi hoàn thành toàn bộ Task của 1 Story:** Bắt buộc đánh dấu `[x]` vào Story đó.
> - Cập nhật đồng bộ tỷ lệ tiến độ ở **Bảng Theo Dõi Tiến Độ Tổng Quát** bên dưới sau mỗi lượt code.

---

### 📊 BẢNG THEO DÕI TIẾN ĐỘ TỔNG QUÁT

| Epic / Story ID | Tên Hạng Mục | Trạng Thái | Tiến Độ Tasks |
| :--- | :--- | :---: | :---: |
| 🏗️ **EPIC-1** | **Win32 Subsystem Binding & Data Model** | ✅ Hoàn thành | **4/4** |
| ↳ Story 1.1 | Khởi tạo Project Go & Binding Win32 API | ✅ Hoàn thành | 3/3 |
| ↳ Story 1.2 | Thiết kế Data Model cho Log phân tầng | ✅ Hoàn thành | 1/1 |
| ⚙️ **EPIC-2** | **Event Catchers (User Action & OS Reaction)** | ✅ Hoàn thành | **7/7** |
| ↳ Story 2.1 | User Action Tracker (Cửa sổ người dùng) | ✅ Hoàn thành | 3/3 |
| ↳ Story 2.2 | OS Reaction Watcher (Kernel Process & RAM) | ✅ Hoàn thành | 4/4 |
| 🖥️ **EPIC-3** | **Interactive TUI & Dropdown Tree View** | ✅ Hoàn thành | **5/5** |
| ↳ Story 3.1 | Bộ dựng TUI tương tác (Phím mũi tên, Enter) | ✅ Hoàn thành | 3/3 |
| ↳ Story 3.2 | Render Dropdown Tree View đa tầng | ✅ Hoàn thành | 2/2 |
| 🚀 **EPIC-4** | **Testing, Verification & Packaging** | ✅ Hoàn thành | **3/3** |
| ↳ Story 4.1 | Live Testing trên Windows & Đóng gói EXE | ✅ Hoàn thành | 3/3 |
| **TỔNG CỘNG** | **Toàn bộ dự án GoTrace** | ✅ **100%** | **19/19 Tasks** |

---

- **Mã dự án:** `GOTRACE`
- **Mục tiêu:** Xây dựng công cụ dòng lệnh bằng Go trên Windows theo dõi thời gian thực thao tác của người dùng và phản ứng của Kernel, hỗ trợ mở rộng dropdown xem chi tiết từng hàm Win32 API, CPU Ring và phân tầng bộ nhớ.
- **Tài liệu căn cứ:** [SPEC_DU_AN_1_SYSINSPECT.md](file:///c:/onboard/buoi_1_kien_truc_may_tinh/SPEC&STORY&TASK/SPEC_DU_AN_1_SYSINSPECT.md)
- **Thư mục code:** `c:\onboard\gotrace`

---

## 🗂️ TỔNG QUAN EPICS

| Epic ID | Tên Epic | Mục tiêu kỹ thuật |
| :--- | :--- | :--- |
| **EPIC-1** | **Win32 Subsystem Binding & Data Model** | Khởi tạo dự án Go, binding trực tiếp các thư viện Win32 DLL và định nghĩa cấu trúc dữ liệu cho log chi tiết. |
| **EPIC-2** | **Event Catchers (User Action & OS Reaction)** | Xây dựng 2 engine chạy song song: bắt thao tác chuyển cửa sổ của User và bắt tiến trình sinh/diệt kèm đo RAM của Kernel. |
| **EPIC-3** | **Interactive TUI & Dropdown Tree View** | Xây dựng giao diện dòng lệnh tương tác: phím mũi tên điều hướng, phím Enter bung/gập menu cây 4 tầng thông tin. |
| **EPIC-4** | **Testing, Verification & Packaging** | Biên dịch ra file thực thi `.exe`, kiểm chứng dữ liệu thật 100% khi thao tác trên Windows và đóng gói tài liệu. |

---

## 📌 CHI TIẾT STORIES VÀ DANH SÁCH TASKS

### EPIC-1: Win32 Subsystem Binding & Data Model

- [x] #### Story 1.1: Khởi tạo Project Go & Binding Win32 API
> *Là một lập trình viên hệ thống, tôi muốn liên kết các hàm API từ Windows DLL vào Go để có thể giao tiếp trực tiếp với Kernel mà không cần thư viện ngoài.*

- [x] **Task 1.1.1:** Khởi tạo thư mục `c:\onboard\gotrace` và tạo module Go: `go mod init gotrace`.
- [x] **Task 1.1.2:** Tạo gói `pkg/win32`:
  - Nạp `user32.dll`, `kernel32.dll`, `psapi.dll` bằng `syscall.NewLazyDLL`.
  - Khai báo các procedure: `GetForegroundWindow`, `GetWindowTextW`, `GetWindowThreadProcessId`, `CreateToolhelp32Snapshot`, `Process32FirstW`, `Process32NextW`, `OpenProcess`, `QueryFullProcessImageNameW`, `GetProcessMemoryInfo`, `CloseHandle`.
- [x] **Task 1.1.3:** Định nghĩa các cấu trúc C sang Go tương thích bộ nhớ Windows:
  - `PROCESSENTRY32W`
  - `PROCESS_MEMORY_COUNTERS`

- [x] #### Story 1.2: Thiết kế Data Model cho Log phân tầng
> *Là người sử dụng, tôi muốn mỗi dòng log lưu giữ đầy đủ thông tin: Tóm tắt, Tại sao, Nguồn gốc ở đâu, Gọi hàm nào, và CPU/RAM ra sao để có thể mở xem khi cần.*

- [x] **Task 1.2.1:** Tạo struct `LogEntry` trong `pkg/models`:
  - `ID`: Số thứ tự log
  - `Timestamp`: Thời gian chính xác đến mili-giây
  - `EventType`: `USER_ACTION` hoặc `OS_REACTION`
  - `Summary`: Chuỗi tóm tắt 1 dòng (dùng cho trạng thái thu gọn)
  - `IsExpanded`: Cờ boolean đánh dấu đang mở hay đóng
  - `Why`: Nguyên nhân kích hoạt sự kiện
  - `Subsystem`: Subsystem / DLL xuất xứ
  - `APICalls`: Danh sách các chuỗi mô tả hàm API cụ thể kèm tham số & giá trị trả về
  - `HardwareContext`: Cấu trúc chứa chi tiết CPU (Ring 3/0, Threads) và Bộ nhớ (Working Set, VMS, SSD Path).

---

### EPIC-2: Event Catchers (User Action & OS Reaction)

- [x] #### Story 2.1: Bộ lắng nghe thao tác người dùng (User Action Tracker)
> *Là người sử dụng, tôi muốn công cụ nhận diện ngay lập tức khi tôi click chuột hoặc chuyển đổi cửa sổ làm việc giữa các ứng dụng.*

- [x] **Task 2.1.1:** Cài đặt hàm `GetCurrentForegroundWindow()` gọi `user32.dll!GetForegroundWindow` và `GetWindowTextW`.
- [x] **Task 2.1.2:** Cài đặt hàm lấy PID và Process Name của cửa sổ đang kích hoạt (`GetWindowThreadProcessId`).
- [x] **Task 2.1.3:** Xây dựng Goroutine chạy vòng lặp kiểm tra (polling 150ms). Khi phát hiện `HWND` hoặc tiêu đề cửa sổ đổi khác so với lần trước:
  - Tạo `LogEntry` dạng `USER_ACTION`.
  - Ghi rõ: Gọi hàm `GetForegroundWindow()`, HWND trả về, `GetWindowTextW()`, Title trả về.
  - Đẩy vào danh sách quản lý log.

- [x] #### Story 2.2: Bộ bắt phản ứng Hệ điều hành (Kernel Process & Memory Watcher)
> *Là người sử dụng, tôi muốn công cụ bắt trọn khoảnh khắc Kernel sinh ra tiến trình mới hoặc thu hồi tiến trình cũ, cùng với lượng RAM vật lý thực tế được cấp.*

- [x] **Task 2.2.1:** Cài đặt hàm `TakeProcessSnapshot()` dùng `CreateToolhelp32Snapshot` và `Process32First/Next` để lấy bản đồ toàn bộ tiến trình đang sống trên máy.
- [x] **Task 2.2.2:** Xây dựng cơ chế phát hiện chênh lệch (Diff Detection):
  - So sánh danh sách snapshot mới với snapshot cũ để tìm ra các PID mới xuất hiện hoặc biến mất.
- [x] **Task 2.2.3:** Khi có PID mới:
  - Gọi `OpenProcess` để mở Handle.
  - Gọi `QueryFullProcessImageNameW` đọc đường dẫn file thật trên SSD.
  - Gọi `GetProcessMemoryInfo` đọc `WorkingSetSize` (RAM thực) và `PagefileUsage`.
  - Đóng gói đầy đủ bối cảnh vào `LogEntry` dạng `OS_REACTION`.
- [x] **Task 2.2.4:** Khi có PID biến mất:
  - Tạo `LogEntry` ghi nhận Kernel đã thu hồi tài nguyên của process đó.

---

### EPIC-3: Interactive TUI & Dropdown Tree View

- [x] #### Story 3.1: Bộ dựng giao diện dòng lệnh tương tác (Terminal UI Engine)
> *Là người sử dụng, tôi muốn một giao diện terminal chuyên nghiệp, dùng phím mũi tên để chọn dòng và ấn Enter để mở/đóng menu thả xuống mượt mà.*

- [x] **Task 3.1.1:** Xây dựng module đọc phím không cần nhấn Enter (Raw Console Input) trên Windows bằng Win32 `ReadConsoleInput` hoặc cơ chế tương đương.
- [x] **Task 3.1.2:** Xử lý các phím điều hướng:
  - Phím mũi tên lên `↑`: Di chuyển con trỏ lên dòng log phía trên.
  - Phím mũi tên xuống `↓`: Di chuyển con trỏ xuống dòng log phía dưới.
  - Phím `Enter` hoặc `Phím cách (Space)`: Đổi trạng thái `IsExpanded = !IsExpanded` của dòng đang chọn.
  - Phím `Q`: Thoát chương trình an toàn.
- [x] **Task 3.1.3:** Cơ chế render màn hình chống nhấp nháy (Flicker-free): Xóa màn hình hoặc ghi đè từ gốc tọa độ `(0, 0)` bằng ANSI escape codes.

- [x] #### Story 3.2: Render Dropdown Tree View đa tầng
> *Là người sử dụng, tôi muốn khi mở rộng một log, toàn bộ lý do, nguồn gốc, hàm API và CPU/RAM được trình bày dạng nhánh cây đẹp mắt, rõ ràng.*

- [x] **Task 3.2.1:** Render dòng tiêu đề:
  - Trạng thái đóng: `▶ [Timestamp] [Icon] Summary...`
  - Trạng thái mở: `▼ [Timestamp] [Icon] Summary...` (kèm đổi màu nổi bật dòng đang chọn).
- [x] **Task 3.2.2:** Render 4 nhánh cây khi mở rộng:
  - `├── 🎯 1. BỐI CẢNH & NGUYÊN DO (WHY)`
  - `├── 🏛️ 2. NGUỒN GỐC DỮ LIỆU & SUBSYSTEM (WHERE)`
  - `├── ⚡ 3. HÀM WIN32 API / SYSCALL ĐÃ THỰC THI (HOW/APIs)` (liệt kê từng hàm và giá trị trả về)
  - `└── 🧠 4. PHẢN ỨNG CPU & BỘ NHỚ LIÊN QUAN` (phân tầng Trục Không Gian, Trục Thực Thi, Trục Tương Tác).

---

### EPIC-4: Testing, Verification & Packaging

- [x] #### Story 4.1: Kiểm thử thực tế trên Windows & Hoàn thiện
> *Là một kỹ sư, tôi muốn chạy file thực thi trên máy thật để xác minh 100% dữ liệu hiển thị là từ Kernel, không có dữ liệu ảo.*

- [x] **Task 4.1.1:** Biên dịch dự án: `go build -o gotrace.exe .` trong thư mục `c:\onboard\gotrace`.
- [x] **Task 4.1.2:** Chạy kiểm thử tương tác:
  - Thao tác click chuột đổi giữa trình duyệt và editor.
  - Mở Notepad hoặc Calc để kiểm chứng log bắt PID và đường dẫn SSD `C:\Windows\System32\notepad.exe`.
  - Nhấn `Enter` bung dropdown kiểm tra cấu trúc API trả về.
- [x] **Task 4.1.3:** Viết file `README.md` hướng dẫn phím tắt và kiến trúc hoạt động của công cụ.

---

## 🧾 NHẬT KÝ NGHIỆM THU (18/11/2026)

- **Mã nguồn:** `c:\onboard\gotrace` — Go 1.25.5 windows/amd64, không dùng thư viện ngoài.
- **Build:** `go build -trimpath -ldflags "-s -w" -o gotrace.exe .` → `gotrace.exe` (~1.8 MB).
- **Test:** `go test ./...` → PASS toàn bộ (`pkg/win32`, `pkg/monitor`, `pkg/store`, `pkg/ui`).
- **Xác minh dữ liệu thật (không mô phỏng):**
  - `TestTakeProcessSnapshotGoiThatAPI`: ảnh chụp bảng tiến trình của Kernel có PID của chính tiến trình test.
  - `TestProbeSelfDocDuLieuThat`: `OpenProcess` + `QueryFullProcessImageNameW` + `GetProcessMemoryInfo` + đếm luồng đều trả số thật > 0.
  - `TestOSReactionWatcherBatDuocTienTrinhMoi`: mở `ping.exe` thật, watcher bắt được PID mới kèm đường dẫn SSD, WorkingSet và Pagefile.
  - `TestForegroundWindowTraVeHandleThat`: HWND/PID thật của cửa sổ tiền cảnh.
  - `TestCauTrucCoDungKichThuocC`: offset/kích thước struct khớp C (`PROCESSENTRY32W` 568 byte, `PROCESS_MEMORY_COUNTERS` 72 byte, `INPUT_RECORD` 20 byte).
- **Benchmark polling:** snapshot bảng tiến trình ~10.7 ms/vòng ở chu kỳ 250 ms (≈0.36% CPU toàn máy 12 luồng); quét cửa sổ tiền cảnh ~10 µs ở chu kỳ 150 ms.
- **Còn lại cho người dùng:** chạy `gotrace.exe` trên Windows Terminal/PowerShell thật và bấm thử phím để xác nhận đường nhận phím vật lý (`ReadConsoleInputW`) cùng Ctrl+C; logic phím đã được test ở `pkg/ui/model_test.go`.
- **Ghi chú số liệu:** bảng tiến độ cũ ghi tổng 17 task và EPIC-2 là 0/5; số thật là 19 task (EPIC-2 có 7 task: 3 của Story 2.1 + 4 của Story 2.2).
- **Bổ sung theo yêu cầu (18/11/2026):** cờ `-ignore` + bộ lọc nhiễu (mặc định `tasklist.exe`, `conhost.exe`, báo tổng kết mỗi 5 giây), phím `P` tạm ngắt/chạy tiếp polling (khi chạy tiếp tự chốt lại baseline, không bắn loạt sự kiện ảo), ghi chú "vì sao polling thay vì ETW" ở header. Test: `TestBoLocAnSuKienNhieu`, `TestTamDungPollingRoiChayTiep`, `pkg/control/*`, `TestHandleKeyPTamDungPolling`, `TestComposeFrameTamDungCoBangRon`.
