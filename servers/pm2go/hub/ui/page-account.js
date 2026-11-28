'use strict';
// Account page (#/account): change password, end other sessions, manage your API tokens.
async function pageAccount() {
  const main = shell('#/account');
  const tokens = h('div');
  const cur = h('input', { type: 'password', autocomplete: 'current-password' });
  const n1 = h('input', { type: 'password', autocomplete: 'new-password' });
  const n2 = h('input', { type: 'password', autocomplete: 'new-password' });
  const name = h('input', { placeholder: 'Name, e.g. deploy script' });
  const cap = h('select', {}, ['viewer', 'operator', 'admin'].filter(r => can(r)).reverse().map(r => h('option', { value: r }, r)));
  const days = h('input', { type: 'number', min: 0, max: 3650, value: 90, style: 'width:90px' });
  const secret = h('div');

  async function load() {
    const list = await api('/api/tokens');
    tokens.replaceChildren(list.length ? h('div', { class: 'tablewrap' }, h('table', {},
      h('tr', {}, ['Name', 'Token', 'Role cap', 'Created', 'Last used', 'Expires', ''].map(x => h('th', {}, x))),
      list.map(t => h('tr', {}, h('td', {}, t.name), h('td', { class: 'mono' }, t.prefix + '…'), h('td', {}, badge(t.role_cap)),
        h('td', {}, dt(t.created_at)), h('td', {}, t.last_used_at ? ago(t.last_used_at) : 'never'), h('td', {}, t.expires_at ? dt(t.expires_at) : 'never'),
        h('td', {}, h('button', { class: 'sm danger', on: { click: async () => {
          if (!confirm('Revoke token ' + t.name + '? Scripts using it stop working.')) return;
          try { await api('/api/tokens/' + t.id, { method: 'DELETE' }); load(); } catch (e) { toast(e.message, true); } } } }, 'Revoke')))))) : h('div', { class: 'empty card' }, 'No API tokens'));
  }
  main.append(h('h1', {}, 'Account'), h('div', { class: 'sub' }, `${ME.username}, role ${ME.role}`),
    h('h2', {}, 'Change password'), h('div', { class: 'card' }, h('div', { class: 'form' },
      h('div', {}, h('label', {}, 'Current password'), cur), h('div', {}, h('label', {}, 'New password (8+ characters)'), n1), h('div', {}, h('label', {}, 'Repeat new password'), n2)),
      h('div', { class: 'row', style: 'margin-top:12px' }, h('button', { class: 'primary', on: { click: async () => {
        if (n1.value !== n2.value) return toast('The new passwords do not match', true);
        try { await api('/api/password', { method: 'POST', body: { current: cur.value, new: n1.value } }); cur.value = n1.value = n2.value = ''; toast('Password changed, your other sessions were ended'); }
        catch (e) { toast(e.message, true); } } } }, 'Change password'),
        h('button', { on: { click: async () => {
          try { await api('/api/sessions/revoke-all', { method: 'POST' }); toast('Other sessions ended'); } catch (e) { toast(e.message, true); } } } }, 'Sign out everywhere else'))),
    h('h2', {}, 'API tokens'),
    h('div', { class: 'sub' }, 'For scripts: send "Authorization: Bearer <token>". A token has at most your role, or the cap you choose. The secret is shown once.'),
    h('div', { class: 'card' }, h('div', { class: 'form' },
      h('div', {}, h('label', {}, 'Name'), name), h('div', {}, h('label', {}, 'Role cap'), cap), h('div', {}, h('label', {}, 'Expires in days (0 = never)'), days)),
      h('div', { style: 'margin-top:12px' }, h('button', { class: 'primary', on: { click: async () => {
        try {
          const r = await api('/api/tokens', { method: 'POST', body: { name: name.value.trim(), role_cap: cap.value, expires_days: Number(days.value) || 0 } });
          name.value = '';
          secret.replaceChildren(h('div', { class: 'sub', style: 'margin-top:12px' }, 'Copy this token now. It cannot be shown again.'), h('div', { class: 'once mono' }, r.token));
          load();
        } catch (e) { toast(e.message, true); } } } }, 'Create token')), secret),
    h('div', { style: 'margin-top:12px' }), tokens);
  await load();
}
registerPage('account', pageAccount);
