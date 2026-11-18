'use strict';

const { inTransaction, countRows } = require('../src/database/db');

const TOTAL_ROWS = 20000;
const MATCH_ROWS = 18000;      // 90%
const MISMATCH_ROWS = 1000;    // 5%  — hai bên đều có, lệch tiền
const MISSING_IN_BANK = 1000;  // 5%  — chỉ nội bộ có
const MISSING_INTERNAL = 1000; //      — chỉ ngân hàng có (bù lại cho đủ 20.000 bên bank)

const SEED_WALLET_ID = 'WAL-SEED';
const DEMO_WALLET_ID = 'WAL-8892';
const DEMO_WALLET_BALANCE = 100000;

function txCode(n) {
  return `TX-${String(n).padStart(6, '0')}`;
}

function amountFor(n) {
  return 1000 + ((n * 137) % 9000);
}

function isoAt(offsetMinutes) {
  return new Date(Date.UTC(2026, 0, 1, 0, 0, 0) - offsetMinutes * 60000).toISOString();
}

/**
 * Sinh 20.000 giao dịch nội bộ (source='SEED', wallet_id='WAL-SEED') và
 * 20.000 dòng sao kê ngân hàng, với phân bố 90/5/5 cố định (không random)
 * để `assertSeedShape()` đo lại được.
 */
function seedMockData(db) {
  const insertBank = db.prepare(
    'INSERT OR IGNORE INTO bank_statement_logs (id, tx_code, amount, bank_timestamp) VALUES (?, ?, ?, ?)'
  );
  const insertTx = db.prepare(
    `INSERT OR IGNORE INTO transactions
       (id, wallet_id, tx_code, type, amount, gateway, status, idempotency_key, source, created_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
  );

  inTransaction(db, () => {
    // Ví kỹ thuật giữ toàn bộ dòng SEED — tách khỏi ví thật.
    db.prepare(
      'INSERT OR IGNORE INTO wallets (id, owner, balance, created_at) VALUES (?, ?, 0, ?)'
    ).run(SEED_WALLET_ID, 'Seed Dataset', isoAt(0));

    for (let n = 0; n < MATCH_ROWS + MISMATCH_ROWS; n += 1) {
      const code = txCode(n);
      const internalAmount = amountFor(n);
      const isMismatch = n >= MATCH_ROWS;
      const bankAmount = isMismatch ? internalAmount + 7 : internalAmount;

      insertTx.run(
        `SEED-${n}`, SEED_WALLET_ID, code, 'DEPOSIT', internalAmount,
        'INTERNAL', 'SUCCESS', null, 'SEED', isoAt(n)
      );
      insertBank.run(`BANK-${n}`, code, bankAmount, isoAt(n));
    }

    // Nội bộ có, ngân hàng không có.
    for (let n = MATCH_ROWS + MISMATCH_ROWS; n < TOTAL_ROWS; n += 1) {
      insertTx.run(
        `SEED-${n}`, SEED_WALLET_ID, txCode(n), 'DEPOSIT', amountFor(n),
        'INTERNAL', 'SUCCESS', null, 'SEED', isoAt(n)
      );
    }

    // Ngân hàng có, nội bộ không có.
    for (let n = 0; n < MISSING_INTERNAL; n += 1) {
      const code = txCode(TOTAL_ROWS + n);
      insertBank.run(`BANK-X${n}`, code, amountFor(n), isoAt(TOTAL_ROWS + n));
    }
  });

  return assertSeedShape(db);
}

/** Tạo ví demo WAL-8892 nếu bảng wallets còn trống. */
function seedDemoWallet(db) {
  if (countRows(db, 'wallets') > 0) return false;
  db.prepare('INSERT INTO wallets (id, owner, balance, created_at) VALUES (?, ?, ?, ?)').run(
    DEMO_WALLET_ID, 'Demo User', DEMO_WALLET_BALANCE, new Date().toISOString()
  );
  return true;
}

/** Seed idempotent: gọi bao nhiêu lần cũng không nhân đôi dữ liệu. */
function seedIfEmpty(db) {
  const walletCreated = seedDemoWallet(db);
  let shape = null;
  if (countRows(db, 'bank_statement_logs') === 0 && countRows(db, 'transactions') === 0) {
    shape = seedMockData(db);
  }
  return { walletCreated, shape };
}

/** Đếm lại phân bố bằng SQL — con số trong tài liệu phải khớp con số ở đây. */
function assertSeedShape(db, { log = false } = {}) {
  const q = (sql) => db.prepare(sql).get().n;
  const shape = {
    internal: q("SELECT count(*) AS n FROM transactions WHERE source = 'SEED'"),
    bank: q('SELECT count(*) AS n FROM bank_statement_logs'),
    match: q(`
      SELECT count(*) AS n FROM transactions t
      JOIN bank_statement_logs b ON b.tx_code = t.tx_code
      WHERE t.source = 'SEED' AND b.amount = t.amount`),
    mismatch: q(`
      SELECT count(*) AS n FROM transactions t
      JOIN bank_statement_logs b ON b.tx_code = t.tx_code
      WHERE t.source = 'SEED' AND b.amount <> t.amount`),
    missingInBank: q(`
      SELECT count(*) AS n FROM transactions t
      WHERE t.source = 'SEED'
        AND NOT EXISTS (SELECT 1 FROM bank_statement_logs b WHERE b.tx_code = t.tx_code)`),
    missingInternal: q(`
      SELECT count(*) AS n FROM bank_statement_logs b
      WHERE NOT EXISTS (SELECT 1 FROM transactions t
                        WHERE t.tx_code = b.tx_code AND t.source = 'SEED')`),
  };

  shape.consistent =
    shape.internal === TOTAL_ROWS &&
    shape.bank === TOTAL_ROWS &&
    shape.match === MATCH_ROWS &&
    shape.mismatch === MISMATCH_ROWS &&
    shape.missingInBank === MISSING_IN_BANK &&
    shape.missingInternal === MISSING_INTERNAL;

  if (log) {
    console.log('[seed] phân bố dữ liệu đối soát:');
    console.table([
      { metric: 'nội bộ (SEED)', value: shape.internal, expected: TOTAL_ROWS },
      { metric: 'ngân hàng', value: shape.bank, expected: TOTAL_ROWS },
      { metric: 'khớp (MATCH)', value: shape.match, expected: MATCH_ROWS },
      { metric: 'lệch tiền', value: shape.mismatch, expected: MISMATCH_ROWS },
      { metric: 'thiếu phía ngân hàng', value: shape.missingInBank, expected: MISSING_IN_BANK },
      { metric: 'thiếu phía nội bộ', value: shape.missingInternal, expected: MISSING_INTERNAL },
    ]);
  }

  return shape;
}

module.exports = {
  TOTAL_ROWS,
  MATCH_ROWS,
  MISMATCH_ROWS,
  MISSING_IN_BANK,
  MISSING_INTERNAL,
  SEED_WALLET_ID,
  DEMO_WALLET_ID,
  DEMO_WALLET_BALANCE,
  seedMockData,
  seedDemoWallet,
  seedIfEmpty,
  assertSeedShape,
};
