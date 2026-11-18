# DANH SÁCH STORIES & TASKS: DỰ ÁN FINTECH PAYMENT & RECONCILIATION ENGINE

## Quản Lý Tiến Độ Triển Khai Hệ Thống Thanh Toán & Đối Soát Chịu Tải Cao (K6 Load-Tested)

> [!IMPORTANT]
> **QUY TẮC BẮT BUỘC KHI CẬP NHẬT TIẾN ĐỘ:**
> 1. Mỗi khi hoàn thành xong 1 Task, lập tức đánh dấu `[x]` vào Task đó.
> 2. Mỗi khi hoàn thành toàn bộ Task trong một Story, lập tức đánh dấu `[x]` vào Story đó.
> 3. Mỗi khi toàn bộ Story của một EPIC xong, đánh dấu `[x]` vào EPIC ở bảng tổng quan.
> 4. Tham chiếu tài liệu đặc tả kỹ thuật chi tiết tại: [SPEC_FINTECH_PAYMENT_K6_ENGINE.md](file:///c:/onboard/buoi_2_ky_thuat_lap_trinh/SPEC&STORY&TASK/SPEC_FINTECH_PAYMENT_K6_ENGINE.md)

- **Thư mục dự án:** [c:\onboard\buoi_2_ky_thuat_lap_trinh\fintech-payment-engine](file:///c:/onboard/buoi_2_ky_thuat_lap_trinh/fintech-payment-engine)
- **Môi trường thực thi:** Node.js **v22.15.0+** — bắt buộc, vì `node:sqlite` chỉ tồn tại từ Node 22.5 (Node 18/20 không chạy được). Zero External Dependencies, kiểm thử bằng `node --test` có sẵn.
- **Kịch bản kiểm thử tải:** K6 Script (`k6-script.js`, **50 VUs**) + Bộ giả lập tải song song (`simulate-load.js`).
- **Định dạng AC:** mỗi Story có khối **AC** — điều kiện nghiệm thu **kiểm chứng được** bằng một lệnh chạy, một query SQL, hoặc một assertion trong test. Không dùng câu mô tả chung chung.

---

## TỔNG QUAN TIẾN ĐỘ THEO EPICS

- [x] **EPIC 1: Dữ Liệu Thử Nghiệm & Cấu Trúc Dữ Liệu Nền Tảng (Data Structures & Mock Generator)**
- [x] **EPIC 2: Lập Trình Hướng Đối Tượng (OOP) — Đóng Gói, Trừu Tượng & Đa Hình**
- [x] **EPIC 3: Động Cơ Đối Soát Dữ Liệu Lớn (Big-O Engine: O(n²) vs O(n)) & Chống Trùng Lặp**
- [x] **EPIC 4: Máy Chủ HTTP Zero-Dependency & Tích Hợp API Chịu Tải**
- [x] **EPIC 5: Bộ Kịch Bản Bắn Tải K6 & Thực Nghiệm Đa Luồng (Race Condition & Benchmark)**
- [x] **EPIC 6: Dashboard Mission Control, Bộ Kiểm Thử & Đóng Gói Deploy**

---

## CHI TIẾT TỪNG STORY VÀ TASK

### EPIC 1: DỮ LIỆU THỬ NGHIỆM, DATABASE SQLITE & CẤU TRÚC DỮ LIỆU

#### [x] Story 1.1: Tích hợp Cơ sở dữ liệu SQLite lõi (node:sqlite) & Trình sinh Mock Data
- [x] **Task 1.1.1:** Viết `src/database/db.js` khởi tạo kết nối SQLite qua `node:sqlite` (DatabaseSync), tạo tự động 4 bảng `wallets`, `transactions`, `bank_statement_logs`, `recon_results`, kèm index trên `transactions(tx_code)` và `transactions(wallet_id, created_at)`.
- [x] **Task 1.1.2:** Bật `PRAGMA journal_mode = WAL` và `PRAGMA busy_timeout = 5000` ngay sau khi mở kết nối (chống `SQLITE_BUSY` khi 50 request ghi đồng thời). File DB đặt tại `data/fintech.db`.
- [x] **Task 1.1.3:** Viết `data/mockFintechData.js` sinh **20.000** dòng `bank_statement_logs` và **20.000** dòng `transactions` với `source = 'SEED'`, `wallet_id = 'WAL-SEED'`, `tx_code` duy nhất — để dữ liệu seed **không trộn** vào lịch sử ví thật.
- [x] **Task 1.1.4:** Cài đặt phân bố lệch dữ liệu thực tế bằng `tx_code`: 90% khớp, 5% lệch tiền, 5% thiếu (một nửa thiếu phía ngân hàng, một nửa thiếu phía nội bộ).
- [x] **Task 1.1.5:** Viết hàm `seedDemoWallet()` tạo ví `WAL-8892` với `balance = 100000` (đúng ví mà K6 và dashboard dùng), chỉ seed khi bảng `wallets` trống.
- [x] **Task 1.1.6:** Viết hàm `assertSeedShape()` đếm lại phân bố 90/5/5 bằng SQL và in ra khi seed — để con số trong tài liệu có phép đo, không phải niềm tin.

> **AC Story 1.1**
> - `node src/server.js` lần đầu tạo `data/fintech.db` với 4 bảng; chạy lần hai **không** nhân đôi dữ liệu.
> - `SELECT count(*) FROM bank_statement_logs` = **20.000**; `SELECT count(*) FROM transactions WHERE source='SEED'` = **20.000**.
> - `assertSeedShape()` in ra đúng tỉ lệ ~90% MATCH, ~5% AMOUNT_MISMATCH, ~5% MISSING (±1%).
> - `SELECT balance FROM wallets WHERE id='WAL-8892'` = **100000**; ví này **không** có dòng `source='SEED'` nào.

#### [x] Story 1.2: Cài đặt Cấu trúc Dữ liệu Stack & Queue thuần JavaScript
- [x] **Task 1.2.1:** Cài đặt `TransactionStack` trong `src/structures/TransactionStack.js` theo LIFO với `push()`, `pop()`, `peek()`, `isEmpty()`.
- [x] **Task 1.2.2:** Cài đặt `PaymentQueue` trong `src/structures/PaymentQueue.js` theo FIFO tối ưu bằng con trỏ chỉ mục (không dùng `shift()` gây $O(n)$).
- [x] **Task 1.2.3:** Viết test `tests/structures.test.js` kiểm tra thứ tự LIFO/FIFO và độ phức tạp amortized $O(1)$.

> **AC Story 1.2**
> - 10.000 lần `push`/`pop` giữ đúng thứ tự LIFO; 10.000 lần `enqueue`/`dequeue` giữ đúng thứ tự FIFO.
> - `grep -c "shift()" src/structures/PaymentQueue.js` = **0**.
> - Thời gian dequeue 500.000 phần tử **< 25×** thời gian 50.000 phần tử (tuyến tính ≈ 10×; nếu dùng `shift()` thì con số này là bậc hai).

---

### EPIC 2: LẬP TRÌNH HƯỚNG ĐỐI TƯỢNG (OOP) THỰC CHIẾN

#### [x] Story 2.1: Đóng gói Ví tiền & Cơ chế Async Mutex chống Race Condition
- [x] **Task 2.1.1:** Cài đặt `Wallet` trong `src/core/Wallet.js` với Private Fields (`#id`, `#balance`, `#owner`, `#queue`, `#historyStack`) — không có setter công khai cho `balance`.
- [x] **Task 2.1.2:** Viết `src/core/Transaction.js`: model giao dịch bất biến (`Object.freeze`), không có method mutating.
- [x] **Task 2.1.3:** Triển khai `deposit(amount)` — validate `Number.isInteger(amount) && amount > 0`, ghi lịch sử vào Stack.
- [x] **Task 2.1.4:** Triển khai `withdraw(amount)` với **promise-chain mutex** (hàng đợi `#queue` nối tiếp nhau), **bắt buộc có `await` giữa bước đọc và bước ghi** để race condition tái hiện được khi không khoá. Không dùng cờ boolean `#isLocked` (cờ này hoặc là no-op khi code đồng bộ, hoặc từ chối request thay vì xếp hàng).
- [x] **Task 2.1.5:** Triển khai `refundLastTransaction()` — `pop()` từ Stack, hoàn tiền $O(1)$.
- [x] **Task 2.1.6:** Ghi write-through xuống SQLite: mỗi `deposit`/`withdraw`/`refund` tạo đúng một dòng trong `transactions` (`source='LIVE'`, `tx_code` sinh bằng `node:crypto`) và cập nhật `wallets.balance` **trong cùng luồng xử lý**.
- [x] **Task 2.1.7:** Nạp lại ví từ DB khi khởi động (`SELECT ... FROM wallets` + dựng lại Stack từ `transactions` gần nhất) — RAM chỉ là cache, DB là nguồn sự thật.

> **AC Story 2.1**
> - `Object.keys(wallet)` không chứa `balance`; truy cập `wallet.#balance` từ ngoài là `SyntaxError`.
> - **50 request đồng thời** `withdraw(100000)` trên ví `balance = 100000` → đúng **1 SUCCESS**, **49 `INSUFFICIENT_FUNDS`**, `balance = 0`, không lần nào âm.
> - Sau mỗi thao tác: `SELECT balance FROM wallets WHERE id='WAL-8892'` khớp với giá trị trả về của API.
> - `withdraw(-1000)`, `withdraw(0)`, `withdraw(1.5)`, `withdraw("100000")` → `INVALID_AMOUNT`, số dư không đổi.
> - Restart server → `GET /api/wallet/balance` vẫn trả đúng số dư trước khi tắt.
> - `refundLastTransaction()` tạo dòng `type='REFUND'` trong DB và cộng lại đúng số tiền (không cộng đôi nếu gọi hai lần).

#### [x] Story 2.2: Hệ thống Cổng thanh toán Đa hình & Trừu tượng (Polymorphism & Abstraction)
- [x] **Task 2.2.1:** Cài đặt lớp cơ sở `IPaymentGateway` trong `src/gateways/IPaymentGateway.js` — `charge()`/`refund()` ném `NotImplementedError` nếu lớp con không override.
- [x] **Task 2.2.2:** Cài đặt `VNPayGateway.js` — ký HMAC-SHA256 bằng `node:crypto`, secret đọc từ `process.env.VNP_HASH_SECRET` (fallback chỉ dùng cho dev, có cảnh báo khi fallback).
- [x] **Task 2.2.3:** Cài đặt `MoMoGateway.js` — mô phỏng tạo mã QR và xử lý callback.
- [x] **Task 2.2.4:** Cài đặt `StripeGateway.js` — kiểm tra tính hợp lệ của token thẻ quốc tế.
- [x] **Task 2.2.5:** Cài đặt `PaymentService.js` — nhận gateway qua constructor (Dependency Injection), không biết gì về tên cổng cụ thể.
- [x] **Task 2.2.6:** Viết `src/gateways/registry.js` — `Map` từ mã cổng (`'VNPAY'`, `'MOMO'`, `'STRIPE'`) sang instance. Đây là **nơi duy nhất** ánh xạ chuỗi → đối tượng, thay cho `if/else` trong router.

> **AC Story 2.2**
> - `new IPaymentGateway().charge()` → ném `NotImplementedError`; cả 3 gateway con đều override đủ 2 method.
> - `registry.get('VNPAY')` trả instance VNPay; `registry.get('XYZ')` → `undefined` và endpoint trả `400 UNKNOWN_GATEWAY`.
> - `grep -c "if.*gateway" src/server.js` = **0** (không rẽ nhánh theo chuỗi cổng).
> - Charge qua VNPAY trả về `signature` là chuỗi hex 64 ký tự (SHA-256).

---

### EPIC 3: ĐỘNG CƠ ĐỐI SOÁT BIG-O & BẢO VỆ CHỐNG TRÙNG LẶP

#### [x] Story 3.1: Động cơ đối soát dữ liệu lớn (Big-O Reconciliation Engine)
- [x] **Task 3.1.1:** Cài đặt `ReconciliationService` trong `src/services/ReconciliationService.js`, chỉ đọc bản ghi `source='SEED'` để dữ liệu thật không làm nhiễu phép đo.
- [x] **Task 3.1.2:** Viết `reconcileNaive()` — vòng lặp + `indexOf()` ẩn (tìm tuyến tính cho từng phần tử), $O(n^2)$.
- [x] **Task 3.1.3:** Viết `reconcileOptimized()` — dựng `Map` theo `tx_code` trước, tra cứu $O(1)$, tổng $O(n)$.
- [x] **Task 3.1.4:** Cả hai hàm trả **cùng một cấu trúc kết quả**: `{ batchId, strategy, durationMs, matches, mismatches: [{ txCode, verdict, internalAmount, bankAmount }] }`, với `verdict ∈ MATCH | AMOUNT_MISMATCH | MISSING_IN_BANK | MISSING_INTERNAL`.
- [x] **Task 3.1.5:** Ghi kết quả từng lượt vào bảng `recon_results` (batch_id, strategy, tx_code, verdict, 2 amount, duration_ms).
- [x] **Task 3.1.6:** Viết CLI `benchmark.js` đo `performance.now()` + `process.memoryUsage().heapUsed` trước/sau, in bảng so sánh. **Không** in con số cam kết cố định — in số đo thật của máy đang chạy.

> **AC Story 3.1**
> - Trên cùng một snapshot, `reconcileNaive()` và `reconcileOptimized()` trả **mảng kết quả giống hệt nhau** (so sánh sâu, cùng thứ tự `tx_code`).
> - Tổng `matches + mismatches.length` = 21.000 (20.000 bản ghi nội bộ + 1.000 dòng chỉ có phía ngân hàng).
> - `SELECT strategy, duration_ms FROM recon_results ORDER BY id DESC LIMIT 2` cho thấy optimized nhanh hơn naive trên **cùng máy**, cùng dữ liệu.
> - `benchmark.js` in ra số đo thật; không có hằng số `18` nào hardcode trong code.

#### [x] Story 3.2: Bảo vệ chống lặp lệnh thanh toán (Idempotency Guard)
- [x] **Task 3.2.1:** Cài đặt `IdempotencyGuard` trong `src/security/IdempotencyGuard.js` dùng `Map` (tra cứu $O(1)$; cần map để gắn TTL cho từng key).
- [x] **Task 3.2.2:** Triển khai `has(key)`, `register(key)`, `checkAndRegister(key)` (**kiểm tra + đăng ký trong cùng một tick**, không có `await` ở giữa).
- [x] **Task 3.2.3:** Namespace khoá theo ví: khoá thật là `` `${walletId}:${idempotencyKey}` ``. Thêm TTL 15 phút (TTL injectable để test không phải chờ thật) và dọn key hết hạn định kỳ.
- [x] **Task 3.2.4:** Nối guard vào `POST /api/wallet/withdraw` và `POST /api/payment/charge`; request trùng → `409 DUPLICATE` **trước khi** chạm vào số dư.

> **AC Story 3.2**
> - 100 request cùng `idempotencyKey` trên cùng ví → **1 SUCCESS**, **99 `409 DUPLICATE`**.
> - Cùng `idempotencyKey` nhưng khác `walletId` → **cả hai** thành công (chứng minh có namespace).
> - Với TTL inject = 50ms: key dùng lại được sau 100ms (chứng minh có TTL).
> - Request bị chặn bởi guard **không** tạo dòng nào trong `transactions` và không đổi `balance`.
> - Ghi rõ trong README/SPEC: guard nằm trong RAM, restart là mất.

---

### EPIC 4: MÁY CHỦ HTTP ZERO-DEPENDENCY & API CHỊU TẢI

#### [x] Story 4.1: Khởi tạo HTTP REST Server bằng `node:http`
- [x] **Task 4.1.1:** Viết `src/server.js` khởi tạo HTTP server cổng 3000 (`process.env.PORT` cho Render), không dùng Express.
- [x] **Task 4.1.2:** Viết router + đọc JSON body có giới hạn **64KB** (quá → `413`), early return, envelope lỗi thống nhất `{"error":{"code","message"}}`.
- [x] **Task 4.1.3:** Endpoint `POST /api/wallet/withdraw` (validate amount là số nguyên dương, guard idempotency, mutex).
- [x] **Task 4.1.4:** Endpoint `POST /api/payment/charge` (registry gateway + guard idempotency + PaymentService).
- [x] **Task 4.1.5:** Endpoint `POST /api/reconcile/naive` và `POST /api/reconcile/optimized`. Naive là **đồng bộ có chủ đích**: đặt cờ `isRunning`, lượt thứ hai nhận `409 RECON_IN_PROGRESS` thay vì chồng thêm một lượt chặn Event Loop.
- [x] **Task 4.1.6:** Endpoint `GET /api/wallet/balance?walletId=...` (đọc từ DB, trả `source:"db"`) và `POST /api/wallet/refund` (body `{"walletId": "..."}`).
- [x] **Task 4.1.7:** Phục vụ file tĩnh `public/` cho `GET /` — chỉ 3 content-type (`text/html`, `text/css`, `text/javascript`), chặn `..`/đường dẫn tuyệt đối → `400 BAD_PATH`.
- [x] **Task 4.1.8:** Endpoint `GET /api/health` trả `{"ok":true}` (dùng cho Render health check và cho script chờ server sẵn sàng).
- [x] **Task 4.1.9:** Thứ tự khởi động: mở DB → bật PRAGMA → load ví từ DB → seed nếu trống → mới `listen()`.

> **AC Story 4.1**
> - 9 route hoạt động đúng contract ở SPEC §6; route lạ → `404 NOT_FOUND`.
> - JSON hỏng/thiếu field/`amount` sai kiểu → `400` kèm `error.code` cụ thể, **không** stack trace ra client.
> - Body > 64KB → `413 PAYLOAD_TOO_LARGE`.
> - `GET /../../etc/passwd` → `400 BAD_PATH`, không trả nội dung file ngoài `public/`.
> - Gọi `/api/reconcile/naive` hai lần đồng thời → request thứ hai nhận `409 RECON_IN_PROGRESS`.
> - `GET /api/wallet/balance` không truyền `walletId` → `400 MISSING_WALLET_ID`; ví không tồn tại → `404 WALLET_NOT_FOUND`.

---

### EPIC 5: BỘ KỊCH BẢN BẮN TẢI K6 & THỰC NGHIỆM ĐA LUỒNG

#### [x] Story 5.1: Viết kịch bản K6 chính thức (`k6-script.js`)
- [x] **Task 5.1.1:** Scenario `withdraw`: **50 VUs**, mỗi VU 1 request `withdraw` 100.000đ vào `WAL-8892` (scenario `restore` gọi `refund` đưa ví về 100.000 trước lượt withdraw).
- [x] **Task 5.1.2:** Scenario `idempotency`: 100 request cùng một `idempotencyKey`.
- [x] **Task 5.1.3:** Scenario `reconcile`: naive chạy **1 VU / 1 iteration**, optimized 20 VUs — đo p95 cả hai. Không bắn tải song song vào naive.
- [x] **Task 5.1.4:** Đặt `options.thresholds` + `check()`: `http_req_duration: p(95)<1000`, và check tổng SUCCESS = 1 / INSUFFICIENT_FUNDS = 49. Dùng `check` cho 4xx nghiệp vụ (K6 mặc định tính non-2xx là failed, nên không đặt threshold `http_req_failed` cho scenario withdraw).
- [x] **Task 5.1.5:** Chạy smoke `k6 run --vus 1 --duration 1s k6-script.js` trước khi demo (bắt lỗi cú pháp), ghi lệnh này vào README.

> **AC Story 5.1**
> - `k6 run --vus 1 --duration 1s k6-script.js` chạy không lỗi cú pháp.
> - Lượt withdraw 50 VUs: check pass với **đúng 1** SUCCESS và **49** INSUFFICIENT_FUNDS; threshold p95 < 1000ms pass.
> - Lượt idempotency: 1 SUCCESS / 99 DUPLICATE.
> - Output K6 in được p95 của naive và optimized; không có threshold nào khiến lượt naive bị tính là "hệ thống hỏng".

#### [x] Story 5.2: Viết script giả lập tải song song nội bộ (`simulate-load.js`)
- [x] **Task 5.2.1:** `simulate-load.js` dùng `Promise.all` + `node:http` bắn **50 request đồng thời** rút tiền, mỗi request ghi lại thời điểm bắt đầu (cùng ~1ms).
- [x] **Task 5.2.2:** In bảng tổng kết: tổng request, SUCCESS, REJECTED theo từng `error.code`, số dư cuối, thời gian chạy.
- [x] **Task 5.2.3:** Viết `index.js` chạy kịch bản đầu-cuối trên Terminal: khởi tạo ví → nạp tiền → bắn 50 request → đối soát Big-O → hoàn tiền bằng Stack. Tự spawn/gọi server hoặc kiểm tra server đã chạy và báo lỗi rõ ràng.

> **AC Story 5.2**
> - 50 request được gửi trong cùng một mili-giây (chênh lệch `startedAt` giữa request đầu và cuối < 20ms).
> - Bảng tổng kết: SUCCESS + REJECTED = 50; số dư cuối = **0** và không âm.
> - `node index.js` chạy trọn kịch bản, exit code 0; nếu server chưa chạy thì in hướng dẫn thay vì crash.

---

### EPIC 6: DASHBOARD MISSION CONTROL, BỘ KIỂM THỬ & ĐÓNG GÓI DEPLOY

#### [x] Story 6.1: Frontend Dashboard (`public/index.html`)
- [x] **Task 6.1.1:** Dựng `public/index.html` — HTML/CSS/JS thuần, dark mode, không build step, không CDN, không npm. Hiển thị số dư `WAL-8892` và tự làm mới mỗi 2 giây từ `GET /api/wallet/balance`.
- [x] **Task 6.1.2:** Nút **"Kích nổ Race Condition"**: bắn 50 request đồng thời từ browser, hiện bảng kết quả (SUCCESS / INSUFFICIENT_FUNDS / số dư sau khi bắn) — để nhìn thấy bằng mắt rằng không có số dư âm.
- [x] **Task 6.1.3:** Nút **"Đối soát"**: gọi lần lượt `/api/reconcile/naive` và `/api/reconcile/optimized`, hiện hai cột thời gian + số bản ghi lệch.
- [x] **Task 6.1.4:** Nút **"Hoàn tiền"** gọi `POST /api/wallet/refund` và hiện danh sách 10 giao dịch gần nhất.

> **AC Story 6.1**
> - Mở `http://localhost:3000` thấy dashboard; không có request nào ra domain ngoài.
> - Số dư trên dashboard khớp `GET /api/wallet/balance?walletId=WAL-8892` tại cùng thời điểm.
> - Sau khi bấm "Kích nổ": bảng hiện đúng 1 SUCCESS và 49 INSUFFICIENT_FUNDS, số dư 0.
> - Bấm "Đối soát": hai cột thời gian là số **đo thật** từ response, không hardcode.
> - Lỗi API hiển thị `error.code` cho người dùng, không nuốt lỗi im lặng.

#### [x] Story 6.2: Bộ kiểm thử tự động bằng `node --test`
- [x] **Task 6.2.1:** `tests/wallet.test.js` — đóng gói (không truy cập được `#balance`), validate amount, 50 request đồng thời → 1 SUCCESS / 49 / số dư 0, refund không cộng đôi.
- [x] **Task 6.2.2:** `tests/idempotency.test.js` — trùng key, namespace theo ví, TTL injectable.
- [x] **Task 6.2.3:** `tests/reconciliation.test.js` — naive và optimized trả cùng kết quả trên cùng snapshot.
- [x] **Task 6.2.4:** Ghi lệnh chạy test (`node --test "tests/*.test.js"`) kèm yêu cầu: không cần `npm install`, không cần mạng, không cần k6.

> **AC Story 6.2**
> - `node --test "tests/*.test.js"` xanh toàn bộ, chạy được trên máy sạch chỉ có Node 22.15.
> - Có ít nhất một test chứng minh bất biến "không bao giờ số dư âm" dưới đồng thời.
> - Test idempotency không chờ 15 phút thật (dùng TTL inject).
> - Test không phụ thuộc thứ tự chạy (mỗi file tự tạo DB tạm `:memory:` hoặc file riêng).

#### [x] Story 6.3: Đóng gói & Deploy
- [x] **Task 6.3.1:** Viết `render.yaml` — `NODE_VERSION=22.15.0`, build rỗng, start `node src/server.js`, kèm comment cảnh báo filesystem Render free là tạm.
- [x] **Task 6.3.2:** Viết `.gitignore` loại `data/*.db`, `*.db-wal`, `*.db-shm` (không commit DB và file WAL).

> **AC Story 6.3**
> - `render.yaml` khai báo đúng Node 22.15.0 và start command; không có `npm install`.
> - `git status` sau khi chạy server không thấy `data/fintech.db` hay file `-wal`/`-shm`.
