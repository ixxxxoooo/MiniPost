/* ==========================================================================
   MiniPost 设计稿原型 · 核心：状态、工作区渲染（标题栏 / 侧栏 / 标签 / 请求 / 响应 / 页面）
   说明：纯原生 JS，字符串模板渲染；所有交互通过 data-a 事件委托分发（见 overlays.js 的 A 表）。
   ========================================================================== */
(function () {
  'use strict';
  const M = window.MP, I = window.I;
  const $ = (s, r) => (r || document).querySelector(s);
  const $$ = (s, r) => Array.from((r || document).querySelectorAll(s));
  const esc = (s) => String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  const MPA = (window.MPA = { $, $$, esc, I });

  /* ---------- 数据索引 ---------- */
  const flat = {};
  (function walk(list, parent) { list.forEach((n) => { flat[n.id] = n; n.parent = parent; if (n.kids) walk(n.kids, n.id); }); })(M.tree, null);
  MPA.flat = flat;
  const countReqs = () => Object.values(flat).filter((n) => n.t === 'r').length;
  const countFolders = () => Object.values(flat).filter((n) => n.t === 'f').length;

  /* ---------- 三套方向的默认交互决策 ---------- */
  const DIR_DEFAULTS = {
    a: { dock: 'bottom', nav: 'seg', tabs: 'strip', env: 'titlebar', settings: 'modal', home: 'grid', fx: 'comet', quick: 'center' },
    b: { dock: 'right', nav: 'rail', tabs: 'titlebar', env: 'titlebar', settings: 'page', home: 'list', fx: 'ring', quick: 'drop' },
    c: { dock: 'bottom', nav: 'seg', tabs: 'strip', env: 'titlebar', settings: 'modal', home: 'grid', fx: 'comet', quick: 'center' },
  };
  MPA.DIR_DEFAULTS = DIR_DEFAULTS;

  const mkTab = (id, view, resp, dirty, code) => ({ kind: 'req', id, view: view || 'params', resp: resp || 'none', rtab: 'body', dirty: !!dirty, code: code || 200, merged: false });

  function fresh() {
    return {
      dir: 'a', scheme: 'mist', mode: 'light', size: 'std', logo: 'fold', static: false,
      opt: Object.assign({}, DIR_DEFAULTS.a),
      view: 'work', setPage: 'general', sideTab: 'req', sideCollapsed: false, q: '', hq: '',
      tabs: [mkTab('r1', 'params', 'ok'), mkTab('r2', 'body', 'none', true), mkTab('r5', 'params', 'none')], active: 0,
      sel: 'r1', open: { f1: true, f2: true, f3: false, f4: false }, histOpen: { '今天': true, '昨天': true },
      con: { open: false, filter: 'all', ex: 0 }, splitPct: 54,
      overlay: null, ovAnim: false, pop: null, toasts: [],
      envSel: 'e1', curEnv: 'e1', homeSel: 'p1', homeQ: '', projId: 'p1', homeSort: '最近更新',
      update: null, last: { m: 'GET', path: '/users?page=1&size=20', code: 200, ms: 128 },
      hist: JSON.parse(JSON.stringify(M.history)), conRows: M.consoleRows.slice(),
      set: { ssl: true, redirect: true, cookies: false, autoBackup: false, autoUpdate: true, lang: '中文', http: '自动', fmt: '自动', timeout: 0, maxMB: 50, interval: 30, fs: 12 },
      cookieDom: 'api.acme.io', imp: { tab: 'file', ok: false, strat: 'merge', over: false },
      qk: { q: '', sel: 0 }, uid: 0,
    };
  }
  const S = (MPA.S = fresh());
  MPA.fresh = fresh;
  MPA.reset = (keep) => { const k = {}; (keep || []).forEach((f) => (k[f] = S[f])); Object.keys(S).forEach((x) => delete S[x]); Object.assign(S, fresh(), k); };
  MPA.setDir = (d) => { S.dir = d; S.opt = Object.assign({}, DIR_DEFAULTS[d]); };

  /* ---------- 工具 ---------- */
  const proj = () => M.projects.find((p) => p.id === S.projId) || M.projects[0];
  const env = () => M.envs.find((e) => e.id === S.curEnv) || null;
  const curTab = () => S.tabs[S.active] || null;
  const reqOf = (id) => Object.assign({}, M.defReq, M.reqs[id] || {});
  const ib = (icon, act, o) => { o = o || {}; return '<button class="ib ' + (o.on ? 'on ' : '') + (o.cls || '') + '" data-a="' + act + '" ' + (o.attr || '') + ' data-tip="' + esc(o.tip || '') + '">' + I(icon, o.sz || 15) + (o.badge ? '<i class="badge"></i>' : '') + '</button>'; };
  /* 项目主题色：存储的是色相基准（hex），显示时取其色相，套用当前配色强调色的明度与彩度，使其与外观配色协调 */
  const hexHue = (hex) => {
    const lin = (c) => { c /= 255; return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4); };
    const [r, g, b] = [1, 3, 5].map((i) => lin(parseInt(hex.slice(i, i + 2), 16)));
    const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b), m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b), s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
    const a = 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s, bb = 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s;
    return Math.round(((Math.atan2(bb, a) * 180) / Math.PI + 360) % 360);
  };
  const pjc = (hex) => (hex ? 'oklch(from var(--ac-base) clamp(.56, l, .8) clamp(.11, c, .17) ' + hexHue(hex) + ')' : 'var(--ac-base)');
  const mbadge = (m) => '<span class="m m-' + m + '">' + M.METHOD_ABBR[m] + '</span>';
  const varHtml = (s) => esc(s).replace(/\{\{([^}]+)\}\}/g, (m0, k) => '<span class="vt ' + (!S.curEnv && !k.startsWith('$') ? 'bad' : '') + '">{{' + k + '}}</span>');
  const cb = (on, act, attr) => '<i class="cb ' + (on ? 'on' : '') + '" ' + (act ? 'data-a="' + act + '" ' : '') + (attr || '') + '>' + I('check', 10) + '</i>';
  const sw = (on, key) => '<i class="sw ' + (on ? 'on' : '') + '" data-a="sw" data-k="' + key + '"></i>';
  const sel = (label, act, attr) => '<span class="sel" ' + (act ? 'data-a="' + act + '" ' : '') + (attr || '') + '><span>' + label + '</span>' + I('chevUD', 12) + '</span>';
  Object.assign(MPA, { proj, env, curTab, reqOf, ib, mbadge, varHtml, cb, sw, sel, mkTab, countReqs, countFolders });
  const logo = (size, kind) => (window.MPLogo ? (kind === 'glyph' ? MPLogo.glyph(S.logo, { size, color: 'accent' }) : MPLogo.tile(S.logo, { size, variant: 'color' })) : '');
  MPA.logo = logo;

  /* ---------- 响应数据 ---------- */
  const jsonCreated = '{\n  "code": 0,\n  "message": "created",\n  "data": {\n    "id": 1026,\n    "name": "Alice",\n    "email": "alice@example.com",\n    "roles": ["admin", "editor"],\n    "createdAt": "2026-11-15T10:02:44Z"\n  }\n}';
  const RESP = {
    200: { text: '200 OK', body: M.jsonOk, ms: 128, size: '1.2 KB', cls: 'ok' },
    201: { text: '201 Created', body: jsonCreated, ms: 246, size: '312 B', cls: 'ok' },
    204: { text: '204 No Content', body: '', ms: 64, size: '0 B', cls: 'ok' },
    422: { text: '422 Unprocessable', body: M.jsonErr, ms: 91, size: '208 B', cls: 'err' },
    9200: { text: '200 OK', body: M.jsonOk, ms: 842, size: '8.2 MB', cls: 'ok', large: true },
  };
  MPA.RESP = RESP;
  const outcomeOf = (id) => ({ r8: 'error', r7: 'stream', r10: 'stream', r6: 422, r2: 201, r4: 204 }[id] || 200);
  MPA.outcomeOf = outcomeOf;

  /* ---------- JSON 高亮代码块 ---------- */
  const RE = /("(?:\\.|[^"\\])*")(\s*:)?|(-?\b\d+(?:\.\d+)?\b)|\b(true|false|null)\b|([{}\[\],])/g;
  function hl(line) {
    return esc(line).replace(RE, (m0, s, colon, num, b, p) => s ? (colon ? '<span class="jk">' + s + '</span><span class="jp">:</span>' : '<span class="js">' + s + '</span>') : num ? '<span class="jn">' + num + '</span>' : b ? '<span class="jb">' + b + '</span>' : '<span class="jp">' + p + '</span>');
  }
  function codeBlock(text, o) {
    o = o || {};
    const lines = text.split('\n');
    return '<div class="code scroll">' + lines.map((l, i) => '<div class="cl ' + (o.hot === i ? 'hot' : '') + '"><span class="n">' + (/[{\[]\s*$/.test(l) ? '<span class="fo">' + I('chevD', 10) + '</span>' : '') + (i + 1) + '</span><span class="t">' + (o.plain ? esc(l) : hl(l)) + '</span></div>').join('') + (o.caret ? '<div class="cl"><span class="n"></span><span class="t"><i class="caret"></i></span></div>' : '') + '</div>';
  }
  MPA.codeBlock = codeBlock;

  /* ==========================================================================
     标题栏
     ========================================================================== */
  function envChip(cls) {
    const e = env();
    return '<span class="chip ' + (cls || '') + '" data-a="envMenu" data-tip="切换环境">' + I('globe', 13) + '<span>' + (e ? esc(e.name) : '无环境') + '</span><span class="pdot" style="background:' + (e ? 'var(--ok)' : 'var(--bd2)') + '"></span>' + I('chevD', 11) + '</span>';
  }
  function projChip() {
    const p = proj();
    return '<span class="chip" data-a="projMenu" data-tip="切换项目"><span class="pav" style="background:' + pjc(p.color) + '">' + esc(p.name[0]) + '</span><span>' + esc(p.name) + '</span>' + I('chevD', 11) + '</span>';
  }
  // 标签是否放进标题栏（方案 C 的悬浮岛占据标题栏中央，所以始终使用标签条）
  const tabsInTitlebar = () => S.opt.tabs === 'titlebar' && S.view === 'work' && S.dir !== 'c';
  function titlebar() {
    const o = S.opt, d = S.dir, rail = o.nav === 'rail';
    const tabsInTb = tabsInTitlebar();
    let h = '<div class="lights"><i></i><i></i><i></i></div>';
    h += ib('sidebar', 'toggleSide', { on: !S.sideCollapsed, tip: '收起 / 展开侧栏 ⌘B' });
    if (d !== 'b') h += ib('arrowL', 'noop', { cls: 'dis', tip: '后退' }) + ib('arrowR', 'noop', { cls: 'dis', tip: '前进' }) + ib('home', 'home', { on: S.view === 'home', tip: '项目主页' });
    if (d === 'a') h += projChip();
    if (d === 'c') {
      h += '<div class="island">' + projChip() + '<span class="sep"></span>' + (o.env === 'titlebar' ? envChip() + '<span class="sep"></span>' : '') + '<span class="cmdk2" data-a="quick">' + I('search', 13) + '<span class="grow">搜索</span><span class="kbd">⌘K</span></span></div>';
    } else if (tabsInTb) {
      h += tabsHtml('in-tb');
    } else {
      h += '<div class="cmdk" data-a="quick">' + I('search', 13) + '<span>搜索请求、环境、命令…</span><span class="kbd">⌘K</span></div>';
    }
    if (!tabsInTb) h += '<span class="sp"></span>';
    if (d === 'b' && !tabsInTb) h += projChip();
    if (d !== 'c' && tabsInTb) h += ib('search', 'quick', { tip: '快速打开 ⌘K' });
    if (d !== 'c' && o.env === 'titlebar') h += envChip();
    if (d === 'b' && S.view === 'work') h += ib('home', 'home', { tip: '项目主页' });
    if (S.update === 'downloading') h += '<span class="updchip" data-a="settings" data-tip="正在下载 2.1.0"><svg class="ic spin" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round"><path d="M12 3a9 9 0 1 0 9 9"/></svg>下载中 62%</span>';
    else if (S.update) h += '<span class="updchip ready" data-a="restart" data-tip="新版本 2.1.0 已就绪">' + I('download', 12) + '重启更新<i class="dotm"></i></span>';
    h += ib(S.mode === 'dark' ? 'sun' : 'moon', 'theme', { tip: S.mode === 'dark' ? '切换为浅色' : '切换为深色' });
    if (!rail) h += ib('sliders', 'settings', { on: S.view === 'settings', tip: '设置 ⌘,' });
    return '<div class="tb">' + h + '</div>';
  }

  /* ==========================================================================
     侧栏
     ========================================================================== */
  function railHtml() {
    const t = S.sideTab, r = (icon, key, tip) => '<button class="ib ' + (S.view === 'work' && t === key && !S.sideCollapsed ? 'cur' : '') + '" data-a="sideTab" data-t="' + key + '" data-tip="' + tip + '">' + I(icon, 17) + '</button>';
    return '<div class="rail"><div style="margin:2px 0 8px;filter:drop-shadow(0 2px 6px color-mix(in oklab,var(--ac) 30%,transparent))">' + logo(26) + '</div>' + r('layers', 'req', '请求') + r('clock', 'hist', '历史') + r('globe', 'env', '环境') + '<span class="sp"></span>' +
      '<button class="ib" data-a="cookies" data-tip="Cookie 管理器">' + I('cookie', 17) + '</button>' +
      '<button class="ib ' + (S.con.open ? 'on' : '') + '" data-a="toggleCon" data-tip="控制台 ⌘J">' + I('terminal', 17) + '</button>' +
      '<button class="ib ' + (S.view === 'settings' ? 'on' : '') + '" data-a="settings" data-tip="设置 ⌘,">' + I('sliders', 17) + '</button></div>';
  }
  function treeRows(list, q, depth) {
    return list.map((n) => {
      if (n.t === 'f') {
        const kids = n.kids.filter((k) => !q || (k.name + k.url + k.m).toLowerCase().includes(q));
        if (q && !kids.length) return '';
        const open = q ? true : S.open[n.id];
        return '<div class="fld ' + (open ? 'open' : '') + '" data-fid="' + n.id + '"><div class="row" data-a="fold" data-id="' + n.id + '" data-ctx="folder:' + n.id + '">' + I('chevR', 11).replace('class="ic"', 'class="ic chev"') + '<span class="fic">' + I('folder', 14) + '</span><span class="nm">' + esc(n.name) + '</span><span class="sub">' + n.kids.length + '</span><span class="acts">' + ib('plus', 'noop', { sz: 13, cls: 'sm' }) + ib('more', 'ctxBtn', { sz: 13, cls: 'sm', attr: 'data-id="' + n.id + '" data-kind="folder"' }) + '</span></div><div class="kw"><div class="kids">' + treeRows(kids, q, depth + 1) + '</div></div></div>';
      }
      if (q && !(n.name + n.url + n.m).toLowerCase().includes(q)) return '';
      return '<div class="row ' + (S.sel === n.id ? 'sel' : '') + '" data-a="openReq" data-id="' + n.id + '" data-ctx="req:' + n.id + '">' + (depth === 0 ? '<span style="width:17px;flex:none"></span>' : '') + mbadge(n.m) + '<span class="nm">' + esc(n.name) + '</span><span class="acts">' + ib('more', 'ctxBtn', { sz: 13, cls: 'sm', attr: 'data-id="' + n.id + '" data-kind="req"' }) + '</span></div>';
    }).join('');
  }
  MPA.renderTree = () => { const el = $('#win .tree'); if (el) el.innerHTML = treeBody(); };
  function treeBody() {
    const q = S.q.trim().toLowerCase();
    if (S.emptyTree) return '<div class="empty-s">' + I('layers', 22) + '<b>还没有请求</b><span>新建一个请求，或导入已有集合</span><div style="display:flex;gap:6px;margin-top:6px"><button class="btn sm pri" data-a="newReq">' + I('plus', 12) + '新建请求</button><button class="btn sm" data-a="importOv" data-tab="file">' + I('upload', 12) + '导入</button></div></div>';
    const rows = treeRows(M.tree, q, 0);
    return rows || '<div class="empty-s">' + I('search', 20) + '<span>无匹配结果</span></div>';
  }
  function sideHead() {
    const rail = S.opt.nav === 'rail';
    const titles = { req: '请求', hist: '历史', env: '环境' };
    let h = '';
    if (S.dir === 'b') h += '<div style="padding:8px 8px 0">' + '<div class="pbtn" data-a="projMenu">' + '<span class="pav" style="background:' + pjc(proj().color) + '">' + esc(proj().name[0]) + '</span><span class="grow ell">' + esc(proj().name) + '</span>' + I('chevUD', 13) + '</div></div>';
    if (rail) h += '<div class="side-t"><span>' + titles[S.sideTab] + '</span><span class="sp"></span>' + sideActions() + '</div>';
    else h += '<div class="side-h"><div class="seg full"><b class="' + (S.sideTab === 'req' ? 'on' : '') + '" data-a="sideTab" data-t="req">' + I('layers', 13) + '请求</b><b class="' + (S.sideTab === 'hist' ? 'on' : '') + '" data-a="sideTab" data-t="hist">' + I('clock', 13) + '历史</b><b class="' + (S.sideTab === 'env' ? 'on' : '') + '" data-a="sideTab" data-t="env">' + I('globe', 13) + '环境</b></div></div>';
    return h;
  }
  function sideActions() {
    if (S.sideTab === 'req') return ib('filePlus', 'newReq', { tip: '新建请求 ⌘N' }) + ib('folderPlus', 'newFolder', { tip: '新建文件夹 ⌘⇧N' }) + ib('upload', 'importOv', { attr: 'data-tab="file"', tip: '导入 ⌘O' });
    if (S.sideTab === 'hist') return ib('refresh', 'noop', { tip: '刷新' }) + ib('trash', 'clearHist', { tip: '清空历史' });
    return ib('plus', 'newEnv', { tip: '新建环境' });
  }
  function sideBody() {
    if (S.sideTab === 'req') {
      return '<div class="sr" style="padding:' + (S.opt.nav === 'rail' ? '4px 8px 6px' : '0 8px 6px') + '"><label class="inp grow">' + I('search', 13) + '<input data-in="q" placeholder="搜索请求…" value="' + esc(S.q) + '"><span class="kbd">⌘K</span></label>' + (S.opt.nav === 'rail' ? '' : sideActions()) + '</div><div class="tree scroll">' + treeBody() + '</div><div class="side-f">' + I('layers', 12) + countReqs() + ' 个请求 · ' + countFolders() + ' 个文件夹</div>';
    }
    if (S.sideTab === 'hist') {
      const q = S.hq.trim().toLowerCase();
      const total = S.hist.reduce((n, g) => n + g.items.length, 0);
      let h = '<div class="sr" style="padding:' + (S.opt.nav === 'rail' ? '4px 8px 6px' : '0 8px 6px') + '"><label class="inp grow">' + I('search', 13) + '<input data-in="hq" placeholder="搜索历史…" value="' + esc(S.hq) + '"></label>' + (S.opt.nav === 'rail' ? '' : sideActions()) + '</div><div class="tree scroll">';
      if (!total) h += '<div class="empty-s">' + I('clock', 22) + '<b>暂无历史记录</b><span>发送请求后会自动记录在这里</span></div>';
      S.hist.forEach((g) => {
        const items = g.items.filter((it) => !q || (it[0] + it[1] + it[2]).toLowerCase().includes(q) || (q === '4xx' && String(it[2])[0] === '4') || (q === '2xx' && String(it[2])[0] === '2'));
        if (!items.length) return;
        const open = S.histOpen[g.day] !== false;
        h += '<div class="grp ' + (open ? 'open' : '') + '" data-a="histFold" data-day="' + g.day + '">' + I('chevR', 11) + g.day + '<span class="cnt">' + items.length + '</span></div>';
        if (open) h += items.map((it, i) => '<div class="row hrow" style="animation:rowIn var(--dur-slow) var(--ease-out) ' + i * 25 + 'ms both" data-a="openHist" data-ctx="hist:0">' + mbadge(it[0]) + '<span class="path">' + esc(it[1].split('?')[0]) + '</span><span class="sc ' + (it[2] === 0 ? 'st-err' : it[2] >= 400 ? 'st-err' : 'st-ok') + '">' + (it[2] || 'ERR') + '</span><span class="tm">' + it[3] + '</span></div>').join('');
      });
      return h + '</div><div class="side-f">' + I('clock', 12) + total + ' 条记录</div>';
    }
    let h = '<div class="tree scroll" style="padding-top:6px">';
    h += M.envs.map((e, i) => '<div class="row" style="height:36px;animation:rowIn var(--dur-slow) var(--ease-out) ' + i * 30 + 'ms both" data-a="openEnv" data-id="' + e.id + '" data-ctx="env:' + e.id + '"><span class="fic">' + I('globe', 15) + '</span><span class="grow"><b style="font-weight:600;display:block">' + esc(e.name) + '</b><span class="mu" style="font-size:10.5px">' + e.vars.length + ' 个变量' + (e.desc ? ' · ' + esc(e.desc) : '') + '</span></span>' + (S.curEnv === e.id ? '<span class="cur-badge">' + I('check', 10) + '当前</span>' : '') + '</div>').join('');
    h += '<div class="row" data-a="newEnv" style="margin-top:4px;color:var(--mu)"><span style="width:17px"></span>' + I('plus', 14) + '<span class="nm">新建环境</span></div></div>';
    return h + '<div class="side-f">' + I('info', 12) + '切换后 {{变量}} 立即替换</div>';
  }
  function sidebar() {
    return '<div class="side ' + (S.sideCollapsed ? 'closed' : '') + '">' + sideHead() + sideBody() + '</div>';
  }
  MPA.renderSide = () => { const el = $('#win .side'); if (el) el.outerHTML = sidebar(); };

  /* ==========================================================================
     标签栏
     ========================================================================== */
  function tabsHtml(cls) {
    const tabs = S.tabs.map((t, i) => {
      const n = t.kind === 'env' ? null : flat[t.id] || t.node;
      const title = t.kind === 'env' ? M.envs.find((e) => e.id === t.id).name : n.name;
      const lead = t.kind === 'env' ? '<span style="color:var(--ac);display:grid">' + I('globe', 13) + '</span>' : mbadge(n.m);
      return '<div class="tab ' + (i === S.active ? 'on ' : '') + (t.fresh ? 'in' : '') + '" data-a="selTab" data-i="' + i + '" data-ctx="tab:' + i + '">' + lead + '<span class="tt">' + esc(title) + '</span>' + (t.dirty ? '<i class="dirty"></i>' : '') + '<span class="x" data-a="closeTab" data-i="' + i + '">' + I('x', 11) + '</span></div>';
    }).join('');
    S.tabs.forEach((t) => (t.fresh = false));
    return '<div class="tabs ' + (cls || '') + '"><div class="tabs-sc">' + tabs + '</div>' + ib('plus', 'newReq', { cls: 'plus sm', sz: 14, tip: '新建请求 ⌘N' }) + (cls === 'in-tb' ? '' : '<span class="sp"></span>' + ib('chevD', 'tabsAll', { cls: 'more sm', tip: '显示所有标签' })) + '</div>';
  }

  /* ==========================================================================
     请求区
     ========================================================================== */
  function kvTable(rows, kind) {
    const two = kind === 'urlenc'; // urlencoded 没有“说明”列
    const cols = kind === 'params' ? ['参数名', '参数值', '说明'] : two ? ['键', '值', '说明'] : ['请求头', '值', '说明'];
    const gt = two ? ' style="grid-template-columns:34px 1fr 1.2fr 34px"' : '';
    const g2 = two ? 'grid-template-columns:34px 1fr 1.2fr 34px;' : '';
    const desc = (v) => (two ? '' : '<span class="c"><input class="d" value="' + esc(v) + '" placeholder="说明"></span>');
    let h = '<div class="kv scroll"><div class="hd"' + gt + '><span></span><span>' + cols[0] + '</span><span>' + cols[1] + '</span>' + (two ? '' : '<span>' + cols[2] + '</span>') + '<span></span></div>';
    rows.forEach((r, i) => {
      const empty = !r[0] && !r[1];
      h += '<div class="kr ' + (r[3] === 0 ? 'off' : '') + (empty ? ' ghost' : '') + '" style="' + g2 + 'animation:rowIn var(--dur-slow) var(--ease-out) ' + i * 24 + 'ms both">' + (empty ? '<span></span>' : '<span>' + cb(r[3] !== 0, 'kvToggle') + '</span>') + '<span class="c"><input value="' + esc(r[0]) + '" placeholder="' + cols[0] + '"></span><span class="c"><input value="' + esc(r[1]) + '" placeholder="' + cols[1] + '"></span>' + desc(r[2]) + '<span>' + (empty ? '' : '<button class="ib sm del" data-a="noop" data-tip="删除">' + I('x', 12) + '</button>') + '</span></div>';
    });
    if (rows.length && (rows[rows.length - 1][0] || rows[rows.length - 1][1])) h += '<div class="kr ghost"' + (two ? ' style="' + g2 + '"' : '') + '><span></span><span class="c"><input placeholder="' + cols[0] + '"></span><span class="c"><input placeholder="' + cols[1] + '"></span>' + desc('') + '<span></span></div>';
    return h + '</div>';
  }
  // form-data 表格：字段名 / 类型（Text、File）/ 字段值；文件字段显示文件名与大小
  function formTable(rows) {
    const cols = '34px 1fr 92px 1.4fr 1fr 34px';
    let h = '<div class="kv scroll"><div class="hd" style="grid-template-columns:' + cols + '"><span></span><span>键</span><span>类型</span><span>值</span><span>说明</span><span></span></div>';
    rows.forEach((r, i) => {
      const file = r[4] === 'file';
      h += '<div class="kr ' + (r[3] === 0 ? 'off' : '') + '" style="grid-template-columns:' + cols + ';animation:rowIn var(--dur-slow) var(--ease-out) ' + i * 24 + 'ms both"><span>' + cb(r[3] !== 0, 'kvToggle') + '</span><span class="c"><input value="' + esc(r[0]) + '" placeholder="键"></span><span class="c"><span class="tsel" data-a="noop">' + (file ? '文件' : '文本') + I('chevUD', 10) + '</span></span><span class="c">' + (file ? '<span class="fchip">' + I('image', 12) + '<span class="ell">' + esc(r[1]) + '</span><em>86 KB</em></span>' : '<input value="' + esc(r[1]) + '" placeholder="值">') + '</span><span class="c"><input class="d" value="' + esc(r[2]) + '" placeholder="说明"></span><span><button class="ib sm del" data-a="noop" data-tip="删除">' + I('x', 12) + '</button></span></div>';
    });
    h += '<div class="kr ghost" style="grid-template-columns:' + cols + '"><span></span><span class="c"><input placeholder="键"></span><span class="c"></span><span class="c"><input placeholder="值"></span><span class="c"><input class="d" placeholder="说明"></span><span></span></div>';
    return h + '</div>';
  }
  function bodyEditor(rq, t) {
    const b = rq.body, types = [['none', 'none'], ['form', 'form-data'], ['raw', 'raw'], ['json', 'JSON'], ['urlenc', 'urlencoded']];
    const bulk = '<span class="lk" data-a="noop">' + I('list', 12) + '批量编辑</span>';
    const right = b.type === 'json' ? '<button class="btn sm ghost" data-a="beautify">' + I('sparkle', 12) + '格式化</button>' : b.type === 'raw' ? '<span class="sel" data-a="noop" style="min-width:96px;height:24px"><span>Text</span>' + I('chevUD', 12) + '</span>' : b.type === 'urlenc' ? bulk : '';
    const h = '<div class="hint sub"><div class="seg">' + types.map((x) => '<b class="' + (b.type === x[0] ? 'on' : '') + '" data-a="bodyType" data-v="' + x[0] + '">' + x[1] + '</b>').join('') + '</div><span class="sp"></span>' + right + '</div>';
    if (b.type === 'none') return h + '<div class="empty-r">' + I('window', 26) + '<div class="big">此请求没有请求体</div><span>选择上方类型来添加 JSON、表单或原始文本</span></div>';
    if (b.type === 'form') return h + formTable(b.rows);
    if (b.type === 'urlenc') return h + kvTable(b.rows, 'urlenc');
    if (b.type === 'raw') return h + codeBlock(b.text, { plain: true, hot: -1 });
    return h + codeBlock(b.text, { hot: t.view === 'body' && !S.static ? 1 : -1, caret: false });
  }
  function authView(rq) {
    const a = rq.auth;
    if (a.type === 'none') return '<div class="auth"><div class="fr"><label>认证类型</label><span>' + sel('None', 'noop', 'style="width:180px"') + '</span></div><div class="empty-r" style="padding:30px 0">' + I('lock', 24) + '<span>此请求不需要认证</span></div></div>';
    if (a.type === 'basic') return '<div class="auth"><div class="fr"><label>认证类型</label><span style="display:flex;gap:10px;align-items:center">' + sel('Basic Auth', 'noop', 'style="width:180px"') + '<span class="mu" style="font-size:11px">None / Basic / Bearer / API Key</span></span></div>' +
      '<div class="fr"><label>用户名</label><label class="inp" style="height:30px"><input value="' + esc(a.user) + '"></label></div>' +
      '<div class="fr"><label>密码</label><label class="inp" style="height:30px"><input class="mono" type="' + (S.authShow ? 'text' : 'password') + '" value="' + esc(a.pass) + '"><button class="ib sm" data-a="authEye" data-tip="显示 / 隐藏">' + I(S.authShow ? 'eyeOff' : 'eye', 14) + '</button></label></div>' +
      '<div class="fr"><label>预览</label><div class="prev">' + I('lock', 12) + '<span class="grow ell">Authorization: Basic <span class="mu">YWxpY2U6aHVudGVyMi1kZW1v</span></span><span class="tag mute">请求头</span></div></div></div>';
    if (a.type === 'apikey') return '<div class="auth"><div class="fr"><label>认证类型</label><span style="display:flex;gap:10px;align-items:center">' + sel('API Key', 'noop', 'style="width:180px"') + '<span class="mu" style="font-size:11px">None / Basic / Bearer / API Key</span></span></div>' +
      '<div class="fr"><label>键名</label><label class="inp" style="height:30px"><input class="mono" value="' + esc(a.key) + '"></label></div>' +
      '<div class="fr"><label>键值</label><label class="inp" style="height:30px"><input class="mono" type="' + (S.authShow ? 'text' : 'password') + '" value="' + esc(a.value) + '"><button class="ib sm" data-a="authEye" data-tip="显示 / 隐藏">' + I(S.authShow ? 'eyeOff' : 'eye', 14) + '</button></label></div>' +
      '<div class="fr"><label>添加位置</label><div class="seg"><b class="on">请求头</b><b>查询参数</b></div></div>' +
      '<div class="fr"><label>预览</label><div class="prev">' + I('lock', 12) + '<span class="grow ell">' + esc(a.key) + ': <span class="mu">' + (S.authShow ? esc(a.value) : '••••••••••••••••') + '</span></span><span class="tag mute">请求头</span></div></div></div>';
    return '<div class="auth"><div class="fr"><label>认证类型</label><span style="display:flex;gap:10px;align-items:center">' + sel('Bearer Token', 'noop', 'style="width:180px"') + '<span class="mu" style="font-size:11px">None / Basic / Bearer / API Key</span></span></div>' +
      '<div class="fr"><label>Token</label><label class="inp" style="height:30px"><input class="mono" type="' + (S.authShow ? 'text' : 'password') + '" value="' + esc(a.token) + '">' + '<button class="ib sm" data-a="authEye" data-tip="显示 / 隐藏">' + I(S.authShow ? 'eyeOff' : 'eye', 14) + '</button></label></div>' +
      '<div class="fr"><label>预览</label><div class="prev">' + I('lock', 12) + '<span class="grow ell">Authorization: Bearer <span class="mu">' + (S.authShow ? esc(a.token) : '••••••••••••••••') + '</span></span><span class="tag mute">请求头</span></div></div>' +
      '<div class="tipbox">' + I('info', 14) + '<span>支持在 Token 中使用 <b>{{token}}</b> 等环境变量；敏感值不会写入日志。</span></div></div>';
  }
  function requestPane(t) {
    const n = flat[t.id] || t.node, rq = reqOf(t.id);
    const url = t.url != null ? t.url : n.url;
    const params = rq.params.filter((r) => r[3] !== 0 && (r[0] || r[1])).length, hdrs = rq.headers.filter((r) => r[3] !== 0).length;
    const busy = t.resp === 'sending';
    const urlRow = '<div class="urlrow"><div class="url ' + (S.focusUrl ? 'focus' : '') + '"><div class="mth m-' + n.m + '" data-a="mthMenu"><span>' + n.m + '</span>' + I('chevD', 12) + '</div><div class="ui" contenteditable="plaintext-only" spellcheck="false">' + varHtml(url) + '</div>' + (S.opt.env === 'urlbar' ? envChip('envchip') : '') + '</div>' +
      '<div class="send ' + (busy ? 'busy' : '') + '" data-a="' + (busy ? 'cancel' : 'send') + '">' + (busy ? '<i class="fill"></i><span class="sb">' + (S.opt.fx === 'ring' ? '<svg class="ic spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M12 3a9 9 0 1 0 9 9" opacity=".9"/></svg>' : I('stop', 13)) + '取消</span>' : '<span class="sb">' + I('send', 14) + '发送<span class="kbd" style="background:color-mix(in oklab,var(--acf) 16%,transparent);border-color:transparent;color:var(--acf)">⌘↵</span></span><span class="sc" data-a="sendMenu">' + I('chevD', 12) + '</span>') + '</div></div>';
    const tabsRow = '<div class="rq-tabs">' + [['params', 'Params', params], ['headers', 'Headers', hdrs], ['body', 'Body', rq.body.type !== 'none' ? 'dot' : 0], ['auth', 'Auth', rq.auth.type !== 'none' ? 'dot' : 0]].map((x) => '<span class="ut ' + (t.view === x[0] ? 'on' : '') + '" data-a="rqView" data-v="' + x[0] + '">' + x[1] + (x[2] === 'dot' ? '<i class="dotm"></i>' : x[2] ? '<span class="cnt ' + (t.view === x[0] ? 'ac' : '') + '">' + x[2] + '</span>' : '') + '</span>').join('') +
      '<div class="crumb"><span class="ell">' + esc(proj().name) + '</span>' + I('chevR', 10) + '<b>' + esc((flat[n.parent] || {}).name || '根目录') + '</b>' + I('chevR', 10) + '<span class="cur ell">' + esc(n.name) + '</span><span class="sep"></span>' + '<button class="btn sm ghost" data-a="save" ' + (t.dirty ? '' : 'style="opacity:.55"') + '>' + I('save', 12) + '保存<span class="kbd">⌘S</span></button>' + ib('copy', 'copyCurl', { cls: 'sm', sz: 13, tip: '复制为 cURL ⌘⇧C' }) + ib('target', 'noop', { cls: 'sm', sz: 13, tip: '在侧栏中定位' }) + '</div></div>';
    let body = '';
    if (t.view === 'params') body = '<div class="rq-body"><div class="hint">查询参数 · 修改后自动同步到地址栏<span class="lk" data-a="noop">' + I('list', 12) + '批量编辑</span></div>' + kvTable(rq.params.length ? rq.params : [['', '', '', 1]], 'params') + '</div>';
    else if (t.view === 'headers') body = '<div class="rq-body"><div class="hint">请求头 · 取消勾选即可临时禁用<span class="lk" data-a="noop">' + I('list', 12) + '批量编辑</span></div>' + kvTable(rq.headers, 'headers') + '</div>';
    else if (t.view === 'body') body = '<div class="rq-body">' + bodyEditor(rq, t) + '</div>';
    else body = '<div class="rq-body">' + authView(rq) + '</div>';
    return '<div class="req"><div class="req-top">' + urlRow + '</div>' + tabsRow + body + '</div>';
  }

  /* ==========================================================================
     响应区
     ========================================================================== */
  function respHeader(t) {
    const r = RESP[t.code] || RESP[200];
    const tabs = [['body', '响应体', 0], ['headers', '响应头', 7], ['cookies', 'Cookies', 2]];
    if (t.resp === 'stream') tabs.unshift(['stream', 'Stream', 7]);
    const live = t.resp === 'sending';
    let h = '<div class="resp-h">' + tabs.map((x) => '<span class="ut ' + (t.rtab === x[0] ? 'on' : '') + '" data-a="rtab" data-v="' + x[0] + '" style="' + (t.resp === 'none' || t.resp === 'error' || live ? 'opacity:.5;pointer-events:none' : '') + '">' + (x[0] === 'stream' ? '<span style="color:var(--info);display:grid">' + I('zap', 12) + '</span>' : '') + x[1] + (x[2] ? '<span class="cnt ' + (t.rtab === x[0] ? 'ac' : '') + '">' + x[2] + '</span>' : '') + '</span>').join('');
    h += '<div class="meta">';
    if (live) h += '<span class="sending-t"><i class="bdot"></i>发送中<span class="tm">' + (S.static ? '0.82s' : '0.00s') + '</span></span>';
    else if (t.resp === 'error') h += '<span class="pill err">' + I('errorC', 12) + '请求失败</span>';
    else if (t.resp === 'stream') h += '<span class="pill ok"><i class="bdot" style="width:6px;height:6px;background:var(--ok)"></i>200 流式</span><span class="mi" data-a="timing">' + I('clock', 12) + '3.02 s</span><span class="mi">' + I('layers', 12) + '7 条事件</span>';
    else if (t.resp === 'ok') h += '<span class="pill ' + r.cls + '">' + (r.cls === 'ok' ? '<i class="pdot" style="width:6px;height:6px;background:var(--ok)"></i>' : '') + r.text + '</span><span class="mi" data-a="timing" data-tip="查看响应时间分解">' + I('clock', 12) + r.ms + ' ms</span><span class="mi" data-a="timing" data-tip="响应大小">' + I('database', 12) + r.size + '</span><span class="mi" data-tip="远端地址 · TLS 1.3">' + I('lock', 12) + '</span>';
    return h + '</div></div>';
  }
  function respBody(t) {
    if (t.resp === 'none') return '<div class="empty-r"><div style="opacity:.9">' + logo(40, 'glyph') + '</div><div class="big">准备就绪</div><span>点击「发送」，响应会显示在这里</span><div class="hints"><span><span class="kbd">⌘↵</span>发送</span><span><span class="kbd">⌘K</span>快速打开</span><span><span class="kbd">⌘I</span>粘贴 cURL</span></div></div>';
    if (t.resp === 'sending') return '<div class="rb"><div class="sk"><i style="width:34%"></i><i style="width:62%"></i><i style="width:48%"></i><i style="width:71%"></i><i style="width:40%"></i><i style="width:56%"></i></div><div class="empty-r" style="flex:none;padding:0 0 18px"><span>正在等待响应首字节…</span></div></div>';
    if (t.resp === 'error') return '<div class="rb scroll"><div class="diag"><h4>' + I('errorC', 16) + '无法发送请求<span class="errcode">ECONNREFUSED</span></h4><div class="mono mu" style="font-size:11.5px">connect ECONNREFUSED 127.0.0.1:8080 · 耗时 31 ms</div><div class="lst"><div>' + I('check', 13) + '检查地址与端口是否正确，服务是否已启动</div><div>' + I('globe', 13) + (S.curEnv ? '当前环境「' + esc(env().name) + '」中的 <span class="vt">{{host}}</span> 指向 localhost' : '未选择环境，<span class="vt bad">{{host}}</span> 无法解析') + '</div><div>' + I('lock', 13) + '若走代理，请在 设置 → 通用 中检查网络配置</div></div><div style="display:flex;gap:8px;margin-top:2px"><button class="btn pri sm" data-a="send">' + I('refresh', 12) + '重试</button><button class="btn sm" data-a="toggleCon">' + I('terminal', 12) + '在控制台查看</button><button class="btn sm ghost" data-a="settings">打开设置</button></div></div></div>';
    if (t.rtab === 'stream' && t.resp === 'stream') {
      const ev = M.streamEvents;
      const head = '<div class="rtool"><div class="seg"><b class="' + (!t.merged ? 'on' : '') + '" data-a="merged" data-v="0">' + I('list', 12) + '事件</b><b class="' + (t.merged ? 'on' : '') + '" data-a="merged" data-v="1">' + I('sparkle', 12) + '合并文本</b></div><span class="sp"></span>' + (t.merged ? '' : '<label class="inp" style="width:150px;height:24px">' + I('search', 12) + '<input placeholder="搜索事件"></label>' + ib('wrap', 'noop', { cls: 'sm', tip: '自动换行' })) + ib('copy', 'copyResp', { cls: 'sm', tip: '复制' }) + ib('trash', 'noop', { cls: 'sm', tip: '清空' }) + '</div>';
      if (t.merged) return '<div class="rb">' + head + '<div class="merged scroll">' + esc(M.mergedText) + '<i class="caret" style="display:inline-block;width:2px;height:15px;background:var(--ac);vertical-align:-3px;margin-left:2px;animation:blink 1s steps(1) infinite"></i><div class="mu" style="margin-top:10px;font-size:11px">把 OpenAI / Anthropic / Gemini / Ollama 的增量片段按顺序拼成完整回答</div></div></div>';
      return '<div class="rb">' + head + '<div class="stream scroll" style="overflow:auto">' + ev.map((e, i) => '<div class="ev" style="animation-delay:' + i * 70 + 'ms"><span class="mu">+' + e[0] + 's</span><span class="evn ' + e[1] + '">' + e[1] + '</span><span class="ell">' + esc(e[2]) + '</span></div>').join('') + '</div></div>';
    }
    if (t.rtab === 'headers') return '<div class="rb"><div class="kv scroll"><div class="hd" style="grid-template-columns:34px 1fr 2.2fr 34px"><span></span><span>名称</span><span>值</span><span></span></div>' + M.respHeaders.map((r, i) => '<div class="kr" style="grid-template-columns:34px 1fr 2.2fr 34px;animation:rowIn var(--dur-slow) var(--ease-out) ' + i * 22 + 'ms both"><span class="ix">' + (i + 1) + '</span><span class="c mono" style="color:var(--info)">' + r[0] + '</span><span class="c mono ell">' + esc(r[1]) + '</span><span><button class="ib sm del" data-a="copyVal" data-tip="复制值">' + I('copy', 12) + '</button></span></div>').join('') + '</div></div>';
    if (t.rtab === 'cookies') return '<div class="rb"><div class="kv scroll"><div class="hd" style="grid-template-columns:34px 1fr 1.6fr 80px 110px 72px"><span></span><span>名称</span><span>值</span><span>路径</span><span>过期</span><span>属性</span></div>' + M.cookies.map((c, i) => '<div class="kr" style="grid-template-columns:34px 1fr 1.6fr 80px 110px 72px;animation:rowIn var(--dur-slow) var(--ease-out) ' + i * 22 + 'ms both"><span class="ix">' + (i + 1) + '</span><span class="c mono">' + c[0] + '</span><span class="c mono ell">' + c[1] + '</span><span class="c mono">' + c[2] + '</span><span class="c mu">' + c[3] + '</span><span class="c" style="gap:4px">' + (c[4] ? '<span class="tag mute">S</span>' : '') + (c[5] ? '<span class="tag mute">H</span>' : '') + '</span></div>').join('') + '</div></div>';
    const r = RESP[t.code] || RESP[200];
    const tool = '<div class="rtool"><span class="fmt" data-a="noop">' + I('code', 12) + 'JSON' + I('chevD', 10) + '</span><span class="fmt ac" data-a="noop">' + I('eye', 12) + '预览</span><span class="sp"></span>' + ib('wrap', 'noop', { cls: 'sm', tip: '自动换行', on: t.wrap }) + ib('expandAll', 'noop', { cls: 'sm', tip: '展开全部' }) + ib('collapseAll', 'noop', { cls: 'sm', tip: '折叠全部' }) + ib('search', 'noop', { cls: 'sm', tip: '查找 ⌘F' }) + '<span class="sep"></span>' + ib('copy', 'copyResp', { cls: 'sm', tip: '复制响应体' }) + ib('download', 'noop', { cls: 'sm', tip: '保存到文件' }) + '</div>';
    if (r.large) return '<div class="rb">' + tool + '<div class="bigcard"><span class="ico">' + I('database', 22) + '</span><b>响应较大（8.2 MB）</b><span>为保持界面流畅，仅显示前 2 MB。复制或保存可获取完整内容。</span><div class="acts"><button class="btn pri sm" data-a="noop">' + I('download', 12) + '保存到文件</button><button class="btn sm" data-a="noop">' + I('copy', 12) + '复制完整内容</button><button class="btn sm ghost" data-a="noop">' + I('eye', 12) + '仍然预览前 2 MB</button></div></div></div>';
    if (t.filter) return '<div class="rb">' + tool + '<div class="rfilter">' + I('filter', 13) + '<input class="mono" value="data.users[*].name"><span class="res">命中 2 项</span><button class="btn sm ghost" data-a="noop">' + I('x', 12) + '清除</button></div>' + codeBlock('[\n  "Alice",\n  "Bob"\n]') + '</div>';
    if (!r.body) return '<div class="rb">' + tool + '<div class="empty-r">' + I('okC', 26) + '<div class="big">空响应</div><span>服务器返回了 204，没有响应体</span></div></div>';
    return '<div class="rb">' + tool + codeBlock(r.body) + '</div>';
  }
  function responsePane(t) {
    return '<div class="resp">' + (t.resp === 'sending' && S.opt.fx === 'comet' ? '<div class="comet"></div>' : '') + respHeader(t) + respBody(t) + '</div>';
  }

  /* ---------- 环境页 / 项目主页 / 欢迎页 ---------- */
  function envPage(t) {
    const e = M.envs.find((x) => x.id === t.id);
    return '<div class="envp"><div class="envp-main"><div class="envp-h"><div class="grow"><h2>' + esc(e.name) + '</h2><p>变量可在请求的任意位置写成 <span class="vt">{{变量名}}</span> 使用，切换环境后自动替换。</p></div>' +
      '<button class="btn dng" data-a="delEnv">' + I('trash', 13) + '</button><button class="btn" data-a="setCurEnv" data-id="' + e.id + '">' + (S.curEnv === e.id ? I('check', 13) + '当前使用' : '设为当前环境') + '</button><button class="btn pri" data-a="toastSave">' + I('save', 13) + '保存</button></div>' +
      '<div class="kv scroll" style="flex:none"><div class="hd" style="grid-template-columns:34px 1fr 2fr 56px"><span></span><span>变量名</span><span>值</span><span></span></div>' + e.vars.map((v, i) => '<div class="kr ' + (v[3] ? '' : 'off') + '" style="grid-template-columns:34px 1fr 2fr 56px;animation:rowIn var(--dur-slow) var(--ease-out) ' + i * 26 + 'ms both"><span>' + cb(!!v[3], 'kvToggle') + '</span><span class="c"><input value="' + esc(v[0]) + '"></span><span class="c"><input type="' + (v[2] ? 'password' : 'text') + '" value="' + esc(v[1]) + '"></span><span style="display:flex;gap:0"><button class="ib sm ' + (v[2] ? 'on' : '') + '" data-a="noop" data-tip="标为密文（遮罩显示）">' + I(v[2] ? 'lock' : 'eye', 12) + '</button><button class="ib sm del" data-a="noop">' + I('x', 12) + '</button></span></div>').join('') + '<div class="kr ghost" style="grid-template-columns:34px 1fr 2fr 56px"><span></span><span class="c"><input placeholder="添加变量"></span><span class="c"><input placeholder="值"></span><span></span></div></div>' +
      '<div class="foot-note">' + I('lock', 12) + '带锁的变量会在界面上遮罩，且不会写入日志。</div></div>' +
      '<div class="envp-side"><div style="display:flex;align-items:center;gap:6px;font-weight:650">' + I('zap', 13) + '动态变量<span class="mu" style="font-weight:450;font-size:11px">发送时求值</span></div>' + M.dynVars.map((d) => '<div class="dv" data-a="copyVal"><code>' + d[0] + '</code><span class="fg2">' + d[1] + '</span><span class="vv ell">' + d[2] + '</span><span class="cp">' + I('copy', 12) + '</span></div>').join('') + '</div></div>';
  }
  function projCard(p, i) {
    return '<div class="pcard ' + (S.homeSel === p.id ? 'sel' : '') + '" style="animation-delay:' + i * 40 + 'ms" data-a="homeSel" data-id="' + p.id + '"><div class="top"><span class="pav" style="background:' + pjc(p.color) + '">' + esc(p.name[0]) + '</span><div class="grow"><div class="nm ell">' + esc(p.name) + '</div><div class="mu" style="font-size:11px">' + p.reqs + ' 个请求 · ' + p.folders + ' 个文件夹</div></div>' + (p.id === S.projId ? '<span class="pill ac">当前</span>' : '') + '</div><div class="bt"><span>' + p.updated + '</span><span class="sp"></span>' + I('globe', 12) + p.envs + ' 个环境</div></div>';
  }
  function projRow(p, i) {
    return '<div class="pr ' + (S.homeSel === p.id ? 'sel' : '') + '" style="animation-delay:' + i * 35 + 'ms" data-a="homeSel" data-id="' + p.id + '"><span class="pav" style="background:' + pjc(p.color) + '">' + esc(p.name[0]) + '</span><b class="ell">' + esc(p.name) + (p.id === S.projId ? ' <span class="pill ac" style="height:16px;margin-left:4px">当前</span>' : '') + '</b><span class="mu ell">' + esc(p.desc || '—') + '</span><span class="mono">' + p.reqs + '</span><span class="mono">' + p.envs + '</span><span class="mu">' + p.updated + '</span>' + ib('more', 'noop', { cls: 'sm', sz: 14 }) + '</div>';
  }
  function homeEmpty() {
    return '<div class="home"><div class="home-main"><div class="home-h"><div class="grow"><h1>项目</h1><p>还没有项目。新建一个项目，把相关请求与环境放在一起。</p></div></div><div class="home-empty"><span class="ico">' + logo(44, 'glyph') + '</span><b>创建你的第一个项目</b><span>项目会保存请求、环境与历史记录，并拥有独立的主题色。</span><div class="new"><label class="inp" style="width:220px;height:30px">' + I('plus', 13) + '<input placeholder="输入项目名，回车新建"></label><button class="btn pri" style="height:30px">新建项目</button></div><button class="btn ghost sm" data-a="importOv" data-tab="file">' + I('upload', 12) + '或从 Postman / OpenAPI 导入</button></div></div></div>';
  }
  function homeView() {
    if (S.noProj) return homeEmpty();
    const p = M.projects.find((x) => x.id === S.homeSel) || M.projects[0];
    const list = M.projects.filter((x) => !S.homeQ || x.name.toLowerCase().includes(S.homeQ.toLowerCase()));
    const main = '<div class="home-main scroll"><div class="home-h"><div class="grow"><h1>项目</h1><p>选择项目继续工作。每个项目有自己的主题色，会应用到选中态、高亮和发送按钮。</p></div><label class="inp" style="width:210px;height:30px">' + I('plus', 13) + '<input data-in="newproj" placeholder="输入项目名，回车新建"></label><button class="btn pri" style="height:30px" data-a="newProj">新建项目</button></div>' +
      '<div class="home-bar"><label class="inp" style="width:200px">' + I('search', 13) + '<input data-in="homeQ" placeholder="搜索项目…" value="' + esc(S.homeQ) + '"></label><span class="mu">共 ' + list.length + ' 个项目</span><span class="sp"></span><div class="seg"><b class="' + (S.opt.home === 'grid' ? 'on' : '') + '" data-a="homeMode" data-v="grid" data-tip="卡片">' + I('grid', 13) + '</b><b class="' + (S.opt.home === 'list' ? 'on' : '') + '" data-a="homeMode" data-v="list" data-tip="列表">' + I('list', 13) + '</b></div>' + sel('排序：' + S.homeSort, 'noop') + '</div>' +
      (S.opt.home === 'grid' ? '<div class="pgrid">' + list.map(projCard).join('') + '</div>' : '<div class="plist"><div class="pr hd"><span></span><span>名称</span><span>描述</span><span>请求</span><span>环境</span><span>更新</span><span></span></div>' + list.map(projRow).join('') + '</div>') + '</div>';
    const side = '<div class="home-side"><div class="hd"><span class="pav" style="background:' + pjc(p.color) + '">' + esc(p.name[0]) + '</span><div><b style="font-size:14px">' + esc(p.name) + '</b><div class="mu" style="font-size:11px">' + p.reqs + ' 个请求 · ' + p.folders + ' 个文件夹</div></div></div>' +
      '<div class="fld-l"><label>项目名称</label><label class="inp" style="height:28px"><input value="' + esc(p.name) + '"></label></div><div class="fld-l"><label>描述</label><label class="inp" style="height:28px"><input value="' + esc(p.desc) + '" placeholder="添加描述"></label></div>' +
      '<div class="fld-l"><label>主题色</label><div class="swatches">' + '<i class="auto ' + (!p.color ? 'on' : '') + '" style="background:var(--ac-base)" data-a="pcolor" data-c="" data-tip="跟随外观配色"></i>' + M.PROJECT_COLORS.map((c) => '<i class="' + (c === p.color ? 'on' : '') + '" style="background:' + pjc(c) + '" data-a="pcolor" data-c="' + c + '"></i>').join('') + '</div></div>' +
      '<div class="fld-l"><label>实时预览</label><div class="livep" style="--ac:' + pjc(p.color) + ';--ac-soft:color-mix(in oklab,' + pjc(p.color) + ' 14%,transparent)"><div style="display:flex;gap:12px;border-bottom:1px solid var(--bd)"><span class="ut on" style="height:24px;padding:0 2px;color:' + pjc(p.color) + '">Params</span><span class="ut" style="height:24px;padding:0 2px">Headers</span></div><div style="display:flex;gap:8px;align-items:center;height:24px;border-radius:6px;padding:0 8px;background:var(--ac-soft)"><span class="m m-GET">GET</span><span style="color:' + pjc(p.color) + ';font-weight:600">示例请求</span></div><div style="display:flex;align-items:center"><span class="mu">发送按钮</span><span class="sp"></span><span class="btn pri sm" style="background:' + pjc(p.color) + ';border-color:' + pjc(p.color) + '">发送</span></div></div></div>' +
      '<span class="sp"></span><button class="btn pri" style="height:30px" data-a="openProj" data-id="' + p.id + '">打开项目</button><div style="display:flex;gap:6px"><button class="btn grow">' + I('pencil', 12) + '重命名</button><button class="btn grow">' + I('upload', 12) + '导出</button><button class="btn dng">' + I('trash', 12) + '</button></div></div>';
    return '<div class="home">' + main + side + '</div>';
  }
  function welcome() {
    const rec = [['GET', '获取用户列表', '用户服务'], ['POST', '创建用户', '用户服务'], ['GET', '订单列表', '订单服务']];
    return '<div class="welcome"><div class="inner"><div class="logo">' + logo(64) + '</div><h2>MiniPost</h2><p>一个简洁、快速的 API 调试工具</p><div class="acts-l">' +
      [['plus', '新建请求', '⌘ N', 'newReq'], ['upload', '导入 Postman / OpenAPI / cURL', '⌘ O', 'importOv'], ['code', '粘贴 cURL 命令', '⌘ I', 'curlOv'], ['search', '快速打开请求、环境、命令', '⌘ K', 'quick']].map((a) => '<div class="a" data-a="' + a[3] + '" data-tab="' + (a[3] === 'curlOv' ? 'curl' : 'file') + '"><span class="ico">' + I(a[0], 13) + '</span><span class="tx">' + a[1] + '</span><span class="kbd">' + a[2] + '</span></div>').join('') + '</div>' +
      '<div class="recent"><h6>最近打开</h6>' + rec.map((r) => '<div class="rr" data-a="openByName" data-n="' + r[1] + '">' + mbadge(r[0]) + '<span class="grow ell">' + r[1] + '</span><span class="mu" style="font-size:11px">' + r[2] + '</span></div>').join('') + '</div></div></div>';
  }
  Object.assign(MPA, { titlebar, sidebar, railHtml, tabsHtml, requestPane, responsePane, envPage, homeView, welcome, kvTable });

  /* ==========================================================================
     控制台 / 状态栏 / 工作区
     ========================================================================== */
  function consolePanel() {
    const rows = S.conRows.filter((r) => S.con.filter === 'all' || (S.con.filter === 'ok' && r[3] >= 200 && r[3] < 400) || (S.con.filter === 'err' && (r[3] === 0 || r[3] >= 400)));
    return '<div class="con"><div class="con-h">' + I('terminal', 13) + '控制台<span class="cnt">' + S.conRows.length + '</span><div class="seg" style="margin-left:6px">' + [['all', '全部'], ['ok', '成功'], ['err', '错误']].map((x) => '<b class="' + (S.con.filter === x[0] ? 'on' : '') + '" data-a="conFilter" data-v="' + x[0] + '" style="height:20px;padding:0 9px">' + x[1] + '</b>').join('') + '</div><span class="sp"></span><button class="btn sm ghost" data-a="conClear">清空</button>' + ib('x', 'toggleCon', { cls: 'sm', sz: 13 }) + '</div><div class="scroll" style="flex:1">' +
      (rows.length ? rows.map((r, i) => '<div class="con-r ' + (S.con.ex === i ? 'ex' : '') + '" data-a="conEx" data-i="' + i + '"><span class="mu">' + r[0] + '</span><span class="' + (r[3] === 0 || r[3] >= 400 ? 'st-err' : 'st-ok') + '">' + (r[3] === 0 || r[3] >= 400 ? '✕' : '←') + '</span>' + mbadge(r[1]) + '<span class="ell">' + varHtml(r[2]) + '</span><span class="' + (r[3] === 0 || r[3] >= 400 ? 'st-err' : 'st-ok') + '" style="font-weight:600">' + (r[3] || 'ERR') + '</span><span class="mu">' + r[4] + '</span><span class="mu">' + r[5] + '</span></div>' + (S.con.ex === i ? '<div class="con-d">GET ' + esc(r[2].replace('{{host}}', 'https://api.dev.acme.io')) + '<br>状态: ' + (r[3] || '连接失败') + ' · 耗时 ' + r[4] + '<br>响应大小: ' + r[5] + '</div>' : '')).join('') : '<div class="empty-r" style="padding:24px">' + I('terminal', 22) + '<span>没有符合筛选的日志</span></div>') + '</div></div>';
  }
  function statusBar() {
    const l = S.last, e = S.opt;
    return '<div class="status"><span class="sbtn ' + (S.con.open ? 'on' : '') + '" data-a="toggleCon">' + I('terminal', 12) + '控制台<span class="cnt" style="height:14px">' + S.conRows.length + '</span></span><span class="sbtn" data-a="cookies">' + I('cookie', 12) + 'Cookies (3)</span>' +
      '<span class="last"><span class="m-' + l.m + '" style="font-weight:700">' + l.m + '</span>' + esc(l.path.length > 30 ? l.path.slice(0, 30) + '…' : l.path) + ' → <span class="' + (l.code >= 400 || !l.code ? 'st-err' : 'st-ok') + '" style="font-weight:700">' + (l.code || 'ERR') + '</span> · ' + l.ms + 'ms</span><span class="sp"></span>' +
      (e.env === 'status' ? envChip() : '') + '<span class="sbtn" data-a="toggleDock" data-tip="切换响应布局">' + I(e.dock === 'bottom' ? 'dockBottom' : 'dockRight', 13) + '</span><span class="sbtn" data-a="toggleSide" data-tip="侧栏 ⌘B">' + I('sidebar', 13) + '</span></div>';
  }
  function workArea() {
    if (S.view === 'home') return '<div class="work">' + homeView() + '</div>';
    if (S.view === 'settings') return '<div class="work">' + (MPA.settingsPage ? MPA.settingsPage() : '') + '</div>';
    const t = curTab();
    if (!t) return '<div class="work">' + welcome() + '</div>';
    if (t.kind === 'env') return '<div class="work">' + envPage(t) + '</div>';
    const right = S.opt.dock === 'right';
    const reqP = requestPane(t), resP = responsePane(t);
    const pct = S.splitPct;
    return '<div class="work"><div class="split ' + (right ? 'right' : '') + '"><div class="pane-req" style="flex:0 0 ' + (right ? 52 : pct) + '%">' + reqP + '</div><div class="gutter" data-a="noop"></div><div class="grow" style="display:flex;min-width:0;min-height:0;flex-direction:column">' + resP.replace('class="resp"', 'class="resp" style="flex:1"') + '</div></div></div>';
  }
  function mainCol() {
    const tabsStrip = !tabsInTitlebar() && S.view === 'work' ? tabsHtml('') : '';
    return '<div class="main">' + tabsStrip + workArea() + (S.con.open ? consolePanel() : '') + statusBar() + '</div>';
  }
  MPA.renderApp = function () {
    const win = $('#win');
    if (!win) return;
    win.dataset.dir = S.dir; win.dataset.scheme = S.scheme; win.dataset.mode = S.mode;
    win.classList.toggle('static', !!S.static);
    $('#app').innerHTML = titlebar() + '<div class="body">' + (S.opt.nav === 'rail' ? railHtml() : '') + sidebar() + mainCol() + '</div>';
  };
})();
