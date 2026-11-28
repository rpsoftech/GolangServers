'use strict';
// Issues page (#/issues).
async function pageIssues() {
  const main = shell('#/issues');
  let status = 'open';
  const body = h('div');
  const tabs = h('div', { class: 'tabs' });
  main.append(h('h1', {}, 'Issues'), h('div', { class: 'sub' }, 'Error lines from process logs, grouped by shape'), tabs, body);
  async function load() {
    tabs.replaceChildren(...['open', 'resolved', 'ignored', ''].map(s => h('button', { class: status === s ? 'on' : '', on: { click: () => { status = s; load(); } } }, s || 'all')));
    try {
      const list = await api('/api/issues' + (status ? '?status=' + status : ''));
      body.replaceChildren(list.length ? h('div', { class: 'tablewrap' }, h('table', {},
        h('tr', {}, ['Last seen', 'Server / process', 'Error', 'Count', 'Status', ''].map((x, i) => h('th', { class: i === 3 ? 'num' : '' }, x))),
        list.map(i => h('tr', {}, h('td', {}, ago(i.last_seen)), h('td', {}, `${i.server} / ${i.name}`),
          h('td', { class: 'mono', style: 'max-width:560px;word-break:break-all' }, i.sample), h('td', { class: 'num' }, i.count), h('td', {}, badge(i.status)),
          h('td', { class: 'row' }, ['resolved', 'ignored', 'open'].filter(s => s !== i.status).map(s => h('button', { class: 'sm', on: { click: async () => {
            await api('/api/issues/status', { method: 'POST', body: { server: i.server, name: i.name, fingerprint: i.fingerprint, status: s } }); load(); } } }, s))))))) : h('div', { class: 'empty card' }, 'No issues'));
    } catch (e) { if (e.message !== 'unauthorized') toast(e.message, true); }
  }
  every(10000, load);
}
registerNav('#/issues', 'Issues');
registerPage('issues', pageIssues);
