'use strict';
// Users page (#/users, admin): create accounts, change roles, disable, reset passwords, end sessions.
async function pageUsers() {
  if (!can('admin')) { shell('#/users').append(h('div', { class: 'empty card' }, 'Admins only')); return; }
  const main = shell('#/users');
  const list = h('div');
  const uname = h('input', { placeholder: 'Username', autocomplete: 'off' });
  const upw = h('input', { type: 'password', placeholder: 'Temporary password (8+)', autocomplete: 'new-password' });
  const urole = h('select', {}, ['viewer', 'operator', 'admin'].map(r => h('option', { value: r }, r)));
  const act = async (fn, ok) => { try { await fn(); if (ok) toast(ok); load(); } catch (e) { toast(e.message, true); } };

  async function load() {
    const users = await api('/api/users');
    list.replaceChildren(h('div', { class: 'tablewrap' }, h('table', {},
      h('tr', {}, ['User', 'Role', 'State', 'Last login', 'Created', ''].map(x => h('th', {}, x))),
      users.map(u => h('tr', {}, h('td', {}, u.username, u.id === ME.id ? h('span', { class: 'sub', style: 'margin:0 0 0 6px' }, '(you)') : null),
        h('td', {}, h('select', { on: { change: e => act(() => api('/api/users/' + u.id, { method: 'PUT', body: { role: e.target.value } }), 'Role updated') } },
          ['viewer', 'operator', 'admin'].map(r => h('option', { value: r, selected: r === u.role }, r)))),
        h('td', {}, u.disabled ? badge('disabled') : badge('active', 'online'), u.must_change_password ? badge('must change password', 'waiting') : null),
        h('td', {}, u.last_login_at ? ago(u.last_login_at) : 'never'), h('td', {}, dt(u.created_at)),
        h('td', { class: 'row' },
          h('button', { class: 'sm', on: { click: () => act(() => api('/api/users/' + u.id, { method: 'PUT', body: { disabled: !u.disabled } }), u.disabled ? 'Enabled' : 'Disabled, sessions ended') } }, u.disabled ? 'Enable' : 'Disable'),
          h('button', { class: 'sm', on: { click: () => {
            const pw = prompt('Temporary password for ' + u.username + ' (8+ characters). They must change it at next sign-in and all their sessions end.');
            if (pw) act(() => api('/api/users/' + u.id + '/password', { method: 'POST', body: { password: pw } }), 'Password reset');
          } } }, 'Reset password'),
          h('button', { class: 'sm', on: { click: () => act(() => api('/api/users/' + u.id + '/revoke-sessions', { method: 'POST' }), 'Sessions ended') } }, 'End sessions'),
          h('button', { class: 'sm danger', on: { click: () => confirm('Delete user ' + u.username + ' and their API tokens?') && act(() => api('/api/users/' + u.id, { method: 'DELETE' }), 'Deleted') } }, 'Delete')))))));
  }
  main.append(h('h1', {}, 'Users'),
    h('div', { class: 'sub' }, 'viewer: read only. operator: viewer plus process commands and issue status. admin: everything, including rules, settings, users and the audit log.'),
    list, h('h2', {}, 'New user'), h('div', { class: 'card' }, h('div', { class: 'form' },
      h('div', {}, h('label', {}, 'Username'), uname), h('div', {}, h('label', {}, 'Temporary password'), upw), h('div', {}, h('label', {}, 'Role'), urole)),
      h('div', { style: 'margin-top:12px' }, h('button', { class: 'primary', on: { click: () => act(async () => {
        await api('/api/users', { method: 'POST', body: { username: uname.value.trim(), password: upw.value, role: urole.value } }); uname.value = upw.value = '';
      }, 'User created, they must change the password at first sign-in') } }, 'Create user'))));
  await load();
}
registerNav('#/users', 'Users', 'admin');
registerPage('users', pageUsers);
