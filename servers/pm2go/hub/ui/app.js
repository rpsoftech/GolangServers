'use strict';
// ---------- helpers ----------
const root = document.getElementById('root');
function h(tag, attrs, ...kids) {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (k === 'class') e.className = v;
    else if (k === 'on') for (const [ev, fn] of Object.entries(v)) e.addEventListener(ev, fn);
    else if (k === 'style') e.style.cssText = v;
    else if (v === true) e.setAttribute(k, '');
    else if (v !== false && v != null) e.setAttribute(k, v);
  }
  for (const kid of kids.flat(Infinity)) {
    if (kid == null || kid === false) continue;
    e.append(kid.nodeType ? kid : document.createTextNode(String(kid)));
  }
  return e;
}
async function api(path, opts = {}) {
  const res = await fetch(path, { credentials: 'same-origin', ...opts,
    headers: opts.body ? { 'Content-Type': 'application/json' } : undefined,
    body: opts.body ? JSON.stringify(opts.body) : undefined });
  if (res.status === 401 && path !== '/api/login') { showLogin(); throw new Error('unauthorized'); }
  if (!res.ok) throw new Error((await res.text()).trim() || res.statusText);
  return res.status === 204 ? null : res.json();
}
function toast(msg, bad) {
  const t = h('div', { class: 'toast' + (bad ? ' bad' : '') }, msg);
  document.body.append(t);
  setTimeout(() => t.remove(), bad ? 6000 : 2500);
}
const fmtBytes = b => b >= 2 ** 30 ? (b / 2 ** 30).toFixed(1) + ' GB' : b >= 2 ** 20 ? (b / 2 ** 20).toFixed(0) + ' MB' : b >= 1024 ? (b / 1024).toFixed(0) + ' KB' : (b || 0) + ' B';
const fmtRate = b => fmtBytes(b) + '/s';
function fmtDur(ms) {
  const s = Math.floor(ms / 1000);
  if (s < 60) return s + 's'; if (s < 3600) return Math.floor(s / 60) + 'm';
  if (s < 172800) return Math.floor(s / 3600) + 'h ' + Math.floor(s % 3600 / 60) + 'm';
  return Math.floor(s / 86400) + 'd';
}
const ago = ts => ts ? fmtDur(Date.now() - ts) + ' ago' : 'never';
const clock = ts => new Date(ts).toLocaleTimeString();
const dt = ts => new Date(ts).toLocaleString();
const level = v => v >= 90 ? 'c' : v >= 70 ? 'w' : '';
function meter(pct) { return h('div', { class: 'meter' }, h('i', { class: level(pct), style: `width:${Math.min(100, Math.max(0, pct))}%` })); }
function badge(text, cls) { return h('span', { class: 'badge ' + (cls || text) }, text); }
function statusCls(s) { return s === 'waiting restart' ? 'waiting' : s; }

let timers = [], sockets = [];
function clearPage() { timers.forEach(clearInterval); timers = []; sockets.forEach(s => s.close()); sockets = []; }
function every(ms, fn) { fn(); timers.push(setInterval(fn, ms)); }

// ---------- charts ----------
function lineChart({ title, series, unit = '', max = null, fmt = v => v.toFixed(1) }) {
  const W = 420, H = 130, L = 38, R = 8, T = 6, B = 18;
  const pts = series.flatMap(s => s.points);
  const box = h('div', { class: 'chart' });
  const latest = series.map(s => s.points.length ? s.points[s.points.length - 1].v : null).filter(v => v != null);
  box.append(h('div', { class: 't' }, h('span', {}, title), h('b', {}, latest.length ? fmt(latest[0]) + unit : '–')));
  if (pts.length < 2) { box.append(h('div', { class: 'empty' }, 'collecting data…')); return box; }
  const t0 = Math.min(...pts.map(p => p.t)), t1 = Math.max(...pts.map(p => p.t));
  let top = max ?? Math.max(...pts.map(p => p.v)) * 1.15; if (top <= 0) top = 1;
  const x = t => L + (t1 === t0 ? 0 : (t - t0) / (t1 - t0)) * (W - L - R);
  const y = v => T + (1 - Math.min(v, top) / top) * (H - T - B);
  const NS = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(NS, 'svg'); svg.setAttribute('viewBox', `0 0 ${W} ${H}`); svg.setAttribute('width', '100%');
  const el = (n, a, txt) => { const e = document.createElementNS(NS, n); for (const k in a) e.setAttribute(k, a[k]); if (txt != null) e.textContent = txt; svg.append(e); return e; };
  for (let i = 0; i <= 3; i++) {
    const v = top * i / 3, yy = y(v);
    el('line', { x1: L, x2: W - R, y1: yy, y2: yy, stroke: '#2a313c', 'stroke-width': 1 });
    el('text', { x: L - 4, y: yy + 3, 'text-anchor': 'end' }, fmt(v));
  }
  el('text', { x: L, y: H - 4 }, clock(t0)); el('text', { x: W - R, y: H - 4, 'text-anchor': 'end' }, clock(t1));
  for (const s of series) {
    if (s.points.length < 2) continue;
    const d = s.points.map((p, i) => (i ? 'L' : 'M') + x(p.t).toFixed(1) + ' ' + y(p.v).toFixed(1)).join(' ');
    el('path', { d, fill: 'none', stroke: s.color, 'stroke-width': 1.6, 'stroke-linejoin': 'round' });
  }
  box.append(svg);
  if (series.length > 1) box.append(h('div', { class: 'legend' }, series.map(s => h('span', {}, h('i', { style: 'background:' + s.color }), s.label))));
  return box;
}
const C = { blue: '#4f8cff', green: '#3fb950', amber: '#d29922', red: '#f85149', purple: '#a371f7' };
const pointsOf = (arr, f) => (arr || []).map(p => ({ t: p.ts, v: f(p) }));

// ---------- login / shell ----------
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
const NAV = [['#/', 'Servers'], ['#/issues', 'Issues'], ['#/alerts', 'Alerts'], ['#/settings', 'Settings']];
function shell(active) {
  const main = h('main');
  root.replaceChildren(
    h('header', {}, h('span', { class: 'brand' }, 'pm2go'),
      h('nav', {}, NAV.map(([href, label]) => h('a', { href, class: active === href ? 'on' : '' }, label))),
      h('button', { class: 'sm', on: { click: async () => { await api('/api/logout', { method: 'POST' }); showLogin(); } } }, 'Sign out')),
    main);
  return main;
}

// ---------- overview ----------
async function pageOverview() {
  const main = shell('#/');
  const strip = h('div', { class: 'strip' }), grid = h('div', { class: 'grid' });
  main.append(h('h1', {}, 'Servers'), h('div', { class: 'sub' }, 'Hosts reporting to this hub'), strip, grid);
  every(3000, async () => {
    try {
      const [servers, alerts] = await Promise.all([api('/api/servers'), api('/api/alerts/active')]);
      strip.replaceChildren(...alerts.filter(a => a.state === 'firing' && !a.suppressed).slice(0, 5).map(alertRow));
      if (!servers.length) { grid.replaceChildren(h('div', { class: 'empty card' }, 'No servers yet. Run  pm2go link --hub <this url> --key <agent key>  on a host (key is under Settings).')); return; }
      grid.replaceChildren(...servers.map(s => {
        const sy = s.system;
        return h('div', { class: 'card click', on: { click: () => location.hash = '#/server/' + encodeURIComponent(s.name) } },
          h('div', { class: 'row spread' }, h('div', { class: 'row' }, h('span', { class: 'dot ' + (s.online ? 'on' : 'off') }), h('b', {}, s.name)),
            s.alert_level ? badge(s.alert_level) : null),
          h('div', { class: 'sub', style: 'margin:2px 0 10px' }, s.online ? s.hostname : 'offline, last seen ' + ago(s.last_seen)),
          sy && s.online ? [
            h('div', { class: 'row spread' }, h('span', {}, 'CPU'), h('span', {}, sy.cpu.toFixed(0) + '%')), meter(sy.cpu),
            h('div', { class: 'row spread' }, h('span', {}, 'Memory'), h('span', {}, `${sy.mem_percent.toFixed(0)}% of ${fmtBytes(sy.mem_total)}`)), meter(sy.mem_percent),
            h('div', { class: 'row spread' }, h('span', {}, 'Disk'), h('span', {}, sy.disk_percent.toFixed(0) + '%')), meter(sy.disk_percent),
          ] : null,
          h('div', { class: 'row', style: 'margin-top:4px' }, h('span', { class: 'sub', style: 'margin:0' }, `${s.processes} processes`),
            s.online_processes ? badge(s.online_processes + ' online', 'online') : null,
            s.errored_processes ? badge(s.errored_processes + ' errored', 'errored') : null));
      }));
    } catch (e) { if (e.message !== 'unauthorized') toast(e.message, true); }
  });
}
function alertRow(a) {
  return h('div', { class: 'alert ' + a.level }, badge(a.level), h('b', {}, a.rule_name),
    h('span', {}, a.server + (a.process ? ' / ' + a.process : '')), h('span', { class: 'd' }, a.detail), h('span', { class: 'sub', style: 'margin:0' }, ago(a.since)));
}

// ---------- server page ----------
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
async function cmd(server, action, target, extra = {}) {
  try { await api(`/api/servers/${encodeURIComponent(server)}/command`, { method: 'POST', body: { action, target, ...extra } }); toast(`${action} ${target}: ok`); }
  catch (e) { toast(`${action} ${target}: ${e.message}`, true); }
}
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
      h('td', { class: 'row' },
        h('button', { class: 'sm', on: { click: () => cmd(server, 'restart', String(p.id)) } }, 'Restart'),
        h('button', { class: 'sm', on: { click: () => cmd(server, 'reload', String(p.id)) } }, 'Reload'),
        p.status === 'stopped' || p.status === 'errored'
          ? h('button', { class: 'sm', on: { click: () => cmd(server, 'restart', String(p.id)) } }, 'Start')
          : h('button', { class: 'sm', on: { click: () => cmd(server, 'stop', String(p.id)) } }, 'Stop'),
        h('button', { class: 'sm danger', on: { click: () => confirm(`Delete ${p.name} #${p.instance}?`) && cmd(server, 'delete', String(p.id)) } }, 'Delete'))))));
}

// ---------- process page ----------
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

// ---------- issues ----------
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

// ---------- alerts ----------
const RULE_TYPES = {
  system_cpu: ['Server CPU %', 'threshold'], system_memory: ['Server memory %', 'threshold'], system_disk: ['Server disk %', 'threshold'],
  system_load: ['Server load per core ×100', 'threshold'], proc_cpu: ['Process CPU %', 'threshold'], proc_memory: ['Process memory MB', 'threshold'],
  proc_status: ['Process errored', ''], restart_loop: ['Restart loop (crashes in window)', 'count'], log_errors: ['Error lines in logs', 'log'], server_offline: ['Server offline', '']
};
async function pageAlerts() {
  const main = shell('#/alerts');
  const active = h('div'), rules = h('div'), hist = h('div');
  main.append(h('div', { class: 'row' }, h('h1', {}, 'Alerts'), h('span', { class: 'right' }, h('button', { class: 'primary', on: { click: () => ruleForm(null, load) } }, 'New rule'))),
    h('div', { class: 'sub' }, 'Rules fire after the condition holds for the set duration. A higher level covers lower ones for the same target.'),
    h('h2', {}, 'Active'), active, h('h2', {}, 'Rules'), rules, h('h2', {}, 'History'), hist);
  async function load() {
    try {
      const [act, rl, hi] = await Promise.all([api('/api/alerts/active'), api('/api/alerts/rules'), api('/api/alerts/history?limit=50')]);
      active.replaceChildren(act.length ? h('div', { class: 'strip' }, act.map(a => h('div', { class: 'alert ' + a.level }, badge(a.level), badge(a.state), a.suppressed ? badge('covered', 'ignored') : null,
        h('b', {}, a.rule_name), h('span', {}, a.server + (a.process ? ' / ' + a.process : '')), h('span', { class: 'd' }, a.detail), h('span', { class: 'sub', style: 'margin:0' }, ago(a.since))))) : h('div', { class: 'empty card' }, 'Nothing firing'));
      rules.replaceChildren(h('div', { class: 'tablewrap' }, h('table', {},
        h('tr', {}, ['On', 'Level', 'Name', 'Condition', 'Scope', 'Repeat', ''].map(x => h('th', {}, x))),
        rl.map(r => h('tr', {}, h('td', {}, h('input', { type: 'checkbox', checked: r.enabled, on: { change: async e => { await api('/api/alerts/rules/' + r.id, { method: 'PUT', body: { ...r, enabled: e.target.checked } }); } } })),
          h('td', {}, badge(r.level)), h('td', {}, r.name), h('td', {}, describeRule(r)), h('td', {}, `${r.server || '*'} / ${r.process || '*'}`),
          h('td', {}, r.repeat_s ? 'every ' + fmtDur(r.repeat_s * 1000) : '–'),
          h('td', { class: 'row' }, h('button', { class: 'sm', on: { click: () => ruleForm(r, load) } }, 'Edit'),
            h('button', { class: 'sm danger', on: { click: async () => { if (confirm('Delete rule ' + r.name + '?')) { await api('/api/alerts/rules/' + r.id, { method: 'DELETE' }); load(); } } } }, 'Delete')))))));
      hist.replaceChildren(hi.length ? h('div', { class: 'tablewrap' }, h('table', {}, h('tr', {}, ['Time', 'State', 'Level', 'Rule', 'Target', 'Detail', 'Telegram'].map(x => h('th', {}, x))),
        hi.map(e => h('tr', {}, h('td', {}, dt(e.ts)), h('td', {}, badge(e.state)), h('td', {}, badge(e.level)), h('td', {}, e.rule_name),
          h('td', {}, e.server + (e.process ? ' / ' + e.process : '')), h('td', {}, e.message), h('td', {}, e.notified ? 'sent' : '–'))))) : h('div', { class: 'empty card' }, 'No history yet'));
    } catch (e) { if (e.message !== 'unauthorized') toast(e.message, true); }
  }
  every(5000, load);
}
function describeRule(r) {
  const d = r.duration_s ? ` for ${fmtDur(r.duration_s * 1000)}` : '';
  switch (r.type) {
    case 'system_cpu': case 'system_memory': case 'system_disk': case 'proc_cpu': return `${RULE_TYPES[r.type][0]} > ${r.threshold}${d}`;
    case 'proc_memory': return `memory > ${r.threshold} MB${d}`;
    case 'system_load': return `load/core > ${r.threshold}%${d}`;
    case 'restart_loop': return `${r.count} crashes in ${fmtDur(r.window_s * 1000)}`;
    case 'log_errors': return `${r.count} log lines in ${fmtDur(r.window_s * 1000)}` + (r.pattern ? ` matching /${r.pattern}/` : '') + (r.stream ? ` (${r.stream})` : '');
    case 'proc_status': return 'process errored' + d;
    case 'server_offline': return 'no data' + (r.duration_s ? ' for ' + fmtDur(r.duration_s * 1000) : '');
  }
  return r.type;
}
async function ruleForm(rule, done) {
  const r = rule || { name: '', enabled: true, level: 'warning', type: 'proc_cpu', server: '', process: '', threshold: 80, duration_s: 300, window_s: 300, count: 5, pattern: '', stream: '', repeat_s: 0, notify_resolved: true, chats: [] };
  const tg = await api('/api/settings/telegram');
  const f = {};
  const mk = (key, type = 'text') => { if (!f[key]) f[key] = h('input', { type, value: r[key] ?? '' }); return f[key]; };
  const inp = (key, label, type = 'text', cls = '') => h('div', { class: cls }, h('label', {}, label), mk(key, type));
  f.level = h('select', {}, ['info', 'warning', 'critical'].map(l => h('option', { value: l, selected: l === r.level }, l)));
  f.type = h('select', { on: { change: sync } }, Object.entries(RULE_TYPES).map(([k, [lbl]]) => h('option', { value: k, selected: k === r.type }, lbl)));
  f.stream = h('select', {}, [['', 'both'], ['out', 'stdout'], ['err', 'stderr']].map(([v, l]) => h('option', { value: v, selected: v === r.stream }, l)));
  f.enabled = h('input', { type: 'checkbox', checked: r.enabled }); f.nr = h('input', { type: 'checkbox', checked: r.notify_resolved });
  const chatBoxes = tg.chats.map(c => ({ c, box: h('input', { type: 'checkbox', checked: (r.chats || []).includes(c.chat_id) }) }));
  const groups = {
    threshold: () => [inp('threshold', 'Threshold', 'number'), inp('duration_s', 'Must hold for (seconds)', 'number')],
    count: () => [inp('count', 'Crash count', 'number'), inp('window_s', 'Window (seconds)', 'number')],
    log: () => [inp('count', 'Matching lines', 'number'), inp('window_s', 'Window (seconds)', 'number'), inp('pattern', 'Regex (empty = error/fatal/panic/exception)'), h('div', {}, h('label', {}, 'Stream'), f.stream)],
    dur: () => [inp('duration_s', 'Must hold for (seconds)', 'number')],
  };
  const dyn = h('div', { class: 'form full', style: 'grid-column:1/-1' });
  function sync() {
    const kind = RULE_TYPES[f.type.value][1];
    dyn.replaceChildren(...(kind === 'threshold' ? groups.threshold() : kind === 'count' ? groups.count() : kind === 'log' ? groups.log() : groups.dur()));
  }
  const modal = h('div', { class: 'modal', on: { click: e => { if (e.target === modal) modal.remove(); } } }, h('div', { class: 'card' },
    h('h1', {}, rule ? 'Edit rule' : 'New rule'),
    h('div', { class: 'form' }, inp('name', 'Name', 'text', 'full'), h('div', {}, h('label', {}, 'Level'), f.level), h('div', {}, h('label', {}, 'Type'), f.type),
      inp('server', 'Server (empty = all)'), inp('process', 'Process (empty = all)'), dyn,
      inp('repeat_s', 'Remind every (seconds, 0 = never)', 'number'),
      h('div', { class: 'full' }, h('label', { class: 'check' }, f.enabled, 'Enabled'), h('label', { class: 'check' }, f.nr, 'Send a message when resolved')),
      h('div', { class: 'full' }, h('label', {}, 'Telegram chats (none ticked = every chat whose minimum level allows it)'),
        chatBoxes.length ? chatBoxes.map(({ c, box }) => h('label', { class: 'check' }, box, `${c.name || c.chat_id} (min ${c.min_level})`)) : h('div', { class: 'sub' }, 'No chats configured under Settings yet.'))),
    h('div', { class: 'row', style: 'margin-top:14px' }, h('button', { class: 'primary', on: { click: async () => {
      const kind = RULE_TYPES[f.type.value][1];
      const vis = { threshold: ['threshold', 'duration_s'], count: ['count', 'window_s'], log: ['count', 'window_s', 'pattern'], '': ['duration_s'] }[kind] || [];
      const num = k => (f[k] && vis.includes(k) ? Number(f[k].value) || 0 : (k === 'repeat_s' ? Number(f[k].value) || 0 : 0));
      const body = { name: f.name.value.trim(), enabled: f.enabled.checked, level: f.level.value, type: f.type.value, server: f.server.value.trim(), process: f.process.value.trim(),
        threshold: num('threshold'), duration_s: num('duration_s'), window_s: num('window_s'), count: num('count'), pattern: kind === 'log' && f.pattern ? f.pattern.value : '', stream: kind === 'log' ? f.stream.value : '',
        repeat_s: num('repeat_s'), notify_resolved: f.nr.checked, chats: chatBoxes.filter(x => x.box.checked).map(x => x.c.chat_id) };
      try { await api(rule ? '/api/alerts/rules/' + rule.id : '/api/alerts/rules', { method: rule ? 'PUT' : 'POST', body }); modal.remove(); done(); }
      catch (e) { toast(e.message, true); } } } }, 'Save'), h('button', { on: { click: () => modal.remove() } }, 'Cancel'))));
  document.body.append(modal);
  sync();
}

// ---------- settings ----------
async function pageSettings() {
  const main = shell('#/settings');
  const [info, tg] = await Promise.all([api('/api/settings/info'), api('/api/settings/telegram')]);
  const origin = location.origin;
  const enabled = h('input', { type: 'checkbox', checked: tg.enabled });
  const token = h('input', { type: 'password', placeholder: tg.token_set ? '•••••••• (saved, leave blank to keep)' : 'Bot token from @BotFather', style: 'width:100%' });
  const chats = [...tg.chats];
  const chatBox = h('div');
  function drawChats() {
    chatBox.replaceChildren(...chats.map((c, i) => h('div', { class: 'row', style: 'margin-bottom:8px' },
      h('input', { placeholder: 'Name', value: c.name, on: { input: e => c.name = e.target.value } }),
      h('input', { placeholder: 'Chat ID', value: c.chat_id, on: { input: e => c.chat_id = e.target.value } }),
      h('select', { on: { change: e => c.min_level = e.target.value } }, ['info', 'warning', 'critical'].map(l => h('option', { value: l, selected: l === c.min_level }, 'min ' + l))),
      h('button', { class: 'sm', on: { click: async () => { try { await api('/api/settings/telegram/test', { method: 'POST', body: { chat_id: c.chat_id } }); toast('Test sent'); } catch (e) { toast(e.message, true); } } } }, 'Send test'),
      h('button', { class: 'sm danger', on: { click: () => { chats.splice(i, 1); drawChats(); } } }, 'Remove'))));
  }
  drawChats();
  main.append(h('h1', {}, 'Settings'),
    h('h2', {}, 'Connect a server'), h('div', { class: 'card' }, h('div', { class: 'sub' }, 'Run on each host (needs the pm2go binary), then the host appears under Servers:'),
      h('div', { class: 'mono', style: 'user-select:all;background:var(--bg);padding:8px;border-radius:6px' }, `pm2go link --hub ${origin} --key ${info.agent_key} --name <machine-name>`)),
    h('h2', {}, 'Telegram notifications'), h('div', { class: 'card' },
      h('label', { class: 'check' }, enabled, 'Send alerts to Telegram'),
      h('label', {}, 'Bot token'), token,
      h('label', {}, 'Chats (a user, group or channel the bot can write to). Each chat gets alerts at or above its minimum level.'), chatBox,
      h('div', { class: 'row' }, h('button', { on: { click: () => { chats.push({ name: '', chat_id: '', min_level: 'warning' }); drawChats(); } } }, 'Add chat'),
        h('button', { class: 'primary', on: { click: async () => {
          try { await api('/api/settings/telegram', { method: 'PUT', body: { enabled: enabled.checked, bot_token: token.value, chats } }); toast('Saved'); pageSettings(); }
          catch (e) { toast(e.message, true); } } } }, 'Save'))),
    h('div', { class: 'sub', style: 'margin-top:12px' }, `Metrics, events and alert history are kept for ${info.retention_days} days.`));
}

// ---------- router ----------
async function route() {
  clearPage();
  const parts = (location.hash || '#/').slice(2).split('/').map(decodeURIComponent);
  try {
    await api('/api/me');
    if (parts[0] === 'server' && parts[2] === 'proc') return await pageProc(parts[1], parts[3]);
    if (parts[0] === 'server') return await pageServer(parts[1]);
    if (parts[0] === 'issues') return await pageIssues();
    if (parts[0] === 'alerts') return await pageAlerts();
    if (parts[0] === 'settings') return await pageSettings();
    return await pageOverview();
  } catch (e) { if (e.message !== 'unauthorized') toast(e.message, true); }
}
window.addEventListener('hashchange', route);
route();
