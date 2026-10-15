(function (App) {
  'use strict';

  var form = App.$('#signin-form');
  var alertBox = App.$('#form-alert');
  var btn = App.$('#submit-btn');
  var fields = {
    username: App.$('#username'),
    password: App.$('#password')
  };
  var homeURL = '/static/view/home.html';
  var redirect = App.safeRedirect(App.queryParam('redirect'), homeURL);
  if (redirect.indexOf('/static/view/signin.html') === 0 || redirect.indexOf('/static/view/signup.html') === 0) {
    redirect = homeURL;
  }

  function showAlert(msg, type) {
    alertBox.textContent = msg;
    alertBox.className = 'form-alert show' + (type ? ' ' + type : '');
  }

  function setFieldError(name, msg) {
    var input = fields[name];
    App.$('#' + name + '-error').textContent = msg || '';
    if (msg) input.setAttribute('aria-invalid', 'true');
    else input.removeAttribute('aria-invalid');
  }

  // 已登录则直接进入首页
  var reason = App.queryParam('reason');
  if (!reason && App.Session.isActive()) {
    window.location.replace(redirect);
    return;
  }
  if (reason === 'expired') {
    showAlert('登录已失效, 请重新登录');
  } else if (reason === 'signup') {
    showAlert('注册成功, 请登录', 'success');
  }
  var presetUser = App.queryParam('username');
  if (presetUser) {
    fields.username.value = presetUser;
    fields.password.focus();
  }

  Object.keys(fields).forEach(function (name) {
    fields[name].addEventListener('input', function () { setFieldError(name, ''); });
  });

  form.addEventListener('submit', function (e) {
    e.preventDefault();
    if (btn.disabled) return;

    var username = fields.username.value.trim();
    var password = fields.password.value;
    setFieldError('username', username ? '' : '请输入用户名');
    setFieldError('password', password ? '' : '请输入密码');
    if (!username) return fields.username.focus();
    if (!password) return fields.password.focus();

    alertBox.className = 'form-alert';
    App.setBusy(btn, true, '登录中...');
    App.api('/user/signin', { username: username, password: password }, { auth: false })
      .then(function (body) {
        App.Session.save(body.data);
        window.location.replace(redirect);
      })
      .catch(function (err) {
        App.setBusy(btn, false);
        showAlert(err.message || '登录失败');
        if (err.code === App.Code.LOGIN_FAILED) {
          fields.password.select();
          fields.password.focus();
        }
      });
  });
})(window.App);
