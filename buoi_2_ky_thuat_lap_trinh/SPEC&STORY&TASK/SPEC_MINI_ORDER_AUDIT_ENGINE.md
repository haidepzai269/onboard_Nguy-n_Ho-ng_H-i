# ĐẶC TẢ KỸ THUẬT (SPECIFICATION): MINI ORDER & AUDIT ENGINE

## Hệ Thống Xử Lý Đơn Hàng & Truy Vết Biến Động Trạng Thái (Node.js Thuần)

> [!IMPORTANT]
> **MỤC TIÊU CỦA ĐẶC TẢ:**
> Tài liệu này được thiết kế nhằm cụ thể hóa toàn bộ kiến thức cốt lõi của **Buổi 2: Kỹ thuật lập trình** (Cấu trúc dữ liệu, Big-O, OOP thực dụng, Clean Code và Debug có hệ thống) thành một ứng dụng thực chiến chạy trên **Node.js thuần** (không cài thêm bất kỳ npm package nào, chạy trực tiếp bằng lệnh `node <file>.js`).
> 
> Hệ thống phục vụ mục đích **"Học để hiểu sâu bản chất, không học vẹt lý thuyết"** thông qua việc tự tay đo đạc, kiểm chứng sự chênh lệch hiệu năng và thiết kế mã nguồn chuẩn công nghiệp.

---

## 1. THÔNG TIN CHUNG DỰ ÁN

- **Tên dự án:** `Mini Order & Audit Engine`
- **Mã dự án:** `MOAE`
- **Vị trí tài liệu:** [SPEC_MINI_ORDER_AUDIT_ENGINE.md](file:///c:/onboard/buoi_2_ky_thuat_lap_trinh/SPEC&STORY&TASK/SPEC_MINI_ORDER_AUDIT_ENGINE.md)
- **Thư mục mã nguồn:** `c:\onboard\buoi_2_ky_thuat_lap_trinh\mini-order-engine`
- **Môi trường thực thi:** **Node.js (v18+ trở lên)**, không phụ thuộc bên ngoài (`Zero-dependency`).
- **Hình thức tương tác:** Giao diện dòng lệnh (CLI Terminal Console).

---

## 2. BỐI CẢNH & VẤN ĐỀ NGHIỆP VỤ (WHY)

Trong thực tế phát triển phần mềm Backend và Web Application:
1. **Lỗi hiệu năng Production:** Các đoạn code nối dữ liệu (Join/Enrichment) viết bằng `Array.find()` chạy rất nhanh trên máy dev (50 bản ghi), nhưng khi lên production với $20.000$ đơn hàng và $2.000$ khách hàng thì vòng lặp ẩn $O(n^2)$ làm treo cứng tiến trình Node.js (Event Loop bị block).
2. **Trạng thái đơn hàng bị sửa đổi tùy tiện:** Đơn hàng đã thanh toán (`PAID`) hoặc đã hủy (`CANCELLED`) vẫn bị các hàm bên ngoài ghi đè tiền hoặc thêm sản phẩm do thiếu cơ chế đóng gói (Encapsulation).
3. **Thao tác hủy đơn và hoàn tác phức tạp:** Khách hàng hoặc Admin cần nút **Undo** để khôi phục trạng thái đơn hàng trước đó nếu thao tác nhầm.
4. **Gửi thông báo nghẽn mạng:** Khi có hàng loạt đơn hàng được xử lý cùng lúc, nếu gửi thông báo trực tiếp sẽ gây nghẽn luồng. Cần cơ chế hàng đợi (Queue) xử lý lần lượt theo nguyên tắc đến trước phục vụ trước (FIFO).

---

## 3. KIẾN TRÚC HỆ THỐNG & LUỒNG DỮ LIỆU (PIPELINE)

```
[Khách hàng / API Input]
       │
       ▼
 ┌─────────────────────────────────────────────────────────────┐
 │ 1. OrderValidator (Single Responsibility - Pure Functions)  │
 │    - Early return kiểm tra tính hợp lệ dữ liệu              │
 │    - Defensive programming (?. và ??)                       │
 └──────────────────────────────┬──────────────────────────────┘
                                │ (Hợp lệ)
                                ▼
 ┌─────────────────────────────────────────────────────────────┐
 │ 2. OrderEnricher (Cấu trúc dữ liệu & Tối ưu Big-O)           │
 │    - Tra cứu User & Product bằng Map (O(1) thay vì O(n))    │
 │    - Kiểm tra Blacklist & Voucher bằng Set.has() (O(1))     │
 └──────────────────────────────┬──────────────────────────────┘
                                │
                                ▼
 ┌─────────────────────────────────────────────────────────────┐
 │ 3. Order Aggregate (OOP: Encapsulation & State Machine)     │
 │    - Bảo vệ #status, #items, #history qua Private Fields    │
 │    - Chuyển trạng thái: CREATED -> PAID -> SHIPPED          │
 └───────────────────────┬──────────────┬──────────────────────┘
                         │              │
                         │              ▼
                         │     ┌───────────────────────────────┐
                         │     │ 4. AuditRollback (Stack LIFO) │
                         │     │    - Push snapshot trạng thái │
                         │     │    - Pop khi gọi hàm undo()   │
                         │     └───────────────────────────────┘
                         ▼
 ┌─────────────────────────────────────────────────────────────┐
 │ 5. NotificationDispatcher (Queue FIFO & Polymorphism)       │
 │    - Enqueue job gửi thông báo vào Hàng đợi FIFO             │
 │    - Đa hình: EmailNotifier, SmsNotifier, ConsoleNotifier   │
 │    - Dependency Injection: Truyền Notifier từ ngoài vào     │
 └─────────────────────────────────────────────────────────────┘
```

---

## 4. MA TRẬN ÁP DỤNG CẤU TRÚC DỮ LIỆU & GIẢI THUẬT

Dự án bắt buộc phải vận dụng và thể hiện rõ nét vai trò của 5 cấu trúc dữ liệu cốt lõi đã học:

| Cấu trúc dữ liệu | Vị trí cài đặt trong dự án | Vai trò kỹ thuật & Độ phức tạp | Đánh đổi (Trade-off) |
| :--- | :--- | :--- | :--- |
| **Hash Map (`Map`)** | `UserRegistry`, `ProductCatalog` | Tra cứu thông tin User & Giá sản phẩm theo ID với độ phức tạp **$O(1)$**. | Tốn thêm $O(n)$ bộ nhớ RAM một lần để đổi lấy tốc độ tra cứu siêu tốc hàng chục ngàn lần. |
| **Set** | `VoucherManager`, `CustomerBlacklist` | Kiểm tra mã giảm giá còn hiệu lực và khách hàng có trong danh sách đen không qua `Set.has()` (**$O(1)$**). | Khử trùng lặp tự động, không quan tâm thứ tự, tối ưu tốc độ kiểm tra tồn tại. |
| **Stack** | `AuditUndoStack` | Lưu lịch sử biến động trạng thái (Snapshot) theo nguyên tắc **LIFO** (Vào sau ra trước). Cho phép `undoLastAction()`. | Giữ lịch sử thao tác gần nhất ở đỉnh Stack; `push`/`pop` có chi phí $O(1)$. |
| **Queue** | `NotificationQueue` | Hàng đợi chứa các tin nhắn/email cần phát đi theo nguyên tắc **FIFO** (Vào trước ra trước). | Đảm bảo tính công bằng: Đơn hàng nào tạo trước thì gửi thông báo trước, tránh nghẽn I/O. |
| **Array `[]`** | `OrderRepository` | Lưu danh sách đơn hàng để duyệt render báo cáo, hỗ trợ phân trang (Pagination) và duyệt tuần tự. | Giữ nguyên thứ tự thời gian tạo đơn, truy cập theo chỉ số (Index) $O(1)$. |

---

## 5. THIẾT KẾ HƯỚNG ĐỐI TƯỢNG (OOP) & NGUYÊN TẮC THIẾT KẾ

### 5.1. Tính đóng gói (Encapsulation) — Class `Order`
- Sử dụng cú pháp ECMAScript Private Field (`#`) để khóa chặt các trường nội tại:
  - `#id`: Mã định danh đơn hàng (Read-only).
  - `#userId`: ID khách hàng.
  - `#status`: Trạng thái (`PENDING`, `PAID`, `CANCELLED`, `COMPLETED`).
  - `#items`: Danh sách sản phẩm (Mảng nội bộ, không cho phép gán trực tiếp từ ngoài).
  - `#totalAmount`: Tổng tiền đã tính toán sau khi áp voucher.
- **Hành vi kiểm soát:**
  - `addItem(productId, quantity, unitPrice)`: Kiểm tra hợp lệ trước khi đẩy vào mảng.
  - `pay(paymentMethod)`: Chỉ cho phép thanh toán khi đơn hàng đang ở trạng thái `PENDING`. Báo lỗi nếu đã `CANCELLED` hoặc đã `PAID`.
  - `cancel(reason)`: Kiểm tra nếu đơn hàng đã giao (`COMPLETED`) thì từ chối hủy.
  - Cung cấp getter công khai: `get status()`, `get totalAmount()`, `get summary()`.

### 5.2. Tính đa hình (Polymorphism) & Dependency Injection — Tầng thông báo
- Định nghĩa chuẩn giao tiếp (Interface Pattern):
  ```javascript
  class INotifier {
    send(recipient, message) {
      throw new Error("Phải cài đặt phương thức send()");
    }
  }
  ```
- Các lớp cài đặt cụ thể:
  - `EmailNotifier`: Giả lập định dạng và gửi email qua SMTP.
  - `SmsNotifier`: Giả lập gửi tin nhắn SMS tới số điện thoại.
  - `MockTestNotifier`: Dùng cho môi trường Unit Test (lưu tin nhắn vào bộ nhớ, không in ra màn hình hoặc gửi ra ngoài).
- Lớp `NotificationService` nhận `notifier` từ bên ngoài qua constructor (Dependency Injection), tuyệt đối **không** dùng chuỗi `if (type === 'email') ... else if ...`.

### 5.3. Nguyên tắc Single Responsibility (Trách nhiệm duy nhất)
- `OrderValidator.js`: Chỉ làm nhiệm vụ kiểm tra tính toàn vẹn của payload đơn hàng (Pure functions, không lưu state).
- `OrderEnricher.js`: Chỉ làm nhiệm vụ ánh xạ ID sang thông tin chi tiết bằng `Map`.
- `AuditService.js`: Chỉ quản lý Stack lịch sử và thực hiện Undo.
- `OrderRepository.js`: Chỉ làm nhiệm vụ lưu trữ và truy vấn danh sách đơn hàng.

---

## 6. KỸ NĂNG LẬP TRÌNH NỀN TẢNG & DEFENSIVE PROGRAMMING

1. **Early Return:** Mọi hàm xử lý nghiệp vụ đều đặt các câu lệnh kiểm tra biên (Edge cases) ở dòng đầu tiên:
   ```javascript
   function validateOrderPayload(payload) {
     if (!payload) return { valid: false, error: "Payload rỗng" };
     if (!payload.userId) return { valid: false, error: "Thiếu thông tin userId" };
     if (!Array.isArray(payload.items) || payload.items.length === 0) {
       return { valid: false, error: "Đơn hàng phải có ít nhất 1 sản phẩm" };
     }
     return { valid: true };
   }
   ```
2. **Defensive Data Access:**
   - Dùng `payload?.customer?.profile?.phone ?? "Chưa cung cấp"`.
   - Tránh truy cập thuộc tính sâu mà không kiểm tra lồng (`Cannot read property of undefined`).
3. **Cấm nuốt lỗi:**
   - Tuyệt đối không viết `catch (e) {}`. Mọi ngoại lệ bắt buộc phải được đóng gói vào object lỗi gồm `{ timestamp, code, originalMessage, context }` và ghi log có tổ chức.

---

## 7. MODULE BENCHMARK: THỰC CHỨNG $O(n^2)$ VS $O(n)$

Hệ thống tích hợp sẵn một script benchmark độc lập `benchmark.js`:
- **Đầu vào thử nghiệm:** Sinh dữ liệu ngẫu nhiên $20.000$ đơn hàng và $2.000$ khách hàng.
- **Kịch bản 1 (Cách tiếp cận ngây thơ - Naive $O(n^2)$):**
  - Dùng `orders.map(order => users.find(u => u.id === order.userId))`.
  - Đo bằng `console.time("Naive O(n^2)")`.
- **Kịch bản 2 (Cách tiếp cận tối ưu - Optimized $O(n)$):**
  - Khởi tạo `new Map(users.map(u => [u.id, u]))`.
  - Dùng `orders.map(order => userMap.get(order.userId))`.
  - Đo bằng `console.time("Optimized O(n)")`.
- **Tiêu chí nghiệm thu (Acceptance Criteria):**
  - Kịch bản 2 phải nhanh hơn Kịch bản 1 tối thiểu **100 lần** trên cùng một tập dữ liệu.

---

## 8. CẤU TRÚC THƯ MỤC TRIỂN KHAI

```text
c:\onboard\buoi_2_ky_thuat_lap_trinh\mini-order-engine\
│
├── data/
│   └── mockDataGenerator.js    # Tạo dữ liệu ngẫu nhiên (Users, Products, Orders)
│
├── src/
│   ├── models/
│   │   └── Order.js            # OOP Encapsulation (Private fields #status, #items)
│   │
│   ├── services/
│   │   ├── OrderValidator.js   # Pure functions, Early return, Defensive checks
│   │   ├── OrderEnricher.js    # Sử dụng Map & Set để nối dữ liệu O(1)
│   │   ├── AuditService.js     # Quản lý Stack Undo biến động trạng thái
│   │   └── notifiers/          # Đa hình & Dependency Injection
│   │       ├── INotifier.js
│   │       ├── EmailNotifier.js
│   │       ├── SmsNotifier.js
│   │       └── MockTestNotifier.js
│   │
│   ├── structures/
│   │   ├── Stack.js            # Cài đặt Stack (LIFO) chuẩn chỉ
│   │   └── Queue.js            # Cài đặt Queue (FIFO) chuẩn chỉ
│   │
│   └── pipeline/
│       └── OrderPipeline.js    # Điều phối toàn bộ luồng xử lý đơn hàng
│
├── benchmark.js                # Chạy đo đạc O(n^2) vs O(n) với console.time
└── index.js                    # File chạy demo tương tác toàn diện trên Terminal
```

---

## 9. KẾ HOẠCH TRIỂN KHAI & DANH SÁCH TASK (IMPLEMENTATION PLAN)

Học viên thực hiện theo thứ tự từng giai đoạn và đánh dấu `[x]` khi hoàn thành:

### Giai đoạn 1: Chuẩn bị Cấu trúc Dữ liệu & Benchmark Hiệu năng
- [ ] **Task 1.1:** Viết `mockDataGenerator.js` sinh ngẫu nhiên 20.000 Orders, 2.000 Users, 500 Vouchers.
- [ ] **Task 1.2:** Viết `benchmark.js` so sánh trực tiếp `Array.find()` ($O(n^2)$) và `Map.get()` ($O(n)$) trên 20.000 orders. Ghi lại kết quả mili-giây.
- [ ] **Task 1.3:** Viết lớp `Stack.js` (push, pop, peek, isEmpty, size) và `Queue.js` (enqueue, dequeue, peek, size).

### Giai đoạn 2: Thiết kế Hướng đối tượng & Đóng gói (OOP)
- [ ] **Task 2.1:** Cài đặt lớp `Order.js` với private fields `#id`, `#userId`, `#items`, `#status`, `#totalAmount`.
- [ ] **Task 2.2:** Cài đặt các phương thức biến đổi trạng thái có validate: `addItem()`, `applyVoucher()`, `pay()`, `cancel()`.
- [ ] **Task 2.3:** Tạo các Notifier (`EmailNotifier`, `SmsNotifier`, `MockTestNotifier`) kế thừa `INotifier` và tích hợp Dependency Injection.

### Giai đoạn 3: Xây dựng Pipeline & Lập trình phòng thủ (Clean Code)
- [ ] **Task 3.1:** Viết `OrderValidator.js` sử dụng kỹ thuật Early Return và kiểm tra input bằng Optional Chaining (`?.`, `??`).
- [ ] **Task 3.2:** Viết `OrderEnricher.js` dùng `Map` để gắn thông tin user, dùng `Set` để kiểm tra voucher hợp lệ.
- [ ] **Task 3.3:** Viết `AuditService.js` sử dụng `Stack` để lưu trữ snapshot lịch sử đơn hàng, hỗ trợ hàm `undoLastChange()`.
- [ ] **Task 3.4:** Đưa thông báo vào `Queue` và xử lý tuần tự theo cơ chế Worker FIFO.

### Giai đoạn 4: Tích hợp & Kiểm thử toàn diện
- [ ] **Task 4.1:** Hoàn thiện `index.js` chạy một kịch bản hoàn chỉnh từ lúc tạo đơn -> tính tiền -> thanh toán -> gửi thông báo -> hủy đơn -> Undo khôi phục trạng thái.
- [ ] **Task 4.2:** Cố tình đưa vào các case dữ liệu bẩn (missing field, sai ID) để kiểm tra cơ chế bắt lỗi không nuốt lỗi (`catch(e)`).
- [ ] **Task 4.3:** Chạy thử nghiệm và chụp output terminal để nghiệm thu.
