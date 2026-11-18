'use strict';

/** Lỗi nghiệp vụ có mã ổn định — server map thẳng ra envelope {error:{code,message}}. */
class DomainError extends Error {
  constructor(code, message, status = 400) {
    super(message);
    this.name = 'DomainError';
    this.code = code;
    this.status = status;
  }
}

class NotImplementedError extends Error {
  constructor(method) {
    super(`${method}() chưa được cài đặt ở lớp con`);
    this.name = 'NotImplementedError';
    this.code = 'NOT_IMPLEMENTED';
    this.status = 500;
  }
}

module.exports = { DomainError, NotImplementedError };
