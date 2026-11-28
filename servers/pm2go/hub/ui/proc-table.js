'use strict';
// Process table shared by the server and process pages.
function procTable(server, list) {
  const names = [...new Set(list.map(p => p.name))];
  return h('div', { class: 'tablewrap' }, h('table', {},
    h('tr', {}, ['ID', 'Name', 'Status', 'PID', 'Uptime', '↺', 'CPU', 'Memory', 'Actions'].map((x, i) => h('th', { class: i > 2 && i < 8 ? 'num' : '' }, x))),
    list.map(p => h('tr', {},
      h('td', { class: 'mono' }, p.id),
      h('td', {}, h('a', { href: `#/server/${encodeURIComponent(server)}/proc/${encodeURIComponent(p.name)}` }, p.name), p.config.instances > 1 ? ` #${p.instance}` : '',
        p.namespace ? h('span', { class: 'sub', style: 'margin:0 0 0 6px' }, p.namespace) : null),
      h('td', {}, badge(p.status, statusCls(p.status))), h('td', { class: 'num mono' }, p.pid || '–'),
      h('td', { class: 'num' }, p.status === 'online' ? fmtDur(p.uptime_ms) : '–'), h('td', { class: 'num' }, p.restarts),
      h('td', { class: 'num' }, p.status === 'online' ? p.cpu.toFixed(1) + '%' : '–'), h('td', { class: 'num' }, p.status === 'online' ? fmtBytes(p.memory) : '–'),
      h('td', { class: 'row' }, !can('operator') ? null : [
        h('button', { class: 'sm', on: { click: () => cmd(server, 'restart', String(p.id)) } }, 'Restart'),
        h('button', { class: 'sm', on: { click: () => cmd(server, 'reload', String(p.id)) } }, 'Reload'),
        p.status === 'stopped' || p.status === 'errored'
          ? h('button', { class: 'sm', on: { click: () => cmd(server, 'restart', String(p.id)) } }, 'Start')
          : h('button', { class: 'sm', on: { click: () => cmd(server, 'stop', String(p.id)) } }, 'Stop'),
        !can('admin') ? null : h('button', { class: 'sm danger', on: { click: () => confirm(`Delete ${p.name} #${p.instance}?`) && cmd(server, 'delete', String(p.id)) } }, 'Delete')])))));
}
