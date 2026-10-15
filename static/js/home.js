(function (App) {
  'use strict';

  if (!App.requireLogin()) return;

  var PAGE_SIZE = 20;
  var el = App.el;
  var state = { offset: 0, total: 0, loading: false };
  var session = App.Session.get();

  var rowsBox = App.$('#file-rows');
  var emptyBox = App.$('#file-empty');
  var loadingBox = App.$('#file-loading');
  var pager = App.$('#pager');

  App.$('#top-username').textContent = session.username;
  App.$('#username').textContent = session.username;

  App.$('#signout-btn').addEventListener('click', function () {
    App.setBusy(this, true, '退出中...');
    App.signout();
  });
  App.$('#refresh-btn').addEventListener('click', function () { loadFiles(); });
  App.$('#prev-btn').addEventListener('click', function () {
    loadFiles(Math.max(0, state.offset - PAGE_SIZE));
  });
  App.$('#next-btn').addEventListener('click', function () {
    loadFiles(state.offset + PAGE_SIZE);
  });

  // ------------------------------------------------------------ 用户信息
  function loadUserInfo() {
    return App.api('/user/info').then(function (body) {
      var d = body.data || {};
      App.$('#username').textContent = d.Username || session.username;
      App.$('#regtime').textContent = d.SignupAt || '-';
      App.$('#lastactive').textContent = d.LastActive || '-';
      // 服务端下发的上传/下载入口可能变化, 每次刷新
      App.Session.saveEntries(d);
    }).catch(function (err) {
      App.toast('获取用户信息失败: ' + err.message, 'error');
    });
  }

  // ------------------------------------------------------------ 文件列表
  function loadFiles(offset) {
    if (state.loading) return Promise.resolve();
    if (typeof offset === 'number') state.offset = offset;
    state.loading = true;
    loadingBox.hidden = false;
    emptyBox.hidden = true;

    return App.api('/file/query', { limit: PAGE_SIZE, offset: state.offset })
      .then(function (body) {
        var data = body.data || {};
        var files = Array.isArray(data.Files) ? data.Files : [];
        state.total = Number(data.Total) || 0;
        // 当前页的文件全部被删除后回到上一页
        if (!files.length && state.offset > 0 && state.total > 0) {
          state.loading = false;
          return loadFiles(Math.max(0, Math.floor((state.total - 1) / PAGE_SIZE) * PAGE_SIZE));
        }
        renderFiles(files);
      })
      .catch(function (err) {
        rowsBox.textContent = '';
        App.toast('获取文件列表失败: ' + err.message, 'error');
      })
      .then(function () {
        state.loading = false;
        loadingBox.hidden = true;
      });
  }

  function renderFiles(files) {
    rowsBox.textContent = '';
    files.forEach(function (f) {
      rowsBox.appendChild(renderRow(f));
    });
    emptyBox.hidden = files.length > 0;
    App.$('#filecount').textContent = String(state.total);

    pager.hidden = state.total <= PAGE_SIZE;
    var from = state.total ? state.offset + 1 : 0;
    var to = Math.min(state.offset + files.length, state.total);
    App.$('#pager-info').textContent = '第 ' + from + '-' + to + ' 条, 共 ' + state.total + ' 条';
    App.$('#prev-btn').disabled = state.offset <= 0;
    App.$('#next-btn').disabled = state.offset + PAGE_SIZE >= state.total;
  }

  function renderRow(f) {
    var nameCell = el('td', { class: 'name' }, [
      el('span', { class: 'fname', title: f.FileName, text: f.FileName }),
      el('span', { class: 'fhash', title: 'SHA1: ' + f.FileHash, text: f.FileHash.slice(0, 16) + '…' })
    ]);
    var ops = el('td', { class: 'ops' }, [el('div', { class: 'ops-btns' }, [
      el('button', {
        class: 'btn btn-sm btn-primary', type: 'button', text: '下载',
        'aria-label': '下载 ' + f.FileName,
        onclick: function () { downloadFile(f, this); }
      }),
      el('button', {
        class: 'btn btn-sm', type: 'button', text: '断点下载',
        title: '支持暂停/继续的分段下载',
        'aria-label': '断点下载 ' + f.FileName,
        onclick: function () { resumableDownload(f); }
      }),
      el('button', {
        class: 'btn btn-sm', type: 'button', text: '重命名',
        'aria-label': '重命名 ' + f.FileName,
        onclick: function () { renameFile(f); }
      }),
      el('button', {
        class: 'btn btn-sm btn-danger', type: 'button', text: '删除',
        'aria-label': '删除 ' + f.FileName,
        onclick: function () { deleteFile(f); }
      })
    ])]);
    return el('tr', { dataset: { hash: f.FileHash } }, [
      nameCell,
      el('td', { class: 'nowrap', title: f.FileSize + ' 字节', text: App.formatSize(f.FileSize) }),
      el('td', { class: 'nowrap col-hide', text: f.UploadAt || '-' }),
      el('td', { class: 'nowrap col-hide', text: f.LastUpdated || '-' }),
      ops
    ]);
  }

  // ------------------------------------------------------------ 操作
  // 获取带签名的临时下载链接(不在链接中暴露登录token)
  function getDownloadURL(f) {
    return App.api(App.entry('download') + '/file/downloadurl', { filehash: f.FileHash })
      .then(function (body) { return body.data; });
  }

  function downloadFile(f, btn) {
    App.setBusy(btn, true, '准备中');
    getDownloadURL(f).then(function (d) {
      App.triggerDownload(d.URL, d.FileName || f.FileName);
      App.toast('开始下载: ' + f.FileName, 'success');
    }).catch(function (err) {
      App.toast('下载失败: ' + err.message, 'error');
    }).then(function () {
      App.setBusy(btn, false);
    });
  }

  function resumableDownload(f) {
    var frame = el('iframe', {
      title: '断点下载 ' + f.FileName,
      src: '/static/view/download.html?embedded=1&filehash=' + encodeURIComponent(f.FileHash)
    });
    App.modal({
      title: '断点下载',
      wide: true,
      body: [frame],
      focus: 'iframe',
      onClose: function () {
        // 关闭弹窗时停止iframe内的下载
        frame.src = 'about:blank';
      }
    });
  }

  function renameFile(f) {
    App.prompt('重命名', '新的文件名', f.FileName, function (v) {
      var msg = App.Validate.fileName(v);
      if (msg) return msg;
      return v === f.FileName ? '文件名未修改' : '';
    }).then(function (newName) {
      if (newName === null) return;
      return App.api('/file/update', { op: 0, filehash: f.FileHash, filename: newName })
        .then(function () {
          App.toast('重命名成功', 'success');
          return loadFiles();
        });
    }).catch(function (err) {
      App.toast('重命名失败: ' + err.message, 'error');
    });
  }

  function deleteFile(f) {
    App.confirm('删除文件', '确定要删除文件“' + f.FileName + '”吗? 删除后无法恢复。', '删除', true)
      .then(function (ok) {
        if (!ok) return;
        return App.api('/file/delete', { filehash: f.FileHash }).then(function () {
          App.toast('已删除: ' + f.FileName, 'success');
          return loadFiles();
        });
      })
      .catch(function (err) {
        App.toast('删除失败: ' + err.message, 'error');
      });
  }

  // 从上传页返回时显示提示
  var uploaded = App.queryParam('uploaded');
  if (uploaded) {
    App.toast('已成功上传 ' + uploaded + ' 个文件', 'success');
    window.history.replaceState(null, '', window.location.pathname);
  }

  loadUserInfo();
  loadFiles(0);
})(window.App);
