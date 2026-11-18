'use strict';

const { DatabaseSync } = require('node:sqlite');
const path = require('node:path');
const fs = require('node:fs');

/**
 * Schema khớp SPEC §5.
 * - balance/amount là INTEGER đồng (VND không có phần lẻ).
 * - transactions.tx_code là KHOÁ ĐỐI SOÁT với bank_statement_logs.tx_code.
 * - transactions.source phân biệt dữ liệu mock ('SEED') với thao tác thật ('LIVE').
 */
const SCHEMA = `
CREATE TABLE IF NOT EXISTS wallets (
    id TEXT PRIMARY KEY,
    owner TEXT NOT NULL,
    balance INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    wallet_id TEXT NOT NULL,
    tx_code TEXT NOT NULL,
    type TEXT NOT NULL,
    amount INTEGER NOT NULL,
    gateway TEXT,
    status TEXT NOT NULL,
    idempotency_key TEXT,
    source TEXT NOT NULL DEFAULT 'LIVE',
    created_at TEXT NOT NULL,
    FOREIGN KEY(wallet_id) REFERENCES wallets(id)
);

CREATE INDEX IF NOT EXISTS idx_tx_code ON transactions(tx_code);
CREATE INDEX IF NOT EXISTS idx_wallet_created ON transactions(wallet_id, created_at);

CREATE TABLE IF NOT EXISTS bank_statement_logs (
    id TEXT PRIMARY KEY,
    tx_code TEXT NOT NULL UNIQUE,
    amount INTEGER NOT NULL,
    bank_timestamp TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS recon_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    batch_id TEXT NOT NULL,
    strategy TEXT NOT NULL,
    tx_code TEXT NOT NULL,
    verdict TEXT NOT NULL,
    internal_amount INTEGER,
    bank_amount INTEGER,
    duration_ms INTEGER NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_recon_batch ON recon_results(batch_id);
`;

const DEFAULT_DB = path.join(__dirname, '..', '..', 'data', 'fintech.db');

/**
 * Mở DB + bật PRAGMA + tạo schema. `file` có thể là ':memory:' cho test.
 */
function openDb(file = process.env.FINTECH_DB || DEFAULT_DB) {
  if (file !== ':memory:') {
    fs.mkdirSync(path.dirname(file), { recursive: true });
  }
  const db = new DatabaseSync(file);
  db.exec('PRAGMA journal_mode = WAL;');
  db.exec('PRAGMA busy_timeout = 5000;');
  initSchema(db);
  return db;
}

function initSchema(db) {
  db.exec(SCHEMA);
  return db;
}

/** Chạy fn trong một transaction; lỗi thì rollback. */
function inTransaction(db, fn) {
  db.exec('BEGIN');
  try {
    const result = fn();
    db.exec('COMMIT');
    return result;
  } catch (err) {
    db.exec('ROLLBACK');
    throw err;
  }
}

function countRows(db, table) {
  return db.prepare(`SELECT count(*) AS n FROM ${table}`).get().n;
}

module.exports = { openDb, initSchema, inTransaction, countRows, DEFAULT_DB };
