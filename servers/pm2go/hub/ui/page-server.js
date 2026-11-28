'use strict';
// Server detail page (#/server/<name>).
async function pageServer(name) {
  const main = shell('#/');
  let range = '1h';
  const head = h('div'), charts = h('div', { class: 'charts' }), procs = h('div');
  const rangeSel = h('select', { on: { change: e => { range = e.target.value; loadCharts(); } } },
    ['15m', '1h', '6h', '24h', '7d'].map(r => h('option', { value: r, selected: r === range }, r)));
  main.append(head, h('div', { class: 'row', style: 'margin:14px 0 8px' }, h('h2', { style: 'margin:0' }, 'Host metrics'), h('span', { class: 'right' }, rangeSel)),
    charts, h('h2', {}, 'Processes'), procs, h('h2', {}, 'Recent events'), h('div', { id: 'events' }));
  async function loadCharts() {
    try {
      const m = await api(`/api/servers/${encodeURIComponent(name)}/metrics?range=${range}`);
      charts.replaceChildren(
        lineChart({ title: 'CPU', unit: '%', max: 100, fmt: v => v.toFixed(0), series: [{ label: 'cpu', color: C.blue, points: pointsOf(m.system, p => p.cpu) }] }),
        lineChart({ title: 'Memory', unit: '%', max: 100, fmt: v => v.toFixed(0), series: [{ label: 'mem', color: C.purple, points: pointsOf(m.system, p => p.mem_pct) }] }),
        lineChart({ title: 'Load (1m)', series: [{ label: 'load', color: C.amber, points: pointsOf(m.system, p => p.load1) }], fmt: v => v.toFixed(2) }),
        lineChart({ title: 'Disk', unit: '%', max: 100, fmt: v => v.toFixed(0), series: [{ label: 'disk', color: C.green, points: pointsOf(m.system, p => p.disk_pct) }] }),
        lineChart({ title: 'Network', unit: '/s', fmt: fmtBytes, series: [{ label: 'in', color: C.green, points: pointsOf(m.system, p => p.rx) }, { label: 'out', color: C.blue, points: pointsOf(m.system, p => p.tx) }] }));
    } catch (e) { if (e.message !== 'unauthorized') toast(e.message, true); }
  }
  async function loadEvents() {
    const ev = await api(`/api/events?server=${encodeURIComponent(name)}&limit=15`);
    document.getElementById('events')?.replaceChildren(ev.length ? h('div', { class: 'tablewrap' }, h('table', {},
      h('tr', {}, ['Time', 'Process', 'Event', 'Status', 'Reason', 'Exit'].map(x => h('th', {}, x))),
      ev.map(e => h('tr', {}, h('td', {}, dt(e.ts)), h('td', {}, e.name + (e.instance ? '#' + e.instance : '')), h('td', {}, e.type),
        h('td', {}, badge(e.status, statusCls(e.status))), h('td', {}, e.reason || ''), h('td', {}, e.exit_code ?? ''))))) : h('div', { class: 'empty' }, 'No events yet'));
  }
  every(3000, async () => {
    try {
      const s = await api('/api/servers/' + encodeURIComponent(name));
      const sn = s.snapshot, sy = sn && sn.system;
      head.replaceChildren(h('div', { class: 'row' }, h('span', { class: 'dot ' + (s.online ? 'on' : 'off') }), h('h1', {}, s.name),
        h('span', { class: 'right row' },
          h('button', { on: { click: () => cmd(name, 'save', '') } }, 'Save process list'))),
        h('div', { class: 'sub' }, `${s.hello.hostname} · ${s.hello.os} · ${s.hello.cpu_count} CPU · agent ${s.hello.version}` + (s.online ? '' : ' · offline, last seen ' + ago(s.last_seen)) + (sy ? ` · up ${fmtDur(sy.uptime_s * 1000)}` : '')));
      procs.replaceChildren(sn && sn.processes.length ? procTable(name, sn.processes) : h('div', { class: 'empty card' }, 'No processes'));
    } catch (e) { if (e.message !== 'unauthorized') toast(e.message, true); }
  });
  loadCharts(); timers.push(setInterval(loadCharts, 15000));
  loadEvents(); timers.push(setInterval(loadEvents, 10000));
}
registerPage('server/:name', pageServer);
