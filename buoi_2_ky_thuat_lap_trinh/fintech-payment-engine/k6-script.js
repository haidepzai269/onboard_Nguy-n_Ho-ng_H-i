/**
 * Kịch bản K6 chính thức — 50 VUs.
 *
 *   k6 run --vus 1 --duration 1s k6-script.js     # smoke, kiểm cú pháp
 *   k6 run k6-script.js                            # full: 50 VUs + idempotency + reconcile
 *
 * Lưu ý: ví WAL-8892 phải đang có 100.000đ trước lượt withdraw
 * (khởi động lại server với DB mới, hoặc gọi /api/wallet/refund).
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

const BASE = __ENV.BASE_URL || 'http://localhost:3000';
const WALLET = __ENV.WALLET_ID || 'WAL-8892';

const withdrawSuccess = new Counter('withdraw_success');
const withdrawRejected = new Counter('withdraw_insufficient');
const idempotencySuccess = new Counter('idem_success');
const idempotencyDuplicate = new Counter('idem_duplicate');

export const options = {
  scenarios: {
    // Chạy TRƯỚC để ví còn tiền: 100 request cùng một idempotencyKey.
    // Chỉ 1 request được trừ tiền, 99 request nhận 409 DUPLICATE.
    idempotency: {
      executor: 'shared-iterations',
      vus: 10,
      iterations: 100,
      exec: 'idempotencyScenario',
    },
    // Hoàn lại khoản charge vừa rồi để ví về đúng 100.000 trước lượt withdraw.
    restore: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 1,
      exec: 'restoreScenario',
      startTime: '5s',
    },
    // 50 VUs × 1 request: kiểm tra Mutex Lock trên cùng một ví.
    withdraw: {
      executor: 'per-vu-iterations',
      vus: 50,
      iterations: 1,
      exec: 'withdrawScenario',
      startTime: '10s',
    },
    // Đối soát tối ưu dưới tải: 20 VUs gửi đều trong 5s.
    reconcile_optimized: {
      executor: 'constant-vus',
      vus: 20,
      duration: '5s',
      exec: 'reconcileOptimizedScenario',
      startTime: '20s',
    },
    // Đối soát naive: 1 VU, 1 iteration — đồng bộ, có chủ đích làm nghẽn loop.
    reconcile_naive: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 1,
      exec: 'reconcileNaiveScenario',
      startTime: '40s',
    },
  },
  thresholds: {
    // 4xx nghiệp vụ (INSUFFICIENT_FUNDS) không phải lỗi hệ thống → không đặt http_req_failed.
    'http_req_duration{scenario:withdraw}': ['p(95)<1000'],
    'http_req_duration{scenario:reconcile_optimized}': ['p(95)<2000'],
    checks: ['rate>0.95'],
  },
};

// Dùng khi chạy smoke bằng CLI (`--vus/--duration` ghi đè toàn bộ scenarios):
// k6 cần một hàm `default` để có việc mà làm.
export default function () {
  withdrawScenario();
}

export function withdrawScenario() {
  const res = http.post(
    `${BASE}/api/wallet/withdraw`,
    JSON.stringify({
      walletId: WALLET,
      amount: 100000,
      idempotencyKey: `K6-W-${__VU}-${__ITER}-${Date.now()}`,
    }),
    { headers: { 'content-type': 'application/json' }, tags: { scenario: 'withdraw' } }
  );

  const body = safeJson(res);
  check(res, {
    'withdraw: 200 hoặc 409': (r) => r.status === 200 || r.status === 409,
  });
  if (res.status === 200) withdrawSuccess.add(1);
  else if (body?.error?.code === 'INSUFFICIENT_FUNDS') withdrawRejected.add(1);
}

export function idempotencyScenario() {
  const res = http.post(
    `${BASE}/api/payment/charge`,
    JSON.stringify({
      walletId: WALLET,
      gateway: 'VNPAY',
      amount: 1000,
      idempotencyKey: 'K6-IDEMPOTENCY-SAME-KEY',
    }),
    { headers: { 'content-type': 'application/json' }, tags: { scenario: 'idempotency' } }
  );

  const body = safeJson(res);
  check(res, {
    'idempotency: 200 hoặc 409 DUPLICATE': (r) =>
      r.status === 200 || (r.status === 409 && body?.error?.code === 'DUPLICATE'),
  });
  if (res.status === 200) idempotencySuccess.add(1);
  if (body?.error?.code === 'DUPLICATE') idempotencyDuplicate.add(1);
}

export function restoreScenario() {
  const res = http.post(`${BASE}/api/wallet/refund`, JSON.stringify({ walletId: WALLET }), {
    headers: { 'content-type': 'application/json' },
    tags: { scenario: 'restore' },
  });
  check(res, { 'restore: refund 200': (r) => r.status === 200 });
}

export function reconcileOptimizedScenario() {
  const res = http.post(`${BASE}/api/reconcile/optimized`, '{}', {
    headers: { 'content-type': 'application/json' },
    tags: { scenario: 'reconcile_optimized' },
  });
  const body = safeJson(res);
  check(res, {
    // 409 RECON_IN_PROGRESS là hành vi đúng khi nhiều VU gọi cùng lúc.
    'optimized: 200 hoặc 409 RECON_IN_PROGRESS': (r) =>
      r.status === 200 || body?.error?.code === 'RECON_IN_PROGRESS',
    'optimized: 200 phải có batchId': () => res.status !== 200 || Boolean(body?.batchId),
  });
  sleep(0.5);
}

export function reconcileNaiveScenario() {
  const res = http.post(`${BASE}/api/reconcile/naive`, '{}', {
    headers: { 'content-type': 'application/json' },
    tags: { scenario: 'reconcile_naive' },
    timeout: '120s',
  });
  check(res, {
    'naive: 200': (r) => r.status === 200,
  });
  sleep(0.5);
}

function safeJson(res) {
  try {
    return res.json();
  } catch {
    return null;
  }
}
