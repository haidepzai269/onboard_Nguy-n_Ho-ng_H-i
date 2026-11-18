'use strict';

const crypto = require('node:crypto');
const { inTransaction } = require('../database/db');

/**
 * BIG-O THỰC CHIẾN: cùng một phép đối soát, hai cách tiếp cận.
 *   - naive:     vòng lặp + find() ẩn  → O(n²)
 *   - optimized: dựng Map theo tx_code → O(n)
 * Cả hai trả CÙNG một cấu trúc kết quả (đã kiểm bằng test) và ghi vào recon_results.
 */
class ReconciliationService {
  #db;

  constructor({ db }) {
    this.#db = db;
  }

  #internalRows() {
    return this.#db
      .prepare("SELECT tx_code, amount FROM transactions WHERE source = 'SEED' ORDER BY tx_code")
      .all();
  }

  #bankRows() {
    return this.#db
      .prepare('SELECT tx_code, amount FROM bank_statement_logs ORDER BY tx_code')
      .all();
  }

  reconcileNaive() {
    const startedAt = performance.now();
    const internal = this.#internalRows().map((r) => ({ code: r.tx_code, amount: r.amount }));
    const bank = this.#bankRows().map((r) => ({ code: r.tx_code, amount: r.amount }));
    const bankCodes = bank.map((b) => b.code);
    const verdicts = [];

    for (const tx of internal) {
      // indexOf() bên trong vòng lặp = tìm tuyến tính cho từng phần tử → O(n²).
      // Đây chính là cái bẫy "lookup O(n) trong vòng lặp" cần thấy bằng mắt.
      const idx = bankCodes.indexOf(tx.code);
      if (idx === -1) {
        verdicts.push({ txCode: tx.code, verdict: 'MISSING_IN_BANK', internalAmount: tx.amount, bankAmount: null });
      } else if (bank[idx].amount === tx.amount) {
        verdicts.push({ txCode: tx.code, verdict: 'MATCH', internalAmount: tx.amount, bankAmount: bank[idx].amount });
      } else {
        verdicts.push({ txCode: tx.code, verdict: 'AMOUNT_MISMATCH', internalAmount: tx.amount, bankAmount: bank[idx].amount });
      }
    }

    const internalCodes = new Set(internal.map((t) => t.code));
    for (const b of bank) {
      if (!internalCodes.has(b.code)) {
        verdicts.push({ txCode: b.code, verdict: 'MISSING_INTERNAL', internalAmount: null, bankAmount: b.amount });
      }
    }

    return this.#finish(verdicts, 'NAIVE', startedAt);
  }

  reconcileOptimized() {
    const startedAt = performance.now();
    const internal = this.#internalRows().map((r) => ({ code: r.tx_code, amount: r.amount }));
    const bank = this.#bankRows().map((r) => ({ code: r.tx_code, amount: r.amount }));

    // Space-Time trade-off: bỏ O(n) bộ nhớ để đổi lấy tra cứu O(1).
    const bankIndex = new Map();
    for (const b of bank) bankIndex.set(b.code, b);

    const verdicts = [];
    for (const tx of internal) {
      const found = bankIndex.get(tx.code);
      if (!found) {
        verdicts.push({ txCode: tx.code, verdict: 'MISSING_IN_BANK', internalAmount: tx.amount, bankAmount: null });
      } else if (found.amount === tx.amount) {
        verdicts.push({ txCode: tx.code, verdict: 'MATCH', internalAmount: tx.amount, bankAmount: found.amount });
      } else {
        verdicts.push({ txCode: tx.code, verdict: 'AMOUNT_MISMATCH', internalAmount: tx.amount, bankAmount: found.amount });
      }
    }

    const internalCodes = new Set(internal.map((t) => t.code));
    for (const b of bank) {
      if (!internalCodes.has(b.code)) {
        verdicts.push({ txCode: b.code, verdict: 'MISSING_INTERNAL', internalAmount: null, bankAmount: b.amount });
      }
    }

    return this.#finish(verdicts, 'OPTIMIZED', startedAt);
  }

  #finish(verdicts, strategy, startedAt) {
    const durationMs = Math.round(performance.now() - startedAt);
    const batchId = crypto.randomUUID();
    const matches = verdicts.reduce((n, v) => n + (v.verdict === 'MATCH' ? 1 : 0), 0);
    const createdAt = new Date().toISOString();

    const insert = this.#db.prepare(
      `INSERT INTO recon_results
         (batch_id, strategy, tx_code, verdict, internal_amount, bank_amount, duration_ms, created_at)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
    );
    inTransaction(this.#db, () => {
      for (const v of verdicts) {
        insert.run(batchId, strategy, v.txCode, v.verdict, v.internalAmount, v.bankAmount, durationMs, createdAt);
      }
    });

    return {
      batchId,
      strategy,
      durationMs,
      total: verdicts.length,
      matches,
      mismatchCount: verdicts.length - matches,
      mismatches: verdicts.filter((v) => v.verdict !== 'MATCH'),
    };
  }
}

module.exports = { ReconciliationService };
