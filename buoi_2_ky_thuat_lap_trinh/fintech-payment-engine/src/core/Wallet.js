'use strict';

const crypto = require('node:crypto');
const { DomainError } = require('./errors');
const { Transaction } = require('./Transaction');
const { TransactionStack } = require('../structures/TransactionStack');
const { inTransaction } = require('../database/db');

const MAX_HISTORY = 1000;

function newTxCode() {
  return `TX-${crypto.randomBytes(6).toString('hex').toUpperCase()}`;
}

function assertAmount(amount) {
  if (!Number.isInteger(amount) || amount <= 0) {
    throw new DomainError('INVALID_AMOUNT', 'amount phải là số nguyên dương (đơn vị đồng)', 400);
  }
}

/**
 * Ví tiền: đóng gói tuyệt đối (private fields), mọi thao tác đi qua một
 * promise-chain mutex duy nhất. DB là nguồn sự thật — mỗi lần đổi số dư
 * đều ghi xuống SQLite trong cùng luồng xử lý, RAM chỉ là cache đọc.
 */
class Wallet {
  #db;
  #id;
  #owner;
  #balance;
  #queue;
  #history;

  constructor({ db, id, owner, balance = 0 }) {
    this.#db = db;
    this.#id = id;
    this.#owner = owner;
    this.#balance = balance;
    this.#queue = Promise.resolve();
    this.#history = new TransactionStack();
  }

  static load(db, id) {
    const row = db.prepare('SELECT id, owner, balance FROM wallets WHERE id = ?').get(id);
    if (!row) {
      throw new DomainError('WALLET_NOT_FOUND', `Không tìm thấy ví ${id}`, 404);
    }
    const wallet = new Wallet({ db, id: row.id, owner: row.owner, balance: row.balance });
    const rows = db
      .prepare(
        `SELECT id, tx_code, type, amount, status
           FROM transactions
          WHERE wallet_id = ? AND source = 'LIVE'
            AND type IN ('DEPOSIT', 'WITHDRAW') AND status = 'SUCCESS'
          ORDER BY rowid DESC
          LIMIT ?`
      )
      .all(id, MAX_HISTORY);
    // rows mới nhất trước → đẩy ngược lại để đỉnh Stack là giao dịch gần nhất.
    for (let i = rows.length - 1; i >= 0; i -= 1) {
      const r = rows[i];
      wallet.#history.push({ id: r.id, txCode: r.tx_code, type: r.type, amount: r.amount });
    }
    return wallet;
  }

  get id() {
    return this.#id;
  }

  get owner() {
    return this.#owner;
  }

  get balance() {
    return this.#balance;
  }

  get pendingOperations() {
    return this.#history.size;
  }

  /** Nối tiếp mọi thao tác trên ví — đây là "khoá" thật, không phải cờ boolean. */
  #enqueue(task) {
    const result = this.#queue.then(() => task());
    this.#queue = result.then(
      () => undefined,
      () => undefined
    );
    return result;
  }

  #insertTransaction({ id, txCode, type, amount, gateway, idempotencyKey, status = 'SUCCESS' }, newBalance) {
    inTransaction(this.#db, () => {
      this.#db
        .prepare(
          `INSERT INTO transactions
             (id, wallet_id, tx_code, type, amount, gateway, status, idempotency_key, source, created_at)
           VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'LIVE', ?)`
        )
        .run(id, this.#id, txCode, type, amount, gateway, status, idempotencyKey, new Date().toISOString());
      this.#db.prepare('UPDATE wallets SET balance = ? WHERE id = ?').run(newBalance, this.#id);
    });
  }

  async deposit(amount) {
    assertAmount(amount);
    return this.#enqueue(async () => {
      const balance = this.#balance + amount;
      const tx = new Transaction({
        id: crypto.randomUUID(),
        walletId: this.#id,
        txCode: newTxCode(),
        type: 'DEPOSIT',
        amount,
        gateway: 'INTERNAL',
        idempotencyKey: null,
        createdAt: new Date().toISOString(),
      });
      this.#insertTransaction({ id: tx.id, txCode: tx.txCode, type: tx.type, amount, gateway: tx.gateway, idempotencyKey: null }, balance);
      this.#balance = balance;
      this.#history.push({ id: tx.id, txCode: tx.txCode, type: tx.type, amount });
      return tx;
    });
  }

  /**
   * Trừ tiền — dùng chung cho withdraw (gateway INTERNAL) và charge (VNPAY/MOMO/STRIPE).
   * `await` bên trong vùng tới hạn là cố ý: nếu ai đó bỏ mutex, race condition
   * sẽ tái hiện ngay (đọc số dư → nhường event loop → ghi).
   */
  async debit(amount, { gateway = 'INTERNAL', idempotencyKey = null, type = 'WITHDRAW' } = {}) {
    assertAmount(amount);
    return this.#enqueue(async () => {
      const current = this.#balance;
      await Promise.resolve(); // điểm nhường event loop
      if (amount > current) {
        throw new DomainError('INSUFFICIENT_FUNDS', `Số dư ${current} không đủ để trừ ${amount}`, 409);
      }
      const balance = current - amount;
      const tx = new Transaction({
        id: crypto.randomUUID(),
        walletId: this.#id,
        txCode: newTxCode(),
        type,
        amount,
        gateway,
        idempotencyKey,
        createdAt: new Date().toISOString(),
      });
      this.#insertTransaction({ id: tx.id, txCode: tx.txCode, type, amount, gateway, idempotencyKey }, balance);
      this.#balance = balance;
      this.#history.push({ id: tx.id, txCode: tx.txCode, type, amount });
      return tx;
    });
  }

  /** Undo giao dịch gần nhất bằng Stack: pop O(1), hoàn tiền, ghi REFUND xuống DB. */
  async refundLastTransaction() {
    return this.#enqueue(async () => {
      const last = this.#history.pop();
      if (!last) {
        throw new DomainError('NOTHING_TO_REFUND', 'Không có giao dịch nào để hoàn', 409);
      }
      const delta = last.type === 'WITHDRAW' ? last.amount : -last.amount;
      const balance = this.#balance + delta;
      if (balance < 0) {
        throw new DomainError('INSUFFICIENT_FUNDS', 'Không thể hoàn: số dư sẽ âm', 409);
      }

      const refund = new Transaction({
        id: crypto.randomUUID(),
        walletId: this.#id,
        txCode: newTxCode(),
        type: 'REFUND',
        amount: last.amount,
        gateway: 'INTERNAL',
        createdAt: new Date().toISOString(),
      });

      inTransaction(this.#db, () => {
        this.#db
          .prepare(
            `INSERT INTO transactions
               (id, wallet_id, tx_code, type, amount, gateway, status, idempotency_key, source, created_at)
             VALUES (?, ?, ?, 'REFUND', ?, 'INTERNAL', 'SUCCESS', NULL, 'LIVE', ?)`
          )
          .run(refund.id, this.#id, refund.txCode, last.amount, refund.createdAt);
        this.#db
          .prepare("UPDATE transactions SET status = 'REFUNDED' WHERE id = ?")
          .run(last.id);
        this.#db.prepare('UPDATE wallets SET balance = ? WHERE id = ?').run(balance, this.#id);
      });

      this.#balance = balance;
      return { status: 'REFUNDED', refundedTxId: last.id, amount: last.amount, balance };
    });
  }

  async recentTransactions(limit = 10) {
    const safe = Math.min(Math.max(Number(limit) || 10, 1), 100);
    return this.#db
      .prepare(
        `SELECT id, tx_code, type, amount, gateway, status, created_at
           FROM transactions
          WHERE wallet_id = ? AND source = 'LIVE'
          ORDER BY rowid DESC
          LIMIT ?`
      )
      .all(this.#id, safe)
      .map((r) => ({
        id: r.id,
        txCode: r.tx_code,
        type: r.type,
        amount: r.amount,
        gateway: r.gateway,
        status: r.status,
        createdAt: r.created_at,
      }));
  }
}

module.exports = { Wallet, MAX_HISTORY };
