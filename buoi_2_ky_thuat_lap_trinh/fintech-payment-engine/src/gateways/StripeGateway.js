'use strict';

const crypto = require('node:crypto');
const { IPaymentGateway } = require('./IPaymentGateway');
const { DomainError } = require('../core/errors');

const TOKEN_RE = /^tok_[A-Za-z0-9]{8,}$/;

/** Stripe: token thẻ quốc tế, chỉ chấp nhận token hợp lệ dạng tok_xxxxxxxx. */
class StripeGateway extends IPaymentGateway {
  constructor() {
    super('STRIPE');
  }

  async charge(amount, { idempotencyKey = '', token = '' } = {}) {
    if (!TOKEN_RE.test(String(token))) {
      throw new DomainError('INVALID_TOKEN', 'Thẻ không hợp lệ: cần token dạng tok_xxxxxxxx', 400);
    }
    const reference = `pi_${crypto.randomBytes(8).toString('hex')}`;
    const signature = crypto
      .createHash('sha256')
      .update(`${reference}|${amount}|${idempotencyKey}`)
      .digest('hex');
    return { provider: 'STRIPE', reference, signature, qrCode: null };
  }

  async refund(reference) {
    return { provider: 'STRIPE', reference, refunded: true };
  }
}

module.exports = { StripeGateway };
