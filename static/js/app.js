/*
 * 文件云盘 - 前端公共模块(会话管理 / 接口请求 / 通用UI组件)
 * 不依赖任何第三方库
 */
(function (window) {
  'use strict';

  var document = window.document;
  var STORAGE_KEYS = ['token', 'username', 'expireAt', 'uploadEntry', 'downloadEntry'];

  // 与后端 common/code.go 保持一致
  var Code = {
    OK: 10000,
    PARAM_INVALID: 10001,
    SERVER_ERROR: 10002,
    REGISTER_FAILED: 10003,
    LOGIN_FAILED: 10004,
    TOKEN_INVALID: 10005,
    USER_NOT_EXISTS: 10006,
    USER_EXISTS: 10007,
    FILE_NOT_EXISTS: 10008,
    FAST_UPLOAD_MISS: 10009,
    UPLOAD_FAILED: 10010,
    CHUNK_INCOMPLETE: 10011,
    FILE_VERIFY_FAILED: 10012,
    FORBIDDEN: 10013,
    URL_EXPIRED: 10014,
    FILE_TOO_LARGE: 10015
  };

  // ---------------------------------------------------------------- 会话
  var Session = {
    get: function () {
      var s = {};
      try {
        STORAGE_KEYS.forEach(function (k) {
          s[k] = window.localStorage.getItem(k) || '';
        });
      } catch (e) {
        STORAGE_KEYS.forEach(function (k) { s[k] = ''; });
      }
      return s;
    },
    save: function (data) {
      try {
        window.localStorage.setItem('token', data.Token || '');
        window.localStorage.setItem('username', data.Username || '');
        window.localStorage.setItem('expireAt', String(data.ExpireAt || ''));
        Session.saveEntries(data);
      } catch (e) {
        throw new Error('浏览器禁用了本地存储, 无法保存登录状态');
      }
    },
    saveEntries: function (data) {
      try {
        if (data.UploadEntry) window.localStorage.setItem('uploadEntry', data.UploadEntry);
        if (data.DownloadEntry) window.localStorage.setItem('downloadEntry', data.DownloadEntry);
      } catch (e) { /* ignore */ }
    },
    clear: function () {
      try {
        STORAGE_KEYS.forEach(function (k) { window.localStorage.removeItem(k); });
      } catch (e) { /* ignore */ }
    },
    // 是否存在未过期的登录状态(最终以服务端校验为准)
    isActive: function () {
      var s = Session.get();
      if (!/^[0-9a-f]{64}$/.test(s.token) || !s.username) return false;
      var exp = parseInt(s.expireAt, 10);
      return !(exp > 0 && exp * 1000 <= Date.now());
    }
  };

  // 服务入口: 优先使用服务端下发的地址, 否则按当前访问的主机名拼接默认端口
  function entry(kind) {
    var s = Session.get();
    var stored = kind === 'upload' ? s.uploadEntry : s.downloadEntry;
    var port = kind === 'upload' ? 28080 : 38080;
    var base = stored || (window.location.protocol + '//' + window.location.hostname + ':' + port);
    if (!/^https?:\/\//i.test(base)) base = 'http://' + base;
    return base.replace(/\/+$/, '');
  }

  // 跳转到登录页, 登录后回到当前页面
  function toSignin(reason) {
    Session.clear();
    var target = window.top !== window ? window.top : window;
    var back = target.location.pathname + target.location.search;
    var url = '/static/view/signin.html?redirect=' + encodeURIComponent(back);
    if (reason) url += '&reason=' + encodeURIComponent(reason);
    target.location.replace(url);
  }

  // 页面初始化时调用: 未登录则跳转登录页
  function requireLogin() {
    if (!Session.isActive()) {
      toSignin('expired');
      return false;
    }
    return true;
  }

  // ---------------------------------------------------------------- 请求
  function ApiError(message, code, status) {
    this.name = 'ApiError';
    this.message = message;
    this.code = code;
    this.status = status;
  }
  ApiError.prototype = Object.create(Error.prototype);
  ApiError.prototype.constructor = ApiError;

  function toForm(data) {
    var form = new URLSearchParams();
    Object.keys(data || {}).forEach(function (k) {
      if (data[k] !== undefined && data[k] !== null) form.append(k, String(data[k]));
    });
    return form;
  }

  function authHeaders(extra) {
    var h = extra || {};
    var token = Session.get().token;
    if (token) h.Authorization = 'Bearer ' + token;
    return h;
  }

  /**
   * 发送请求并解析统一格式的响应 {code, msg, data}
   * - code !== 10000 时 reject(ApiError)
   * - token失效时自动跳转登录页
   */
  function request(url, opts) {
    opts = opts || {};
    var init = {
      method: opts.method || 'POST',
      headers: opts.auth === false ? {} : authHeaders(),
      signal: opts.signal
    };
    if (opts.data) {
      init.body = toForm(opts.data);
    }
    return window.fetch(url, init).then(function (resp) {
      return resp.text().then(function (text) {
        var body = null;
        try {
          body = text ? JSON.parse(text) : null;
        } catch (e) {
          body = null;
        }
        if (!body || typeof body.code !== 'number') {
          throw new ApiError(resp.ok ? '服务返回了无法识别的数据' : '请求失败 (HTTP ' + resp.status + ')', 0, resp.status);
        }
        if (body.code === Code.TOKEN_INVALID && opts.auth !== false) {
          toSignin('expired');
          throw new ApiError(body.msg || '登录已失效, 请重新登录', body.code, resp.status);
        }
        if (body.code !== Code.OK) {
          throw new ApiError(body.msg || '操作失败', body.code, resp.status);
        }
        return body;
      });
    }, function (err) {
      if (err && err.name === 'AbortError') throw err;
      throw new ApiError('网络错误, 无法连接到服务器', 0, 0);
    });
  }

  function api(path, data, opts) {
    opts = opts || {};
    opts.data = data;
    return request(path, opts);
  }

  // ---------------------------------------------------------------- 格式化
  function formatSize(bytes) {
    var n = Number(bytes);
    if (!isFinite(n) || n < 0) return '-';
    if (n < 1024) return n + ' B';
    var units = ['KB', 'MB', 'GB', 'TB'];
    var i = -1;
    do {
      n /= 1024;
      i++;
    } while (n >= 1024 && i < units.length - 1);
    return (n >= 100 ? n.toFixed(0) : n >= 10 ? n.toFixed(1) : n.toFixed(2)) + ' ' + units[i];
  }

  function formatSpeed(bytesPerSec) {
    return formatSize(Math.max(0, Math.round(bytesPerSec))) + '/s';
  }

  function formatDuration(sec) {
    if (!isFinite(sec) || sec < 0) return '--';
    sec = Math.round(sec);
    if (sec < 60) return sec + '秒';
    if (sec < 3600) return Math.floor(sec / 60) + '分' + (sec % 60) + '秒';
    return Math.floor(sec / 3600) + '小时' + Math.floor((sec % 3600) / 60) + '分';
  }

  // ---------------------------------------------------------------- DOM
  /**
   * 创建元素: el('button', {class: 'btn', onclick: fn}, ['文本', childNode])
   * 文本一律作为textNode插入, 避免XSS
   */
  function el(tag, attrs, children) {
    var node = document.createElement(tag);
    Object.keys(attrs || {}).forEach(function (k) {
      var v = attrs[k];
      if (v === undefined || v === null || v === false) return;
      if (k === 'class') node.className = v;
      else if (k === 'text') node.textContent = v;
      else if (k.indexOf('on') === 0 && typeof v === 'function') node.addEventListener(k.slice(2), v);
      else if (k === 'dataset') Object.keys(v).forEach(function (d) { node.dataset[d] = v[d]; });
      else if (v === true) node.setAttribute(k, '');
      else node.setAttribute(k, String(v));
    });
    [].concat(children || []).forEach(function (c) {
      if (c === undefined || c === null || c === false) return;
      node.appendChild(typeof c === 'string' || typeof c === 'number' ? document.createTextNode(String(c)) : c);
    });
    return node;
  }

  function $(sel, root) {
    return (root || document).querySelector(sel);
  }

  function setBusy(btn, busy, busyText) {
    if (!btn) return;
    if (busy) {
      if (!btn.dataset.text) btn.dataset.text = btn.textContent;
      btn.disabled = true;
      btn.textContent = '';
      btn.appendChild(el('span', { class: 'spinner', 'aria-hidden': 'true' }));
      btn.appendChild(document.createTextNode(' ' + (busyText || btn.dataset.text)));
    } else {
      btn.disabled = false;
      if (btn.dataset.text) btn.textContent = btn.dataset.text;
      delete btn.dataset.text;
    }
  }

  // ---------------------------------------------------------------- 提示
  function toast(message, type, duration) {
    var box = $('.toasts');
    if (!box) {
      box = el('div', { class: 'toasts', role: 'status', 'aria-live': 'polite' });
      document.body.appendChild(box);
    }
    var t = el('div', { class: 'toast ' + (type || '') , text: message });
    box.appendChild(t);
    window.setTimeout(function () {
      if (t.parentNode) t.parentNode.removeChild(t);
    }, duration || (type === 'error' ? 5000 : 3000));
    return t;
  }

  // 模态框, 返回 Promise; 按Esc或点击遮罩关闭(视为取消)
  function modal(opts) {
    return new Promise(function (resolve) {
      var lastFocus = document.activeElement;
      var mask = el('div', { class: 'modal-mask' });
      var titleId = 'modal-title-' + Date.now();
      var box = el('div', {
        class: 'modal' + (opts.wide ? ' wide' : ''),
        role: 'dialog',
        'aria-modal': 'true',
        'aria-labelledby': titleId
      });
      var closed = false;

      function close(value) {
        if (closed) return;
        closed = true;
        document.removeEventListener('keydown', onKey, true);
        if (mask.parentNode) mask.parentNode.removeChild(mask);
        if (opts.onClose) opts.onClose();
        if (lastFocus && lastFocus.focus) lastFocus.focus();
        resolve(value);
      }

      function onKey(e) {
        if (e.key === 'Escape') {
          e.preventDefault();
          close(opts.cancelValue);
        } else if (e.key === 'Tab') {
          // 焦点限制在对话框内
          var focusable = box.querySelectorAll('button, input, iframe, [tabindex]:not([tabindex="-1"])');
          if (!focusable.length) return;
          var first = focusable[0];
          var last = focusable[focusable.length - 1];
          if (e.shiftKey && document.activeElement === first) {
            e.preventDefault();
            last.focus();
          } else if (!e.shiftKey && document.activeElement === last) {
            e.preventDefault();
            first.focus();
          }
        }
      }

      var header = el('div', { class: 'modal-header' }, [
        el('h3', { id: titleId, text: opts.title || '提示' }),
        el('button', { class: 'modal-close', type: 'button', 'aria-label': '关闭', text: '×', onclick: function () { close(opts.cancelValue); } })
      ]);
      var body = el('div', { class: 'modal-body' }, opts.body);
      box.appendChild(header);
      box.appendChild(body);
      if (opts.buttons && opts.buttons.length) {
        var footer = el('div', { class: 'modal-footer' });
        opts.buttons.forEach(function (b) {
          footer.appendChild(el('button', {
            class: 'btn ' + (b.class || ''),
            type: 'button',
            text: b.text,
            onclick: function () {
              var v = typeof b.value === 'function' ? b.value() : b.value;
              if (v === undefined && b.value) return; // 校验未通过, 保持打开
              close(v);
            }
          }));
        });
        box.appendChild(footer);
      }
      mask.appendChild(box);
      mask.addEventListener('mousedown', function (e) {
        if (e.target === mask) close(opts.cancelValue);
      });
      document.addEventListener('keydown', onKey, true);
      document.body.appendChild(mask);
      // 焦点在iframe内时键盘事件不会冒泡到父页面, 需单独监听(仅同源iframe)
      Array.prototype.forEach.call(box.querySelectorAll('iframe'), function (frame) {
        frame.addEventListener('load', function () {
          try {
            frame.contentWindow.document.addEventListener('keydown', function (e) {
              if (e.key === 'Escape') {
                e.preventDefault();
                close(opts.cancelValue);
              }
            });
          } catch (err) { /* 跨域iframe无法监听 */ }
        });
      });
      if (opts.onOpen) opts.onOpen(box, close);
      var focusTarget = box.querySelector(opts.focus || '.modal-footer .btn-primary, .modal-footer .btn-danger, .modal-footer .btn');
      if (focusTarget) focusTarget.focus();
    });
  }

  function confirmDialog(title, message, okText, danger) {
    return modal({
      title: title,
      body: [el('p', { style: 'margin:0', text: message })],
      cancelValue: false,
      buttons: [
        { text: '取消', value: false },
        { text: okText || '确定', class: danger ? 'btn-danger solid' : 'btn-primary', value: true }
      ]
    });
  }

  /**
   * 输入对话框; validate(value) 返回错误信息字符串或空
   * 返回 Promise<string|null>, 取消时为null
   */
  function promptDialog(title, label, value, validate) {
    var input = el('input', { class: 'input', type: 'text', id: 'prompt-input', autocomplete: 'off' });
    input.value = value || '';
    var err = el('div', { class: 'error', role: 'alert' });
    function submit() {
      var v = input.value.trim();
      var msg = validate ? validate(v) : '';
      if (msg) {
        err.textContent = msg;
        input.setAttribute('aria-invalid', 'true');
        input.focus();
        return undefined;
      }
      return v;
    }
    return modal({
      title: title,
      cancelValue: null,
      focus: '#prompt-input',
      body: [el('div', { class: 'field', style: 'margin:0' }, [
        el('label', { for: 'prompt-input', text: label }), input, err
      ])],
      buttons: [
        { text: '取消', value: null },
        { text: '确定', class: 'btn-primary', value: submit }
      ],
      onOpen: function (box, close) {
        input.addEventListener('input', function () {
          err.textContent = '';
          input.removeAttribute('aria-invalid');
        });
        input.addEventListener('keydown', function (e) {
          if (e.key === 'Enter' && !e.isComposing) {
            e.preventDefault();
            var v = submit();
            if (v !== undefined) close(v);
          }
        });
        // 选中文件名主体(不含扩展名), 便于直接修改
        window.setTimeout(function () {
          var dot = input.value.lastIndexOf('.');
          input.focus();
          input.setSelectionRange(0, dot > 0 ? dot : input.value.length);
        }, 0);
      }
    });
  }

  // ---------------------------------------------------------------- 校验(与后端规则保持一致)
  var Validate = {
    username: function (v) {
      if (!v) return '请输入用户名';
      if (!/^[\p{L}\p{N}_.\-]{3,32}$/u.test(v)) return '用户名需为3~32位字母、数字、下划线、点或中划线';
      return '';
    },
    password: function (v) {
      if (!v) return '请输入密码';
      var n = Array.from(v).length;
      if (n < 6 || n > 64 || new TextEncoder().encode(v).length > 72) return '密码长度需为6~64位';
      return '';
    },
    fileName: function (v) {
      if (!v) return '文件名不能为空';
      if (v === '.' || v === '..') return '文件名无效';
      if (/[\/\\]/.test(v)) return '文件名不能包含 / 或 \\';
      // eslint-disable-next-line no-control-regex
      if (/[\u0000-\u001f\u007f-\u009f]/.test(v)) return '文件名不能包含控制字符';
      if (Array.from(v).length > 255) return '文件名不能超过255个字符';
      return '';
    }
  };

  // 仅允许站内跳转, 防止开放重定向
  function safeRedirect(target, fallback) {
    if (typeof target === 'string' && /^\/(?!\/)/.test(target) && target.indexOf('\\') < 0) {
      return target;
    }
    return fallback;
  }

  function queryParam(name) {
    return new URLSearchParams(window.location.search).get(name);
  }

  // 触发浏览器下载
  function triggerDownload(url, filename) {
    var a = el('a', { href: url, rel: 'noopener', style: 'display:none' });
    if (filename) a.setAttribute('download', filename);
    document.body.appendChild(a);
    a.click();
    window.setTimeout(function () {
      if (a.parentNode) a.parentNode.removeChild(a);
    }, 1000);
  }

  function signout() {
    var done = function () {
      Session.clear();
      window.location.replace('/static/view/signin.html');
    };
    return api('/user/signout').then(done, done);
  }

  window.App = {
    Code: Code,
    Session: Session,
    ApiError: ApiError,
    entry: entry,
    requireLogin: requireLogin,
    toSignin: toSignin,
    request: request,
    api: api,
    authHeaders: authHeaders,
    formatSize: formatSize,
    formatSpeed: formatSpeed,
    formatDuration: formatDuration,
    el: el,
    $: $,
    setBusy: setBusy,
    toast: toast,
    modal: modal,
    confirm: confirmDialog,
    prompt: promptDialog,
    Validate: Validate,
    safeRedirect: safeRedirect,
    queryParam: queryParam,
    triggerDownload: triggerDownload,
    signout: signout
  };
})(window);
