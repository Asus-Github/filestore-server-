// API 端到端测试: node test/e2e/api.test.mjs
// 前置: 已启动全部服务(source test/e2e/env.sh && ./service/start-all.sh)
import crypto from 'node:crypto';
import assert from 'node:assert/strict';

const GW = process.env.GW || 'http://127.0.0.1:18080';
const UP = process.env.UP || 'http://127.0.0.1:28080';
const DL = process.env.DL || 'http://127.0.0.1:38080';
const OK = 10000;

let passed = 0;
const failures = [];
async function test(name, fn) {
  try {
    await fn();
    passed++;
    console.log('  ✓ ' + name);
  } catch (e) {
    failures.push(name);
    console.log('  ✗ ' + name + '\n      ' + (e && e.stack ? e.stack.split('\n').slice(0, 3).join('\n      ') : e));
  }
}

const sha1 = (buf) => crypto.createHash('sha1').update(buf).digest('hex');
const rand = (n) => crypto.randomBytes(n);

async function call(url, data, token, extra = {}) {
  const headers = { ...(extra.headers || {}) };
  if (token) headers.Authorization = 'Bearer ' + token;
  let body = extra.body;
  if (data) body = new URLSearchParams(data);
  const resp = await fetch(url, { method: extra.method || 'POST', headers, body, redirect: 'manual' });
  const text = await resp.text();
  let json = null;
  try { json = JSON.parse(text); } catch { /* not json */ }
  return { status: resp.status, json, text, headers: resp.headers };
}

async function signupAndLogin(prefix = 'u') {
  const username = prefix + crypto.randomBytes(4).toString('hex');
  const password = 'Passw0rd!';
  const r1 = await call(GW + '/user/signup', { username, password });
  assert.equal(r1.json.code, OK, 'signup: ' + r1.text);
  const r2 = await call(GW + '/user/signin', { username, password });
  assert.equal(r2.json.code, OK, 'signin: ' + r2.text);
  return { username, password, token: r2.json.data.Token, data: r2.json.data };
}

async function listFiles(token, limit = 100, offset = 0) {
  const r = await call(GW + '/file/query', { limit, offset }, token);
  assert.equal(r.json.code, OK, r.text);
  return r.json.data;
}

// 与前端一致: 原始文件名单独作为表单字段提交(multipart文件名中的 " 会被转义为 %22)
async function upload(token, buf, name, filehash, { rawOnly = false } = {}) {
  const fd = new FormData();
  if (filehash) fd.append('filehash', filehash);
  if (!rawOnly) fd.append('filename', name);
  fd.append('file', new Blob([buf]), name);
  return call(UP + '/file/upload', null, token, { body: fd });
}

async function getDownloadURL(token, filehash) {
  const r = await call(DL + '/file/downloadurl', { filehash }, token);
  return r;
}

async function fetchBytes(url, headers = {}) {
  const resp = await fetch(url, { headers });
  return { status: resp.status, headers: resp.headers, buf: Buffer.from(await resp.arrayBuffer()) };
}

async function mpUpload(token, buf, name, chunkSizeHint, { skip = [], stopAfter = Infinity } = {}) {
  const filehash = sha1(buf);
  const init = await call(UP + '/file/mpupload/init', { filehash, filesize: buf.length, filename: name }, token);
  assert.equal(init.json.code, OK, init.text);
  const info = init.json.data;
  let sent = 0;
  for (let i = 1; i <= info.ChunkCount; i++) {
    if (info.ChunkExists.includes(i) || skip.includes(i)) continue;
    if (sent >= stopAfter) break;
    const part = buf.subarray((i - 1) * info.ChunkSize, i * info.ChunkSize);
    const r = await call(`${UP}/file/mpupload/uppart?uploadid=${info.UploadID}&index=${i}`, null, token,
      { body: part, headers: { 'Content-Type': 'application/octet-stream' } });
    assert.equal(r.json.code, OK, 'uppart ' + i + ': ' + r.text);
    sent++;
  }
  return { info, filehash };
}

console.log('API E2E against', GW, UP, DL);

// ------------------------------------------------------------------ 账号
console.log('\n[账号]');
let alice;
await test('注册+登录成功, 返回64位token与可用的上传/下载入口', async () => {
  alice = await signupAndLogin('alice');
  assert.match(alice.token, /^[0-9a-f]{64}$/);
  assert.ok(alice.data.ExpireAt > Date.now() / 1000);
  assert.equal(alice.data.UploadEntry, 'http://127.0.0.1:28080');
  assert.equal(alice.data.DownloadEntry, 'http://127.0.0.1:38080');
  assert.equal(alice.data.Location, '/static/view/home.html');
});

await test('重复注册返回 10007 用户已存在', async () => {
  const r = await call(GW + '/user/signup', { username: alice.username, password: 'another1' });
  assert.equal(r.json.code, 10007, r.text);
});

await test('注册参数校验(用户名过短/密码过短/非法字符)', async () => {
  for (const [u, p] of [['ab', 'goodpass'], ['validname', '123'], ['bad name', 'goodpass'], ['<script>', 'goodpass']]) {
    const r = await call(GW + '/user/signup', { username: u, password: p });
    assert.equal(r.json.code, 10001, `${u}/${p}: ${r.text}`);
  }
});

await test('中文用户名可注册登录', async () => {
  const u = '测试用户' + crypto.randomBytes(2).toString('hex');
  assert.equal((await call(GW + '/user/signup', { username: u, password: 'goodpass' })).json.code, OK);
  const r = await call(GW + '/user/signin', { username: u, password: 'goodpass' });
  assert.equal(r.json.code, OK, r.text);
});

await test('错误密码/不存在用户 登录失败 10004 且提示一致', async () => {
  const r1 = await call(GW + '/user/signin', { username: alice.username, password: 'wrongpass' });
  const r2 = await call(GW + '/user/signin', { username: 'nouser_' + Date.now(), password: 'wrongpass' });
  assert.equal(r1.json.code, 10004);
  assert.equal(r2.json.code, 10004);
  assert.equal(r1.json.msg, r2.json.msg);
});

await test('用户信息: 返回当前token所属用户的信息', async () => {
  const r = await call(GW + '/user/info', {}, alice.token);
  assert.equal(r.json.code, OK, r.text);
  assert.equal(r.json.data.Username, alice.username);
  assert.match(r.json.data.SignupAt, /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);
  // 注册时间应接近当前时间(时区正确)
  const diff = Math.abs(new Date(r.json.data.SignupAt.replace(' ', 'T')).getTime() - Date.now());
  assert.ok(diff < 5 * 60 * 1000, 'signup time drift ' + diff);
});

await test('鉴权: 无token/伪造token/格式错误 均返回401+10005', async () => {
  for (const tok of [null, 'a'.repeat(64), 'x'.repeat(40), alice.token.slice(0, 63) + 'z']) {
    const r = await call(GW + '/user/info', {}, tok);
    assert.equal(r.status, 401, r.text);
    assert.equal(r.json.code, 10005);
  }
});

await test('鉴权: 旧版 query 参数 token 仍然可用, 且 username 与 token 不一致时拒绝', async () => {
  const ok = await call(`${GW}/user/info?token=${alice.token}&username=${encodeURIComponent(alice.username)}`, {});
  assert.equal(ok.json.code, OK, ok.text);
  const bad = await call(`${GW}/user/info?token=${alice.token}&username=someoneelse`, {});
  assert.equal(bad.json.code, 10005, bad.text);
});

let bob;
await test('多处登录: 同一用户的多个token同时有效', async () => {
  bob = await signupAndLogin('bob');
  const r = await call(GW + '/user/signin', { username: bob.username, password: bob.password });
  const t2 = r.json.data.Token;
  assert.notEqual(t2, bob.token);
  assert.equal((await call(GW + '/user/info', {}, bob.token)).json.code, OK);
  assert.equal((await call(GW + '/user/info', {}, t2)).json.code, OK);
  // 退出第二个会话, 第一个仍有效
  assert.equal((await call(GW + '/user/signout', {}, t2)).json.code, OK);
  assert.equal((await call(GW + '/user/info', {}, t2)).json.code, 10005);
  assert.equal((await call(GW + '/user/info', {}, bob.token)).json.code, OK);
});

// ------------------------------------------------------------------ 普通上传 / 秒传 / 列表
console.log('\n[上传与列表]');
const small = rand(300 * 1024 + 7);
const smallHash = sha1(small);
// 含HTML/引号的文件名, 用于验证转义与XSS防护(浏览器提交的文件名不会包含路径分隔符)
const smallName = '报告 <img src=x onerror=alert(1)> "q" & \'s\'.bin';

await test('普通上传: 成功并返回服务端计算的hash', async () => {
  const r = await upload(alice.token, small, smallName, smallHash);
  assert.equal(r.json.code, OK, r.text);
  assert.equal(r.json.data.FileHash, smallHash);
  assert.equal(r.json.data.FileSize, small.length);
  assert.equal(r.json.data.FileName, smallName);
});

await test('普通上传: 客户端hash与内容不一致时拒绝 10012', async () => {
  const r = await upload(alice.token, rand(1000), 'x.bin', 'a'.repeat(40));
  assert.equal(r.json.code, 10012, r.text);
});

await test('普通上传: 未登录拒绝(401)', async () => {
  const r = await upload(null, rand(10), 'x.bin');
  assert.equal(r.status, 401);
  assert.equal(r.json.code, 10005);
});

await test('普通上传: 空文件/文件名含路径时被规范化', async () => {
  const r = await upload(alice.token, Buffer.alloc(0), '../../etc/empty.txt');
  assert.equal(r.json.code, OK, r.text);
  assert.equal(r.json.data.FileName, 'empty.txt');
  assert.equal(r.json.data.FileHash, sha1(Buffer.alloc(0)));
});

await test('文件列表: 按上传时间倒序, 包含文件名/大小/时间', async () => {
  const d = await listFiles(alice.token);
  assert.equal(d.Total, 2);
  assert.equal(d.Files.length, 2);
  assert.equal(d.Files[0].FileName, 'empty.txt');
  const f = d.Files.find((x) => x.FileHash === smallHash);
  assert.equal(f.FileName, smallName);
  assert.equal(f.FileSize, small.length);
  assert.match(f.UploadAt, /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);
});

await test('秒传: 其他用户上传过的文件可秒传(且不需要重新上传内容)', async () => {
  const r = await call(UP + '/file/fastupload', { filehash: smallHash, filename: 'bob的副本.bin', filesize: small.length }, bob.token);
  assert.equal(r.json.code, OK, r.text);
  const d = await listFiles(bob.token);
  assert.equal(d.Total, 1);
  assert.equal(d.Files[0].FileName, 'bob的副本.bin');
});

await test('秒传: 不存在的文件返回 10009, 大小不一致也不能秒传', async () => {
  const r1 = await call(UP + '/file/fastupload', { filehash: sha1(rand(10)), filename: 'n.bin', filesize: 10 }, bob.token);
  assert.equal(r1.json.code, 10009, r1.text);
  const r2 = await call(UP + '/file/fastupload', { filehash: smallHash, filename: 'n.bin', filesize: 1 }, bob.token);
  assert.equal(r2.json.code, 10009, r2.text);
});

await test('秒传: 参数校验', async () => {
  const r = await call(UP + '/file/fastupload', { filehash: 'zz', filename: 'n.bin', filesize: 1 }, bob.token);
  assert.equal(r.json.code, 10001);
});

await test('分页: limit/offset 生效', async () => {
  const p1 = await listFiles(alice.token, 1, 0);
  const p2 = await listFiles(alice.token, 1, 1);
  assert.equal(p1.Total, 2);
  assert.equal(p1.Files.length, 1);
  assert.equal(p2.Files.length, 1);
  assert.notEqual(p1.Files[0].FileHash, p2.Files[0].FileHash);
});

// ------------------------------------------------------------------ 下载
console.log('\n[下载]');
let dlURL;
await test('获取下载链接: 签名链接, 不包含登录token', async () => {
  const r = await getDownloadURL(alice.token, smallHash);
  assert.equal(r.json.code, OK, r.text);
  dlURL = r.json.data.URL;
  assert.ok(dlURL.startsWith(DL + '/file/download?'), dlURL);
  assert.ok(!dlURL.includes(alice.token));
  assert.equal(r.json.data.FileName, smallName);
  assert.equal(r.json.data.FileSize, small.length);
});

await test('下载: 内容正确, Content-Disposition 支持中文/特殊字符', async () => {
  const r = await fetchBytes(dlURL);
  assert.equal(r.status, 200);
  assert.equal(sha1(r.buf), smallHash);
  const cd = r.headers.get('content-disposition');
  assert.ok(cd.startsWith('attachment;'), cd);
  const m = /filename\*=UTF-8''([^;]+)/.exec(cd);
  assert.equal(decodeURIComponent(m[1]), smallName);
  assert.equal(r.headers.get('accept-ranges'), 'bytes');
});

await test('下载: Range 请求返回 206 与正确的 Content-Range', async () => {
  const r = await fetchBytes(dlURL, { Range: 'bytes=100-1123' });
  assert.equal(r.status, 206);
  assert.equal(r.headers.get('content-range'), `bytes 100-1123/${small.length}`);
  assert.ok(r.buf.equals(small.subarray(100, 1124)));
  const tail = await fetchBytes(dlURL, { Range: `bytes=${small.length - 10}-` });
  assert.equal(tail.status, 206);
  assert.ok(tail.buf.equals(small.subarray(small.length - 10)));
  const bad = await fetchBytes(dlURL, { Range: `bytes=${small.length + 10}-` });
  assert.equal(bad.status, 416);
});

await test('下载: 跨域请求可读取 Content-Range/Content-Disposition 头', async () => {
  const pre = await fetch(dlURL, { method: 'OPTIONS', headers: { Origin: GW, 'Access-Control-Request-Method': 'GET', 'Access-Control-Request-Headers': 'range' } });
  assert.ok(pre.status === 204 || pre.status === 200, 'preflight ' + pre.status);
  assert.ok((pre.headers.get('access-control-allow-headers') || '').toLowerCase().includes('range'));
  const r = await fetch(dlURL, { headers: { Origin: GW, Range: 'bytes=0-9' } });
  const expose = (r.headers.get('access-control-expose-headers') || '').toLowerCase();
  assert.ok(expose.includes('content-range') && expose.includes('content-disposition'), expose);
});

await test('下载: 签名被篡改/过期 返回403; 用他人用户名签名无效', async () => {
  const u = new URL(dlURL);
  u.searchParams.set('sign', 'f'.repeat(64));
  assert.equal((await fetchBytes(u.toString())).status, 403);
  const u2 = new URL(dlURL);
  u2.searchParams.set('expires', String(Math.floor(Date.now() / 1000) + 99999));
  assert.equal((await fetchBytes(u2.toString())).status, 403);
  const u3 = new URL(dlURL);
  u3.searchParams.set('username', bob.username);
  assert.equal((await fetchBytes(u3.toString())).status, 403);
});

await test('下载: 也支持 Authorization 头直接下载', async () => {
  const r = await fetchBytes(`${DL}/file/download?filehash=${smallHash}`, { Authorization: 'Bearer ' + alice.token });
  assert.equal(r.status, 200);
  assert.equal(sha1(r.buf), smallHash);
});

await test('越权: 不能获取/下载不属于自己的文件', async () => {
  const carol = await signupAndLogin('carol');
  const r = await getDownloadURL(carol.token, smallHash);
  assert.equal(r.json.code, 10008, r.text);
  const r2 = await fetch(`${DL}/file/download?filehash=${smallHash}`, { headers: { Authorization: 'Bearer ' + carol.token } });
  const j = await r2.json();
  assert.equal(j.code, 10008);
  // 旧版漏洞: 仅凭 username 参数即可下载他人文件
  const r3 = await fetch(`${DL}/file/download?filehash=${smallHash}&username=${alice.username}`);
  assert.equal(r3.status, 401);
});

// ------------------------------------------------------------------ 重命名 / 删除
console.log('\n[重命名与删除]');
await test('重命名: 成功并返回新文件信息; 列表中可见', async () => {
  const r = await call(GW + '/file/update', { op: 0, filehash: smallHash, filename: '新名字 v2.bin' }, alice.token);
  assert.equal(r.json.code, OK, r.text);
  assert.equal(r.json.data.FileName, '新名字 v2.bin');
  const d = await listFiles(alice.token);
  assert.equal(d.Files.find((x) => x.FileHash === smallHash).FileName, '新名字 v2.bin');
  // 不影响bob的文件名
  const b = await listFiles(bob.token);
  assert.equal(b.Files[0].FileName, 'bob的副本.bin');
});

await test('重命名: 文件名校验(空/含斜杠/超长), 不支持的op', async () => {
  for (const name of ['', '   ', 'a/b', 'a\\b', 'x'.repeat(256)]) {
    const r = await call(GW + '/file/update', { op: 0, filehash: smallHash, filename: name }, alice.token);
    assert.equal(r.json.code, 10001, JSON.stringify(name) + r.text);
  }
  const r = await call(GW + '/file/update', { op: 1, filehash: smallHash, filename: 'ok' }, alice.token);
  assert.equal(r.json.code, 10001);
});

await test('重命名: 不能重命名他人的文件/不存在的文件', async () => {
  const fake = sha1(rand(5));
  const r = await call(GW + '/file/update', { op: 0, filehash: fake, filename: 'x.bin' }, alice.token);
  assert.equal(r.json.code, 10008, r.text);
  const carol = await signupAndLogin('carol');
  const r2 = await call(GW + '/file/update', { op: 0, filehash: smallHash, filename: 'hacked.bin' }, carol.token);
  assert.equal(r2.json.code, 10008, r2.text);
});

await test('删除: 删除后列表中消失, 不能再下载; 再次上传(秒传)可恢复', async () => {
  const r = await call(GW + '/file/delete', { filehash: smallHash }, alice.token);
  assert.equal(r.json.code, OK, r.text);
  const d = await listFiles(alice.token);
  assert.ok(!d.Files.some((x) => x.FileHash === smallHash));
  assert.equal((await getDownloadURL(alice.token, smallHash)).json.code, 10008);
  // 旧签名链接也失效
  const old = await fetch(dlURL);
  assert.equal((await old.json()).code, 10008);
  // 不影响bob
  assert.equal((await listFiles(bob.token)).Total, 1);
  // 重复删除
  assert.equal((await call(GW + '/file/delete', { filehash: smallHash }, alice.token)).json.code, 10008);
  // 秒传恢复
  const fu = await call(UP + '/file/fastupload', { filehash: smallHash, filename: '恢复.bin', filesize: small.length }, alice.token);
  assert.equal(fu.json.code, OK, fu.text);
  assert.ok((await listFiles(alice.token)).Files.some((x) => x.FileHash === smallHash && x.FileName === '恢复.bin'));
});

// ------------------------------------------------------------------ 分块上传
console.log('\n[分块上传]');
const big = rand(3 * 1024 * 1024 + 12345); // 测试环境分块大小为1MB -> 4个分块
const bigHash = sha1(big);

await test('分块上传: init -> uppart(全部) -> complete, 内容可正确下载', async () => {
  const { info } = await mpUpload(alice.token, big, '大文件.dat');
  assert.equal(info.ChunkCount, 4);
  assert.match(info.UploadID, /^[0-9a-f]{32}$/);
  const r = await call(UP + '/file/mpupload/complete', { uploadid: info.UploadID, filehash: bigHash, filesize: big.length, filename: '大文件.dat' }, alice.token);
  assert.equal(r.json.code, OK, r.text);
  const u = await getDownloadURL(alice.token, bigHash);
  const dl = await fetchBytes(u.json.data.URL);
  assert.equal(sha1(dl.buf), bigHash);
});

await test('分块上传: 断点续传 — 重新 init 返回同一 uploadId 及已完成分块', async () => {
  const data = rand(2 * 1024 * 1024 + 500); // 3个分块
  const { info } = await mpUpload(bob.token, data, 'resume.dat', 0, { stopAfter: 2 });
  const again = await call(UP + '/file/mpupload/init', { filehash: sha1(data), filesize: data.length, filename: 'resume.dat' }, bob.token);
  assert.equal(again.json.data.UploadID, info.UploadID);
  assert.deepEqual(again.json.data.ChunkExists, [1, 2]);
  // 未传完时 complete 返回 10011 及已完成分块
  const early = await call(UP + '/file/mpupload/complete', { uploadid: info.UploadID }, bob.token);
  assert.equal(early.json.code, 10011, early.text);
  assert.deepEqual(early.json.data.ChunkExists, [1, 2]);
  // 续传剩余分块
  await mpUpload(bob.token, data, 'resume.dat');
  const r = await call(UP + '/file/mpupload/complete', { uploadid: info.UploadID }, bob.token);
  assert.equal(r.json.code, OK, r.text);
  assert.ok((await listFiles(bob.token)).Files.some((f) => f.FileHash === sha1(data) && f.FileName === 'resume.dat'));
  // 完成后会话被清理, 再次 init 会得到新的会话
  const fresh = await call(UP + '/file/mpupload/init', { filehash: sha1(data), filesize: data.length, filename: 'resume.dat' }, bob.token);
  assert.notEqual(fresh.json.data.UploadID, info.UploadID);
  assert.deepEqual(fresh.json.data.ChunkExists, []);
  await call(UP + '/file/mpupload/cancel', { uploadid: fresh.json.data.UploadID }, bob.token);
});

await test('分块上传: 分块大小错误/序号越界被拒绝', async () => {
  const data = rand(1024 * 1024 + 10);
  const init = await call(UP + '/file/mpupload/init', { filehash: sha1(data), filesize: data.length, filename: 'bad.dat' }, alice.token);
  const id = init.json.data.UploadID;
  const short = await call(`${UP}/file/mpupload/uppart?uploadid=${id}&index=1`, null, alice.token, { body: data.subarray(0, 100) });
  assert.equal(short.json.code, 10010, short.text);
  const over = await call(`${UP}/file/mpupload/uppart?uploadid=${id}&index=3`, null, alice.token, { body: data.subarray(0, 10) });
  assert.equal(over.json.code, 10001, over.text);
  const tooLong = await call(`${UP}/file/mpupload/uppart?uploadid=${id}&index=2`, null, alice.token, { body: rand(11) });
  assert.equal(tooLong.json.code, 10010, tooLong.text);
  await call(UP + '/file/mpupload/cancel', { uploadid: id }, alice.token);
});

await test('分块上传: 内容与声明的hash不一致时合并校验失败 10012', async () => {
  const data = rand(1024 * 1024 + 10);
  const fakeHash = sha1(rand(8));
  const init = await call(UP + '/file/mpupload/init', { filehash: fakeHash, filesize: data.length, filename: 'evil.dat' }, alice.token);
  const id = init.json.data.UploadID;
  for (let i = 1; i <= 2; i++) {
    const r = await call(`${UP}/file/mpupload/uppart?uploadid=${id}&index=${i}`, null, alice.token, { body: data.subarray((i - 1) * 1048576, i * 1048576) });
    assert.equal(r.json.code, OK, r.text);
  }
  const r = await call(UP + '/file/mpupload/complete', { uploadid: id }, alice.token);
  assert.equal(r.json.code, 10012, r.text);
  // 不应写入任何文件记录
  assert.equal((await call(UP + '/file/fastupload', { filehash: fakeHash, filename: 'x', filesize: data.length }, bob.token)).json.code, 10009);
});

await test('分块上传: 不能操作他人的上传任务; uploadid 非法(路径穿越/命令注入)被拒绝', async () => {
  const data = rand(1024 * 1024 + 1);
  const init = await call(UP + '/file/mpupload/init', { filehash: sha1(data), filesize: data.length, filename: 'mine.dat' }, alice.token);
  const id = init.json.data.UploadID;
  const r = await call(`${UP}/file/mpupload/uppart?uploadid=${id}&index=1`, null, bob.token, { body: data.subarray(0, 1048576) });
  assert.equal(r.json.code, 10001, r.text);
  const c = await call(UP + '/file/mpupload/complete', { uploadid: id }, bob.token);
  assert.equal(c.json.code, 10001, c.text);
  for (const bad of ['../../etc', 'x; touch /tmp/pwned', id + '/../x']) {
    const rr = await call(`${UP}/file/mpupload/uppart?uploadid=${encodeURIComponent(bad)}&index=1`, null, alice.token, { body: Buffer.from('x') });
    assert.equal(rr.json.code, 10001, rr.text);
  }
  assert.equal((await call(UP + '/file/mpupload/cancel', { uploadid: id }, alice.token)).json.code, OK);
  assert.equal((await call(UP + '/file/mpupload/cancel', { uploadid: id }, alice.token)).json.code, 10001);
});

await test('分块上传: 并发上传分块及重复上传同一分块', async () => {
  const data = rand(5 * 1024 * 1024 + 3);
  const h = sha1(data);
  const init = await call(UP + '/file/mpupload/init', { filehash: h, filesize: data.length, filename: 'concurrent.dat' }, bob.token);
  const { UploadID: id, ChunkCount: n, ChunkSize: cs } = init.json.data;
  const jobs = [];
  for (let i = 1; i <= n; i++) {
    const part = data.subarray((i - 1) * cs, i * cs);
    jobs.push(call(`${UP}/file/mpupload/uppart?uploadid=${id}&index=${i}`, null, bob.token, { body: part }));
    if (i === 2) jobs.push(call(`${UP}/file/mpupload/uppart?uploadid=${id}&index=${i}`, null, bob.token, { body: part }));
  }
  const results = await Promise.all(jobs);
  results.forEach((r) => assert.equal(r.json.code, OK, r.text));
  const [c1, c2] = await Promise.all([
    call(UP + '/file/mpupload/complete', { uploadid: id }, bob.token),
    call(UP + '/file/mpupload/complete', { uploadid: id }, bob.token)
  ]);
  const codes = [c1.json.code, c2.json.code].sort();
  assert.ok(codes.includes(OK), JSON.stringify([c1.json, c2.json]));
  const u = await getDownloadURL(bob.token, h);
  assert.equal(sha1((await fetchBytes(u.json.data.URL)).buf), h);
});

// ------------------------------------------------------------------ 静态页面
console.log('\n[静态资源]');
await test('页面及脚本可访问, 无外部CDN依赖', async () => {
  for (const p of ['signin', 'signup', 'home', 'upload', 'download']) {
    const r = await fetch(`${GW}/static/view/${p}.html`);
    assert.equal(r.status, 200, p);
    const html = await r.text();
    assert.ok(!/src="https?:\/\//.test(html) && !/href="https?:\/\//.test(html), p + ' has external deps');
    for (const m of html.matchAll(/(?:src|href)="(\/static\/[^"]+)"/g)) {
      const a = await fetch(GW + m[1]);
      assert.equal(a.status, 200, m[1]);
    }
  }
  assert.equal((await fetch(GW + '/static/js/hash-worker.js')).status, 200);
  assert.equal((await fetch(GW + '/user/signin', { redirect: 'manual' })).status, 302);
});

// ------------------------------------------------------------------ 退出登录
console.log('\n[退出登录]');
await test('退出登录后token失效', async () => {
  const r = await call(GW + '/user/signout', {}, alice.token);
  assert.equal(r.json.code, OK);
  assert.equal((await call(GW + '/user/info', {}, alice.token)).json.code, 10005);
  assert.equal((await upload(alice.token, rand(10), 'x')).json.code, 10005);
  assert.equal((await getDownloadURL(alice.token, bigHash)).json.code, 10005);
});

console.log(`\n${passed} passed, ${failures.length} failed`);
if (failures.length) {
  console.log('FAILED:\n - ' + failures.join('\n - '));
  process.exit(1);
}
