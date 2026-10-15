/*
 * 断点下载: 通过 Range 请求分段下载文件, 支持暂停/继续/失败重试, 完成后校验sha1并保存
 * 页面参数: filehash(必填), embedded=1(嵌入弹窗时使用)
 */
(function (App) {
  'use strict';

  if (!App.requireLogin()) return;

  var SEGMENT = 4 * 1024 * 1024;
  var MAX_RETRIES = 5;
  // 超过该大小时不在浏览器中计算sha1(避免占用过多时间), 仅校验大小
  var VERIFY_MAX = 512 * 1024 * 1024;
  // 分段下载的数据暂存于内存, 限制最大文件大小
  var MAX_IN_MEMORY = 2 * 1024 * 1024 * 1024;

  var filehash = (App.queryParam('filehash') || '').toLowerCase();
  if (App.queryParam('embedded') === '1') App.$('#dl-page').classList.add('embedded');
  else App.$('#dl-back').hidden = false;

  var ui = {
    name: App.$('#dl-name'),
    hash: App.$('#dl-hash'),
    progress: App.$('#dl-progress'),
    bar: App.$('#dl-bar'),
    status: App.$('#dl-status'),
    detail: App.$('#dl-detail'),
    toggle: App.$('#dl-toggle'),
    save: App.$('#dl-save')
  };

  var st = {
    url: '',
    urlExpireAt: 0,
    filename: 'download',
    total: -1,
    parts: [],
    loaded: 0,
    paused: false,
    running: false,
    done: false,
    controller: null,
    blob: null,
    speed: { at: 0, bytes: 0, value: 0 }
  };

  function setStatus(text, cls) {
    ui.status.textContent = text;
    ui.progress.className = 'progress lg ' + (cls || '');
  }

  function render() {
    var percent = st.total > 0 ? st.loaded / st.total * 100 : 0;
    if (st.total === 0) percent = 100;
    ui.bar.style.width = percent.toFixed(1) + '%';
    ui.progress.setAttribute('aria-valuenow', String(Math.round(percent)));
    var text = App.formatSize(st.loaded) + ' / ' + (st.total >= 0 ? App.formatSize(st.total) : '?') +
      ' · ' + percent.toFixed(1) + '%';
    if (st.running && st.speed.value > 0) {
      text += ' · ' + App.formatSpeed(st.speed.value);
      if (st.total > 0) text += ' · 剩余 ' + App.formatDuration((st.total - st.loaded) / st.speed.value);
    }
    ui.detail.textContent = text;
  }

  function trackSpeed() {
    var now = Date.now();
    if (!st.speed.at) {
      st.speed = { at: now, bytes: st.loaded, value: 0 };
    } else if (now - st.speed.at >= 500) {
      var inst = (st.loaded - st.speed.bytes) / ((now - st.speed.at) / 1000);
      st.speed.value = st.speed.value ? st.speed.value * 0.6 + inst * 0.4 : inst;
      st.speed.at = now;
      st.speed.bytes = st.loaded;
    }
  }

  // 获取(或刷新)签名下载链接
  function refreshURL() {
    return App.api(App.entry('download') + '/file/downloadurl', { filehash: filehash }).then(function (body) {
      var d = body.data || {};
      st.url = d.URL;
      st.urlExpireAt = Number(d.ExpireAt) || 0;
      st.filename = d.FileName || st.filename;
      if (st.total < 0 && typeof d.FileSize === 'number') st.total = d.FileSize;
      ui.name.textContent = st.filename;
      ui.name.title = st.filename;
      document.title = '断点下载 ' + st.filename + ' - 文件云盘';
    });
  }

  function ensureURL() {
    // 链接即将过期时提前刷新
    if (st.url && (!st.urlExpireAt || st.urlExpireAt * 1000 - Date.now() > 60 * 1000)) return Promise.resolve();
    return refreshURL();
  }

  // 解析 Content-Range: bytes start-end/total
  function parseContentRange(h) {
    var m = /bytes\s+(\d+)-(\d+)\/(\d+|\*)/i.exec(h || '');
    if (!m) return null;
    return { start: +m[1], end: +m[2], total: m[3] === '*' ? -1 : +m[3] };
  }

  function HttpError(message, status, retryable, refresh) {
    this.name = 'HttpError';
    this.message = message;
    this.status = status;
    this.retryable = retryable;
    this.refresh = refresh;
  }
  HttpError.prototype = Object.create(Error.prototype);

  // 下载一个分段 [start, end]
  function fetchSegment(start, end) {
    st.controller = window.AbortController ? new window.AbortController() : null;
    return window.fetch(st.url, {
      headers: { Range: 'bytes=' + start + '-' + end },
      signal: st.controller ? st.controller.signal : undefined,
      cache: 'no-store'
    }).then(function (resp) {
      if (resp.status === 206) {
        var cr = parseContentRange(resp.headers.get('Content-Range'));
        if (!cr || cr.start !== start) throw new HttpError('服务端返回的数据范围不正确', 206, true);
        if (cr.total >= 0) st.total = cr.total;
        return resp.arrayBuffer().then(function (buf) {
          if (buf.byteLength !== cr.end - cr.start + 1) throw new HttpError('分段数据不完整', 206, true);
          return buf;
        });
      }
      if (resp.status === 200) {
        // 服务端不支持Range: 返回的是完整文件, 只能整体使用
        return resp.arrayBuffer().then(function (buf) {
          if (start !== 0) throw new HttpError('服务端不支持断点续传', 200, false);
          st.total = buf.byteLength;
          return buf;
        });
      }
      if (resp.status === 416) {
        throw new HttpError('请求范围无效', 416, false);
      }
      return resp.text().then(function (text) {
        var msg = '下载失败 (HTTP ' + resp.status + ')';
        var code = 0;
        try {
          var j = JSON.parse(text);
          if (j && j.msg) msg = j.msg;
          code = j && j.code;
        } catch (e) { /* ignore */ }
        if (code === App.Code.TOKEN_INVALID) App.toSignin('expired');
        // 链接过期/无效时刷新链接后重试
        var refresh = resp.status === 403 || code === App.Code.URL_EXPIRED;
        throw new HttpError(msg, resp.status, refresh || resp.status >= 500, refresh);
      });
    });
  }

  function sleep(ms) {
    return new Promise(function (r) { window.setTimeout(r, ms); });
  }

  async function loop() {
    if (st.running || st.done) return;
    st.running = true;
    st.speed = { at: 0, bytes: 0, value: 0 };
    setStatus('下载中');
    ui.toggle.textContent = '暂停';
    ui.toggle.disabled = false;
    var retries = 0;
    try {
      while (!st.paused && (st.total < 0 || st.loaded < st.total)) {
        await ensureURL();
        if (st.paused) break;
        var end = st.total > 0 ? Math.min(st.loaded + SEGMENT, st.total) - 1 : st.loaded + SEGMENT - 1;
        if (st.total === 0) break;
        try {
          var buf = await fetchSegment(st.loaded, end);
          st.parts.push(new Uint8Array(buf));
          st.loaded += buf.byteLength;
          retries = 0;
          trackSpeed();
          render();
        } catch (err) {
          if (err && err.name === 'AbortError') break; // 暂停
          if (!err.retryable || retries >= MAX_RETRIES) throw err;
          retries++;
          if (err.refresh) st.url = '';
          setStatus('网络异常, ' + retries + '/' + MAX_RETRIES + ' 次重试中...', 'paused');
          await sleep(1000 * retries);
          if (!st.paused) setStatus('下载中');
        }
      }
    } catch (err) {
      st.running = false;
      setStatus('下载失败: ' + (err.message || err), 'error');
      ui.toggle.textContent = '重试';
      render();
      return;
    }
    st.running = false;
    st.controller = null;
    render();
    if (st.paused) {
      setStatus('已暂停, 已下载 ' + App.formatSize(st.loaded), 'paused');
      ui.toggle.textContent = '继续';
      return;
    }
    await finish();
  }

  async function finish() {
    st.done = true;
    ui.toggle.disabled = true;
    st.blob = new Blob(st.parts, { type: 'application/octet-stream' });
    st.parts = [];
    if (st.blob.size !== st.total) {
      st.done = false;
      st.loaded = 0;
      st.blob = null;
      setStatus('下载的文件大小不一致, 请重新下载', 'error');
      ui.toggle.disabled = false;
      ui.toggle.textContent = '重新下载';
      return;
    }
    if (st.blob.size <= VERIFY_MAX) {
      setStatus('正在校验文件完整性...');
      var hasher = new window.Sha1();
      var CHUNK = 4 * 1024 * 1024;
      for (var off = 0; off < st.blob.size; off += CHUNK) {
        hasher.update(new Uint8Array(await st.blob.slice(off, off + CHUNK).arrayBuffer()));
        await sleep(0);
      }
      if (hasher.hex() !== filehash) {
        st.done = false;
        st.loaded = 0;
        st.blob = null;
        setStatus('文件校验失败(内容不一致), 请重新下载', 'error');
        ui.toggle.disabled = false;
        ui.toggle.textContent = '重新下载';
        return;
      }
    }
    setStatus('下载完成, 文件校验通过', 'success');
    ui.toggle.textContent = '已完成';
    ui.save.hidden = false;
    saveBlob();
  }

  function saveBlob() {
    if (!st.blob) return;
    var url = URL.createObjectURL(st.blob);
    App.triggerDownload(url, st.filename);
    window.setTimeout(function () { URL.revokeObjectURL(url); }, 60 * 1000);
  }

  ui.toggle.addEventListener('click', function () {
    if (st.done) return;
    if (st.running) {
      st.paused = true;
      if (st.controller) st.controller.abort();
      return;
    }
    if (st.loaded === 0) st.parts = [];
    st.paused = false;
    loop();
  });
  ui.save.addEventListener('click', saveBlob);

  window.addEventListener('beforeunload', function (e) {
    if (st.running) {
      e.preventDefault();
      e.returnValue = '文件正在下载, 离开页面将中断下载';
      return e.returnValue;
    }
  });

  if (!/^[0-9a-f]{40}$/.test(filehash)) {
    ui.name.textContent = '参数错误';
    setStatus('缺少有效的文件hash', 'error');
    return;
  }
  ui.hash.textContent = 'SHA1: ' + filehash;
  refreshURL().then(function () {
    render();
    // 分段数据保存在内存中, 超大文件请使用普通下载(浏览器原生下载同样支持断点续传)
    if (st.total > MAX_IN_MEMORY) {
      setStatus('文件超过 ' + App.formatSize(MAX_IN_MEMORY) + ', 请使用文件列表中的“下载”按钮', 'error');
      ui.toggle.disabled = true;
      return;
    }
    loop();
  }).catch(function (err) {
    ui.name.textContent = '无法下载';
    setStatus('获取文件信息失败: ' + err.message, 'error');
  });
})(window.App);
