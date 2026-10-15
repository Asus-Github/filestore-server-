(function (App) {
  'use strict';

  var form = App.$('#signup-form');
  var alertBox = App.$('#form-alert');
  var btn = App.$('#submit-btn');
  var fields = {
    username: App.$('#username'),
    password: App.$('#password'),
    passwordc: App.$('#passwordc')
  };

  function showAlert(msg, type) {
    alertBox.textContent = msg;
    alertBox.className = 'form-alert show' + (type ? ' ' + type : '');
  }

  function setFieldError(name, msg) {
    var input = fields[name];
    App.$('#' + name + '-error').textContent = msg || '';
    if (msg) input.setAttribute('aria-invalid', 'true');
    else input.removeAttribute('aria-invalid');
    return !msg;
  }

  function validate(name) {
    var v = fields[name].value;
    switch (name) {
      case 'username':
        return setFieldError(name, App.Validate.username(v.trim()));
      case 'password':
        return setFieldError(name, App.Validate.password(v));
      case 'passwordc':
        if (!v) return setFieldError(name, '请再次输入密码');
        return setFieldError(name, v === fields.password.value ? '' : '两次输入的密码不一致');
    }
    return true;
  }

  Object.keys(fields).forEach(function (name) {
    fields[name].addEventListener('input', function () { setFieldError(name, ''); });
    fields[name].addEventListener('blur', function () {
      if (fields[name].value) validate(name);
    });
  });

  form.addEventListener('submit', function (e) {
    e.preventDefault();
    if (btn.disabled) return;

    alertBox.className = 'form-alert';
    var firstInvalid = null;
    Object.keys(fields).forEach(function (name) {
      if (!validate(name) && !firstInvalid) firstInvalid = fields[name];
    });
    if (firstInvalid) {
      firstInvalid.focus();
      return;
    }

    var username = fields.username.value.trim();
    App.setBusy(btn, true, '注册中...');
    App.api('/user/signup', { username: username, password: fields.password.value }, { auth: false })
      .then(function () {
        showAlert('注册成功, 正在跳转到登录页...', 'success');
        window.setTimeout(function () {
          window.location.replace('/static/view/signin.html?reason=signup&username=' + encodeURIComponent(username));
        }, 1000);
      })
      .catch(function (err) {
        App.setBusy(btn, false);
        if (err.code === App.Code.USER_EXISTS) {
          setFieldError('username', err.message);
          fields.username.focus();
          return;
        }
        showAlert(err.message || '注册失败');
      });
  });
})(window.App);
