# GoTrace — Real-Time Interactive Deep-Trace Tool

Công cụ dòng lệnh (TUI) chạy trên Windows, theo dõi thời gian thực **thao tác người dùng** và
**phản ứng của Kernel**, cho phép mở rộng từng dòng log để xem đầy đủ: bối cảnh kích hoạt,
subsystem/dữ liệu gốc, từng lời gọi Win32 API kèm giá trị trả về thật, và trạng thái CPU/Bộ nhớ.

Không có dữ liệu mô phỏng. Mọi HWND, PID, WorkingSetSize, đường dẫn `.exe` đều là kết quả thật
của lời gọi Win32 API.

## Yêu cầu

- Windows 10/11 (Windows Terminal hoặc PowerShell/cmd). Cần console thật: chương trình đọc phím
  bằng `ReadConsoleInputW` và bật ANSI cho stdout, nên không dùng được khi stdout bị redirect.
- Go 1.25.5 trở lên nếu tự build (không cần thư viện ngoài, chỉ dùng `syscall` của stdlib).

## Build & chạy

```bash
cd c:\onboard\gotrace
go build -o gotrace.exe .
.\gotrace.exe
```

## Phím tắt

| Phím | Tác dụng |
| :--- | :--- |
| `↑` / `↓` | Chọn dòng log phía trên / phía dưới |
| `Enter` hoặc `Space` | Mở / đóng menu chi tiết (dropdown tree) của dòng đang chọn |
| `PgUp` / `PgDn` | Cuộn tay một trang nội dung (dùng khi cây chi tiết cao hơn cả khung nhìn) |
| `P` | Tạm dừng / chạy tiếp polling (log đứng yên để đọc, không cần Ctrl+C) |
| `C` | Xoá toàn bộ log đang hiển thị và làm sạch màn hình |
| `Q` | Thoát an toàn (Ctrl+C cũng thoát và khôi phục terminal) |

Khi đang ở dòng mới nhất, chế độ **tự động cuộn** bật và màn hình bám log mới. Dùng `↑` để rời
khỏi đáy; cuộn xuống dòng cuối sẽ bật lại tự động cuộn. Footer luôn hiển thị log đang chọn và
chu kỳ polling.

Mở một log gần đáy sẽ tự cuộn để thấy trọn khối chi tiết (tiêu đề + cả 4 nhánh cây) thay vì chỉ
thấy dòng tiêu đề. Nếu khối chi tiết cao hơn cả khung nhìn, khung nhìn neo ở dòng tiêu đề để đọc
từ đầu; dùng `PgDn` để cuộn xuống phần còn lại (`PgUp` quay lại). Khi cuộn tay, tự động cuộn tạm
tắt; chọn dòng khác hoặc quay về dòng mới nhất để bật lại.

## Vì sao polling, và vì sao có bộ lọc

Windows không có kênh "push" sự kiện process cho tiến trình user thường (không có tương đương
SSE của web). Cơ chế đẩy thật là **ETW realtime session** (`Microsoft-Windows-Kernel-Process`
với `EVENT_TRACE_REAL_TIME_MODE`), nhưng nó cần quyền **Administrator**, phải tự quản lý session
và parse event qua `Tdh.dll`, và event có thể rơi khi tải cao. Vì vậy GoTrace chủ động **polling**
bằng `CreateToolhelp32Snapshot` (250 ms cho process, 150 ms cho cửa sổ tiền cảnh) — đúng thứ mà
bài nghiệm thu cần: bạn thấy tận mắt lời gọi Syscall → Kernel và giá trị trả về thật.

Đổi lại, polling nhìn thấy cả những tiến trình sống rất ngắn do công cụ khác liên tục spawn
(`tasklist.exe`, `conhost.exe` của trình giám sát/agent). GoTrace **lọc** chúng theo tên, đếm lại
và in một dòng tổng kết mỗi 5 giây, thay vì để log bị nhiễu:

```bash
.\gotrace.exe                                  # lọc mặc định: tasklist.exe, conhost.exe
.\gotrace.exe -ignore "tasklist.exe,conhost.exe,git.exe"
.\gotrace.exe -ignore ""                       # tắt lọc, xem toàn bộ sự kiện thật
```

Footer hiển thị `Đã lọc: N`; số sự kiện bị ẩn không bị mất dấu vì dòng tổng kết ghi rõ tên và số lần.

## Kiến trúc

```
main.go                     nối console + store + 2 monitor + UI, bắt Ctrl+C
pkg/win32/                  tầng duy nhất nói chuyện với DLL (không phụ thuộc package khác)
  win32.go                  NewLazyDLL: user32, kernel32, psapi + CloseHandle
  types.go                  struct C ↔ Go (PROCESSENTRY32W, PROCESS_MEMORY_COUNTERS,
                            THREADENTRY32, INPUT_RECORD...) với offset/kích thước thật
  window.go                 GetForegroundWindow, GetWindowTextW, GetClassNameW,
                            GetWindowThreadProcessId
  process.go                CreateToolhelp32Snapshot, Process32FirstW/NextW,
                            Thread32First/Next, OpenProcess,
                            QueryFullProcessImageNameW, GetProcessMemoryInfo
  console.go                raw console input (ReadConsoleInputW), bật VT/ANSI, kích thước
pkg/models/                 LogEntry + HardwareContext (dữ liệu của một dòng log)
pkg/store/                  danh sách log có mutex, giới hạn 1000 dòng, kênh báo thay đổi
pkg/monitor/                UserActionTracker (150ms) và OSReactionWatcher (250ms)
pkg/ui/                     composeFrame (ghép cả khung hình) + vòng lặp phím
```

Luồng dữ liệu: monitor gọi Win32 → tạo `LogEntry` → `store.Add` → UI nhận tín hiệu → ghép lại
toàn bộ khung hình vào **một `WriteString` duy nhất** (không nhấp nháy) → ghi ra stdout.

## Win32 API thật được gọi cho từng loại sự kiện

**User Action** (đổi cửa sổ tiền cảnh): `GetForegroundWindow`, `GetWindowTextW`,
`GetWindowThreadProcessId`, `GetClassNameW`, sau đó mở tiến trình sở hữu cửa sổ bằng
`OpenProcess` → `QueryFullProcessImageNameW` → `GetProcessMemoryInfo` →
`CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD)` + `Thread32Next`.

**OS Reaction** (process sinh ra / biến mất): `CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS)` +
`Process32FirstW` / `Process32NextW`, so sánh với ảnh chụp trước để tìm PID mới/biến mất. Với PID
mới: `OpenProcess`, `QueryFullProcessImageNameW`, `GetProcessMemoryInfo`, đếm luồng thật, PPID
lấy từ `PROCESSENTRY32W`. Với PID biến mất: ghi nhận thời điểm PID rời khỏi Process Active Table
(không bịa số byte đã giải phóng, vì PID đã bị thu hồi).

## Cấu trúc một dòng log mở rộng

```
▼ [11:25:05.120] ⚙️ [OS REACTION] Kernel tạo Process mới: "chrome.exe" [PID: 19504]
  │
  ├── 🎯 1. BỐI CẢNH & NGUYÊN DO (WHY):
  ├── 🏛️ 2. NGUỒN GỐC DỮ LIỆU & SUBSYSTEM (WHERE):
  ├── ⚡ 3. HÀM WIN32 API / SYSCALL ĐÃ THỰC THI (HOW/APIs):
  └── 🧠 4. PHẢN ỨNG CỦA CPU & BỘ NHỚ LIÊN QUAN:
      ├── Trục Không Gian (Storage -> RAM)
      ├── Trục Thực Thi (Process Memory: Working Set, Commit Size)
      └── Trục Tương Tác (CPU Rings: Ring 3 ↔ Ring 0, số luồng)
```

## Kiểm thử

```bash
go test ./...        # chạy toàn bộ
go test ./pkg/win32 -v
```

Các test đáng chú ý (chạy trên Windows thật, gọi API thật):

| Test | Chứng minh |
| :--- | :--- |
| `pkg/win32/TestTakeProcessSnapshotGoiThatAPI` | Bảng tiến trình đọc từ Kernel có PID của chính tiến trình test |
| `pkg/win32/TestProbeSelfDocDuLieuThat` | `OpenProcess` + đường dẫn `.exe` thật + WorkingSet/Pagefile > 0 + số luồng > 0 |
| `pkg/win32/TestForegroundWindowTraVeHandleThat` | `GetForegroundWindow` + PID thật của cửa sổ tiền cảnh |
| `pkg/win32/TestCauTrucCoDungKichThuocC` | Offset/kích thước struct khớp C (`PROCESSENTRY32W` = 568 byte...) |
| `pkg/monitor/TestOSReactionWatcherBatDuocTienTrinhMoi` | Mở `ping.exe` thật, watcher bắt được kèm `QueryFullProcessImageNameW` + `GetProcessMemoryInfo` |
| `pkg/monitor/TestBoLocAnSuKienNhieu` | `ping.exe` bị lọc: không có dòng log riêng, nhưng bộ đếm và dòng tổng kết bộ lọc vẫn ghi nhận |
| `pkg/monitor/TestTamDungPollingRoiChayTiep` | Tạm dừng thì không ghi log; chạy tiếp không bắn loạt sự kiện của quãng đã dừng |
| `pkg/ui/TestComposeFrame*` | Khung hình đúng số dòng, có ghi chú polling/ETW, băng-rôn tạm dừng, chế độ không ANSI không lọt mã điều khiển |

## Chi phí polling (đo bằng benchmark)

| Vòng quét | Chi phí | Chu kỳ | Ước tính tải |
| :--- | ---: | ---: | :--- |
| `TakeProcessSnapshot` (OS reaction) | ~10.7 ms | 250 ms | ~4.3% một nhân, ~0.36% toàn máy 12 luồng |
| `QueryForegroundWindow` (user action) | ~10 µs | 150 ms | dưới 0.01% một nhân |

Đo bằng `go test ./pkg/win32 -run XXX -bench . -benchtime=100x` trên i5-12450H.

## Ghi chú & giới hạn đã biết

- Một số tiến trình hệ thống (chạy quyền cao hơn) sẽ từ chối `OpenProcess`; GoTrace hiển thị
  nguyên văn lỗi thay vì điền số giả.
- Emoji chỉ đẹp trên terminal có font hỗ trợ (Windows Terminal); cmd.exe cũ có thể hiện ô vuông.
- Nếu terminal không bật được `ENABLE_VIRTUAL_TERMINAL_PROCESSING`, UI tự chuyển sang chế độ
  không màu (không in mã ANSI).
- Log được giới hạn 1000 dòng gần nhất để bộ nhớ không tăng vô hạn.
- Chi phí snapshot phụ thuộc số tiến trình đang sống; máy càng nhiều process thì mỗi vòng quét càng lâu.
