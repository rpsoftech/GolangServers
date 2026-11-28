'use strict';
// Alerts page and rule editor modal (#/alerts).
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
registerNav('#/alerts', 'Alerts');
registerPage('alerts', pageAlerts);
