'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const { openDb } = require('../src/database/db');
const { ReconciliationService } = require('../src/services/ReconciliationService');

function fixture() {
  const db = openDb(':memory:');
  db.prepare('INSERT INTO wallets (id, owner, balance, created_at) VALUES (?, ?, 0, ?)')
    .run('WAL-SEED', 'Seed', new Date().toISOString());

  const tx = db.prepare(
    `INSERT INTO transactions (id, wallet_id, tx_code, type, amount, gateway, status, idempotency_key, source, created_at)
     VALUES (?, 'WAL-SEED', ?, 'DEPOSIT', ?, 'INTERNAL', 'SUCCESS', NULL, 'SEED', ?)`
  );
  const bank = db.prepare('INSERT INTO bank_statement_logs (id, tx_code, amount, bank_timestamp) VALUES (?, ?, ?, ?)');
  const now = new Date().toISOString();

  tx.run('i1', 'TX-A', 100, now);
  tx.run('i2', 'TX-B', 200, now);
  tx.run('i3', 'TX-C', 300, now);
  bank.run('b1', 'TX-A', 100, now);
  bank.run('b2', 'TX-B', 250, now);
  bank.run('b4', 'TX-D', 400, now);

  return db;
}

test('naive và optimized cho cùng kết quả trên cùng snapshot', () => {
  const db = fixture();
  const service = new ReconciliationService({ db });

  const naive = service.reconcileNaive();
  const optimized = service.reconcileOptimized();

  assert.equal(naive.strategy, 'NAIVE');
  assert.equal(optimized.strategy, 'OPTIMIZED');
  assert.deepEqual(optimized.mismatches, naive.mismatches);
  assert.equal(optimized.matches, naive.matches);
  assert.equal(optimized.total, naive.total);
  db.close();
});

test('phân loại đúng 4 loại verdict', () => {
  const db = fixture();
  const { mismatches, matches, total } = new ReconciliationService({ db }).reconcileOptimized();

  const byCode = Object.fromEntries(mismatches.map((m) => [m.txCode, m.verdict]));
  assert.equal(matches, 1);
  assert.equal(total, 4);
  assert.equal(byCode['TX-B'], 'AMOUNT_MISMATCH');
  assert.equal(byCode['TX-C'], 'MISSING_IN_BANK');
  assert.equal(byCode['TX-D'], 'MISSING_INTERNAL');

  const mismatchRow = mismatches.find((m) => m.txCode === 'TX-B');
  assert.equal(mismatchRow.internalAmount, 200);
  assert.equal(mismatchRow.bankAmount, 250);
  db.close();
});

test('không đọc dữ liệu LIVE vào phép đo', () => {
  const db = fixture();
  db.prepare(
    `INSERT INTO transactions (id, wallet_id, tx_code, type, amount, gateway, status, idempotency_key, source, created_at)
     VALUES ('live1', 'WAL-SEED', 'TX-LIVE', 'DEPOSIT', 999999, 'INTERNAL', 'SUCCESS', NULL, 'LIVE', ?)`
  ).run(new Date().toISOString());

  const result = new ReconciliationService({ db }).reconcileOptimized();
  assert.equal(result.total, 4, 'dòng LIVE không được tính vào đối soát');
  assert.ok(!result.mismatches.some((m) => m.txCode === 'TX-LIVE'));
  db.close();
});

test('kết quả được ghi vào recon_results, mỗi batch đúng số dòng', () => {
  const db = fixture();
  const service = new ReconciliationService({ db });
  const optimized = service.reconcileOptimized();
  const naive = service.reconcileNaive();

  const rows = db.prepare('SELECT strategy, verdict FROM recon_results WHERE batch_id = ?').all(optimized.batchId);
  assert.equal(rows.length, 4);
  assert.ok(rows.every((r) => r.strategy === 'OPTIMIZED'));
  assert.equal(rows.filter((r) => r.verdict === 'MATCH').length, 1);

  const naiveRows = db.prepare('SELECT count(*) AS n FROM recon_results WHERE batch_id = ?').get(naive.batchId).n;
  assert.equal(naiveRows, 4);
  assert.notEqual(optimized.batchId, naive.batchId, 'mỗi lượt một batchId riêng');
  db.close();
});
