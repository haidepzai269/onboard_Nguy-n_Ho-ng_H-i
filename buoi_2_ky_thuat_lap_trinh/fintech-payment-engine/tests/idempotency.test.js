'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const { IdempotencyGuard } = require('../src/security/IdempotencyGuard');

test('checkAndRegister: lần đầu false, lần sau true', () => {
  const guard = new IdempotencyGuard();
  assert.equal(guard.checkAndRegister('W1', 'KEY-A'), false);
  assert.equal(guard.checkAndRegister('W1', 'KEY-A'), true);
  assert.equal(guard.has('W1', 'KEY-A'), true);
});

test('namespace theo ví: cùng key ở hai ví khác nhau không chặn nhau', () => {
  const guard = new IdempotencyGuard();
  assert.equal(guard.checkAndRegister('W1', 'SAME'), false);
  assert.equal(guard.checkAndRegister('W2', 'SAME'), false, 'ví khác phải được xử lý bình thường');
  assert.equal(guard.checkAndRegister('W1', 'SAME'), true);
});

test('không có key → không chặn và không ghi vào store', () => {
  const guard = new IdempotencyGuard();
  assert.equal(guard.checkAndRegister('W1', null), false);
  assert.equal(guard.checkAndRegister('W1', null), false);
  assert.equal(guard.checkAndRegister('W1', ''), false);
  assert.equal(guard.size, 0);
});

test('TTL inject: hết hạn thì cho phép lại, không phải chờ 15 phút thật', () => {
  let now = 1_000_000;
  const guard = new IdempotencyGuard({ ttlMs: 60_000, now: () => now });

  assert.equal(guard.checkAndRegister('W1', 'K'), false);
  now += 59_999;
  assert.equal(guard.checkAndRegister('W1', 'K'), true, 'còn 1ms thì vẫn bị chặn');

  now += 1;
  assert.equal(guard.checkAndRegister('W1', 'K'), false, 'hết TTL thì key được dùng lại');
  assert.equal(guard.size, 1, 'entry mới thay entry cũ');
});

test('dọn key hết hạn định kỳ để store không phình', () => {
  let now = 0;
  const guard = new IdempotencyGuard({ ttlMs: 1000, now: () => now });

  for (let i = 0; i < 2000; i += 1) {
    guard.checkAndRegister('W1', `K-${i}`);
    now += 10; // mỗi key sống 100 vòng
  }
  assert.equal(guard.checkAndRegister('W1', 'KEY-MOI'), false, 'key mới vẫn được chấp nhận');
  assert.ok(guard.size < 500, `store phải được dọn key hết hạn, hiện tại ${guard.size}`);
});
