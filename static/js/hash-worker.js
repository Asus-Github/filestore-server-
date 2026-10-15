/*
 * 在 Web Worker 中计算文件 SHA-1, 避免大文件哈希时阻塞页面
 * 输入消息: { id, file, sliceSize }
 * 输出消息: { id, type: 'progress', loaded, total } | { id, type: 'done', hash } | { id, type: 'error', message }
 */
/* global importScripts, Sha1, FileReaderSync */
importScripts('/static/js/sha1.js');

function readSlice(blob) {
  if (typeof blob.arrayBuffer === 'function') {
    return blob.arrayBuffer();
  }
  return Promise.resolve(new FileReaderSync().readAsArrayBuffer(blob));
}

self.onmessage = async function (e) {
  var msg = e.data || {};
  var file = msg.file;
  var sliceSize = msg.sliceSize || 4 * 1024 * 1024;
  try {
    var hasher = new Sha1();
    var total = file.size;
    var lastReport = 0;
    for (var offset = 0; offset < total; offset += sliceSize) {
      var buf = await readSlice(file.slice(offset, Math.min(offset + sliceSize, total)));
      hasher.update(new Uint8Array(buf));
      var loaded = Math.min(offset + sliceSize, total);
      var now = Date.now();
      if (now - lastReport > 100 || loaded === total) {
        lastReport = now;
        self.postMessage({ id: msg.id, type: 'progress', loaded: loaded, total: total });
      }
    }
    self.postMessage({ id: msg.id, type: 'done', hash: hasher.hex() });
  } catch (err) {
    self.postMessage({ id: msg.id, type: 'error', message: String((err && err.message) || err) });
  }
};
