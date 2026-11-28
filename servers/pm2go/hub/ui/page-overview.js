'use strict';
// Overview page (#/).
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
registerNav('#/', 'Servers');
registerPage('', pageOverview);
