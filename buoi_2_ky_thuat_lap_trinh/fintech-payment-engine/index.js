'use strict';

/**
 * Kịch bản đầu-cuối trên Terminal (không cần mở trình duyệt, không cần k6):
 *
 *   node index.js
 *
 * Khởi tạo ví → nạp tiền → bắn 50 request đồng thời → đối soát Big-O → hoàn tiền bằng Stack.
 * Dùng DB riêng `data/demo-run.db` và xoá trước mỗi lượt để số liệu lặp lại được.
 */
const fs = require('node:fs');
const path = require('node:path');

const { start } = require('./src/server');
const { fireConcurrent, get } = require('./simulate-load');
const { DEMO_WALLET_ID } = require('./data/mockFintechData');

const DB_FILE = path.join(__dirname, 'data', 'demo-run.db');

function resetDbFile(file) {
  for (const suffix of ['', '-wal', '-shm']) {
    try {
      fs.rmSync(`${file}${suffix}`, { force: true });
    } catch {
      /* file không tồn tại */
    }
  }
}

function section(title) {
  console.log(`\n${'─'.repeat(64)}\n${title}\n${'─'.repeat(64)}`);
}

async function main() {
  resetDbFile(DB_FILE);

  const { server, db, port } = await start({ port: 0, dbFile: DB_FILE });
  console.log(`Server demo chạy tại http://127.0.0.1:${port} (DB: data/demo-run.db)`);

  try {
    section('1. Số dư ban đầu (đọc từ SQLite)');
    const before = await get(port, `/api/wallet/balance?walletId=${DEMO_WALLET_ID}`);
    console.log(`Ví ${before.walletId} — ${before.owner}: ${before.balance}đ (nguồn: ${before.source})`);

    section('2. Bắn 50 request rút 100.000đ đồng thời (giả lập K6)');
    const outcome = await fireConcurrent({ port, walletId: DEMO_WALLET_ID, amount: 100000, count: 50 });
    console.table([
      { metric: 'SUCCESS', value: outcome.byCode.SUCCESS || 0 },
      { metric: 'INSUFFICIENT_FUNDS', value: outcome.byCode.INSUFFICIENT_FUNDS || 0 },
      { metric: 'Chênh lệch thời điểm gửi (ms)', value: outcome.spreadMs },
    ]);
    const afterWithdraw = await get(port, `/api/wallet/balance?walletId=${DEMO_WALLET_ID}`);
    console.log(`Số dư sau khi bắn: ${afterWithdraw.balance}đ ${afterWithdraw.balance < 0 ? '— ÂM: SAI!' : '(không âm)'}`);

    section('3. Đối soát O(n²) vs O(n) trên 20.000 bản ghi');
    const optimized = await (await fetch(`http://127.0.0.1:${port}/api/reconcile/optimized`, { method: 'POST' })).json();
    const naive = await (await fetch(`http://127.0.0.1:${port}/api/reconcile/naive`, { method: 'POST' })).json();
    console.table([
      { strategy: 'Optimized (Map)', durationMs: optimized.durationMs, matches: optimized.matches, mismatches: optimized.mismatches, total: optimized.total },
      { strategy: 'Naive (indexOf)', durationMs: naive.durationMs, matches: naive.matches, mismatches: naive.mismatches, total: naive.total },
    ]);
    if (optimized.matches !== naive.matches || optimized.mismatches !== naive.mismatches) {
      throw new Error('Hai thuật toán cho kết quả khác nhau — đối soát sai.');
    }
    console.log(`Cùng kết quả (${optimized.matches} khớp / ${optimized.mismatches} lệch) — chỉ khác thời gian.`);

    section('4. Hoàn tiền bằng Stack (LIFO)');
    const refund = await (await fetch(`http://127.0.0.1:${port}/api/wallet/refund`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ walletId: DEMO_WALLET_ID }),
    })).json();
    console.log(refund);
    const restored = await get(port, `/api/wallet/balance?walletId=${DEMO_WALLET_ID}`);
    console.log(`Số dư sau hoàn tiền: ${restored.balance}đ`);

    section('5. 5 giao dịch gần nhất');
    const history = await get(port, `/api/wallet/transactions?walletId=${DEMO_WALLET_ID}&limit=5`);
    console.table(history.transactions.map((t) => ({
      loại: t.type, số_tiền: t.amount, cổng: t.gateway, trạng_thái: t.status,
    })));

    section('Xong');
    console.log('Mở dashboard: `node src/server.js` rồi vào http://localhost:3000');
  } finally {
    await new Promise((resolve) => server.close(resolve));
    db.close();
  }
}

main().catch((err) => {
  console.error('Lỗi kịch bản:', err.message);
  process.exit(1);
});
