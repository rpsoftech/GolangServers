'use strict';
// Settings page (#/settings).
async function pageSettings() {
  const main = shell('#/settings');
  const admin = can('admin');
  const [info, tg] = await Promise.all([api('/api/settings/info'), admin ? api('/api/settings/telegram') : { enabled: false, token_set: false, chats: [] }]);
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
    !admin ? null : [h('h2', {}, 'Connect a server'), h('div', { class: 'card' }, h('div', { class: 'sub' }, 'Run on each host (needs the pm2go binary), then the host appears under Servers:'),
      h('div', { class: 'mono', style: 'user-select:all;background:var(--bg);padding:8px;border-radius:6px' }, `pm2go link --hub ${origin} --key ${info.agent_key} --name <machine-name>`))],
    !admin ? null : [h('h2', {}, 'Telegram notifications'), h('div', { class: 'card' },
      h('label', { class: 'check' }, enabled, 'Send alerts to Telegram'),
      h('label', {}, 'Bot token'), token,
      h('label', {}, 'Chats (a user, group or channel the bot can write to). Each chat gets alerts at or above its minimum level.'), chatBox,
      h('div', { class: 'row' }, h('button', { on: { click: () => { chats.push({ name: '', chat_id: '', min_level: 'warning' }); drawChats(); } } }, 'Add chat'),
        h('button', { class: 'primary', on: { click: async () => {
          try { await api('/api/settings/telegram', { method: 'PUT', body: { enabled: enabled.checked, bot_token: token.value, chats } }); toast('Saved'); pageSettings(); }
          catch (e) { toast(e.message, true); } } } }, 'Save')))],
    h('div', { class: 'sub', style: 'margin-top:12px' }, `Metrics, events and alert history are kept for ${info.retention_days} days.`));
}
registerNav('#/settings', 'Settings');
registerPage('settings', pageSettings);
