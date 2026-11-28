'use strict';
// Login screen.
function showLogin() {
  clearPage(); root.replaceChildren();
  const pw = h('input', { type: 'password', placeholder: 'Admin password', autofocus: true });
  const go = async () => {
    try { await api('/api/login', { method: 'POST', body: { password: pw.value } }); route(); }
    catch (e) { toast(e.message, true); }
  };
  pw.addEventListener('keydown', e => { if (e.key === 'Enter') go(); });
  root.append(h('div', { class: 'login card' }, h('h1', {}, 'pm2go'), h('div', { class: 'sub' }, 'Sign in to the dashboard'),
    pw, h('div', { style: 'margin-top:12px' }, h('button', { class: 'primary', on: { click: go } }, 'Sign in'))));
}
