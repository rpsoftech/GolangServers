'use strict';
// Core: DOM helper, API client, formatters, page lifecycle, nav and router registries, shell.
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
async function cmd(server, action, target, extra = {}) {
  try { await api(`/api/servers/${encodeURIComponent(server)}/command`, { method: 'POST', body: { action, target, ...extra } }); toast(`${action} ${target}: ok`); }
  catch (e) { toast(`${action} ${target}: ${e.message}`, true); }
}

// ---------- nav and shell ----------
// Nav order follows registration order, which follows <script> order in index.html.
// registerNav(href, label, minRole): the link is shown only to users whose role is at least minRole
// (the server enforces roles anyway; hiding is only a convenience).
const NAV = [];
function registerNav(href, label, minRole = 'viewer') { NAV.push([href, label, minRole]); }
// ME is the signed-in user from /api/me: { username, role, effective_role, must_change_password, ... }.
let ME = null;
const ROLE_RANK = { viewer: 1, operator: 2, admin: 3 };
const can = min => !!ME && ROLE_RANK[ME.effective_role] >= ROLE_RANK[min];
function shell(active) {
  const main = h('main');
  root.replaceChildren(
    h('header', {}, h('span', { class: 'brand' }, 'pm2go'),
      h('nav', {}, NAV.filter(n => can(n[2])).map(([href, label]) => h('a', { href, class: active === href ? 'on' : '' }, label))),
      h('a', { href: '#/account', class: 'who' + (active === '#/account' ? ' on' : ''), title: 'Account, password and API tokens' }, ME ? `${ME.username} (${ME.effective_role})` : ''),
      h('button', { class: 'sm', on: { click: async () => { await api('/api/logout', { method: 'POST' }); ME = null; showLogin(); } } }, 'Sign out')),
    main);
  return main;
}

// ---------- router ----------
// registerPage('server/:name/proc/:proc', fn) calls fn(name, proc). A pattern matches when its
// segments are a prefix of the hash path; the pattern with most segments wins. '' matches anything
// and is the fallback page.
const PAGES = [];
function registerPage(pattern, fn) {
  PAGES.push({ segs: pattern ? pattern.split('/') : [], fn });
  PAGES.sort((a, b) => b.segs.length - a.segs.length);
}
function matchPage(parts) {
  for (const p of PAGES) {
    if (p.segs.length > parts.length && p.segs.length) continue;
    const args = [];
    if (p.segs.every((s, i) => s[0] === ':' ? (args.push(parts[i]), true) : s === parts[i])) return { fn: p.fn, args };
  }
  return null;
}
async function route() {
  clearPage();
  const parts = (location.hash || '#/').slice(2).split('/').map(decodeURIComponent);
  try {
    ME = await api('/api/me');
    if (ME.must_change_password) return showPasswordChange();
    const m = matchPage(parts);
    if (m) return await m.fn(...m.args);
  } catch (e) { if (e.message !== 'unauthorized') toast(e.message, true); }
}
