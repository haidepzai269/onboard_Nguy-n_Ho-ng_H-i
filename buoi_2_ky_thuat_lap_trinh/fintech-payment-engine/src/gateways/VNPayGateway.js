'use strict';

const crypto = require('node:crypto');
const { IPaymentGateway } = require('./IPaymentGateway');

let warned = false;

function secret() {
  if (process.env.VNP_HASH_SECRET) return process.env.VNP_HASH_SECRET;
  if (!warned) {
    warned = true;
    console.warn('[VNPay] VNP_HASH_SECRET chưa được đặt — dùng secret dev (KHÔNG dùng cho thật).');
  }
  return 'dev-only-vnp-hash-secret';
}

/** Ký HMAC-SHA256 đúng kiểu VNPay: txnRef|amount|createdAt. */
class VNPayGateway extends IPaymentGateway {
  constructor() {
    super('VNPAY');
  }

  async charge(amount, { idempotencyKey = '' } = {}) {
    const reference = `VNP-${crypto.randomBytes(5).toString('hex').toUpperCase()}`;
    const data = `${reference}|${amount}|${idempotencyKey}`;
    const signature = crypto.createHmac('sha256', secret()).update(data).digest('hex');
    return { provider: 'VNPAY', reference, signature, qrCode: null };
  }

  async refund(reference, amount) {
    const signature = crypto
      .createHmac('sha256', secret())
      .update(`REFUND|${reference}|${amount}`)
      .digest('hex');
    return { provider: 'VNPAY', reference, signature, refunded: true };
  }
}

module.exports = { VNPayGateway };
