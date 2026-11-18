'use strict';

const { VNPayGateway } = require('./VNPayGateway');
const { MoMoGateway } = require('./MoMoGateway');
const { StripeGateway } = require('./StripeGateway');

/**
 * Nơi DUY NHẤT ánh xạ chuỗi mã cổng → instance.
 * Nhờ nó, server không cần một chuỗi if/else nào để chọn cổng.
 */
const gatewayRegistry = new Map();

function register(gateway) {
  gatewayRegistry.set(gateway.code.toUpperCase(), gateway);
  return gateway;
}

register(new VNPayGateway());
register(new MoMoGateway());
register(new StripeGateway());

function getGateway(code) {
  if (!code) return undefined;
  return gatewayRegistry.get(String(code).toUpperCase());
}

function listGateways() {
  return [...gatewayRegistry.keys()];
}

module.exports = { gatewayRegistry, register, getGateway, listGateways };
