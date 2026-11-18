'use strict';

/**
 * FIFO — hàng đợi xử lý lệnh thanh toán ngầm.
 * Dùng con trỏ `#head` thay cho Array#shift() để dequeue là O(1) amortized
 * (shift() dời toàn bộ mảng → O(n) mỗi lần).
 */
class PaymentQueue {
  #items = [];
  #head = 0;

  enqueue(item) {
    this.#items.push(item);
    return this;
  }

  dequeue() {
    if (this.isEmpty()) return undefined;
    const item = this.#items[this.#head];
    this.#items[this.#head] = undefined; // giải phóng tham chiếu
    this.#head += 1;
    if (this.#head > 64 && this.#head * 2 >= this.#items.length) {
      this.#items = this.#items.slice(this.#head);
      this.#head = 0;
    }
    return item;
  }

  peek() {
    return this.isEmpty() ? undefined : this.#items[this.#head];
  }

  isEmpty() {
    return this.#head >= this.#items.length;
  }

  get size() {
    return this.#items.length - this.#head;
  }
}

module.exports = { PaymentQueue };
