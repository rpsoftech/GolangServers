'use strict';
// Audit log (#/audit, admin): who did what, from where. Never shows secrets (the server never stores them).
async function pageAudit() {
  if (!can('admin')) { shell('#/audit').append(h('div', { class: 'empty card' }, 'Admins only')); return; }
  const main = shell('#/audit');
  const table = h('div');
  const user = h('input', { placeholder: 'User' });
  const action = h('input', { placeholder: 'Action, e.g. login or user.*' });
  const failed = h('input', { type: 'checkbox' });
  let rows = [];
  const query = before => '/api/audit?limit=100' + (user.value ? '&user=' + encodeURIComponent(user.value.trim()) : '') +
    (action.value ? '&action=' + encodeURIComponent(action.value.trim()) : '') + (failed.checked ? '&failed=1' : '') + (before ? '&before=' + before : '');
  function draw(more) {
    table.replaceChildren(...[rows.length ? h('div', { class: 'tablewrap' }, h('table', {},
      h('tr', {}, ['Time', 'User', 'IP', 'Action', 'Target', 'Detail', 'Result'].map(x => h('th', {}, x))),
      rows.map(e => h('tr', {}, h('td', {}, dt(e.ts)), h('td', {}, e.user || '–'), h('td', { class: 'mono' }, e.ip), h('td', {}, e.action),
        h('td', {}, e.target), h('td', {}, e.detail), h('td', {}, e.success ? badge('ok', 'online') : badge('failed', 'errored')))))) : h('div', { class: 'empty card' }, 'No matching entries'),
      more ? h('div', { style: 'margin-top:10px' }, h('button', { on: { click: async () => { const r = await api(query(rows[rows.length - 1].id)); rows = rows.concat(r); draw(r.length === 100); } } }, 'Older')) : null].filter(Boolean));
  }
  async function load() { try { rows = await api(query()); draw(rows.length === 100); } catch (e) { toast(e.message, true); } }
  main.append(h('h1', {}, 'Audit log'), h('div', { class: 'sub' }, 'Sign-ins, commands, rule, settings, user and token changes. Kept for the retention period.'),
    h('div', { class: 'row', style: 'margin-bottom:12px' }, user, action, h('label', { class: 'check', style: 'margin:0' }, failed, 'failures only'),
      h('button', { on: { click: load } }, 'Search')), table);
  await load();
}
registerNav('#/audit', 'Audit', 'admin');
registerPage('audit', pageAudit);
