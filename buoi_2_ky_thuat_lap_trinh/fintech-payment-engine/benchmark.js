'use strict';

/**
 * Benchmark CLI: đo đối soát O(n²) vs O(n) trên 20.000 bản ghi THẬT trong SQLite.
 * Không in con số cam kết — mọi số liệu lấy từ máy đang chạy.
 *
 *   node benchmark.js
 */
const { openDb } = require('./src/database/db');
const { seedIfEmpty, assertSeedShape } = require('./data/mockFintechData');
const { ReconciliationService } = require('./src/services/ReconciliationService');

function heapMb() {
  if (global.gc) global.gc();
  return Math.round((process.memoryUsage().heapUsed / 1024 / 1024) * 100) / 100;
}

function run(label, fn, service) {
  const before = heapMb();
  const startedAt = performance.now();
  const result = fn.call(service);
  const wall = Math.round(performance.now() - startedAt);
  const after = heapMb();
  return {
    label,
    durationMs: result.durationMs,
    wallMs: wall,
    heapDeltaMb: Math.round((after - before) * 100) / 100,
    matches: result.matches,
    mismatches: result.mismatchCount,
    total: result.total,
    batchId: result.batchId,
  };
}

function main() {
  const db = openDb();
  const { shape } = seedIfEmpty(db);
  const measured = shape || assertSeedShape(db);
  console.log(`Dữ liệu: ${measured.internal} bản ghi nội bộ, ${measured.bank} dòng sao kê ngân hàng`);
  console.log(`Phân bố: MATCH ${measured.match}, lệch tiền ${measured.mismatch}, thiếu ${measured.missingInBank + measured.missingInternal}\n`);

  const service = new ReconciliationService({ db });
  // optimized trước để phép đo không bị ảnh hưởng bởi cache DB của lần chạy trước.
  const optimized = run('Optimized O(n)   ', service.reconcileOptimized, service);
  const naive = run('Naive O(n²)      ', service.reconcileNaive, service);
  console.table([optimized, naive]);

  const speedup = optimized.durationMs > 0
    ? Math.round((naive.durationMs / optimized.durationMs) * 10) / 10
    : null;
  console.log(`\nTỉ lệ: naive chậm hơn optimized ${speedup}× (đo trên máy này, không phải hằng số).`);
  console.log(`Kết quả đã lưu vào recon_results: batch ${optimized.batchId} và ${naive.batchId}`);
  db.close();
}

main();
