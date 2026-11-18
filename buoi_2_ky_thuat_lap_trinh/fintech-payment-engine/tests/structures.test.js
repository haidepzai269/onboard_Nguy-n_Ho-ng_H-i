'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const { TransactionStack } = require('../src/structures/TransactionStack');
const { PaymentQueue } = require('../src/structures/PaymentQueue');

test('TransactionStack là LIFO', () => {
  const stack = new TransactionStack();
  assert.ok(stack.isEmpty());
  stack.push({ id: 1 }).push({ id: 2 }).push({ id: 3 });

  assert.equal(stack.size, 3);
  assert.deepEqual(stack.peek(), { id: 3 });
  assert.deepEqual(stack.pop(), { id: 3 });
  assert.deepEqual(stack.pop(), { id: 2 });
  assert.deepEqual(stack.pop(), { id: 1 });
  assert.equal(stack.pop(), undefined);
  assert.ok(stack.isEmpty());
});

test('PaymentQueue là FIFO và dequeue đúng thứ tự với 10.000 phần tử', () => {
  const queue = new PaymentQueue();
  const N = 10000;
  for (let i = 0; i < N; i += 1) queue.enqueue(i);

  assert.equal(queue.size, N);
  assert.equal(queue.peek(), 0);

  for (let i = 0; i < N; i += 1) {
    assert.equal(queue.dequeue(), i, `phần tử thứ ${i} phải đúng thứ tự FIFO`);
  }
  assert.equal(queue.dequeue(), undefined);
  assert.ok(queue.isEmpty());
  assert.equal(queue.size, 0);
});

test('PaymentQueue xen kẽ enqueue/dequeue vẫn đúng thứ tự', () => {
  const queue = new PaymentQueue();
  queue.enqueue('a');
  queue.enqueue('b');
  assert.equal(queue.dequeue(), 'a');
  queue.enqueue('c');
  assert.equal(queue.dequeue(), 'b');
  assert.equal(queue.dequeue(), 'c');
  assert.equal(queue.peek(), undefined);
});

function measureDequeue(n) {
  const queue = new PaymentQueue();
  for (let i = 0; i < n; i += 1) queue.enqueue(i);
  const startedAt = performance.now();
  for (let i = 0; i < n; i += 1) queue.dequeue();
  return performance.now() - startedAt;
}

test('dequeue tuyến tính: 500.000 phần tử không chậm hơn 25× 50.000 phần tử', () => {
  measureDequeue(10000); // warmup, tránh JIT làm sai phép so sánh
  const small = measureDequeue(50000);
  const large = measureDequeue(500000);
  // Con trỏ #head cho tỉ lệ ~10×. Nếu dùng shift() của Array thì 500.000 phần tử
  // sẽ tốn gấp hàng trăm lần (mỗi shift dời cả mảng), vượt xa ngưỡng này.
  const ratio = large / Math.max(small, 0.001);
  assert.ok(ratio < 25, `tỉ lệ ${ratio.toFixed(1)}× — dequeue không còn tuyến tính`);
});
