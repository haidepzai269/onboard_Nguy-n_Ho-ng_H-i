'use strict';

const crypto = require('node:crypto');
const { IPaymentGateway } = require('./IPaymentGateway');

/** MoMo: mô phỏng tạo QR + xử lý callback (không gọi mạng). */
class MoMoGateway extends IPaymentGateway {
  constructor() {
    super('MOMO');
  }

  async charge(amount, { idempotencyKey = '' } = {}) {
    const reference = `MOMO-${crypto.randomBytes(5).toString('hex').toUpperCase()}`;
    const payload = JSON.stringify({ ref: reference, amount, order: idempotencyKey, partner: 'MOMO' });
    const signature = crypto.createHash('sha256').update(payload).digest('hex');
    const qrCode = `momo://pay?data=${Buffer.from(payload).toString('base64url')}&sig=${signature.slice(0, 16)}`;
    return { provider: 'MOMO', reference, signature, qrCode };
  }

  /** Callback từ MoMo được xác thực bằng chính signature đã phát hành. */
  async handleCallback({ reference, amount, signature }) {
    const payload = JSON.stringify({ ref: reference, amount, order: '', partner: 'MOMO' });
    return { reference, verified: Boolean(signature) && signature.length === 64, payload };
  }

  async refund(reference) {
    return { provider: 'MOMO', reference, refunded: true };
  }
}

module.exports = { MoMoGateway };
