'use strict';

const { NotImplementedError } = require('../core/errors');

/** Hợp đồng chung của mọi cổng thanh toán (Abstraction). */
class IPaymentGateway {
  constructor(code) {
    this.code = code;
  }

  // eslint-disable-next-line no-unused-vars
  async charge(amount, options) {
    throw new NotImplementedError('IPaymentGateway.charge');
  }

  // eslint-disable-next-line no-unused-vars
  async refund(reference, amount) {
    throw new NotImplementedError('IPaymentGateway.refund');
  }
}

module.exports = { IPaymentGateway };
