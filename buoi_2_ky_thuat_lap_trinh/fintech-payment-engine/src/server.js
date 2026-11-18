'use strict';

const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');

const { openDb } = require('./database/db');
const { seedIfEmpty } = require('../data/mockFintechData');
const { Wallet } = require('./core/Wallet');
const { PaymentService } = require('./services/PaymentService');
const { ReconciliationService } = require('./services/ReconciliationService');
const { IdempotencyGuard } = require('./security/IdempotencyGuard');
const { DomainError } = require('./core/errors');
const { listGateways } = require('./gateways/registry');

const PUBLIC_DIR = path.join(__dirname, '..', 'public');
const MAX_BODY_BYTES = 64 * 1024;
const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
};

function sendJson(res, status, payload) {
  const body = JSON.stringify(payload);
  res.writeHead(status, {
    'content-type': 'application/json; charset=utf-8',
    'content-length': Buffer.byteLength(body),
    'cache-control': 'no-store',
  });
  res.end(body);
}

function sendError(res, status, code, message) {
  sendJson(res, status, { error: { code, message } });
}

function readJsonBody(req) {
  return new Promise((resolve, reject) => {
    let size = 0;
    const chunks = [];
    req.on('data', (chunk) => {
      size += chunk.length;
      if (size > MAX_BODY_BYTES) {
        reject(new DomainError('PAYLOAD_TOO_LARGE', 'Body vượt quá 64KB', 413));
        req.destroy();
        return;
      }
      chunks.push(chunk);
    });
    req.on('end', () => {
      const raw = Buffer.concat(chunks).toString('utf8').trim();
      if (!raw) return resolve({});
      try {
        return resolve(JSON.parse(raw));
      } catch {
        return reject(new DomainError('INVALID_JSON', 'Body không phải JSON hợp lệ', 400));
      }
    });
    req.on('error', reject);
  });
}

function serveStatic(pathname, res) {
  const relative = pathname === '/' ? '/index.html' : pathname;
  const resolved = path.resolve(PUBLIC_DIR, `.${relative}`);
  if (!resolved.startsWith(PUBLIC_DIR + path.sep)) {
    return sendError(res, 400, 'BAD_PATH', 'Đường dẫn không hợp lệ');
  }
  let stat;
  try {
    stat = fs.statSync(resolved);
  } catch {
    return sendError(res, 404, 'NOT_FOUND', 'Không tìm thấy tài nguyên');
  }
  if (!stat.isFile()) {
    return sendError(res, 404, 'NOT_FOUND', 'Không tìm thấy tài nguyên');
  }
  const ext = path.extname(resolved).toLowerCase();
  const type = MIME[ext];
  if (!type) {
    return sendError(res, 400, 'BAD_PATH', `Không phục vụ loại file ${ext}`);
  }
  res.writeHead(200, { 'content-type': type, 'cache-control': 'no-store' });
  fs.createReadStream(resolved).pipe(res);
  return undefined;
}

/**
 * Tạo app: mở DB → seed → load ví → trả handler HTTP.
 * `start()` chỉ là phần listen; nhờ vậy index.js và test dùng lại được.
 */
function createApp({ db, guard = new IdempotencyGuard() } = {}) {
  if (!db) throw new Error('createApp cần một kết nối db');

  const seeded = seedIfEmpty(db);
  if (seeded.shape && !seeded.shape.consistent) {
    console.warn('[seed] phân bố dữ liệu KHÔNG khớp kỳ vọng 90/5/5');
  }

  const payments = new PaymentService();
  const reconciliation = new ReconciliationService({ db });
  const walletCache = new Map();
  let naiveRunning = false;

  function getWallet(walletId) {
    if (!walletId) throw new DomainError('MISSING_WALLET_ID', 'Thiếu walletId', 400);
    if (!walletCache.has(walletId)) {
      walletCache.set(walletId, Wallet.load(db, walletId));
    }
    return walletCache.get(walletId);
  }

  async function handle(req, res) {
    const method = req.method || 'GET';
    // Chặn traversal trên URL thô (trước khi URL parser chuẩn hoá '../').
    if (req.url.includes('..')) {
      return sendError(res, 400, 'BAD_PATH', 'Đường dẫn không hợp lệ');
    }
    const url = new URL(req.url, `http://${req.headers.host || 'localhost'}`);
    const route = `${method} ${url.pathname}`;

    try {
      if (route === 'GET /api/health') {
        return sendJson(res, 200, { ok: true, gateways: listGateways() });
      }

      if (route === 'GET /api/wallet/balance') {
        const walletId = url.searchParams.get('walletId');
        if (!walletId) throw new DomainError('MISSING_WALLET_ID', 'Thiếu tham số walletId', 400);
        const row = db.prepare('SELECT id, owner, balance FROM wallets WHERE id = ?').get(walletId);
        if (!row) throw new DomainError('WALLET_NOT_FOUND', `Không tìm thấy ví ${walletId}`, 404);
        return sendJson(res, 200, { walletId: row.id, owner: row.owner, balance: row.balance, source: 'db' });
      }

      if (route === 'GET /api/wallet/transactions') {
        const wallet = getWallet(url.searchParams.get('walletId'));
        const limit = url.searchParams.get('limit') || 10;
        return sendJson(res, 200, { walletId: wallet.id, transactions: await wallet.recentTransactions(limit) });
      }

      if (route === 'POST /api/wallet/withdraw') {
        const body = await readJsonBody(req);
        if (!Number.isInteger(body.amount) || body.amount <= 0) {
          throw new DomainError('INVALID_AMOUNT', 'amount phải là số nguyên dương', 400);
        }
        const wallet = getWallet(body.walletId);
        if (guard.checkAndRegister(wallet.id, body.idempotencyKey)) {
          throw new DomainError('DUPLICATE', 'idempotencyKey đã được dùng cho ví này', 409);
        }
        const tx = await wallet.debit(body.amount, {
          gateway: 'INTERNAL',
          idempotencyKey: body.idempotencyKey || null,
        });
        return sendJson(res, 200, { status: 'SUCCESS', balance: wallet.balance, txId: tx.id, txCode: tx.txCode });
      }

      if (route === 'POST /api/payment/charge') {
        const body = await readJsonBody(req);
        if (!Number.isInteger(body.amount) || body.amount <= 0) {
          throw new DomainError('INVALID_AMOUNT', 'amount phải là số nguyên dương', 400);
        }
        const wallet = getWallet(body.walletId);
        if (guard.checkAndRegister(wallet.id, body.idempotencyKey)) {
          throw new DomainError('DUPLICATE', 'idempotencyKey đã được dùng cho ví này', 409);
        }
        const result = await payments.charge({
          wallet,
          gatewayCode: body.gateway,
          amount: body.amount,
          idempotencyKey: body.idempotencyKey || null,
          token: body.token,
        });
        return sendJson(res, 200, result);
      }

      if (route === 'POST /api/wallet/refund') {
        const body = await readJsonBody(req);
        const wallet = getWallet(body.walletId);
        const result = await wallet.refundLastTransaction();
        return sendJson(res, 200, result);
      }

      if (route === 'POST /api/reconcile/naive' || route === 'POST /api/reconcile/optimized') {
        if (naiveRunning) {
          throw new DomainError('RECON_IN_PROGRESS', 'Đang có một lượt đối soát chạy — thử lại sau', 409);
        }
        naiveRunning = true;
        try {
          // Nhường event loop một nhịp để request thứ hai kịp thấy cờ "đang chạy"
          // (naive là code đồng bộ, sẽ chặn loop ngay sau đây).
          await new Promise((resolve) => setImmediate(resolve));
          const result = route.endsWith('naive')
            ? reconciliation.reconcileNaive()
            : reconciliation.reconcileOptimized();
          return sendJson(res, 200, {
            batchId: result.batchId,
            strategy: result.strategy,
            durationMs: result.durationMs,
            total: result.total,
            matches: result.matches,
            mismatches: result.mismatchCount,
            samples: result.mismatches.slice(0, 50),
          });
        } finally {
          naiveRunning = false;
        }
      }

      if (method === 'GET' && !url.pathname.startsWith('/api/')) {
        return serveStatic(url.pathname, res);
      }

      return sendError(res, 404, 'NOT_FOUND', `Không có route ${route}`);
    } catch (err) {
      if (err instanceof DomainError) {
        return sendError(res, err.status, err.code, err.message);
      }
      console.error('[server] lỗi không mong đợi:', err);
      return sendError(res, 500, 'INTERNAL', 'Lỗi hệ thống');
    }
  }

  return { handle, db, walletCache, reconciliation, guard, seeded };
}

function start({ port = Number(process.env.PORT) || 3000, dbFile } = {}) {
  const db = openDb(dbFile);
  const app = createApp({ db });
  const server = http.createServer((req, res) => {
    app.handle(req, res).catch((err) => {
      console.error('[server] handler crash:', err);
      if (!res.headersSent) sendError(res, 500, 'INTERNAL', 'Lỗi hệ thống');
    });
  });

  return new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, () => {
      resolve({ server, db, app, port: server.address().port });
    });
  });
}

if (require.main === module) {
  start()
    .then(({ port, app }) => {
      const shape = app.seeded.shape;
      console.log(`FinTech engine: http://localhost:${port}`);
      if (shape) console.log(`Seed: ${shape.internal} nội bộ / ${shape.bank} ngân hàng (nhất quán: ${shape.consistent})`);
    })
    .catch((err) => {
      console.error('Không khởi động được server:', err);
      process.exit(1);
    });
}

module.exports = { createApp, start, PUBLIC_DIR };
