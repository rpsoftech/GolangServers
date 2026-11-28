'use strict';
// Login screen and the forced password change that follows a bootstrap or reset password.
function showLogin() {
  clearPage(); root.replaceChildren(); ME = null;
  const user = h('input', { placeholder: 'Username', autocomplete: 'username', autofocus: true, style: 'width:100%' });
  const pw = h('input', { type: 'password', placeholder: 'Password', autocomplete: 'current-password', style: 'width:100%;margin-top:8px' });
  const go = async () => {
    try { await api('/api/login', { method: 'POST', body: { username: user.value.trim(), password: pw.value } }); route(); }
    catch (e) { toast(e.message, true); }
  };
  for (const i of [user, pw]) i.addEventListener('keydown', e => { if (e.key === 'Enter') go(); });
  root.append(h('div', { class: 'login card' }, h('h1', {}, 'pm2go'), h('div', { class: 'sub' }, 'Sign in to the dashboard'),
    user, pw, h('div', { style: 'margin-top:12px' }, h('button', { class: 'primary', on: { click: go } }, 'Sign in'))));
}

// Shown instead of any page while the account still has a temporary password.
function showPasswordChange() {
  clearPage(); root.replaceChildren();
  const cur = h('input', { type: 'password', placeholder: 'Current password', autocomplete: 'current-password', style: 'width:100%' });
  const n1 = h('input', { type: 'password', placeholder: 'New password (8+ characters)', autocomplete: 'new-password', style: 'width:100%;margin-top:8px' });
  const n2 = h('input', { type: 'password', placeholder: 'Repeat new password', autocomplete: 'new-password', style: 'width:100%;margin-top:8px' });
  const go = async () => {
    if (n1.value !== n2.value) return toast('The new passwords do not match', true);
    try { await api('/api/password', { method: 'POST', body: { current: cur.value, new: n1.value } }); toast('Password changed'); route(); }
    catch (e) { toast(e.message, true); }
  };
  n2.addEventListener('keydown', e => { if (e.key === 'Enter') go(); });
  root.append(h('div', { class: 'login card' }, h('h1', {}, 'Choose a new password'),
    h('div', { class: 'sub' }, `Signed in as ${ME.username}. The current password is temporary and must be changed before you continue.`),
    cur, n1, n2, h('div', { class: 'row', style: 'margin-top:12px' }, h('button', { class: 'primary', on: { click: go } }, 'Change password'),
      h('button', { on: { click: async () => { await api('/api/logout', { method: 'POST' }); showLogin(); } } }, 'Sign out'))));
}
