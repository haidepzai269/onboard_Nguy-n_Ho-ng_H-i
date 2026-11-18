'use strict';

/**
 * LIFO — lịch sử giao dịch của ví, dùng cho Undo/Refund.
 * push/pop/peek/isEmpty đều O(1).
 */
class TransactionStack {
  #items;

  constructor(items = []) {
    this.#items = [...items];
  }

  push(item) {
    this.#items.push(item);
    return this;
  }

  pop() {
    return this.#items.pop();
  }

  peek() {
    return this.#items[this.#items.length - 1];
  }

  isEmpty() {
    return this.#items.length === 0;
  }

  get size() {
    return this.#items.length;
  }

  toArray() {
    return [...this.#items];
  }
}

module.exports = { TransactionStack };
