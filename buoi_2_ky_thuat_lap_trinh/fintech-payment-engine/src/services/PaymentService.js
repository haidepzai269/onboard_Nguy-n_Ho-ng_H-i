'use strict';

const { DomainError } = require('../core/errors');
const { getGateway } = require('../gateways/registry');

/**
 * Tầng Service điều phối. Nhận gateway registry qua constructor
 * (Dependency Injection) thay vì tự `new VNPayGateway()` bên trong.
 */
class PaymentService {
  #registry;

  constructor({ registry = getGateway, ...rest } = {}) {
    this.#registry = typeof registry === 'function' ? registry : getGateway;
    Object.assign(this, rest);
  }

  async charge({ wallet, gatewayCode, amount, idempotencyKey = null, token = '' }) {
    const gateway = this.#registry(gatewayCode);
    if (!gateway) {
      throw new DomainError('UNKNOWN_GATEWAY', `Cổng thanh toán không hỗ trợ: ${gatewayCode}`, 400);
    }

    const receipt = await gateway.charge(amount, { idempotencyKey, token });
    const tx = await wallet.debit(amount, { gateway: gateway.code, idempotencyKey });

    return {
      status: 'SUCCESS',
      provider: receipt.provider,
      reference: receipt.reference,
      signature: receipt.signature,
      qrCode: receipt.qrCode || null,
      txId: tx.id,
      balance: wallet.balance,
    };
  }
}

module.exports = { PaymentService };
