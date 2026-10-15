(function (App) {
  'use strict';

  if (!App.requireLogin()) return;

  var el = App.el;
  // 不超过该大小的文件使用普通上传, 更大的文件使用分块上传(支持断点续传)
  var NORMAL_UPLOAD_MAX = 16 * 1024 * 1024;
  var PART_CONCURRENCY = 3;
  var PART_RETRIES = 3;
  var MAX_ACTIVE_TASKS = 2;
  var ACTIVE_STATES = ['hashing', 'checking', 'uploading', 'merging'];

  var tasks = [];
  var nextId = 1;
  var listBox = App.$('#tasks');
  var fileInput = App.$('#file');
  var dropzone = App.$('#dropzone');

  App.$('#top-username').textContent = App.Session.get().username;
  App.$('#signout-btn').addEventListener('click', function () {
    var btn = this;
    var ask = hasActiveTasks()
      ? App.confirm('退出登录', '还有文件正在上传, 退出登录将中断上传, 确定退出吗?', '退出', true)
      : Promise.resolve(true);
    ask.then(function (ok) {
      if (!ok) return;
      tasks.forEach(function (t) { abortTask(t, 'cancel'); });
      App.setBusy(btn, true, '退出中...');
      App.signout();
    });
  });

  // ------------------------------------------------------------ 选择文件
  fileInput.addEventListener('change', function () {
    addFiles(fileInput.files);
    fileInput.value = ''; // 允许再次选择同一文件
  });
  dropzone.addEventListener('keydown', function (e) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      fileInput.click();
    }
  });
  ['dragenter', 'dragover'].forEach(function (type) {
    dropzone.addEventListener(type, function (e) {
      e.preventDefault();
      dropzone.classList.add('dragover');
    });
  });
  ['dragleave', 'drop'].forEach(function (type) {
    dropzone.addEventListener(type, function (e) {
      e.preventDefault();
      dropzone.classList.remove('dragover');
    });
  });
  dropzone.addEventListener('drop', function (e) {
    if (e.dataTransfer && e.dataTransfer.files) addFiles(e.dataTransfer.files);
  });
  // 拖拽到页面其他位置时不要让浏览器直接打开文件
  window.addEventListener('dragover', function (e) { e.preventDefault(); });
  window.addEventListener('drop', function (e) { e.preventDefault(); });

  window.addEventListener('beforeunload', function (e) {
    if (hasActiveTasks() && App.Session.isActive()) {
      e.preventDefault();
      e.returnValue = '文件正在上传, 离开页面将中断上传';
      return e.returnValue;
    }
  });

  function hasActiveTasks() {
    return tasks.some(function (t) { return t.state === 'waiting' || ACTIVE_STATES.indexOf(t.state) >= 0; });
  }

  function addFiles(fileList) {
    Array.prototype.forEach.call(fileList || [], function (file) {
      var dup = tasks.some(function (t) {
        return t.state !== 'canceled' && t.state !== 'success' && t.state !== 'error' &&
          t.file.name === file.name && t.file.size === file.size && t.file.lastModified === file.lastModified;
      });
      if (dup) {
        App.toast('文件已在上传队列中: ' + file.name, 'warning');
        return;
      }
      var msg = App.Validate.fileName(file.name);
      var task = createTask(file);
      if (msg) {
        setState(task, 'error', msg);
        return;
      }
    });
    schedule();
  }

  // ------------------------------------------------------------ 任务调度
  function schedule() {
    var active = tasks.filter(function (t) { return ACTIVE_STATES.indexOf(t.state) >= 0; }).length;
    tasks.forEach(function (t) {
      if (active < MAX_ACTIVE_TASKS && t.state === 'waiting') {
        active++;
        run(t);
      }
    });
    renderSummary();
  }

  function newCtl() {
    return {
      reason: null,
      xhrs: [],
      abortController: window.AbortController ? new window.AbortController() : null,
      worker: null
    };
  }

  function ctlError(ctl) {
    var err = new Error(ctl.reason === 'pause' ? '已暂停' : '已取消');
    err.name = ctl.reason === 'pause' ? 'PauseError' : 'CancelError';
    return err;
  }

  function checkCtl(ctl) {
    if (ctl.reason) throw ctlError(ctl);
  }

  function abortTask(task, reason) {
    var ctl = task.ctl;
    if (!ctl || ctl.reason) return;
    ctl.reason = reason;
    ctl.xhrs.slice().forEach(function (x) { x.abort(); });
    if (ctl.abortController) ctl.abortController.abort();
    if (ctl.worker) {
      ctl.worker.terminate();
      if (ctl.onWorkerAbort) ctl.onWorkerAbort();
    }
  }

  function run(task) {
    var ctl = newCtl();
    task.ctl = ctl;
    task.mode = '';
    // 继续/重试时复用已计算的指纹
    setState(task, task.hash ? 'checking' : 'hashing');

    Promise.resolve()
      .then(function () { return task.hash || hashFile(task, ctl); })
      .then(function (hash) {
        task.hash = hash;
        checkCtl(ctl);
        setState(task, 'checking');
        return tryFastUpload(task, ctl);
      })
      .then(function (hit) {
        checkCtl(ctl);
        if (hit) {
          task.mode = '秒传';
          return;
        }
        if (task.file.size <= NORMAL_UPLOAD_MAX) {
          task.mode = '普通上传';
          return normalUpload(task, ctl);
        }
        task.mode = '分块上传';
        return multipartUpload(task, ctl, 0);
      })
      .then(function () {
        setState(task, 'success');
      })
      .catch(function (err) {
        if (task.ctl !== ctl) return; // 已被新的执行替代
        if (err && (err.name === 'PauseError' || (err.name === 'AbortError' && ctl.reason === 'pause'))) {
          setState(task, 'paused');
        } else if (err && (err.name === 'CancelError' || err.name === 'AbortError')) {
          setState(task, 'canceled');
        } else {
          if (err && err.code === App.Code.FILE_VERIFY_FAILED) task.hash = ''; // 文件可能已被修改, 重试时重新计算
          setState(task, 'error', (err && err.message) || '上传失败');
        }
      })
      .then(function () {
        if (task.ctl === ctl) task.ctl = null;
        schedule();
      });
  }

  // ------------------------------------------------------------ 计算文件sha1
  // 优先在 Web Worker 中计算, 不阻塞页面; Worker 不可用时退回主线程分片计算
  function hashFile(task, ctl) {
    return new Promise(function (resolve, reject) {
      var worker = null;
      try {
        worker = new window.Worker('/static/js/hash-worker.js');
      } catch (e) {
        worker = null;
      }
      if (!worker) {
        hashInMainThread(task, ctl).then(resolve, reject);
        return;
      }
      function cleanup() {
        worker.terminate();
        ctl.worker = null;
        ctl.onWorkerAbort = null;
      }
      ctl.worker = worker;
      ctl.onWorkerAbort = function () {
        ctl.worker = null;
        reject(ctlError(ctl));
      };
      worker.onmessage = function (e) {
        var m = e.data || {};
        if (m.type === 'progress') {
          updateProgress(task, m.loaded, m.total, 'hash');
        } else if (m.type === 'done') {
          cleanup();
          resolve(m.hash);
        } else if (m.type === 'error') {
          cleanup();
          reject(new Error('读取文件失败: ' + m.message));
        }
      };
      worker.onerror = function (e) {
        if (e && e.preventDefault) e.preventDefault();
        cleanup();
        hashInMainThread(task, ctl).then(resolve, reject);
      };
      worker.postMessage({ id: task.id, file: task.file });
    });
  }

  async function hashInMainThread(task, ctl) {
    var SLICE = 2 * 1024 * 1024;
    var hasher = new window.Sha1();
    var file = task.file;
    for (var offset = 0; offset < file.size; offset += SLICE) {
      checkCtl(ctl);
      var buf = await file.slice(offset, Math.min(offset + SLICE, file.size)).arrayBuffer();
      hasher.update(new Uint8Array(buf));
      updateProgress(task, Math.min(offset + SLICE, file.size), file.size, 'hash');
      await sleep(0); // 让出主线程, 保持页面响应
    }
    checkCtl(ctl);
    return hasher.hex();
  }

  function sleep(ms) {
    return new Promise(function (resolve) { window.setTimeout(resolve, ms); });
  }

  // ------------------------------------------------------------ 请求
  // 使用 XHR 发送上传请求以获得上传进度
  function xhrSend(ctl, url, body, onProgress, contentType) {
    return new Promise(function (resolve, reject) {
      var xhr = new XMLHttpRequest();
      xhr.open('POST', url, true);
      var token = App.Session.get().token;
      if (token) xhr.setRequestHeader('Authorization', 'Bearer ' + token);
      if (contentType) xhr.setRequestHeader('Content-Type', contentType);
      if (onProgress && xhr.upload) {
        xhr.upload.onprogress = function (e) { onProgress(e.loaded, e.total); };
      }
      function finish() {
        var i = ctl.xhrs.indexOf(xhr);
        if (i >= 0) ctl.xhrs.splice(i, 1);
      }
      xhr.onload = function () {
        finish();
        var resp = null;
        try {
          resp = JSON.parse(xhr.responseText);
        } catch (e) {
          resp = null;
        }
        if (!resp || typeof resp.code !== 'number') {
          reject(new App.ApiError('请求失败 (HTTP ' + xhr.status + ')', 0, xhr.status));
        } else if (resp.code === App.Code.TOKEN_INVALID) {
          App.toSignin('expired');
          reject(new App.ApiError(resp.msg || '登录已失效, 请重新登录', resp.code, xhr.status));
        } else if (resp.code !== App.Code.OK) {
          reject(new App.ApiError(resp.msg || '上传失败', resp.code, xhr.status));
        } else {
          resolve(resp);
        }
      };
      xhr.onerror = function () {
        finish();
        reject(new App.ApiError('网络错误, 无法连接到上传服务', 0, 0));
      };
      xhr.onabort = function () {
        finish();
        var err = new Error('aborted');
        err.name = 'AbortError';
        reject(err);
      };
      ctl.xhrs.push(xhr);
      xhr.send(body);
    });
  }

  function apiCall(ctl, path, data) {
    return App.api(App.entry('upload') + path, data, {
      signal: ctl.abortController ? ctl.abortController.signal : undefined
    });
  }

  // 秒传: 服务端已存在相同文件时直接完成
  function tryFastUpload(task, ctl) {
    return apiCall(ctl, '/file/fastupload', {
      filehash: task.hash,
      filename: task.file.name,
      filesize: task.file.size
    }).then(function () {
      return true;
    }, function (err) {
      if (err && err.code === App.Code.FAST_UPLOAD_MISS) return false;
      throw err;
    });
  }

  function normalUpload(task, ctl) {
    setState(task, 'uploading');
    var form = new FormData();
    form.append('filehash', task.hash);
    // 单独提交原始文件名: 浏览器会转义multipart文件名中的 " 等字符
    form.append('filename', task.file.name);
    form.append('file', task.file, task.file.name);
    return xhrSend(ctl, App.entry('upload') + '/file/upload', form, function (loaded, total) {
      updateProgress(task, total ? Math.round(loaded / total * task.file.size) : 0, task.file.size, 'upload');
    });
  }

  // ------------------------------------------------------------ 分块上传
  // 1. init: 服务端返回 uploadId 及已上传的分块(断点续传)
  // 2. 并发上传缺失的分块, 失败自动重试
  // 3. complete: 服务端合并分块并校验hash
  function multipartUpload(task, ctl, attempt) {
    setState(task, 'uploading');
    return apiCall(ctl, '/file/mpupload/init', {
      filehash: task.hash,
      filesize: task.file.size,
      filename: task.file.name
    }).then(function (body) {
      checkCtl(ctl);
      var info = body.data;
      task.uploadId = info.UploadID;
      var done = {};
      (info.ChunkExists || []).forEach(function (i) { done[i] = true; });
      var chunkSize = Number(info.ChunkSize);
      var count = Number(info.ChunkCount);
      var sizeOf = function (i) {
        return i < count ? chunkSize : task.file.size - (count - 1) * chunkSize;
      };

      var doneBytes = 0;
      var pending = [];
      for (var i = 1; i <= count; i++) {
        if (done[i]) doneBytes += sizeOf(i);
        else pending.push(i);
      }
      task.resumedBytes = doneBytes;
      var inflight = {};
      var report = function () {
        var cur = doneBytes;
        Object.keys(inflight).forEach(function (k) { cur += inflight[k]; });
        task.uploadedBytes = doneBytes;
        updateProgress(task, cur, task.file.size, 'upload');
      };
      report();

      function uploadPart(index) {
        var start = (index - 1) * chunkSize;
        var blob = task.file.slice(start, start + sizeOf(index));
        var url = App.entry('upload') + '/file/mpupload/uppart?uploadid=' +
          encodeURIComponent(task.uploadId) + '&index=' + index;
        var tryOnce = function (n) {
          checkCtl(ctl);
          return xhrSend(ctl, url, blob, function (loaded) {
            inflight[index] = loaded;
            report();
          }, 'application/octet-stream').then(function () {
            delete inflight[index];
            doneBytes += sizeOf(index);
            report();
          }, function (err) {
            delete inflight[index];
            report();
            if (err.name === 'AbortError' || ctl.reason) throw ctl.reason ? ctlError(ctl) : err;
            // 业务错误(如会话过期)不重试; 网络/服务错误重试
            var retryable = !err.code || err.code === App.Code.SERVER_ERROR || err.code === App.Code.UPLOAD_FAILED;
            if (retryable && n < PART_RETRIES) {
              return sleep(800 * (n + 1)).then(function () { return tryOnce(n + 1); });
            }
            throw err;
          });
        };
        return tryOnce(0);
      }

      // 固定并发数的工作队列
      var queue = pending.slice();
      var firstErr = null;
      function worker() {
        if (firstErr || !queue.length) return Promise.resolve();
        var idx = queue.shift();
        return uploadPart(idx).then(worker, function (err) {
          if (!firstErr) firstErr = err;
          // 一个分块失败后停止其他分块
          queue.length = 0;
          ctl.xhrs.slice().forEach(function (x) { x.abort(); });
        });
      }
      var workers = [];
      for (var w = 0; w < Math.min(PART_CONCURRENCY, pending.length); w++) workers.push(worker());

      return Promise.all(workers).then(function () {
        if (ctl.reason) throw ctlError(ctl);
        if (firstErr) throw firstErr;
        setState(task, 'merging');
        return apiCall(ctl, '/file/mpupload/complete', {
          uploadid: task.uploadId,
          filehash: task.hash,
          filesize: task.file.size,
          filename: task.file.name
        });
      });
    }).catch(function (err) {
      // 分块不完整(例如服务端分块文件丢失)时重新走一遍续传流程
      if (err && err.code === App.Code.CHUNK_INCOMPLETE && attempt < 2 && !ctl.reason) {
        return multipartUpload(task, ctl, attempt + 1);
      }
      throw err;
    });
  }

  // ------------------------------------------------------------ 任务UI
  var STATE_TEXT = {
    waiting: '等待中',
    hashing: '计算文件指纹',
    checking: '检查秒传',
    uploading: '上传中',
    merging: '服务端合并校验中',
    paused: '已暂停',
    success: '上传成功',
    error: '上传失败',
    canceled: '已取消'
  };

  function createTask(file) {
    var task = {
      id: nextId++,
      file: file,
      state: 'waiting',
      hash: '',
      mode: '',
      uploadId: '',
      speed: { at: 0, bytes: 0, value: 0 }
    };
    var bar = el('div', { class: 'bar' });
    var progress = el('div', {
      class: 'progress', role: 'progressbar', 'aria-valuemin': '0', 'aria-valuemax': '100', 'aria-valuenow': '0',
      'aria-label': file.name + ' 上传进度'
    }, [bar]);
    var status = el('span', { class: 'status' });
    var detail = el('span', { class: 'detail' });
    var pauseBtn = el('button', { class: 'btn btn-sm', type: 'button', text: '暂停', onclick: function () { togglePause(task); } });
    var retryBtn = el('button', { class: 'btn btn-sm', type: 'button', text: '重试', hidden: true, onclick: function () { retry(task); } });
    var cancelBtn = el('button', { class: 'btn btn-sm btn-danger', type: 'button', text: '取消', onclick: function () { cancel(task); } });
    var removeBtn = el('button', { class: 'btn btn-sm', type: 'button', text: '移除', hidden: true, onclick: function () { remove(task); } });
    var node = el('li', { class: 'task' }, [
      el('div', { class: 'row' }, [
        el('span', { class: 'tname', title: file.name, text: file.name }),
        el('span', { class: 'muted', text: App.formatSize(file.size) }),
        el('span', { class: 'actions' }, [pauseBtn, retryBtn, cancelBtn, removeBtn])
      ]),
      progress,
      el('div', { class: 'meta' }, [status, detail])
    ]);
    task.ui = {
      node: node, bar: bar, progress: progress, status: status, detail: detail,
      pauseBtn: pauseBtn, retryBtn: retryBtn, cancelBtn: cancelBtn, removeBtn: removeBtn
    };
    tasks.push(task);
    listBox.appendChild(node);
    setState(task, 'waiting');
    return task;
  }

  function setState(task, state, message) {
    task.state = state;
    task.message = message || '';
    var ui = task.ui;
    var text = STATE_TEXT[state] || state;
    if (state === 'success' && task.mode) text += ' (' + task.mode + ')';
    if (state === 'uploading' && task.mode) text = task.mode + '中';
    if (message) text += ': ' + message;
    ui.status.textContent = text;
    ui.status.className = 'status ' + (state === 'success' ? 'success' : state === 'error' || state === 'canceled' ? 'error' : state === 'paused' ? 'paused' : '');
    ui.progress.className = 'progress ' + (state === 'success' ? 'success' : state === 'error' ? 'error' : state === 'paused' ? 'paused' : '');

    var active = ACTIVE_STATES.indexOf(state) >= 0;
    var finished = state === 'success' || state === 'canceled';
    ui.pauseBtn.hidden = !(active || state === 'paused') || state === 'merging';
    ui.pauseBtn.textContent = state === 'paused' ? '继续' : '暂停';
    ui.retryBtn.hidden = state !== 'error';
    ui.cancelBtn.hidden = finished || state === 'merging';
    ui.removeBtn.hidden = !(finished || state === 'error');

    if (state === 'success') {
      setBar(task, 100);
      ui.detail.textContent = task.hash ? 'SHA1 ' + task.hash.slice(0, 12) + '…' : '';
    } else if (state === 'paused' && task.uploadedBytes > 0) {
      // 暂停时进度只保留已完成的分块, 未完成的分块继续时会重新上传
      setBar(task, task.uploadedBytes / task.file.size * 100);
      ui.detail.textContent = '已上传 ' + App.formatSize(task.uploadedBytes) + ' / ' +
        App.formatSize(task.file.size) + ', 点击继续可断点续传';
    } else if (state === 'checking' || state === 'merging') {
      ui.detail.textContent = '';
    } else if (state !== 'uploading' && state !== 'hashing') {
      ui.detail.textContent = '';
    }
    if (state === 'hashing' || state === 'uploading') {
      task.speed = { at: Date.now(), bytes: -1, value: 0 };
      // 进入新阶段时进度条从0开始(否则会短暂显示上一阶段"计算指纹"的100%)
      setBar(task, 0);
    }
    renderSummary();
  }

  function setBar(task, percent) {
    var p = Math.max(0, Math.min(100, percent));
    task.ui.bar.style.width = p.toFixed(1) + '%';
    task.ui.progress.setAttribute('aria-valuenow', String(Math.round(p)));
  }

  // phase: 'hash' 计算指纹(进度条占0~100%, 灰色语义), 'upload' 上传
  function updateProgress(task, loaded, total, phase) {
    var percent = total > 0 ? loaded / total * 100 : 100;
    setBar(task, percent);
    var now = Date.now();
    var sp = task.speed;
    if (sp.bytes < 0) {
      sp.at = now;
      sp.bytes = loaded;
    } else if (now - sp.at >= 500) {
      var inst = (loaded - sp.bytes) / ((now - sp.at) / 1000);
      sp.value = sp.value ? sp.value * 0.6 + inst * 0.4 : inst;
      sp.at = now;
      sp.bytes = loaded;
    }
    var text = (phase === 'hash' ? '指纹 ' : '') + percent.toFixed(1) + '% · ' +
      App.formatSize(loaded) + ' / ' + App.formatSize(total);
    if (sp.value > 0 && loaded < total) {
      text += ' · ' + App.formatSpeed(sp.value);
      if (phase === 'upload') text += ' · 剩余 ' + App.formatDuration((total - loaded) / sp.value);
    }
    if (phase === 'upload' && task.resumedBytes > 0) text += ' · 断点续传';
    task.ui.detail.textContent = text;
  }

  function togglePause(task) {
    if (task.state === 'paused') {
      setState(task, 'waiting');
      schedule();
    } else if (ACTIVE_STATES.indexOf(task.state) >= 0 && task.state !== 'merging') {
      abortTask(task, 'pause');
    }
  }

  function retry(task) {
    if (task.state !== 'error') return;
    setBar(task, 0);
    setState(task, 'waiting');
    schedule();
  }

  function cancel(task) {
    if (task.ctl) {
      abortTask(task, 'cancel');
    } else {
      setState(task, 'canceled');
    }
    // 通知服务端清理已上传的分块
    if (task.uploadId) {
      App.api(App.entry('upload') + '/file/mpupload/cancel', { uploadid: task.uploadId }).catch(function () { });
      task.uploadId = '';
    }
    schedule();
  }

  function remove(task) {
    var i = tasks.indexOf(task);
    if (i >= 0) tasks.splice(i, 1);
    if (task.ui.node.parentNode) task.ui.node.parentNode.removeChild(task.ui.node);
    renderSummary();
  }

  // 全部完成后提示返回文件列表
  var summaryShown = false;
  function renderSummary() {
    var ok = tasks.filter(function (t) { return t.state === 'success'; }).length;
    if (!tasks.length || hasActiveTasks()) {
      summaryShown = false;
      return;
    }
    var failed = tasks.filter(function (t) { return t.state === 'error'; }).length;
    if (summaryShown || !ok) return;
    summaryShown = true;
    var back = App.$('#back-btn');
    back.href = '/static/view/home.html?uploaded=' + ok;
    App.toast(failed ? ok + ' 个文件上传成功, ' + failed + ' 个失败' : '全部上传完成, 共 ' + ok + ' 个文件', failed ? 'warning' : 'success');
  }
})(window.App);
