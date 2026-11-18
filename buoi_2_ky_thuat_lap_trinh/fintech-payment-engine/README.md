# FinTech Payment & Reconciliation Engine

Demo Buổi 2 — Kỹ thuật lập trình. Node.js thuần, **không dependency**, SQLite qua `node:sqlite`.

## Yêu cầu

- Node.js **22.15.0** trở lên (dùng `node:sqlite`, có sẵn từ Node 22.5).
- Không cần `npm install`, không cần mạng, không cần database server.

## Chạy

```bash
node src/server.js        # http://localhost:3000 (PORT qua biến môi trường)
```

Lần chạy đầu tự tạo `data/fintech.db`, seed 20.000 giao dịch nội bộ + 20.000 dòng sao kê
và ví demo `WAL-8892` với 100.000đ.

## Kiểm thử

```bash
node --test "tests/*.test.js"   # 21 test, ~0.3s
node --test                     # tự dò toàn bộ file *.test.js
```

## Đo hiệu năng & bắn tải

```bash
node benchmark.js               # in bảng O(n²) vs O(n) trên dữ liệu thật
node index.js                   # kịch bản đầu-cuối, tự dựng DB riêng
node simulate-load.js           # 50 request đồng thời vào server đang chạy
```

K6 (nếu đã cài):

```bash
k6 run --vus 1 --duration 1s k6-script.js   # smoke, bắt lỗi cú pháp
k6 run k6-script.js                         # full 50 VUs + idempotency + reconcile
```

> Trước lượt withdraw của K6, ví phải có 100.000đ. Sau một lượt chạy, gọi
> `POST /api/wallet/refund` hoặc xoá `data/fintech.db` rồi khởi động lại.

## API

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/api/health` | `{ok:true}`, dùng cho health check |
| GET | `/api/wallet/balance?walletId=` | đọc trực tiếp từ DB |
| GET | `/api/wallet/transactions?walletId=&limit=` | lịch sử giao dịch LIVE |
| POST | `/api/wallet/withdraw` | `{walletId, amount, idempotencyKey?}` |
| POST | `/api/payment/charge` | `{walletId, gateway, amount, idempotencyKey?, token?}` |
| POST | `/api/wallet/refund` | `{walletId}` — hoàn giao dịch gần nhất (Stack) |
| POST | `/api/reconcile/naive` | O(n²), đồng bộ, có cờ chống chạy chồng |
| POST | `/api/reconcile/optimized` | O(n) bằng Map |

Lỗi luôn có dạng `{"error":{"code","message"}}`.

## Cấu trúc

```
src/
  database/   db.js            — mở SQLite, PRAGMA WAL + busy_timeout, schema
  structures/ TransactionStack (LIFO), PaymentQueue (FIFO con trỏ)
  core/       Wallet (private fields + promise-chain mutex), Transaction (frozen)
  gateways/   IPaymentGateway + VNPay/MoMo/Stripe + registry (Map)
  services/   PaymentService (DI), ReconciliationService (naive/optimized)
  security/   IdempotencyGuard (namespace + TTL)
  server.js   node:http, 8 route + static public/
public/index.html             — dashboard vanilla, dark mode
```

## Triển khai

`render.yaml` khai báo Node 22.15.0, build rỗng, start `node src/server.js`.
Filesystem của Render Free là **ephemeral**: DB mất sau mỗi lần restart/deploy.
