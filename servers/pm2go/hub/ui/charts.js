'use strict';
// Line chart and chart colours.
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
