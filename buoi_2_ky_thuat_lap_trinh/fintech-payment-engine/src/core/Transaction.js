'use strict';

/**
 * Model giao dịch bất biến. Mọi instance bị Object.freeze:
 * không thể sửa amount/status sau khi tạo — sai lệch dữ liệu tài chính
 * thường bắt đầu từ một object bị mutate ngoài ý muốn.
 */
class Transaction {
  constructor({ id, walletId, txCode, type, amount, gateway = null, status = 'SUCCESS',
    idempotencyKey = null, source = 'LIVE', createdAt = new Date().toISOString() }) {
    Object.assign(this, {
      id, walletId, txCode, type, amount, gateway, status, idempotencyKey, source, createdAt,
    });
    Object.freeze(this);
  }
}

module.exports = { Transaction };
