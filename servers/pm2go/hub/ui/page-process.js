'use strict';
// Process detail page with live logs (#/server/<name>/proc/<proc>).
async function pageProc(server, name) {
  const main = shell('#/');
  let range = '1h';
  const info = h('div'), charts = h('div', { class: 'charts' });
  const logs = h('div', { class: 'logs' });
  main.append(h('div', { class: 'sub' }, h('a', { href: '#/server/' + encodeURIComponent(server) }, '← ' + server)), info,
    h('div', { class: 'row', style: 'margin:14px 0 8px' }, h('h2', { style: 'margin:0' }, 'Metrics'),
      h('span', { class: 'right' }, h('select', { on: { change: e => { range = e.target.value; loadCharts(); } } }, ['15m', '1h', '6h', '24h', '7d'].map(r => h('option', { value: r, selected: r === range }, r))))),
    charts, h('div', { class: 'row', style: 'margin:14px 0 8px' }, h('h2', { style: 'margin:0' }, 'Live logs'),
      h('span', { class: 'right' }, h('button', { class: 'sm', on: { click: () => cmd(server, 'flush', name) } }, 'Flush logs'))), logs);
  async function loadCharts() {
    try {
      const m = await api(`/api/servers/${encodeURIComponent(server)}/metrics?range=${range}`);
      const pr = m.procs[name];
      charts.replaceChildren(
        lineChart({ title: 'CPU (all instances)', unit: '%', fmt: v => v.toFixed(1), series: [{ label: 'cpu', color: C.blue, points: pointsOf(pr, p => p.cpu) }] }),
        lineChart({ title: 'Memory', fmt: fmtBytes, series: [{ label: 'rss', color: C.purple, points: pointsOf(pr, p => p.mem) }] }),
        lineChart({ title: 'Restarts (cumulative)', fmt: v => v.toFixed(0), series: [{ label: 'restarts', color: C.amber, points: pointsOf(pr, p => p.restarts) }] }));
    } catch (e) { if (e.message !== 'unauthorized') toast(e.message, true); }
  }
  every(3000, async () => {
    try {
      const s = await api('/api/servers/' + encodeURIComponent(server));
      const list = (s.snapshot?.processes || []).filter(p => p.name === name);
      info.replaceChildren(h('div', { class: 'row' }, h('h1', {}, name),
        h('span', { class: 'right row' },
          h('button', { on: { click: () => cmd(server, 'restart', name) } }, 'Restart all'),
          h('button', { on: { click: () => cmd(server, 'reload', name) } }, 'Reload'),
          h('button', { on: { click: () => { const n = parseInt(prompt('Number of instances', String(list.length)), 10); if (n >= 0) cmd(server, 'scale', name, { count: n }); } } }, 'Scale'))),
        list.length ? procTable(server, list) : h('div', { class: 'empty card' }, 'Process not found'),
        list[0] ? h('div', { class: 'card', style: 'margin-top:12px' }, h('div', { class: 'kv' },
          ...[['Script', list[0].config.script], ['Args', (list[0].config.args || []).join(' ')], ['Working dir', list[0].config.cwd], ['Autorestart', list[0].config.autorestart === false ? 'off' : 'on'],
            ['Max restarts', list[0].config.max_restarts || 16], ['Max memory restart', list[0].config.max_memory_restart ? list[0].config.max_memory_restart + ' MB' : 'off'],
            ['Cron restart', list[0].config.cron_restart || 'off'], ['Watch', list[0].config.watch ? 'on' : 'off'], ['Out log', list[0].out_log], ['Error log', list[0].err_log]
          ].flatMap(([k, v]) => [h('span', {}, k), h('span', { class: 'mono' }, v)]))) : null);
    } catch (e) { if (e.message !== 'unauthorized') toast(e.message, true); }
  });
  loadCharts(); timers.push(setInterval(loadCharts, 15000));
  // live logs
  const ws = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/api/logs/ws?server=${encodeURIComponent(server)}&proc=${encodeURIComponent(name)}`);
  sockets.push(ws);
  ws.onmessage = ev => {
    const m = JSON.parse(ev.data);
    const stick = logs.scrollTop + logs.clientHeight >= logs.scrollHeight - 30;
    logs.append(h('div', { class: m.stream === 'err' ? 'err' : '' }, h('span', { class: 'ts' }, clock(m.ts) + ' '), m.line));
    while (logs.childElementCount > 1000) logs.firstChild.remove();
    if (stick) logs.scrollTop = logs.scrollHeight;
  };
  ws.onclose = () => logs.append(h('div', { class: 'ts' }, '— log stream closed —'));
}
registerPage('server/:name/proc/:proc', pageProc);
