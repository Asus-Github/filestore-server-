/*
 * 增量计算 SHA-1 (纯JS实现)
 * - 支持分片 update, 大文件无需一次性读入内存
 * - 不依赖 crypto.subtle (非 https / localhost 页面中不可用)
 * 用法: var h = new Sha1(); h.update(uint8Array); ...; h.hex();
 */
(function (root) {
  'use strict';

  function Sha1() {
    this._h = new Int32Array([0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476, 0xc3d2e1f0]);
    this._w = new Int32Array(80);
    this._block = new Uint8Array(64);
    this._blockLen = 0;
    this._bytes = 0; // 已处理的总字节数
  }

  Sha1.prototype._compress = function (buf, off) {
    var w = this._w;
    var h = this._h;
    var i, t;
    for (i = 0; i < 16; i++) {
      var j = off + i * 4;
      w[i] = (buf[j] << 24) | (buf[j + 1] << 16) | (buf[j + 2] << 8) | buf[j + 3];
    }
    for (i = 16; i < 80; i++) {
      t = w[i - 3] ^ w[i - 8] ^ w[i - 14] ^ w[i - 16];
      w[i] = (t << 1) | (t >>> 31);
    }
    var a = h[0], b = h[1], c = h[2], d = h[3], e = h[4];
    for (i = 0; i < 80; i++) {
      var f, k;
      if (i < 20) {
        f = (b & c) | (~b & d);
        k = 0x5a827999;
      } else if (i < 40) {
        f = b ^ c ^ d;
        k = 0x6ed9eba1;
      } else if (i < 60) {
        f = (b & c) | (b & d) | (c & d);
        k = 0x8f1bbcdc;
      } else {
        f = b ^ c ^ d;
        k = 0xca62c1d6;
      }
      t = (((a << 5) | (a >>> 27)) + f + e + k + w[i]) | 0;
      e = d;
      d = c;
      c = (b << 30) | (b >>> 2);
      b = a;
      a = t;
    }
    h[0] = (h[0] + a) | 0;
    h[1] = (h[1] + b) | 0;
    h[2] = (h[2] + c) | 0;
    h[3] = (h[3] + d) | 0;
    h[4] = (h[4] + e) | 0;
  };

  /** @param {Uint8Array|ArrayBuffer} data */
  Sha1.prototype.update = function (data) {
    var buf = data instanceof Uint8Array ? data : new Uint8Array(data);
    var len = buf.length;
    var pos = 0;
    this._bytes += len;
    // 先补齐上次剩余的不完整块
    if (this._blockLen > 0) {
      var need = 64 - this._blockLen;
      var n = len < need ? len : need;
      this._block.set(buf.subarray(0, n), this._blockLen);
      this._blockLen += n;
      pos = n;
      if (this._blockLen < 64) {
        return this;
      }
      this._compress(this._block, 0);
      this._blockLen = 0;
    }
    // 直接处理完整的块, 避免复制
    for (; pos + 64 <= len; pos += 64) {
      this._compress(buf, pos);
    }
    if (pos < len) {
      this._block.set(buf.subarray(pos), 0);
      this._blockLen = len - pos;
    }
    return this;
  };

  /** 返回小写hex摘要; 调用后不应再 update */
  Sha1.prototype.hex = function () {
    var bitLenHi = Math.floor(this._bytes / 0x20000000); // bytes*8 的高32位
    var bitLenLo = (this._bytes * 8) >>> 0;
    var padLen = this._blockLen < 56 ? 56 - this._blockLen : 120 - this._blockLen;
    var pad = new Uint8Array(padLen + 8);
    pad[0] = 0x80;
    pad[padLen] = bitLenHi >>> 24;
    pad[padLen + 1] = (bitLenHi >>> 16) & 0xff;
    pad[padLen + 2] = (bitLenHi >>> 8) & 0xff;
    pad[padLen + 3] = bitLenHi & 0xff;
    pad[padLen + 4] = bitLenLo >>> 24;
    pad[padLen + 5] = (bitLenLo >>> 16) & 0xff;
    pad[padLen + 6] = (bitLenLo >>> 8) & 0xff;
    pad[padLen + 7] = bitLenLo & 0xff;
    var bytes = this._bytes;
    this.update(pad);
    this._bytes = bytes;

    var out = '';
    for (var i = 0; i < 5; i++) {
      out += ('00000000' + (this._h[i] >>> 0).toString(16)).slice(-8);
    }
    return out;
  };

  root.Sha1 = Sha1;
  if (typeof module !== 'undefined' && module.exports) {
    module.exports = Sha1;
  }
})(typeof self !== 'undefined' ? self : this);
