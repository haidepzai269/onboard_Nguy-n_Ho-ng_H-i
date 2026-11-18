'use strict';

/**
 * Chống double-spending bằng Set O(1).
 * Khoá luôn được namespace theo ví: `${walletId}:${key}` — nếu không,
 * một key của ví A sẽ chặn nhầm ví B.
 * Có TTL để Set không phình mãi dưới tải.
 */
class IdempotencyGuard {
  #store = new Map(); // key → expiresAt
  #ttlMs;
  #now;

  constructor({ ttlMs = 15 * 60 * 1000, now = () => Date.now() } = {}) {
    this.#ttlMs = ttlMs;
    this.#now = now;
  }

  #key(walletId, key) {
    return `${walletId}:${key}`;
  }

  #prune(now) {
    for (const [k, expiresAt] of this.#store) {
      if (expiresAt <= now) this.#store.delete(k);
    }
  }

  has(walletId, key) {
    if (!key) return false;
    const entry = this.#store.get(this.#key(walletId, key));
    if (!entry) return false;
    if (entry <= this.#now()) {
      this.#store.delete(this.#key(walletId, key));
      return false;
    }
    return true;
  }

  register(walletId, key) {
    if (!key) return;
    this.#store.set(this.#key(walletId, key), this.#now() + this.#ttlMs);
  }

  /** Kiểm tra + đăng ký trong cùng một tick (không có await ở giữa). */
  checkAndRegister(walletId, key) {
    if (!key) return false;
    if (this.#store.size > 1000) this.#prune(this.#now());
    if (this.has(walletId, key)) return true;
    this.register(walletId, key);
    return false;
  }

  get size() {
    return this.#store.size;
  }
}

module.exports = { IdempotencyGuard };
