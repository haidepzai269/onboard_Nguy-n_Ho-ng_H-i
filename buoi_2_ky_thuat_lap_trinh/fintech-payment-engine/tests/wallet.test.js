'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const { openDb } = require('../src/database/db');
const { seedDemoWallet, DEMO_WALLET_ID, DEMO_WALLET_BALANCE } = require('../data/mockFintechData');
const { Wallet } = require('../src/core/Wallet');
const { Transaction } = require('../src/core/Transaction');

function freshWallet() {
  const db = openDb(':memory:');
  seedDemoWallet(db);
  return { db, wallet: Wallet.load(db, DEMO_WALLET_ID) };
}

test('đóng gói: số dư chỉ đọc, private field không lộ ra ngoài', () => {
  const { db, wallet } = freshWallet();
  assert.equal(wallet.balance, DEMO_WALLET_BALANCE);
  assert.throws(() => { wallet.balance = 1; }, TypeError, 'getter-only phải ném TypeError khi gán');
  assert.equal(wallet['#balance'], undefined, 'private field không truy cập được bằng tên');
  assert.equal(wallet.balance, DEMO_WALLET_BALANCE, 'số dư không đổi sau khi gán thất bại');
  db.close();
});

test('Transaction là bất biến', () => {
  const tx = new Transaction({
    id: 't1', walletId: 'w1', txCode: 'TX-1', type: 'DEPOSIT',
    amount: 1000, gateway: 'INTERNAL', idempotencyKey: null, createdAt: new Date().toISOString(),
  });
  assert.ok(Object.isFrozen(tx));
  assert.throws(() => { 'use strict'; tx.amount = 1; }, TypeError);
  assert.equal(tx.amount, 1000);
});

test('validate amount: chỉ số nguyên dương', async () => {
  const { db, wallet } = freshWallet();
  for (const bad of [0, -1, 1.5, '100', null, undefined, NaN, Infinity]) {
    await assert.rejects(() => wallet.deposit(bad), (err) => err.code === 'INVALID_AMOUNT', `deposit(${String(bad)})`);
    await assert.rejects(() => wallet.debit(bad), (err) => err.code === 'INVALID_AMOUNT', `debit(${String(bad)})`);
  }
  assert.equal(wallet.balance, DEMO_WALLET_BALANCE, 'không có giao dịch nào bị ghi');
  db.close();
});

test('50 request đồng thời trên cùng ví: 1 SUCCESS, 49 INSUFFICIENT_FUNDS, số dư 0', async () => {
  const { db, wallet } = freshWallet();
  const results = await Promise.allSettled(Array.from({ length: 50 }, () => wallet.debit(100000)));

  const ok = results.filter((r) => r.status === 'fulfilled').length;
  const insufficient = results.filter((r) => r.status === 'rejected' && r.reason.code === 'INSUFFICIENT_FUNDS').length;

  assert.equal(ok, 1, 'chỉ đúng một request được trừ tiền');
  assert.equal(insufficient, 49, '49 request còn lại bị chặn vì thiếu tiền');
  assert.equal(wallet.balance, 0);
  assert.ok(wallet.balance >= 0, 'bất biến: không bao giờ số dư âm');

  const dbBalance = db.prepare('SELECT balance FROM wallets WHERE id = ?').get(DEMO_WALLET_ID).balance;
  assert.equal(dbBalance, 0, 'DB là nguồn sự thật và cũng bằng 0');
  const liveRows = db.prepare("SELECT count(*) AS n FROM transactions WHERE source = 'LIVE'").get().n;
  assert.equal(liveRows, 1, 'chỉ một dòng giao dịch được ghi');
  db.close();
});

test('nhiều đợt đồng thời liên tiếp vẫn giữ bất biến không âm', async () => {
  const { db, wallet } = freshWallet();
  await wallet.deposit(50000);
  const batches = await Promise.all(
    Array.from({ length: 4 }, () =>
      Promise.allSettled(Array.from({ length: 25 }, () => wallet.debit(10000)))
    )
  );
  const ok = batches.flat().filter((r) => r.status === 'fulfilled').length;
  assert.equal(ok, 15, '100.000 + 50.000 = 150.000 → đúng 15 lần rút 10.000');
  assert.equal(wallet.balance, 0);
  db.close();
});

test('hoàn tiền bằng Stack: trả đúng giao dịch gần nhất, ghi DB, không cộng đôi sau reload', async () => {
  const { db, wallet } = freshWallet();
  await wallet.debit(30000);
  assert.equal(wallet.balance, 70000);

  const refund = await wallet.refundLastTransaction();
  assert.equal(refund.amount, 30000);
  assert.equal(wallet.balance, 100000);

  const row = db.prepare("SELECT status FROM transactions WHERE id = ?").get(refund.refundedTxId);
  assert.equal(row.status, 'REFUNDED', 'giao dịch gốc được đánh dấu đã hoàn');

  const reloaded = Wallet.load(db, DEMO_WALLET_ID);
  assert.equal(reloaded.balance, 100000, 'nạp lại từ DB vẫn đúng, không cộng đôi');

  const refundedTxStillInStack = (await reloaded.recentTransactions(5)).some(
    (t) => t.id === refund.refundedTxId && t.status === 'SUCCESS'
  );
  assert.equal(refundedTxStillInStack, false);
  db.close();
});

test('ví không có giao dịch → NOTHING_TO_REFUND', async () => {
  const { db } = freshWallet();
  db.prepare('INSERT INTO wallets (id, owner, balance, created_at) VALUES (?, ?, ?, ?)')
    .run('W-EMPTY', 'Empty', 0, new Date().toISOString());
  const wallet = Wallet.load(db, 'W-EMPTY');
  await assert.rejects(() => wallet.refundLastTransaction(), (err) => err.code === 'NOTHING_TO_REFUND');
  db.close();
});

test('ví không tồn tại → WALLET_NOT_FOUND', () => {
  const db = openDb(':memory:');
  assert.throws(() => Wallet.load(db, 'KHONG-CO'), (err) => err.code === 'WALLET_NOT_FOUND' && err.status === 404);
  db.close();
});
