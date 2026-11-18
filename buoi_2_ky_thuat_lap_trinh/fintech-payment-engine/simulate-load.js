'use strict';

/**
 * Bắn 50 request ĐỒNG THỜI vào cùng một ví (không cần cài k6).
 *
 *   node simulate-load.js
 *   node simulate-load.js --requests 50 --amount 100000 --wallet WAL-8892 --port 3000
 *
 * Kết quả kỳ vọng với ví 100.000đ: đúng 1 SUCCESS, 49 INSUFFICIENT_FUNDS, số dư cuối 0.
 */
const http = require('node:http');

function post(port, pathname, body) {
  return new Promise((resolve, reject) => {
    const payload = JSON.stringify(body);
    const startedAt = performance.now();
    const req = http.request(
      {
        host: '127.0.0.1',
        port,
        path: pathname,
        method: 'POST',
        headers: { 'content-type': 'application/json', 'content-length': Buffer.byteLength(payload) },
      },
      (res) => {
        let raw = '';
        res.on('data', (chunk) => { raw += chunk; });
        res.on('end', () => {
          let parsed = null;
          try { parsed = JSON.parse(raw); } catch { /* body không phải JSON */ }
          resolve({ status: res.statusCode, body: parsed, startedAt, durationMs: performance.now() - startedAt });
        });
      }
    );
    req.on('error', reject);
    req.write(payload);
    req.end();
  });
}

function get(port, pathname) {
  return new Promise((resolve, reject) => {
    http
      .get({ host: '127.0.0.1', port, path: pathname }, (res) => {
        let raw = '';
        res.on('data', (c) => { raw += c; });
        res.on('end', () => {
          try { resolve(JSON.parse(raw)); } catch { resolve({ raw }); }
        });
      })
      .on('error', reject);
  });
}

async function fireConcurrent({ port, walletId, amount, count = 50 }) {
  const requests = Array.from({ length: count }, (_, i) =>
    post(port, '/api/wallet/withdraw', {
      walletId,
      amount,
      idempotencyKey: `SIM-${Date.now()}-${i}`,
    })
  );
  const results = await Promise.all(requests);
  const byCode = {};
  for (const r of results) {
    const code = r.status === 200 ? 'SUCCESS' : r.body?.error?.code || `HTTP_${r.status}`;
    byCode[code] = (byCode[code] || 0) + 1;
  }
  const starts = results.map((r) => r.startedAt);
  return {
    results,
    byCode,
    spreadMs: Math.round(Math.max(...starts) - Math.min(...starts)),
    wallMs: Math.round(Math.max(...results.map((r) => r.durationMs))),
  };
}

async function main(argv = process.argv.slice(2)) {
  const arg = (flag, fallback) => {
    const i = argv.indexOf(flag);
    return i >= 0 && argv[i + 1] !== undefined ? argv[i + 1] : fallback;
  };
  const port = Number(arg('--port', process.env.PORT || 3000));
  const walletId = arg('--wallet', 'WAL-8892');
  const amount = Number(arg('--amount', 100000));
  const count = Number(arg('--requests', 50));

  const before = await get(port, `/api/wallet/balance?walletId=${walletId}`);
  if (before.error) {
    console.error('Không đọc được số dư:', before.error, '\nServer đã chạy chưa? `node src/server.js`');
    process.exitCode = 1;
    return;
  }

  console.log(`Bắn ${count} request đồng thời: rút ${amount}đ từ ${walletId} (số dư ${before.balance}đ)\n`);
  const outcome = await fireConcurrent({ port, walletId, amount, count });
  const after = await get(port, `/api/wallet/balance?walletId=${walletId}`);

  console.table([
    { metric: 'Tổng request', value: count },
    { metric: 'SUCCESS', value: outcome.byCode.SUCCESS || 0 },
    { metric: 'INSUFFICIENT_FUNDS', value: outcome.byCode.INSUFFICIENT_FUNDS || 0 },
    { metric: 'Mã lỗi khác', value: Object.entries(outcome.byCode).filter(([k]) => !['SUCCESS', 'INSUFFICIENT_FUNDS'].includes(k)).map(([k, v]) => `${k}:${v}`).join(', ') || '—' },
    { metric: 'Chênh lệch thời điểm gửi (ms)', value: outcome.spreadMs },
    { metric: 'Số dư trước', value: before.balance },
    { metric: 'Số dư sau', value: after.balance },
    { metric: 'Số dư âm?', value: after.balance < 0 ? 'CÓ — SAI' : 'không' },
  ]);

  if (after.balance < 0) {
    console.error('FAIL: số dư âm — mutex không hoạt động.');
    process.exitCode = 1;
  }
}

if (require.main === module) {
  main().catch((err) => {
    console.error('Lỗi:', err.message);
    process.exit(1);
  });
}

module.exports = { fireConcurrent, post, get, main };
