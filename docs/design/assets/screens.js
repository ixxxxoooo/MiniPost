/* ==========================================================================
   MiniPost 设计稿原型 · 屏幕预设、顶部控制条与启动
   URL 参数：dir=a|b|c  scheme=mist|ink|jade|amber|rose  mode=light|dark  screen=<id>
            chrome=0（隐藏控制条，画廊嵌入用）  size=compact|std|wide  logo=fold|mono|post|stamp
            以及交互决策 dock/nav/tabs/env/settings/home/fx/quick
   ========================================================================== */
(function () {
  'use strict';
  const MPA = window.MPA, S = MPA.S, M = window.MP, I = window.I, { $, esc } = MPA;

  /* ---------- 屏幕预设 ---------- */
  const tab = (id, view, resp, code, rtab, extra) => Object.assign(MPA.mkTab(id, view || 'params', resp || 'ok', false, code || 200), { rtab: rtab || 'body' }, extra || {});
  const withTabs = (s, list, active) => { s.tabs = list; s.active = active || 0; s.sel = list[active || 0].id; };
  const pop = (s, p) => { s.pop = p; };
  const ov = (s, o) => { s.overlay = o; s.ovAnim = false; };
  const SCREENS = [
    ['工作区', [
      ['welcome', '欢迎页（空状态）', (s) => { s.tabs = []; s.active = -1; s.sel = ''; }],
      ['req-params', '请求 · Params', () => {}],
      ['req-headers', '请求 · Headers', (s) => { s.tabs[0].view = 'headers'; }],
      ['req-body', '请求 · Body（JSON）', (s) => withTabs(s, [tab('r1'), tab('r2', 'body', 'ok', 201), tab('r5', 'params', 'none')], 1)],
      ['req-auth', '请求 · Auth', (s) => { s.tabs[0].view = 'auth'; }],
      ['req-body-form', '请求 · Body（form-data）', (s) => withTabs(s, [tab('r3', 'body', 'none'), tab('r1'), tab('r5', 'params', 'none')], 0)],
      ['req-body-urlenc', '请求 · Body（urlencoded）', (s) => withTabs(s, [tab('r11', 'body', 'none'), tab('r1'), tab('r5', 'params', 'none')], 0)],
      ['req-body-raw', '请求 · Body（raw 文本）', (s) => withTabs(s, [tab('r12', 'body', 'none'), tab('r1'), tab('r5', 'params', 'none')], 0)],
      ['req-auth-basic', '请求 · Auth（Basic）', (s) => withTabs(s, [tab('r4', 'auth', 'none'), tab('r1'), tab('r5', 'params', 'none')], 0)],
      ['req-auth-apikey', '请求 · Auth（API Key）', (s) => withTabs(s, [tab('r9', 'auth', 'none'), tab('r1'), tab('r5', 'params', 'none')], 0)],
      ['req-dirty', '请求 · 未保存的修改', (s) => { s.tabs[0].dirty = true; s.tabs[2].dirty = true; }],
    ]],
    ['响应', [
      ['resp-ok', '响应 · 成功（JSON）', () => {}],
      ['resp-headers', '响应 · 响应头', (s) => { s.tabs[0].rtab = 'headers'; }],
      ['resp-cookies', '响应 · Cookies', (s) => { s.tabs[0].rtab = 'cookies'; }],
      ['resp-timing', '响应 · 时间分解', (s) => pop(s, { type: 'timing', anchor: '.resp-h .meta .mi', align: 'right' })],
      ['resp-sending', '响应 · 发送中', (s) => { s.tabs[0].resp = 'sending'; }],
      ['resp-large', '响应 · 超大响应（截断卡片）', (s) => withTabs(s, [tab('r1', 'params', 'ok', 9200), tab('r2', 'body', 'none'), tab('r5', 'params', 'none')], 0)],
      ['resp-filter', '响应 · JSON 过滤', (s) => { s.tabs[0].filter = true; }],
      ['resp-error', '响应 · 失败诊断', (s) => withTabs(s, [tab('r8', 'params', 'error'), tab('r1'), tab('r5', 'params', 'none')], 0)],
      ['resp-422', '响应 · 4xx 业务错误', (s) => withTabs(s, [tab('r6', 'body', 'ok', 422), tab('r1'), tab('r5', 'params', 'none')], 0)],
      ['resp-204', '响应 · 空响应（204）', (s) => withTabs(s, [tab('r4', 'params', 'ok', 204), tab('r1'), tab('r5', 'params', 'none')], 0)],
      ['resp-empty', '响应 · 尚未发送', (s) => { s.tabs[0].resp = 'none'; }],
      ['resp-stream', '响应 · SSE 事件流', (s) => withTabs(s, [tab('r7', 'body', 'stream', 200, 'stream'), tab('r1'), tab('r5', 'params', 'none')], 0)],
      ['resp-merged', '响应 · 流式合并文本', (s) => withTabs(s, [tab('r7', 'body', 'stream', 200, 'stream', { merged: true }), tab('r1'), tab('r5', 'params', 'none')], 0)],
    ]],
    ['侧栏与页面', [
      ['hist', '侧栏 · 历史', (s) => { s.sideTab = 'hist'; }],
      ['env-list', '侧栏 · 环境列表', (s) => { s.sideTab = 'env'; }],
      ['env-page', '环境编辑页', (s) => { s.sideTab = 'env'; s.tabs.push({ kind: 'env', id: 'e1' }); s.active = 3; }],
      ['home-grid', '项目主页 · 卡片', (s) => { s.view = 'home'; s.opt.home = 'grid'; }],
      ['home-list', '项目主页 · 列表', (s) => { s.view = 'home'; s.opt.home = 'list'; }],
      ['console', '控制台', (s) => { s.con.open = true; s.con.ex = 0; }],
      ['side-search', '侧栏 · 搜索过滤', (s) => { s.q = 'user'; s.sel = 'r1'; }],
      ['home-empty', '项目主页 · 零项目', (s) => { s.view = 'home'; s.opt.home = 'grid'; s.noProj = true; }],
      ['side-empty', '侧栏 · 还没有请求', (s) => { s.emptyTree = true; s.tabs = []; s.active = -1; s.sel = ''; }],
      ['tb-update', '标题栏 · 更新胶囊（下载中）', (s) => { s.update = 'downloading'; }],
      ['side-collapsed', '侧栏已收起', (s) => { s.sideCollapsed = true; }],
    ]],
    ['设置', [
      ['set-general', '设置 · 通用', (s) => { s.setPage = 'general'; MPA.openSettings(s); }],
      ['set-appearance', '设置 · 外观', (s) => { s.setPage = 'appearance'; MPA.openSettings(s); }],
      ['set-data', '设置 · 数据', (s) => { s.setPage = 'data'; MPA.openSettings(s); }],
      ['set-shortcuts', '设置 · 快捷键', (s) => { s.setPage = 'shortcuts'; MPA.openSettings(s); }],
      ['set-about', '设置 · 关于', (s) => { s.setPage = 'about'; MPA.openSettings(s); }],
      ['set-update', '设置 · 关于（下载更新中）', (s) => { s.setPage = 'about'; s.upd = 'downloading'; MPA.openSettings(s); }],
    ]],
    ['导入与数据', [
      ['imp-file', '导入 · 文件（初始）', (s) => { s.imp = { tab: 'file', ok: false, strat: 'merge', over: false }; ov(s, { type: 'import' }); }],
      ['imp-drag', '导入 · 拖入悬停', (s) => { s.imp = { tab: 'file', ok: false, strat: 'merge', over: true }; ov(s, { type: 'import' }); }],
      ['imp-ok', '导入 · 识别结果与冲突策略', (s) => { s.imp = { tab: 'file', ok: true, strat: 'merge', over: false }; ov(s, { type: 'import' }); }],
      ['imp-url', '导入 · URL', (s) => { s.imp = { tab: 'url', ok: true, strat: 'merge', over: false }; ov(s, { type: 'import' }); }],
      ['imp-curl', '导入 · cURL', (s) => { s.imp = { tab: 'curl', ok: true, strat: 'merge', over: false }; ov(s, { type: 'import' }); }],
      ['cookies', 'Cookie 管理器', (s) => ov(s, { type: 'cookies' })],
    ]],
    ['菜单与浮层', [
      ['quick', '快速打开 ⌘K（默认）', (s) => { s.qk = { q: '', sel: 1 }; ov(s, { type: 'quick' }); }],
      ['quick-q', '快速打开 · 输入搜索', (s) => { s.qk = { q: 'user', sel: 0 }; ov(s, { type: 'quick' }); }],
      ['quick-empty', '快速打开 · 无结果', (s) => { s.qk = { q: 'zzz', sel: 0 }; ov(s, { type: 'quick' }); }],
      ['proj-menu', '项目切换菜单', (s) => pop(s, { type: 'project', anchor: s.dir === 'b' ? '#win .pbtn' : '[data-a=projMenu]' })],
      ['env-menu', '环境切换菜单', (s) => pop(s, { type: 'env', anchor: '[data-a=envMenu]', align: 'right' })],
      ['ctx-req', '右键菜单 · 请求', (s) => { s.sel = 'r2'; pop(s, { type: 'ctx', kind: 'req', anchor: '#win .tree .row[data-id=r2]' }); }],
      ['ctx-folder', '右键菜单 · 文件夹', (s) => pop(s, { type: 'ctx', kind: 'folder', anchor: '#win .tree .row[data-id=f2]' })],
      ['ctx-tab', '右键菜单 · 标签', (s) => pop(s, { type: 'ctx', kind: 'tab', anchor: '#win .tab[data-i="0"]' })],
      ['tabs-list', '标签列表下拉', (s) => pop(s, { type: 'tabsAll', anchor: '[data-a=tabsAll]', align: 'right' })],
      ['mth-menu', '请求方法下拉', (s) => pop(s, { type: 'mth', anchor: '.url .mth' })],
      ['send-menu', '发送下拉', (s) => pop(s, { type: 'send', anchor: '.send', align: 'right' })],
    ]],
    ['对话框与提示', [
      ['dlg-save', '对话框 · 保存更改', (s) => { s.tabs[1].dirty = true; s.active = 1; ov(s, { type: 'save' }); }],
      ['dlg-delete', '对话框 · 删除确认', (s) => ov(s, { type: 'delete' })],
      ['dlg-prompt', '对话框 · 重命名输入', (s) => ov(s, { type: 'prompt' })],
      ['dlg-restore', '对话框 · 恢复备份', (s) => ov(s, { type: 'restore' })],
      ['toast-all', '提示 · 成功/警告/错误/信息', (s) => { s.toasts = [{ id: 1, type: 'err', title: '请求失败', msg: 'ECONNREFUSED：无法连接到 127.0.0.1:8080' }, { id: 2, type: 'warn', title: '响应较大（8.2 MB）', msg: '仅显示前 2 MB，复制或保存可获取完整内容。' }, { id: 3, type: 'ok', title: '', msg: '已复制 cURL 命令' }, { id: 4, type: 'info', title: '', msg: '已取消请求' }]; }],
      ['toast-update', '提示 · 更新就绪 / 撤销删除', (s) => { s.update = 'ready'; s.toasts = [{ id: 1, type: 'ac', title: '新版本 2.1.0 已就绪', msg: '重启后生效，当前打开的标签会自动恢复。', acts: [['重启更新', 'restart'], ['稍后', 'toastDismissAll']] }, { id: 2, type: 'info', title: '已删除“订单服务”', msg: '3 个请求已移入回收，可立即撤销。', acts: [['撤销 ⌘Z', 'toastDismissAll']] }]; }],
    ]],
  ];
  MPA.SCREENS = SCREENS;
  MPA.openSettings = (s) => { if (s.opt.settings === 'page') s.view = 'settings'; else { s.overlay = { type: 'settings' }; s.ovAnim = false; } };
  const flatScreens = SCREENS.reduce((a, g) => a.concat(g[1]), []);

  const KEEP = ['dir', 'scheme', 'mode', 'size', 'logo', 'static', 'opt', 'themeSrc'];
  MPA.go = function (id, silent) {
    MPA.reset(KEEP);
    const sc = flatScreens.find((x) => x[0] === id) || flatScreens[1];
    sc[2](S);
    S.screen = sc[0];
    if (!silent) MPA.renderAll();
  };
  MPA.renderAll = function () { MPA.renderApp(); MPA.renderOverlay(); MPA.renderPop(); MPA.renderToasts(); fit(); };

  /* ---------- 窗口尺寸与缩放 ---------- */
  const SIZES = { compact: [980, 640], std: [1180, 740], wide: [1440, 860] };
  function fit() {
    const win = $('#win'), box = $('#scaler');
    if (!win || !box) return;
    const [w, h] = SIZES[S.size] || SIZES.std;
    win.style.width = w + 'px'; win.style.height = h + 'px';
    if (S.static) { box.style.transform = 'none'; box.style.width = w + 'px'; box.style.height = h + 'px'; return; }
    const stage = $('#stage'), avail = stage.clientWidth - 48, availH = window.innerHeight - 70 - 28;
    const k = Math.min(1, avail / w, availH / h);
    box.style.transform = 'scale(' + k + ')'; box.style.width = w + 'px'; box.style.height = h + 'px';
    stage.style.height = Math.round(h * k + 24) + 'px';
  }
  window.addEventListener('resize', fit);

  /* ---------- 控制条 ---------- */
  const DECISIONS = [
    ['nav', '侧栏导航', [['seg', '分段标签'], ['rail', '活动栏']], '请求 / 历史 / 环境 的入口形态'],
    ['tabs', '标签位置', [['strip', '独立标签条'], ['titlebar', '并入标题栏']], '并入标题栏可多出约 34px 纵向空间'],
    ['dock', '响应布局', [['bottom', '底部'], ['right', '右侧']], '宽屏下左右分栏更利于对照'],
    ['env', '环境切换位置', [['titlebar', '标题栏'], ['urlbar', '地址栏内'], ['status', '底栏']], '离发送动作越近，越不容易发错环境'],
    ['settings', '设置呈现', [['modal', '弹窗'], ['page', '整页']], '整页容纳更多内容且可深链'],
    ['home', '项目主页', [['grid', '卡片'], ['list', '列表']], '项目多时列表更高效'],
    ['fx', '发送反馈', [['comet', '顶部彗星线'], ['ring', '按钮内进度']], '等待动画的呈现位置'],
    ['quick', '快速打开', [['center', '居中面板'], ['drop', '标题栏下拉']], '⌘K 面板形态'],
  ];
  const SCHEME_DOTS = { mist: '#4f52d9', ink: '#17171a', jade: '#08806f', amber: '#c2570c', rose: '#d6336c' };
  const SCHEME_NAMES = { mist: '雾蓝', ink: '墨白', jade: '松石', amber: '琥珀', rose: '玫瑰' };
  function bar() {
    const el = $('#bar');
    if (!el) return;
    const dirs = [['a', 'A', '静雅'], ['b', 'B', '聚焦'], ['c', 'C', '流光']];
    el.innerHTML = '<div class="b-brand">' + (window.MPLogo ? MPLogo.glyph(S.logo, { size: 20, color: 'accent' }) : '') + '<b>MiniPost</b><span>设计稿</span></div>' +
      '<div class="b-grp"><label>方案</label><div class="b-seg">' + dirs.map((d) => '<button data-b="dir" data-v="' + d[0] + '" class="' + (S.dir === d[0] ? 'on' : '') + '"><i>' + d[1] + '</i>' + d[2] + '</button>').join('') + '</div></div>' +
      '<div class="b-grp"><label>配色</label><div class="b-dots">' + Object.keys(SCHEME_DOTS).map((k) => '<button data-b="scheme" data-v="' + k + '" title="' + SCHEME_NAMES[k] + '" class="' + (S.scheme === k ? 'on' : '') + '" style="--c:' + SCHEME_DOTS[k] + '"></button>').join('') + '</div><button class="b-ic" data-b="mode" title="切换明暗">' + I(S.mode === 'dark' ? 'moon' : 'sun', 14) + '</button></div>' +
      '<div class="b-grp"><label>屏幕</label><select data-b="screen">' + SCREENS.map((g) => '<optgroup label="' + g[0] + '">' + g[1].map((s) => '<option value="' + s[0] + '" ' + (S.screen === s[0] ? 'selected' : '') + '>' + s[1] + '</option>').join('') + '</optgroup>').join('') + '</select></div>' +
      '<div class="b-grp"><button class="b-btn" data-b="decisions">' + I('sliders', 13) + '交互方案<span class="b-n" id="b-n"></span></button></div>' +
      '<div class="b-grp"><label>Logo</label><select data-b="logo">' + (window.MPLogo ? MPLogo.list.map((l) => '<option value="' + l.id + '" ' + (S.logo === l.id ? 'selected' : '') + '>' + l.name + (l.recommended ? ' ★' : '') + '</option>').join('') : '') + '</select></div>' +
      '<div class="b-grp"><label>窗口</label><div class="b-seg">' + [['compact', '紧凑'], ['std', '标准'], ['wide', '宽']].map((s) => '<button data-b="size" data-v="' + s[0] + '" class="' + (S.size === s[0] ? 'on' : '') + '">' + s[1] + '</button>').join('') + '</div></div>' +
      '<span style="flex:1"></span><nav class="b-nav"><a href="index.html">总览</a><a href="gallery.html">画廊</a><a href="logo.html">Logo</a><a href="system.html">设计系统</a></nav>';
    const n = DECISIONS.filter((d) => S.opt[d[0]] !== MPA.DIR_DEFAULTS[S.dir][d[0]]).length;
    const nb = $('#b-n'); if (nb) { nb.textContent = n || ''; nb.style.display = n ? 'inline-grid' : 'none'; }
  }
  MPA.syncBar = bar;
  function decisionsPanel(open) {
    let el = $('#dec');
    if (!open) { if (el) el.remove(); return; }
    if (!el) { el = document.createElement('div'); el.id = 'dec'; document.body.appendChild(el); }
    el.innerHTML = '<div class="d-h"><b>交互方案</b><span>每一项都可单独切换，立即在原型中生效；「重置」回到当前方案的推荐组合。</span><button data-b="resetopt">重置</button><button data-b="decisions" class="x">✕</button></div>' + DECISIONS.map((d) => '<div class="d-r"><div class="d-t"><b>' + d[1] + '</b><span>' + d[3] + '</span></div><div class="b-seg">' + d[2].map((o) => '<button data-b="opt" data-k="' + d[0] + '" data-v="' + o[0] + '" class="' + (S.opt[d[0]] === o[0] ? 'on' : '') + '">' + o[1] + (MPA.DIR_DEFAULTS[S.dir][d[0]] === o[0] ? '<em>推荐</em>' : '') + '</button>').join('') + '</div></div>').join('');
  }
  function onBar(ev) {
    const t = ev.target.closest('[data-b]');
    if (!t) return;
    const k = t.dataset.b, v = t.dataset.v != null ? t.dataset.v : t.value;
    if (t.tagName === 'SELECT' && ev.type === 'click') return;
    if (t.tagName !== 'SELECT' && ev.type === 'change') return;
    if (k === 'dir') { MPA.setDir(v); MPA.go(S.screen); }
    else if (k === 'scheme') { S.scheme = v; MPA.renderAll(); }
    else if (k === 'mode') { S.mode = S.mode === 'dark' ? 'light' : 'dark'; S.themeSrc = null; MPA.renderAll(); }
    else if (k === 'screen') MPA.go(v);
    else if (k === 'logo') { S.logo = v; MPA.renderAll(); }
    else if (k === 'size') { S.size = v; fit(); }
    else if (k === 'decisions') { decisionsPanel(!$('#dec')); return; }
    else if (k === 'opt') { S.opt[t.dataset.k] = v; MPA.go(S.screen); decisionsPanel(true); }
    else if (k === 'resetopt') { S.opt = Object.assign({}, MPA.DIR_DEFAULTS[S.dir]); MPA.go(S.screen); decisionsPanel(true); }
    bar();
  }

  /* ---------- 启动 ---------- */
  function init() {
    const q = new URLSearchParams(location.search);
    const win = $('#win');
    if (!win) return; // 画廊等页面只复用屏幕清单，不需要渲染原型
    S.static = q.get('chrome') === '0';
    if (['a', 'b', 'c'].includes(q.get('dir'))) MPA.setDir(q.get('dir'));
    if (SCHEME_DOTS[q.get('scheme')]) S.scheme = q.get('scheme');
    if (['light', 'dark'].includes(q.get('mode'))) S.mode = q.get('mode');
    if (SIZES[q.get('size')]) S.size = q.get('size');
    if (window.MPLogo && MPLogo.list.some((l) => l.id === q.get('logo'))) S.logo = q.get('logo');
    else if (window.MPLogo) { const r = MPLogo.list.find((l) => l.recommended); if (r) S.logo = r.id; }
    DECISIONS.forEach((d) => { const v = q.get(d[0]); if (v && d[2].some((o) => o[0] === v)) S.opt[d[0]] = v; });
    document.body.classList.toggle('static', S.static);
    MPA.go(q.get('screen') || 'req-params', true);
    MPA.bind();
    MPA.renderAll();
    if (!S.static) {
      bar();
      $('#bar').addEventListener('click', onBar);
      $('#bar').addEventListener('change', onBar);
      document.addEventListener('click', (ev) => { const d = $('#dec'); if (d && !d.contains(ev.target) && !ev.target.closest('[data-b=decisions]')) decisionsPanel(false); });
    }
    fit();
  }
  document.addEventListener('DOMContentLoaded', init);
})();
