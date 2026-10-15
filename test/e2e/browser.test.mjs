// 浏览器端到端测试(真实Chrome): node test/e2e/browser.test.mjs
// 前置: 已启动全部服务; 本机安装了 Google Chrome (或设置 CHROME_PATH)
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import assert from 'node:assert/strict';
import { chromium } from 'playwright-core';

const GW = process.env.GW || 'http://127.0.0.1:18080';
const CHROME = process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const HEADLESS = process.env.HEADED ? false : true;
const ART = path.join(path.dirname(new URL(import.meta.url).pathname), 'artifacts');
fs.mkdirSync(ART, { recursive: true });
const TMP = fs.mkdtempSync(path.join(os.tmpdir(), 'fs-e2e-'));

const sha1 = (buf) => crypto.createHash('sha1').update(buf).digest('hex');
let passed = 0;
const failures = [];
const pageErrors = [];

const browser = await chromium.launch({ executablePath: CHROME, headless: HEADLESS });
const context = await browser.newContext({ acceptDownloads: true, viewport: { width: 1280, height: 860 } });
const page = await context.newPage();
page.setDefaultTimeout(20000);
page.on('pageerror', (e) => pageErrors.push('pageerror: ' + e.message));
page.on('console', (m) => {
  if (m.type() === 'error' && !/401|Failed to load resource/.test(m.text())) pageErrors.push('console: ' + m.text());
});
page.on('dialog', (d) => {
  pageErrors.push('unexpected native dialog: ' + d.message());
  d.dismiss();
});

async function test(name, fn) {
  try {
    if (await page.locator('.modal-mask').count()) await page.keyboard.press('Escape');
    await fn();
    passed++;
    console.log('  ✓ ' + name);
  } catch (e) {
    failures.push(name);
    const shot = path.join(ART, failures.length + '.png');
    await page.screenshot({ path: shot, fullPage: true }).catch(() => {});
    console.log('  ✗ ' + name + '\n      ' + String(e && e.stack ? e.stack.split('\n').slice(0, 4).join('\n      ') : e) + '\n      screenshot: ' + shot);
  }
}

function makeFile(name, size) {
  const p = path.join(TMP, name);
  const buf = crypto.randomBytes(size);
  fs.writeFileSync(p, buf);
  return { path: p, name, size, hash: sha1(buf), buf };
}

async function toastText() {
  return page.locator('.toast').allInnerTexts();
}

async function waitToast(re) {
  await page.waitForFunction((src) => {
    const r = new RegExp(src);
    return Array.from(document.querySelectorAll('.toast')).some((t) => r.test(t.textContent));
  }, re.source);
}

// 按路径等待跳转(不能用正则匹配整个URL: 登录页的 redirect 参数中也包含目标页面名)
async function waitPath(p, opts) {
  await page.waitForURL((u) => u.pathname === p, opts);
}

// Chrome保存文件时会把文件名中的非法字符替换为 _
const chromeSafeName = (name) => name.replace(/[<>:"/\\|?*]/g, '_');

async function rowByName(name) {
  return page.locator('#file-rows tr', { has: page.locator('.fname', { hasText: name }) });
}

const user = 'web' + crypto.randomBytes(3).toString('hex');
const password = 'Secret123';

console.log('Browser E2E against', GW, HEADLESS ? '(headless)' : '(headed)');

// ------------------------------------------------------------------ 未登录访问
await test('未登录访问首页 -> 跳转登录页并携带redirect', async () => {
  await page.goto(GW + '/static/view/home.html');
  await page.waitForURL(/signin\.html\?redirect=/);
  assert.match(await page.locator('#form-alert').innerText(), /登录已失效/);
});

// ------------------------------------------------------------------ 注册
await test('注册页: 前端校验(用户名过短/两次密码不一致)', async () => {
  await page.goto(GW + '/static/view/signup.html');
  await page.fill('#username', 'ab');
  await page.fill('#password', 'Secret123');
  await page.fill('#passwordc', 'Secret124');
  await page.click('#submit-btn');
  assert.match(await page.locator('#username-error').innerText(), /3~32位/);
  assert.match(await page.locator('#passwordc-error').innerText(), /不一致/);
  assert.equal(await page.locator('#username').getAttribute('aria-invalid'), 'true');
});

await test('注册成功后跳转登录页并预填用户名', async () => {
  await page.fill('#username', user);
  await page.fill('#passwordc', password);
  await page.click('#submit-btn');
  await waitPath('/static/view/signin.html');
  await page.waitForURL(/reason=signup/);
  assert.equal(await page.inputValue('#username'), user);
  assert.match(await page.locator('#form-alert').innerText(), /注册成功/);
});

await test('重复注册提示用户名已被注册', async () => {
  await page.goto(GW + '/static/view/signup.html');
  await page.fill('#username', user);
  await page.fill('#password', password);
  await page.fill('#passwordc', password);
  await page.click('#submit-btn');
  await page.waitForFunction(() => /已被注册/.test(document.querySelector('#username-error').textContent));
});

// ------------------------------------------------------------------ 登录
await test('登录: 错误密码提示, 按钮恢复可用', async () => {
  await page.goto(GW + '/static/view/signin.html');
  await page.fill('#username', user);
  await page.fill('#password', 'wrong-pass');
  await page.press('#password', 'Enter');
  await page.waitForFunction(() => /用户名或密码错误/.test(document.querySelector('#form-alert').textContent));
  assert.equal(await page.locator('#submit-btn').isDisabled(), false);
});

await test('登录成功 -> 首页显示用户信息与空文件列表', async () => {
  await page.fill('#password', password);
  await page.click('#submit-btn');
  await waitPath('/static/view/home.html');
  await page.waitForFunction((u) => document.querySelector('#username').textContent === u, user);
  await page.waitForFunction(() => /\d{4}-\d{2}-\d{2}/.test(document.querySelector('#regtime').textContent));
  await page.locator('#file-empty').waitFor({ state: 'visible' });
  assert.equal(await page.locator('#filecount').innerText(), '0');
  const token = await page.evaluate(() => localStorage.getItem('token'));
  assert.match(token, /^[0-9a-f]{64}$/);
});

await test('已登录访问登录页 -> 直接进入首页', async () => {
  await page.goto(GW + '/static/view/signin.html');
  await waitPath('/static/view/home.html');
});

// ------------------------------------------------------------------ 上传
// 测试环境: FS_CHUNK_SIZE=1MB; 前端 <=16MB 走普通上传, 更大走分块上传
const xssName = '<img src=x onerror="window.__xss=1">报告.txt';
const fSmall = makeFile('small.bin', 200 * 1024 + 3);
const fXss = { ...makeFile('xss.tmp', 1234) };
const xssPath = path.join(TMP, xssName);
fs.renameSync(fXss.path, xssPath);
fXss.path = xssPath;
fXss.name = xssName;
const fBig = makeFile('big-multipart.bin', 18 * 1024 * 1024 + 777);

async function uploadViaPage(files) {
  await page.goto(GW + '/static/view/upload.html');
  await page.setInputFiles('#file', files.map((f) => f.path));
  for (const f of files) {
    const task = page.locator('.task', { has: page.locator('.tname', { hasText: f.name }) });
    await task.locator('.status.success').waitFor({ timeout: 120000 });
  }
}

await test('上传页: 普通上传 + 含HTML的文件名(多文件同时选择)', async () => {
  await uploadViaPage([fSmall, fXss]);
  const statuses = await page.locator('.task .status').allInnerTexts();
  assert.ok(statuses.every((s) => /上传成功/.test(s)), statuses.join('|'));
  assert.ok(statuses.some((s) => /普通上传/.test(s)));
  await waitToast(/全部上传完成/);
});

await test('上传页: 大文件分块上传, 进度条到100%', async () => {
  await uploadViaPage([fBig]);
  const task = page.locator('.task', { has: page.locator('.tname', { hasText: fBig.name }) });
  assert.match(await task.locator('.status').innerText(), /分块上传/);
  assert.equal(await task.locator('.progress').getAttribute('aria-valuenow'), '100');
});

await test('上传页: 再次上传同一文件 -> 秒传', async () => {
  await uploadViaPage([fSmall]);
  const task = page.locator('.task', { has: page.locator('.tname', { hasText: fSmall.name }) });
  assert.match(await task.locator('.status').innerText(), /秒传/);
});

await test('上传页: 分块上传暂停/继续(断点续传)', async () => {
  const f = makeFile('pause-resume.bin', 24 * 1024 * 1024 + 5);
  await page.goto(GW + '/static/view/upload.html');
  // 限速, 保证有时间点击暂停
  const cdp = await context.newCDPSession(page);
  await cdp.send('Network.enable');
  await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 20, downloadThroughput: -1, uploadThroughput: 3 * 1024 * 1024 });
  await page.setInputFiles('#file', f.path);
  const task = page.locator('.task', { has: page.locator('.tname', { hasText: f.name }) });
  await page.waitForFunction(() => {
    const p = document.querySelector('.task .progress');
    const v = p ? Number(p.getAttribute('aria-valuenow')) : 0;
    // 进度包含正在上传中的分块(最多3个=12.5%), 等待到30%以上保证已有分块完成
    return /分块上传中/.test(document.querySelector('.task .status').textContent) && v >= 30 && v <= 80;
  }, null, { timeout: 60000 });
  await task.locator('button', { hasText: '暂停' }).click();
  await task.locator('.status.paused').waitFor();
  const pausedAt = Number(await task.locator('.progress').getAttribute('aria-valuenow'));
  assert.ok(pausedAt > 0 && pausedAt < 100, 'paused at ' + pausedAt);
  assert.match(await task.locator('.detail').innerText(), /已上传 .* 点击继续可断点续传/);
  await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
  await task.locator('button', { hasText: '继续' }).click();
  await task.locator('.status.success').waitFor({ timeout: 120000 });
  assert.match(await task.locator('.detail, .meta').first().innerText(), /SHA1|断点续传|100/);
  fBig.resumed = f;
});

await test('上传页: 取消上传后可移除任务', async () => {
  const f = makeFile('cancel-me.bin', 20 * 1024 * 1024);
  await page.goto(GW + '/static/view/upload.html');
  const cdp = await context.newCDPSession(page);
  await cdp.send('Network.enable');
  await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 20, downloadThroughput: -1, uploadThroughput: 1024 * 1024 });
  await page.setInputFiles('#file', f.path);
  const task = page.locator('.task', { has: page.locator('.tname', { hasText: f.name }) });
  await page.waitForFunction(() => /上传中/.test(document.querySelector('.task .status').textContent), null, { timeout: 60000 });
  await task.locator('button', { hasText: '取消' }).click();
  await task.locator('.status', { hasText: '已取消' }).waitFor();
  await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
  await task.locator('button', { hasText: '移除' }).click();
  assert.equal(await page.locator('.task').count(), 0);
});

// ------------------------------------------------------------------ 文件列表
await test('首页文件列表: 显示上传的文件, 文件名按文本渲染(无XSS)', async () => {
  await page.goto(GW + '/static/view/home.html');
  await page.locator('#file-rows tr').first().waitFor();
  const names = await page.locator('#file-rows .fname').allInnerTexts();
  for (const f of [fSmall, fXss, fBig, fBig.resumed]) assert.ok(names.includes(f.name), 'missing ' + f.name + ' in ' + names);
  assert.ok(!names.includes('cancel-me.bin'));
  assert.equal(await page.evaluate(() => window.__xss), undefined);
  assert.equal(await page.locator('#file-rows img').count(), 0);
  assert.equal(await page.locator('#filecount').innerText(), '4');
  const sizeText = await (await rowByName(fBig.name)).locator('td').nth(1).innerText();
  assert.match(sizeText, /18\.0 MB/);
});

// ------------------------------------------------------------------ 下载
await test('普通下载: 文件内容与上传一致, 文件名正确(含特殊字符)', async () => {
  for (const f of [fSmall, fXss]) {
    const row = await rowByName(f.name);
    const [dl] = await Promise.all([page.waitForEvent('download'), row.locator('button', { hasText: /^下载$/ }).click()]);
    const p = await dl.path();
    assert.equal(sha1(fs.readFileSync(p)), f.hash, f.name);
    assert.equal(dl.suggestedFilename(), chromeSafeName(f.name));
  }
});

await test('断点下载弹窗: 分段下载, 暂停/继续, 校验通过后保存文件', async () => {
  // 先限速再打开弹窗, 否则本机传输18MB几乎瞬间完成, 来不及暂停
  const cdp = await context.newCDPSession(page);
  await cdp.send('Network.enable');
  await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 20, downloadThroughput: 4 * 1024 * 1024, uploadThroughput: -1 });
  const row = await rowByName(fBig.name);
  await row.locator('button', { hasText: '断点下载' }).click();
  const frame = page.frameLocator('.modal iframe');
  await frame.locator('#dl-toggle', { hasText: '暂停' }).waitFor();
  await page.waitForFunction(() => {
    const f = document.querySelector('.modal iframe');
    const p = f && f.contentDocument && f.contentDocument.querySelector('#dl-progress');
    const v = p ? Number(p.getAttribute('aria-valuenow')) : 0;
    return v >= 10 && v <= 80;
  }, null, { timeout: 30000 });
  await frame.locator('#dl-toggle').click();
  await frame.locator('#dl-status', { hasText: '已暂停' }).waitFor();
  const pausedAt = Number(await frame.locator('#dl-progress').getAttribute('aria-valuenow'));
  assert.ok(pausedAt > 0 && pausedAt < 100, 'paused at ' + pausedAt);
  await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
  const [dl] = await Promise.all([
    page.waitForEvent('download', { timeout: 120000 }),
    frame.locator('#dl-toggle').click()
  ]);
  await frame.locator('#dl-status', { hasText: '校验通过' }).waitFor();
  assert.equal(sha1(fs.readFileSync(await dl.path())), fBig.hash);
  assert.equal(dl.suggestedFilename(), fBig.name);
  await page.keyboard.press('Escape');
  assert.equal(await page.locator('.modal').count(), 0);
});

await test('断点下载页: 独立打开, 下载完成', async () => {
  const p2 = await context.newPage();
  p2.on('pageerror', (e) => pageErrors.push('pageerror(dl): ' + e.message));
  const [dl] = await Promise.all([
    p2.waitForEvent('download', { timeout: 60000 }),
    p2.goto(GW + '/static/view/download.html?filehash=' + fSmall.hash)
  ]);
  assert.equal(sha1(fs.readFileSync(await dl.path())), fSmall.hash);
  assert.equal(await p2.locator('#dl-back').isVisible(), true);
  await p2.close();
});

// ------------------------------------------------------------------ 重命名 / 删除
await test('重命名: 对话框校验, 取消不生效, 确认后列表更新', async () => {
  let row = await rowByName(fSmall.name);
  await row.locator('button', { hasText: '重命名' }).click();
  await page.locator('#prompt-input').waitFor();
  await page.fill('#prompt-input', 'a/b.txt');
  await page.locator('.modal-footer .btn-primary').click();
  assert.match(await page.locator('.modal .error').innerText(), /不能包含/);
  await page.locator('.modal-footer .btn', { hasText: '取消' }).click();
  assert.ok(await (await rowByName(fSmall.name)).count() === 1);

  row = await rowByName(fSmall.name);
  await row.locator('button', { hasText: '重命名' }).click();
  await page.fill('#prompt-input', '重命名后 & "引号".bin');
  await page.press('#prompt-input', 'Enter');
  await waitToast(/重命名成功/);
  await (await rowByName('重命名后 & "引号".bin')).waitFor();
  fSmall.name = '重命名后 & "引号".bin';
});

await test('删除: 确认框取消不删除, 确认后删除', async () => {
  let row = await rowByName(fXss.name);
  await row.locator('button', { hasText: '删除' }).click();
  await page.locator('.modal-footer .btn', { hasText: '取消' }).click();
  assert.equal(await (await rowByName(fXss.name)).count(), 1);
  row = await rowByName(fXss.name);
  await row.locator('button', { hasText: '删除' }).click();
  await page.locator('.modal-footer .btn-danger').click();
  await waitToast(/已删除/);
  await page.waitForFunction(() => document.querySelector('#filecount').textContent === '3');
  assert.equal(await (await rowByName(fXss.name)).count(), 0);
});

await test('刷新页面后状态保持', async () => {
  await page.reload();
  await page.locator('#file-rows tr').first().waitFor();
  assert.equal(await page.locator('#file-rows tr').count(), 3);
});

// ------------------------------------------------------------------ 退出登录 / token失效
await test('token在服务端失效 -> 操作时自动跳转登录页, 登录后回到原页面', async () => {
  const token = await page.evaluate(() => localStorage.getItem('token'));
  await page.evaluate(async (t) => {
    await fetch('/user/signout', { method: 'POST', headers: { Authorization: 'Bearer ' + t } });
  }, token);
  await page.goto(GW + '/static/view/upload.html');
  await page.setInputFiles('#file', fSmall.path);
  await page.waitForURL(/signin\.html\?redirect=/);
  await page.fill('#username', user);
  await page.fill('#password', password);
  await page.click('#submit-btn');
  await waitPath('/static/view/upload.html');
  await page.locator('#dropzone').waitFor();
});

await test('退出登录 -> 回到登录页, 本地登录状态被清除', async () => {
  await page.goto(GW + '/static/view/home.html');
  await page.locator('#signout-btn').click();
  await waitPath('/static/view/signin.html');
  assert.equal(await page.evaluate(() => localStorage.getItem('token')), null);
  await page.goto(GW + '/static/view/home.html');
  await waitPath('/static/view/signin.html');
});

await test('登录页 redirect 参数不允许跳转到外部站点', async () => {
  await page.goto(GW + '/static/view/signin.html?redirect=' + encodeURIComponent('//evil.example.com/x'));
  await page.fill('#username', user);
  await page.fill('#password', password);
  await page.click('#submit-btn');
  await page.waitForURL((u) => u.origin === GW && u.pathname === '/static/view/home.html');
});

await test('移动端视口下页面可用', async () => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(GW + '/static/view/home.html');
  await page.locator('#file-rows tr').first().waitFor();
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  assert.ok(overflow <= 1, 'horizontal overflow ' + overflow);
  await page.screenshot({ path: path.join(ART, 'home-mobile.png'), fullPage: true });
  await page.setViewportSize({ width: 1280, height: 860 });
  await page.screenshot({ path: path.join(ART, 'home-desktop.png'), fullPage: true });
});

await test('全程无JS运行时错误/原生弹窗', async () => {
  assert.deepEqual(pageErrors, []);
});

await browser.close();
fs.rmSync(TMP, { recursive: true, force: true });
console.log(`\n${passed} passed, ${failures.length} failed`);
if (failures.length) {
  console.log('FAILED:\n - ' + failures.join('\n - '));
  process.exit(1);
}
