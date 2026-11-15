/* ==========================================================================
   MiniPost 设计稿原型 · 弹层、气泡菜单、提示与交互分发
   ========================================================================== */
(function () {
  'use strict';
  const MPA = window.MPA, M = window.MP, S = MPA.S, I = window.I;
  const { $, $$, esc, ib, mbadge, cb, sw, sel, flat, proj, env, curTab } = MPA;
  const A = (MPA.A = {});

  /* ---------- 设置内容（弹窗与整页共用） ---------- */
  const SCHEMES = [['mist', '雾蓝', ['#4f52d9', '#eef0f5', '#1b1e29']], ['ink', '墨白', ['#17171a', '#efefed', '#17171a']], ['jade', '松石', ['#08806f', '#ebf0ee', '#17201e']], ['amber', '琥珀', ['#c2570c', '#f3efe8', '#231d14']], ['rose', '玫瑰', ['#d6336c', '#f5edf0', '#261a1f']]];
  const NAV = [['general', '通用', 'sliders'], ['appearance', '外观', 'palette'], ['data', '数据', 'database'], ['shortcuts', '快捷键', 'keyboard'], ['about', '关于', 'info']];
  const srow = (t, d, ctl) => '<div class="srow"><div class="tx"><b>' + t + '</b><span>' + d + '</span></div>' + ctl + '</div>';
  const numf = (v, unit) => '<label class="inp" style="width:104px"><input style="text-align:right" value="' + v + '"><span class="unit">' + unit + '</span></label>';
  function miniWin(scheme, mode, half) {
    const inner = (m, w) => '<div data-scheme="' + scheme + '" data-mode="' + m + '" style="display:flex;width:' + w + ';background:var(--bg)"><div class="ms" style="background:var(--side);width:36%;padding:7px 5px;display:grid;gap:3px;align-content:start"><i style="background:var(--fg);opacity:.35;width:70%"></i><i style="background:var(--fg);opacity:.18;width:50%"></i><i style="background:var(--fg);opacity:.18;width:60%"></i></div><div class="mc" style="flex:1;padding:7px;display:grid;gap:3px;align-content:start"><i style="background:var(--ac);width:60%"></i><i style="background:var(--fg);opacity:.2"></i><i style="background:var(--fg);opacity:.2;width:75%"></i></div></div>';
    return '<div class="mini">' + (half ? inner('light', '50%') + inner('dark', '50%') : inner(mode, '100%')) + '</div>';
  }
  function settingsBody(page) {
    const st = S.set;
    if (page === 'general') {
      return '<div class="sgrp">请求</div>' + srow('HTTP 版本', '发送请求时优先使用的 HTTP 版本。', sel(st.http, 'noop', 'style="width:120px"')) + srow('请求超时', '单位毫秒，0 表示不主动超时。', numf(st.timeout, 'ms')) + srow('最大响应大小', '单位 MB，0 表示不限制。', numf(st.maxMB, 'MB')) + srow('响应格式检测', '自动按 Content-Type 识别，JSON 优先使用 JSON 视图。', sel(st.fmt, 'noop', 'style="width:120px"')) +
        '<div class="sgrp">安全与网络</div>' + srow('SSL 证书校验', '关闭后会跳过证书校验，仅建议在调试环境使用。', sw(st.ssl, 'ssl')) + srow('自动跟随重定向', '301 / 302 / 307 / 308，行为与 Postman 一致。', sw(st.redirect, 'redirect')) + srow('禁用 Cookies', '开启后不再自动保存与发送 Cookie。', sw(st.cookies, 'cookies'));
    }
    if (page === 'appearance') {
      const cur = S.mode;
      return '<div class="sgrp">外观主题</div><div class="thumbs">' + [['light', '浅色'], ['dark', '深色'], ['system', '跟随系统']].map((x) => '<div class="thumb ' + ((S.themeSrc || cur) === x[0] ? 'on' : '') + '" data-a="modeSet" data-v="' + x[0] + '">' + miniWin(S.scheme, x[0], x[0] === 'system') + '<div class="lb"><span>' + x[1] + '</span></div></div>').join('') + '</div>' +
        '<div class="sgrp">配色<span class="tag">新增 2 套</span></div><div class="thumbs five">' + SCHEMES.map((x) => '<div class="thumb ' + (S.scheme === x[0] ? 'on' : '') + '" data-a="schemeSet" data-v="' + x[0] + '">' + miniWin(x[0], S.mode) + '<div class="lb"><span>' + x[1] + '</span><span class="dots">' + x[2].map((c) => '<i style="background:' + c + '"></i>').join('') + '</span></div></div>').join('') + '</div>' +
        '<div class="mu" style="font-size:11px;margin:2px 0 4px">项目主题色仍然可用：它会覆盖这里的强调色，只影响选中态、高亮和发送按钮。</div>' +
        srow('语言', '立即生效，并记住你的选择。', '<div class="seg"><b class="' + (st.lang === '中文' ? 'on' : '') + '" data-a="lang" data-v="中文">中文</b><b class="' + (st.lang === 'English' ? 'on' : '') + '" data-a="lang" data-v="English">English</b></div>') +
        srow('界面字体', '调整菜单、标签、按钮和面板的文字大小。', '<div class="fs-chips">' + [10, 11, 12, 13, 14, 15, 16].map((n) => '<b class="' + (st.fs === n ? 'on' : '') + '" data-a="fs" data-v="' + n + '">' + n + '</b>').join('') + '</div>') +
        srow('界面密度 <span class="tag">建议新增</span>', '紧凑为默认；舒适会放大行高与间距，适合触控板与大屏。', '<div class="seg"><b class="on">紧凑</b><b>标准</b><b>舒适</b></div>') +
        srow('Logo 样式 <span class="tag">建议新增</span>', '应用图标跟随主题色（着色版），或固定使用品牌色。', '<div class="seg"><b class="on">跟随主题</b><b>品牌色</b></div>');
    }
    if (page === 'data') {
      return srow('自动备份', '定时备份请求、环境和历史记录。', sw(st.autoBackup, 'autoBackup')) + srow('备份间隔', '范围 5 – 1440 分钟。', numf(st.interval, '分钟')) +
        '<div class="sgrp">立即操作</div><div style="display:flex;gap:8px;padding:6px 0"><button class="btn" data-a="toast" data-t="ok" data-m="备份完成|已保存到 ~/.minipost/backups/2026-11-15-1042.zip">' + I('download', 13) + '立即备份</button><button class="btn" data-a="restoreDlg">' + I('upload', 13) + '恢复备份</button></div><div class="mu" style="font-size:11px">恢复会覆盖当前数据，建议先备份一次。</div>' +
        '<div class="sgrp">最近备份</div>' + [['2026-11-15 10:42', '48 KB'], ['2026-11-14 22:10', '46 KB']].map((b) => '<div class="bk">' + I('database', 14) + '<span class="grow mono">' + b[0] + '</span><span class="mu">' + b[1] + '</span><button class="btn sm ghost">恢复</button></div>').join('') +
        '<div class="sgrp">数据位置</div><div style="display:flex;align-items:center;gap:10px"><span class="mono grow" style="color:var(--fg2)">~/.minipost</span><button class="btn sm" data-a="noop">' + I('folderOpen', 12).replace('folderOpen', 'folderOpen') + '在 Finder 中显示</button></div>';
    }
    if (page === 'shortcuts') return '<div style="padding-top:6px">' + M.shortcuts.map((s) => '<div class="sc-r"><span>' + s[0] + '</span><span class="kbd">' + s[1] + '</span></div>').join('') + '</div>';
    const u = S.upd || 'latest';
    const updCard = u === 'downloading' ? '<div class="upd"><div style="display:flex;align-items:center;gap:8px"><b>正在下载 2.1.0</b><span class="mu">62% · 8.4 / 13.5 MB</span><span class="sp"></span><button class="btn sm">取消</button></div><div class="pr"><i></i></div><span class="mu" style="font-size:11px">下载好后会提示你重启，当前打开的标签会自动恢复。</span></div>' :
      u === 'ready' ? '<div class="upd" style="border-color:var(--ac-line);background:var(--ac-soft)"><div style="display:flex;align-items:center;gap:8px"><span class="toast ti ac" style="display:grid;width:20px;height:20px;border-radius:50%;background:var(--ac);color:var(--acf);place-items:center">' + I('download', 12) + '</span><b>2.1.0 已下载完成</b><span class="sp"></span><button class="btn pri sm" data-a="restart">重启更新</button></div><span class="fg2" style="font-size:11px">· 新增 5 套配色与响应时间瀑布图 · 修复 SSE 长连接断流<br>重启后生效，当前打开的标签会自动恢复。</span></div>' :
      '<div class="upd"><div style="display:flex;align-items:center;gap:8px"><span style="color:var(--ok);display:grid">' + I('okC', 16) + '</span><b>已是最新版本</b><span class="sp"></span><button class="btn sm" data-a="checkUpd">检查更新</button></div></div>';
    return '<div style="display:grid;justify-items:center;gap:6px;padding:14px 0 8px;text-align:center"><div style="filter:drop-shadow(0 8px 18px color-mix(in oklab,var(--ac) 30%,transparent))">' + MPA.logo(72) + '</div><b style="font-size:16px;margin-top:4px">MiniPost</b><span class="mu">版本 2.1.0 (build 214)</span><span class="fg2" style="max-width:420px;line-height:1.55">轻量的 HTTP 调试工具。支持多项目、环境变量、历史记录、响应预览与 Cookie 管理。</span></div>' + updCard + srow('自动下载更新', '每小时在后台检查一次，发现新版本自动下载，下载好后提示你重启。', sw(st.autoUpdate, 'autoUpdate')) +
      '<div style="display:grid;gap:6px;margin-top:8px;color:var(--fg2)"><div style="display:flex;gap:8px;align-items:center"><span style="color:var(--ok);display:grid">' + I('okC', 13) + '</span>数据只保存在本机 ~/.minipost，仅所有者可读写。</div><div style="display:flex;gap:8px;align-items:center"><span style="color:var(--ok);display:grid">' + I('okC', 13) + '</span>日志不记录请求体、令牌和 Cookie。</div></div>';
  }
  const pageTitle = (p) => NAV.find((n) => n[0] === p)[1];
  function settingsFoot(p) { return '<div class="set-cf"><span>修改会立即生效并自动保存</span>' + (p === 'general' ? '<button class="btn sm" data-a="noop">恢复默认</button>' : '') + '</div>'; }
  function settingsModal() {
    const p = S.setPage;
    return '<div class="dlg set"><div class="set-n"><h6>设置</h6>' + NAV.map((n) => '<div class="ni ' + (p === n[0] ? 'on' : '') + '" data-a="setPage" data-p="' + n[0] + '">' + I(n[2], 14) + n[1] + '</div>').join('') + '<span class="ver">MiniPost 2.1.0</span></div><div class="set-c"><div class="set-ch"><h3>' + pageTitle(p) + '</h3>' + ib('x', 'closeOv', { sz: 14 }) + '</div><div class="set-cb scroll">' + settingsBody(p) + '</div>' + settingsFoot(p) + '</div></div>';
  }
  MPA.settingsPage = function () {
    const p = S.setPage;
    return '<div class="set" style="display:flex;width:auto;height:100%;max-height:none;flex:1;animation:paneIn var(--dur-slow) var(--ease-out)"><div class="set-n" style="width:196px;padding-top:10px"><div class="btn ghost" style="justify-content:flex-start;margin-bottom:8px" data-a="closeSettings">' + I('arrowL', 13) + '返回</div>' + NAV.map((n) => '<div class="ni ' + (p === n[0] ? 'on' : '') + '" data-a="setPage" data-p="' + n[0] + '">' + I(n[2], 14) + n[1] + '</div>').join('') + '<span class="ver">MiniPost 2.1.0</span></div><div class="set-c"><div class="set-ch"><h3>' + pageTitle(p) + '</h3></div><div class="set-cb scroll"><div style="max-width:660px;margin:0 auto;padding:4px 0">' + settingsBody(p) + '</div></div>' + settingsFoot(p) + '</div></div>';
  };

  /* ---------- 导入 ---------- */
  function importDlg() {
    const im = S.imp;
    const tabs = [['file', '文件导入', 'upload'], ['url', 'URL 导入', 'link'], ['curl', 'cURL 导入', 'code']];
    let body = '';
    if (im.tab === 'file') {
      body = '<div class="drop ' + (im.over ? 'over' : '') + '" data-a="impPick">' + I('upload', 22) + '<b>' + (im.over ? '松开鼠标即可导入' : '选择或拖入导入文件') + '</b><span style="font-size:11px">Postman Collection / Environment / OpenAPI / Swagger（JSON / YAML）</span></div>' +
        (im.ok ? '<div class="res"><div class="rh">' + I('okC', 15) + '<span style="color:var(--ok)">已识别：Postman Collection v2.1</span><span class="sp"></span><span class="mu" style="font-weight:450">acme-open-api.json · 86 KB</span></div><div class="stats"><span class="pill info">24 个请求</span><span class="pill">6 个文件夹</span><span class="pill">3 个环境变量</span><span class="pill warn">2 个同名冲突</span></div></div><div class="fld-l"><label>遇到同名请求时</label><div class="strat">' +
          [['merge', '合并', '保留已有，仅新增缺失项', 'layers'], ['over', '覆盖', '用导入内容替换同名项', 'refresh'], ['copy', '保留两者', '同名项另存为副本', 'copy']].map((x) => '<div class="sc3 ' + (im.strat === x[0] ? 'on' : '') + '" data-a="impStrat" data-v="' + x[0] + '"><b>' + I(x[3], 13) + x[1] + '</b><span>' + x[2] + '</span></div>').join('') + '</div></div>'
          : '<div class="res"><div class="mu" style="font-size:11px">识别结果</div><b>尚未选择文件</b><span class="fg2">选择文件后将自动识别类型。</span></div>');
    } else if (im.tab === 'url') {
      body = '<div class="fld-l"><label>文件地址</label><label class="inp" style="height:30px">' + I('link', 13) + '<input class="mono" placeholder="https://example.com/openapi.json" value="' + (S.static || im.ok ? 'https://api.acme.io/openapi.json' : '') + '"></label></div><div class="res"><div class="rh">' + I('info', 14) + '支持直接导入公开的 OpenAPI / Swagger 地址</div><span class="fg2">私有地址请先下载为文件再导入。</span></div>';
    } else {
      body = '<div class="ta focus">curl -X POST \'https://api.acme.io/users\' \\\n  -H \'Content-Type: application/json\' \\\n  -H \'Authorization: Bearer {{token}}\' \\\n  -d \'{"name":"Alice","email":"alice@example.com"}\'</div><div class="parsed">' + I('okC', 14) + '已解析<span class="tag"><b class="m m-POST">POST</b></span><span class="tag">api.acme.io/users</span><span class="tag">2 个请求头</span><span class="tag">JSON 请求体</span></div><div class="mu" style="font-size:11px">按 <span class="kbd">Enter</span> 导入，<span class="kbd">Shift Enter</span> 换行</div>';
    }
    const label = im.tab === 'file' ? '导入文件' : im.tab === 'url' ? '导入地址' : '导入 cURL';
    const can = im.tab === 'curl' || im.ok;
    return '<div class="dlg imp"><div class="dlg-h"><span class="ico">' + I('upload', 15) + '</span><div class="grow"><h3>导入</h3><p>支持文件、URL 与 cURL 导入</p></div>' + ib('x', 'closeOv', { sz: 14 }) + '</div><div class="dlg-b"><div class="seg full">' + tabs.map((t) => '<b class="' + (im.tab === t[0] ? 'on' : '') + '" data-a="impTab" data-v="' + t[0] + '">' + I(t[2], 13) + t[1] + '</b>').join('') + '</div>' + body + '</div><div class="dlg-f"><button class="btn" data-a="closeOv">取消</button><button class="btn pri ' + (can ? '' : 'dis') + '" data-a="doImport">' + label + '</button></div></div>';
  }

  /* ---------- Cookie 管理器 ---------- */
  function cookieDlg() {
    const rows = [['sid', 'a1b2c3d4e5f6', '/', '会话', 1, 1], ['_ga', 'GA1.2.1048.1731', '/', '2028-11-15', 0, 0]];
    return '<div class="dlg ck"><div class="ck-l"><div style="display:flex;align-items:center;gap:8px;font-weight:650;padding:0 2px">' + I('cookie', 14) + 'Cookie 管理器</div><div style="display:flex;gap:6px"><label class="inp grow">' + I('plus', 13) + '<input placeholder="输入域名"></label><button class="btn">添加</button></div><label class="inp">' + I('search', 13) + '<input placeholder="搜索 Cookie"></label><div class="scroll grow">' + M.cookieDomains.map((d) => '<div class="row ' + (d.d === S.cookieDom ? 'sel' : '') + '" style="margin:0" data-a="ckDom" data-d="' + d.d + '"><span class="fic">' + I('globe', 14) + '</span><span class="nm">' + d.d + '</span><span class="sub">' + d.n + '</span></div>').join('') + '</div></div>' +
      '<div class="ck-r"><div class="dlg-h" style="border:0"><span style="color:var(--mu);display:grid">' + I('globe', 15) + '</span><h3>' + S.cookieDom + '</h3><span class="cnt">' + rows.length + '</span><span class="sp"></span><button class="btn sm">' + I('copy', 12) + '粘贴识别</button><button class="btn sm">' + I('plus', 12) + '添加 Cookie</button><button class="btn sm">清空当前域名</button><button class="btn sm dng">清空全部</button>' + ib('x', 'closeOv', { sz: 14 }) + '</div>' +
      '<div class="ck-t hd"><span style="text-align:center">启用</span><span>名称</span><span>值</span><span>路径</span><span>过期时间</span><span>Secure</span><span>HttpOnly</span><span></span></div>' + rows.map((r) => '<div class="ck-t rw"><span style="text-align:center">' + cb(true, 'kvToggle') + '</span><span><input value="' + r[0] + '"></span><span><input value="' + r[1] + '"></span><span><input value="' + r[2] + '"></span><span><input value="' + r[3] + '"></span><span style="text-align:center">' + cb(!!r[4], 'kvToggle') + '</span><span style="text-align:center">' + cb(!!r[5], 'kvToggle') + '</span><span><button class="ib sm" data-a="noop">' + I('trash', 13) + '</button></span></div>').join('') + '<div class="empty-r" style="padding:30px;flex:1">' + I('cookie', 22) + '<span>勾选「启用」的 Cookie 会随请求自动发送</span></div></div></div>';
  }

  /* ---------- 快速打开 ⌘K ---------- */
  function hlMatch(text, q) {
    if (!q) return esc(text);
    const i = text.toLowerCase().indexOf(q.toLowerCase());
    return i < 0 ? esc(text) : esc(text.slice(0, i)) + '<mark>' + esc(text.slice(i, i + q.length)) + '</mark>' + esc(text.slice(i + q.length));
  }
  function quickItems() {
    const q = S.qk.q.trim().toLowerCase();
    const reqs = Object.values(flat).filter((n) => n.t === 'r' && (!q || (n.name + n.url + n.m).toLowerCase().includes(q))).slice(0, q ? 6 : 4);
    const envs = M.envs.filter((e) => !q || e.name.toLowerCase().includes(q) || '环境'.includes(q));
    const cmds = [['plus', '新建请求', '⌘N'], ['code', '导入 cURL', '⌘I'], ['sidebar', '显示 / 隐藏侧栏', '⌘B'], ['terminal', '显示 / 隐藏控制台', '⌘J'], ['moon', '切换明暗主题', ''], ['sliders', '设置', '⌘,']].filter((c) => !q || c[1].toLowerCase().includes(q));
    const out = [];
    if (reqs.length) out.push(['请求', reqs.map((n) => ({ kind: 'req', n }))]);
    if (envs.length) out.push(['环境', envs.map((e) => ({ kind: 'env', e }))]);
    if (cmds.length) out.push(['命令', cmds.map((c) => ({ kind: 'cmd', c }))]);
    return out;
  }
  function quickList() {
    const groups = quickItems();
    if (!groups.length) return '<div class="empty-r" style="padding:34px">' + I('search', 22) + '<span>没有匹配的结果</span></div>';
    let idx = 0, h = '';
    groups.forEach((g) => {
      h += '<div class="qk-g">' + g[0] + '</div>';
      g[1].forEach((it) => {
        const on = idx === S.qk.sel;
        const q = S.qk.q.trim();
        if (it.kind === 'req') h += '<div class="qk-r ' + (on ? 'on' : '') + '" data-a="qkPick" data-id="' + it.n.id + '">' + mbadge(it.n.m) + '<span class="nm"><b>' + hlMatch(it.n.name, q) + '</b><em>' + esc((flat[it.n.parent] || {}).name || '根目录') + '</em></span>' + (on ? '<span class="kbd">↵</span>' : '') + '</div>';
        else if (it.kind === 'env') h += '<div class="qk-r ' + (on ? 'on' : '') + '" data-a="qkEnv" data-id="' + it.e.id + '"><span class="ico">' + I('globe', 12) + '</span><span class="nm"><b>' + hlMatch(it.e.name, q) + '</b><em>切换为当前环境</em></span>' + (S.curEnv === it.e.id ? '<span class="cur-badge">当前</span>' : on ? '<span class="kbd">↵</span>' : '') + '</div>';
        else h += '<div class="qk-r ' + (on ? 'on' : '') + '" data-a="qkCmd" data-k="' + it.c[1] + '"><span class="ico">' + I(it.c[0], 12) + '</span><span class="nm"><b>' + hlMatch(it.c[1], q) + '</b></span>' + (it.c[2] ? '<span class="kbd">' + it.c[2] + '</span>' : '') + '</div>';
        idx++;
      });
    });
    return h;
  }
  MPA.quickCount = () => quickItems().reduce((n, g) => n + g[1].length, 0);
  function quickDlg() {
    const drop = S.opt.quick === 'drop';
    return '<div class="dlg qk ' + (drop ? 'drop-mode' : '') + '"><div class="qk-i">' + I('search', 16) + '<input data-in="qk" placeholder="搜索请求、环境、命令…" value="' + esc(S.qk.q) + '" autocomplete="off"><span class="kbd">esc</span></div><div class="qk-l scroll" id="qkl">' + quickList() + '</div><div class="qk-f"><span><span class="kbd">↑↓</span>选择</span><span><span class="kbd">↵</span>打开</span><span><span class="kbd">⌘↵</span>新标签</span><span class="sp"></span><span>' + MPA.quickCount() + ' 个结果</span></div></div>';
  }
  MPA.renderQuickList = () => { const el = $('#qkl'); if (el) el.innerHTML = quickList(); };

  /* ---------- 小对话框 ---------- */
  function dlgSmall(o) {
    return '<div class="dlg dsm"><div class="dlg-h"><h3>' + o.title + '</h3><span class="sp"></span>' + ib('x', 'closeOv', { sz: 13 }) + '</div><div class="dlg-b">' + o.body + '</div><div class="dlg-f">' + (o.note || '') + o.foot + '</div></div>';
  }
  const dlgs = {
    save: () => dlgSmall({ title: '保存更改', body: '<div class="msg"><span class="ico warn">' + I('warn', 17) + '</span><div><b>请求“创建用户”有未保存的修改</b><span>关闭后这些修改将会丢失，是否先保存？</span></div></div><label class="chk" data-a="chk">' + cb(false) + '关闭 Tab 时默认保存并关闭，不再询问</label><label class="chk" data-a="chk">' + cb(false) + '关闭 Tab 时默认丢弃并关闭，不再询问</label>', foot: '<button class="btn" data-a="dlgDiscard">不保存</button><button class="btn" data-a="closeOv">取消</button><button class="btn pri" data-a="dlgSave">保存并关闭</button>' }),
    delete: () => dlgSmall({ title: '删除“订单服务”？', body: '<div class="msg"><span class="ico err">' + I('trash', 17) + '</span><div><b>文件夹内的 3 个请求也会被删除</b><span>删除后可立即撤销（<span class="kbd">⌘Z</span>）。</span></div></div>', foot: '<button class="btn" data-a="closeOv">取消</button><button class="btn pri dng" data-a="dlgDelete">删除</button>' }),
    prompt: () => dlgSmall({ title: '重命名请求', body: '<div class="fld-l"><label>名称</label><label class="inp focus" style="height:30px;border-color:var(--ac);box-shadow:0 0 0 3px var(--ac-soft)"><input value="获取用户列表" id="pin"></label></div>', foot: '<button class="btn" data-a="closeOv">取消</button><button class="btn pri" data-a="closeOv">确定</button>' }),
    restore: () => dlgSmall({ title: '恢复备份？', body: '<div class="msg"><span class="ico warn">' + I('warn', 17) + '</span><div><b>将用 2026-11-15 10:42 的备份覆盖当前数据</b><span>建议恢复前先立即备份一次，恢复后应用会重新加载。</span></div></div>', foot: '<button class="btn" data-a="closeOv">取消</button><button class="btn" data-a="closeOv">先备份</button><button class="btn pri dng" data-a="closeOv">恢复</button>' }),
    clear: () => dlgSmall({ title: '清空历史记录？', body: '<div class="msg"><span class="ico err">' + I('trash', 17) + '</span><div><b>此操作无法撤销</b><span>将删除当前项目的全部 8 条历史记录。</span></div></div>', foot: '<button class="btn" data-a="closeOv">取消</button><button class="btn pri dng" data-a="doClear">清空</button>' }),
  };

  MPA.renderOverlay = function () {
    const ov = S.overlay, host = $('#ov');
    if (!host) return;
    if (!ov) { host.innerHTML = ''; return; }
    let inner = '', cls = '';
    if (ov.type === 'settings') inner = settingsModal();
    else if (ov.type === 'import') inner = importDlg();
    else if (ov.type === 'cookies') inner = cookieDlg();
    else if (ov.type === 'quick') { inner = quickDlg(); cls = S.opt.quick === 'drop' ? 'clear' : 'top'; }
    else if (dlgs[ov.type]) inner = dlgs[ov.type]();
    host.innerHTML = '<div class="scrim ' + cls + ' ' + (S.ovAnim ? 'anim' : '') + '" data-a="scrimClick">' + inner + '</div>';
    S.ovAnim = false;
    const inp = host.querySelector('input[data-in=qk]'); if (inp && !S.static) { inp.focus(); inp.setSelectionRange(inp.value.length, inp.value.length); }
  };

  /* ---------- 气泡菜单 ---------- */
  const CTX = {
    req: [['打开', 'send', ''], ['重命名', 'pencil', '⌘E'], ['复制', 'copy', '⌘D'], ['复制为 cURL', 'code', '⌘⇧C'], ['导出 JSON…', 'upload', ''], ['移动到…', 'move', ''], '-', ['删除', 'trash', '⌫', 1]],
    folder: [['新建请求', 'filePlus', '⌘N'], ['新建文件夹', 'folderPlus', '⌘⇧N'], ['展开 / 折叠', 'chevR', ''], '-', ['重命名', 'pencil', '⌘E'], ['复制文件夹', 'copy', '⌘D'], ['导出 JSON…', 'upload', ''], '-', ['删除', 'trash', '⌫', 1]],
    tab: [['保存', 'save', '⌘S'], ['复制 cURL', 'code', ''], ['复制地址', 'link', ''], ['在侧栏中定位', 'target', ''], '-', ['关闭', 'x', '⌘W'], ['关闭其他', 'x', ''], ['关闭右侧', 'x', ''], ['关闭全部', 'x', '']],
    hist: [['打开', 'send', ''], ['复制地址', 'link', ''], '-', ['删除此记录', 'trash', '', 1]],
    env: [['打开', 'globe', ''], ['设为当前环境', 'check', ''], ['复制', 'copy', ''], '-', ['删除环境', 'trash', '', 1]],
  };
  function popHtml(p) {
    if (p.type === 'project') return '<div class="pop" style="width:260px"><div style="display:flex;gap:6px;padding:2px 2px 6px"><label class="inp grow">' + I('search', 12) + '<input placeholder="搜索或新建项目…"></label><button class="btn sm" style="height:26px">新建</button></div>' + M.projects.map((x) => '<div class="pi ' + (x.id === S.projId ? 'on' : '') + '" data-a="setProj" data-id="' + x.id + '"><span class="pav" style="background:' + x.color + ';width:16px;height:16px;font-size:9px">' + esc(x.name[0]) + '</span><span class="grow ell">' + esc(x.name) + '</span>' + (x.id === S.projId ? '<span class="cur-badge">当前</span>' : '<span class="mu" style="font-size:10.5px">' + x.updated + '</span>') + '</div>').join('') + '<div class="hsep"></div><div class="pi" data-a="home">' + I('grid', 13) + '全部项目…<span class="kbd">⌘⇧H</span></div></div>';
    if (p.type === 'env') return '<div class="pop r" style="width:230px"><div class="ph-s">切换后，请求里的 {{变量名}} 会立即换成对应值。</div><div class="pi ' + (!S.curEnv ? 'on' : '') + '" data-a="setEnv" data-id="">' + I('globe', 13) + '无环境' + (!S.curEnv ? '<span style="margin-left:auto;color:var(--ac)">' + I('check', 13) + '</span>' : '') + '</div>' + M.envs.map((e) => '<div class="pi ' + (S.curEnv === e.id ? 'on' : '') + '" data-a="setEnv" data-id="' + e.id + '">' + I('globe', 13) + '<span class="grow">' + esc(e.name) + '</span><span class="mu" style="font-size:10.5px">' + e.vars.length + ' 变量</span>' + (S.curEnv === e.id ? '<span style="color:var(--ac)">' + I('check', 13) + '</span>' : '') + '</div>').join('') + '<div class="hsep"></div><div class="pi" data-a="manageEnv">' + I('sliders', 13) + '管理环境…<span class="kbd">⌘⌥E</span></div></div>';
    if (p.type === 'ctx') return '<div class="pop ctx">' + CTX[p.kind].map((it) => it === '-' ? '<div class="hsep"></div>' : '<div class="pi ' + (it[3] ? 'dng' : '') + '" data-a="ctxDo" data-k="' + it[0] + '" data-kind="' + p.kind + '">' + I(it[1], 13) + '<span>' + it[0] + '</span>' + (it[2] ? '<span class="kbd">' + it[2] + '</span>' : '') + '</div>').join('') + '</div>';
    if (p.type === 'mth') return '<div class="pop" style="width:130px">' + ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'].map((m) => '<div class="pi" data-a="setMth" data-m="' + m + '"><span class="m m-' + m + '" style="font-size:11.5px">' + m + '</span></div>').join('') + '</div>';
    if (p.type === 'send') return '<div class="pop r" style="width:210px"><div class="pi" data-a="send">' + I('send', 13) + '发送<span class="kbd">⌘↵</span></div><div class="pi" data-a="toast" data-t="info" data-m="发送并保存|响应将保存到下载文件夹">' + I('download', 13) + '发送并保存响应…</div><div class="pi" data-a="copyCurl">' + I('code', 13) + '复制为 cURL<span class="kbd">⌘⇧C</span></div></div>';
    if (p.type === 'tabsAll') return '<div class="pop r" style="width:320px"><div class="ph-s">打开的标签 · ' + S.tabs.length + '</div><div class="pl scroll">' + S.tabs.map((t, i) => { const n = t.kind === 'env' ? { m: 'GET', name: '环境', parent: '' } : flat[t.id] || t.node; const fd = t.kind === 'env' ? '环境管理' : (flat[n.parent] || {}).name || '根目录'; return '<div class="pi ' + (i === S.active ? 'on' : '') + '" style="--i:' + i + '" data-a="selTab" data-i="' + i + '">' + (t.kind === 'env' ? '<span class="ico">' + I('globe', 12) + '</span>' : mbadge(n.m)) + '<span class="grow ell">' + esc(n.name) + '</span><em class="fd">' + esc(fd) + '</em>' + (t.dirty ? '<i class="pdot" style="background:var(--ac)"></i>' : '') + (i === S.active ? I('check', 13) : '') + '</div>'; }).join('') + '</div></div>';
    if (p.type === 'timing') {
      const rows = [['准备', 0, 3, 'var(--mu)', '3 ms'], ['Socket 初始化', 3, 5, 'var(--warn)', '2 ms'], ['DNS 查询', 8, 9, 'var(--post)', '9 ms'], ['TCP 握手', 17, 14, 'var(--put)', '14 ms'], ['SSL 握手', 31, 26, 'var(--patch)', '26 ms'], ['等待首字节 (TTFB)', 57, 58, 'var(--ac)', '58 ms'], ['下载', 115, 9, 'var(--ok)', '9 ms'], ['处理', 124, 4, 'var(--mu)', '4 ms']];
      return '<div class="timing"><h5>' + I('clock', 14) + '响应时间<b>128 ms</b></h5>' + rows.map((r, i) => '<div class="tr"><span>' + r[0] + '</span><span class="bar"><i style="left:' + (r[1] / 128 * 100) + '%;width:' + Math.max(r[2] / 128 * 100, 2) + '%;background:' + r[3] + ';animation-delay:' + i * 40 + 'ms"></i></span><span class="v">' + r[4] + '</span></div>').join('') + '<div class="tinfo"><span>HTTP 版本</span><b>HTTP/2</b><span>TLS 协议</span><b>TLSv1.3 · AES-256-GCM</b><span>远端地址</span><b>203.0.113.7:443</b><span>证书</span><b>*.acme.io · 有效期至 2027-03-02</b></div></div>';
    }
    return '';
  }
  MPA.renderPop = function () {
    const host = $('#pp');
    if (!host) return;
    const p = S.pop;
    if (!p) { host.innerHTML = ''; return; }
    host.innerHTML = '<div class="scrim clear" data-a="closePop" style="z-index:55;place-items:start">' + popHtml(p) + '</div>';
    const el = host.querySelector('.pop, .timing'), win = $('#win');
    if (!el) return;
    const wr = win.getBoundingClientRect(), k = wr.width / win.offsetWidth || 1;
    let x = p.x, y = p.y;
    if (p.anchor) {
      const a = win.querySelector(p.anchor);
      if (a && p.type === 'ctx') { const r = a.getBoundingClientRect(); x = (r.left - wr.left) / k + 70; y = (r.top - wr.top) / k + 12; }
      else if (a) { const r = a.getBoundingClientRect(); x = (p.align === 'right' ? r.right - wr.left : r.left - wr.left) / k; y = (r.bottom - wr.top) / k + 5; }
    }
    if (x == null) { x = 400; y = 100; }
    el.style.top = y + 'px';
    const w = el.offsetWidth;
    if (p.align === 'right') el.style.left = Math.max(8, x - w) + 'px'; else el.style.left = Math.min(x, win.offsetWidth - w - 8) + 'px';
    const h = el.offsetHeight; if (y + h > win.offsetHeight - 8) el.style.top = Math.max(8, win.offsetHeight - h - 8) + 'px';
  };

  /* ---------- Toast ---------- */
  const TICON = { ok: 'check', warn: 'warn', err: 'errorC', info: 'info', ac: 'download' };
  MPA.renderToasts = function () {
    const host = $('#ts');
    if (!host) return;
    host.innerHTML = S.toasts.map((t) => '<div class="toast ' + t.type + (t.out ? ' out' : '') + '" data-tid="' + t.id + '"><span class="ti ' + t.type + '">' + I(TICON[t.type] || 'info', 12) + '</span><div class="tx">' + (t.title ? '<b>' + esc(t.title) + '</b>' : '') + '<span>' + esc(t.msg) + '</span>' + (t.acts ? '<div class="ta2">' + t.acts.map((a, i) => '<button class="btn sm ' + (i === 0 ? 'pri' : '') + '" data-a="' + a[1] + '">' + a[0] + '</button>').join('') + '</div>' : '') + '</div><button class="ib sm" data-a="toastClose" data-tid="' + t.id + '">' + I('x', 12) + '</button><i class="ttl"></i></div>').join('');
  };
  MPA.toast = function (type, title, msg, acts, ttl) {
    const id = ++S.uid;
    S.toasts.unshift({ id, type, title, msg, acts });
    S.toasts = S.toasts.slice(0, 4);
    MPA.renderToasts();
    if (!S.static) setTimeout(() => A.toastClose(null, { tid: id }), ttl || 5000);
  };
  A.toastClose = (el, d) => { const id = +(d || el.dataset).tid; const t = S.toasts.find((x) => x.id === id); if (!t) return; t.out = true; MPA.renderToasts(); setTimeout(() => { S.toasts = S.toasts.filter((x) => x.id !== id); MPA.renderToasts(); }, 190); };
  const T = MPA.toast;

  /* ==========================================================================
     交互分发
     ========================================================================== */
  const full = () => MPA.renderAll();
  const openOv = (o) => { S.overlay = o; S.pop = null; S.ovAnim = true; MPA.renderOverlay(); MPA.renderPop(); };
  const closeOv = () => { S.overlay = null; MPA.renderOverlay(); };
  const openTab = (id) => {
    const i = S.tabs.findIndex((t) => t.kind === 'req' && t.id === id);
    if (i >= 0) S.active = i;
    else { S.tabs.push(Object.assign(MPA.mkTab(id, 'params', 'none'), { fresh: true })); S.active = S.tabs.length - 1; }
    S.sel = id; S.view = 'work'; full();
  };
  const openEnvTab = (id) => {
    const i = S.tabs.findIndex((t) => t.kind === 'env' && t.id === id);
    if (i >= 0) S.active = i; else { S.tabs.push({ kind: 'env', id, fresh: true }); S.active = S.tabs.length - 1; }
    S.envSel = id; S.view = 'work'; full();
  };
  let uidN = 0;
  Object.assign(A, {
    noop() {},
    toggleSide() { S.sideCollapsed = !S.sideCollapsed; const el = $('#win .side'); if (el) el.classList.toggle('closed', S.sideCollapsed); $$('#win [data-a=toggleSide]').forEach((b) => b.classList.toggle('on', !S.sideCollapsed)); },
    sideTab(el) { S.sideTab = el.dataset.t; S.view = 'work'; if (S.sideCollapsed) S.sideCollapsed = false; full(); },
    fold(el) { const id = el.dataset.id; S.open[id] = !S.open[id]; const f = $('#win .fld[data-fid=' + id + ']'); if (f) f.classList.toggle('open', S.open[id]); },
    histFold(el) { S.histOpen[el.dataset.day] = S.histOpen[el.dataset.day] === false; MPA.renderSide(); },
    openReq(el) { openTab(el.dataset.id); },
    openHist() { openTab('r1'); },
    openByName(el) { const n = Object.values(flat).find((x) => x.name === el.dataset.n); if (n) openTab(n.id); },
    openEnv(el) { openEnvTab(el.dataset.id); },
    selTab(el) { S.active = +el.dataset.i; const t = curTab(); if (t && t.kind === 'req') S.sel = t.id; S.view = 'work'; S.pop = null; full(); },
    closeTab(el, d, ev) {
      ev.stopPropagation(); const i = +el.dataset.i, t = S.tabs[i];
      if (t.dirty) { S.active = i; return openOv({ type: 'save' }); }
      S.tabs.splice(i, 1); if (S.active >= S.tabs.length) S.active = S.tabs.length - 1; full();
    },
    tabsAll(el) { S.pop = { type: 'tabsAll', anchor: '[data-a=tabsAll]', align: 'right' }; MPA.renderPop(); },
    newReq() {
      const id = 'n' + ++uidN; flat[id] = { t: 'r', id, m: 'GET', name: '新请求', url: '', parent: 'f1' };
      S.tabs.push(Object.assign(MPA.mkTab(id, 'params', 'none', true), { fresh: true })); S.active = S.tabs.length - 1; S.sel = id; S.view = 'work'; full();
      T('info', '', '已创建「新请求」，输入地址后按 ⌘↵ 发送');
    },
    newFolder() { openOv({ type: 'prompt' }); },
    newEnv() { T('info', '', '已创建「新环境」'); },
    rqView(el) { const t = curTab(); t.view = el.dataset.v; full(); },
    rtab(el) { const t = curTab(); t.rtab = el.dataset.v; full(); },
    merged(el) { const t = curTab(); t.merged = el.dataset.v === '1'; full(); },
    mthMenu() { S.pop = { type: 'mth', anchor: '.url .mth' }; MPA.renderPop(); },
    setMth(el) { const t = curTab(); const n = flat[t.id] || t.node; n.m = el.dataset.m; t.dirty = true; S.pop = null; full(); },
    sendMenu() { S.pop = { type: 'send', anchor: '.send', align: 'right' }; MPA.renderPop(); },
    send() {
      S.pop = null; const t = curTab(); if (!t || t.kind !== 'req') return;
      const oc = MPA.outcomeOf(t.id); t.resp = 'sending'; t.t0 = Date.now(); t.rtab = 'body'; full();
      clearInterval(S._tick); clearTimeout(S._st);
      S._tick = setInterval(() => { const el = $('#win .sending-t .tm'); if (el) el.textContent = ((Date.now() - t.t0) / 1000).toFixed(2) + 's'; }, 70);
      S._st = setTimeout(() => {
        clearInterval(S._tick); const n = flat[t.id] || t.node, path = '/' + (n.url || '').replace('{{host}}/', '');
        const now = new Date(), hhmm = now.toTimeString().slice(0, 5);
        if (oc === 'error') { t.resp = 'error'; S.last = { m: n.m, path, code: 0, ms: 31 }; S.hist[0].items.unshift([n.m, path, 0, hhmm]); S.conRows.unshift([now.toTimeString().slice(0, 8), n.m, n.url, 0, '—', '—']); T('err', '请求失败', 'ECONNREFUSED：无法连接到 127.0.0.1:8080'); }
        else if (oc === 'stream') { t.resp = 'stream'; t.rtab = 'stream'; S.last = { m: n.m, path, code: 200, ms: 3020 }; S.hist[0].items.unshift([n.m, path, 200, hhmm]); S.conRows.unshift([now.toTimeString().slice(0, 8), n.m, n.url, 200, '3.02 s', '412 B']); }
        else { t.resp = 'ok'; t.code = oc; const r = MPA.RESP[oc]; S.last = { m: n.m, path, code: oc, ms: r.ms }; S.hist[0].items.unshift([n.m, path, oc, hhmm]); S.conRows.unshift([now.toTimeString().slice(0, 8), n.m, n.url, oc, r.ms + ' ms', r.size]); if (oc >= 400) T('warn', '请求已返回 ' + oc, '服务器返回了错误状态，请检查参数。'); }
        full();
      }, oc === 'stream' ? 2400 : 1500);
    },
    cancel() { clearInterval(S._tick); clearTimeout(S._st); const t = curTab(); t.resp = 'none'; full(); T('info', '', '已取消请求'); },
    save() { const t = curTab(); if (t) { t.dirty = false; full(); T('ok', '', '请求已保存'); } },
    bodyType(el) { const t = curTab(); const rq = MPA.reqOf(t.id); const ty = el.dataset.v; const body = ty === 'form' || ty === 'urlenc' ? { type: ty, rows: rq.body.rows || [['', '', '', 1]] } : ty === 'raw' || ty === 'json' ? { type: ty, text: rq.body.text || (ty === 'json' ? '{\n  "name": "Alice"\n}' : '') } : { type: 'none' }; window.MP.reqs[t.id] = Object.assign({}, rq, { body }); full(); },
    beautify() { T('ok', '', 'JSON 已格式化'); },
    copyCurl() { S.pop = null; MPA.renderPop(); T('ok', '', '已复制 cURL 命令'); },
    copyResp() { T('ok', '', '已复制响应体'); },
    copyVal() { T('ok', '', '已复制'); },
    kvToggle(el, d, ev) { ev.stopPropagation(); el.classList.toggle('on'); const r = el.closest('.kr, .ck-t'); if (r && r.classList.contains('kr')) r.classList.toggle('off', !el.classList.contains('on')); },
    authEye() { S.authShow = !S.authShow; full(); },
    dock() {},
    toggleDock() { S.opt.dock = S.opt.dock === 'bottom' ? 'right' : 'bottom'; full(); },
    toggleCon() { S.con.open = !S.con.open; full(); },
    conFilter(el) { S.con.filter = el.dataset.v; full(); },
    conEx(el) { S.con.ex = S.con.ex === +el.dataset.i ? -1 : +el.dataset.i; full(); },
    conClear() { S.conRows = []; full(); },
    home() { S.view = S.view === 'home' ? 'work' : 'home'; S.pop = null; full(); },
    homeSel(el) { S.homeSel = el.dataset.id; full(); },
    homeMode(el) { S.opt.home = el.dataset.v; full(); },
    openProj(el) { S.projId = el.dataset.id; S.view = 'work'; T('ok', '', '已切换到「' + proj().name + '」'); full(); },
    newProj() { T('ok', '', '已创建新项目'); },
    pcolor(el) { const p = M.projects.find((x) => x.id === S.homeSel); p.color = el.dataset.c; full(); },
    setProj(el) { S.projId = el.dataset.id; S.pop = null; full(); T('ok', '', '已切换到「' + proj().name + '」'); },
    projMenu() { S.pop = { type: 'project', anchor: S.dir === 'b' ? '#win .pbtn' : '[data-a=projMenu]' }; MPA.renderPop(); },
    envMenu() { S.pop = { type: 'env', anchor: '[data-a=envMenu]', align: 'right' }; MPA.renderPop(); },
    setEnv(el) { S.curEnv = el.dataset.id; S.pop = null; full(); },
    setCurEnv(el) { S.curEnv = el.dataset.id; full(); T('ok', '', '已切换环境'); },
    manageEnv() { S.pop = null; S.sideTab = 'env'; openEnvTab(S.curEnv || 'e1'); },
    delEnv() { openOv({ type: 'delete' }); },
    toastSave() { T('ok', '', '环境已保存'); },
    timing() { S.pop = { type: 'timing', anchor: '.resp-h .meta .mi', align: 'right' }; MPA.renderPop(); },
    closePop(el, d, ev) { if (ev.target === el) { S.pop = null; MPA.renderPop(); } },
    scrimClick(el, d, ev) { if (ev.target === el && !(S.overlay && S.overlay.type === 'settings' && false)) closeOv(); },
    closeOv() { closeOv(); },
    theme() { S.mode = S.mode === 'dark' ? 'light' : 'dark'; S.themeSrc = null; MPA.renderAll(); MPA.syncBar && MPA.syncBar(); },
    settings() { if (S.opt.settings === 'page') { S.view = 'settings'; S.pop = null; full(); } else openOv({ type: 'settings' }); },
    closeSettings() { S.view = 'work'; full(); },
    setPage(el) { S.setPage = el.dataset.p; if (S.view === 'settings') full(); else MPA.renderOverlay(); },
    sw(el) { const k = el.dataset.k; S.set[k] = !S.set[k]; el.classList.toggle('on', S.set[k]); },
    modeSet(el) { const v = el.dataset.v; S.themeSrc = v; S.mode = v === 'system' ? 'light' : v; full(); MPA.syncBar && MPA.syncBar(); },
    schemeSet(el) { S.scheme = el.dataset.v; full(); MPA.syncBar && MPA.syncBar(); },
    lang(el) { S.set.lang = el.dataset.v; full(); },
    fs(el) { S.set.fs = +el.dataset.v; full(); },
    checkUpd() { S.upd = 'checking'; MPA.renderAll(); setTimeout(() => { S.upd = 'downloading'; full(); setTimeout(() => { S.upd = 'ready'; S.update = 'ready'; full(); T('ac', '新版本 2.1.0 已就绪', '重启后生效，当前打开的标签会自动恢复。', [['重启更新', 'restart'], ['稍后', 'toastDismissAll']], 9000); }, 2200); }, 700); },
    restart() { T('info', '', '（设计稿）此处会重启应用'); },
    updateInfo() { S.setPage = 'about'; S.upd = 'ready'; A.settings(); },
    toastDismissAll() { S.toasts = []; MPA.renderToasts(); },
    toast(el) { const [title, msg] = el.dataset.m.split('|'); T(el.dataset.t, title, msg || ''); },
    restoreDlg() { openOv({ type: 'restore' }); },
    importOv(el) { S.imp = { tab: el.dataset.tab || 'file', ok: false, strat: 'merge', over: false }; openOv({ type: 'import' }); },
    curlOv() { S.imp = { tab: 'curl', ok: false, strat: 'merge', over: false }; openOv({ type: 'import' }); },
    impTab(el) { S.imp.tab = el.dataset.v; MPA.renderOverlay(); },
    impPick() { S.imp.ok = true; S.imp.over = false; MPA.renderOverlay(); },
    impStrat(el) { S.imp.strat = el.dataset.v; MPA.renderOverlay(); },
    doImport() { closeOv(); T('ok', '导入完成', '新增 22 个请求，合并 2 个同名请求'); },
    cookies() { openOv({ type: 'cookies' }); },
    ckDom(el) { S.cookieDom = el.dataset.d; MPA.renderOverlay(); },
    quick() { S.qk = { q: '', sel: 0 }; openOv({ type: 'quick' }); },
    qkPick(el) { closeOv(); openTab(el.dataset.id); },
    qkEnv(el) { S.curEnv = el.dataset.id; closeOv(); full(); T('ok', '', '已切换环境'); },
    qkCmd(el) { const k = el.dataset.k; closeOv(); if (k === '新建请求') A.newReq(); else if (k === '导入 cURL') A.curlOv(); else if (k.includes('侧栏')) A.toggleSide(); else if (k.includes('控制台')) A.toggleCon(); else if (k.includes('主题')) A.theme(); else A.settings(); },
    ctxBtn(el, d, ev) { ev.stopPropagation(); const r = el.getBoundingClientRect(); A._ctx(el.dataset.kind, r.left, r.bottom); },
    ctxDo(el) { const k = el.dataset.k; S.pop = null; MPA.renderPop(); if (k === '删除' || k === '删除环境') openOv({ type: 'delete' }); else if (k === '重命名') openOv({ type: 'prompt' }); else if (k.startsWith('复制')) T('ok', '', '已复制'); else if (k === '打开') A.openReq({ dataset: { id: S.sel } }); else T('info', '', k); },
    dlgSave() { const t = curTab(); t.dirty = false; closeOv(); S.tabs.splice(S.active, 1); S.active = Math.max(0, S.active - 1); full(); T('ok', '', '已保存并关闭'); },
    dlgDiscard() { S.tabs.splice(S.active, 1); S.active = Math.max(0, S.active - 1); closeOv(); full(); },
    dlgDelete() { closeOv(); T('info', '已删除“订单服务”', '3 个请求已移入回收，可立即撤销。', [['撤销 ⌘Z', 'toastDismissAll']]); },
    clearHist() { openOv({ type: 'clear' }); },
    doClear() { S.hist = [{ day: '今天', items: [] }]; closeOv(); full(); },
    chk(el) { const c = el.querySelector('.cb'); c.classList.toggle('on'); },
  });
  A._ctx = function (kind, cx, cy) {
    const win = $('#win'), wr = win.getBoundingClientRect(), k = wr.width / win.offsetWidth || 1;
    S.pop = { type: 'ctx', kind, x: (cx - wr.left) / k, y: (cy - wr.top) / k };
    MPA.renderPop();
  };

  /* ---------- 事件委托 ---------- */
  MPA.bind = function () {
    const win = $('#win');
    win.addEventListener('click', (ev) => {
      const el = ev.target.closest('[data-a]');
      if (!el || !win.contains(el)) return;
      const f = A[el.dataset.a];
      if (f) f(el, el.dataset, ev);
    });
    win.addEventListener('contextmenu', (ev) => {
      const el = ev.target.closest('[data-ctx]');
      if (!el) return;
      ev.preventDefault();
      const [kind, id] = el.dataset.ctx.split(':');
      if (kind === 'req') S.sel = id;
      A._ctx(kind, ev.clientX, ev.clientY);
    });
    win.addEventListener('input', (ev) => {
      const k = ev.target.dataset && ev.target.dataset.in;
      if (!k) return;
      const v = ev.target.value;
      if (k === 'q') { S.q = v; MPA.renderTree(); }
      else if (k === 'hq') { S.hq = v; const pos = ev.target.selectionStart; MPA.renderSide(); const i = $('#win input[data-in=hq]'); if (i) { i.focus(); i.setSelectionRange(pos, pos); } }
      else if (k === 'qk') { S.qk.q = v; S.qk.sel = 0; MPA.renderQuickList(); const c = $('#win .qk-f .sp + span'); if (c) c.textContent = MPA.quickCount() + ' 个结果'; }
      else if (k === 'homeQ') { S.homeQ = v; const pos = ev.target.selectionStart; full(); const i = $('#win input[data-in=homeQ]'); if (i) { i.focus(); i.setSelectionRange(pos, pos); } }
    });
    // 拖入文件的演示：导入对话框内悬停
    win.addEventListener('dragover', (ev) => { if (S.overlay && S.overlay.type === 'import') { ev.preventDefault(); if (!S.imp.over) { S.imp.over = true; MPA.renderOverlay(); } } });
    // 气泡提示
    let tipTimer, tipEl;
    const hideTip = () => { clearTimeout(tipTimer); if (tipEl) { tipEl.remove(); tipEl = null; } };
    win.addEventListener('mouseover', (ev) => {
      const el = ev.target.closest('[data-tip]');
      if (!el || !el.dataset.tip || S.static) return hideTip();
      clearTimeout(tipTimer);
      tipTimer = setTimeout(() => {
        hideTip(); const r = el.getBoundingClientRect(), wr = win.getBoundingClientRect(), k = wr.width / win.offsetWidth || 1;
        tipEl = document.createElement('div'); tipEl.className = 'tip'; tipEl.textContent = el.dataset.tip; win.appendChild(tipEl);
        let x = (r.left + r.width / 2 - wr.left) / k - tipEl.offsetWidth / 2; x = Math.max(6, Math.min(x, win.offsetWidth - tipEl.offsetWidth - 6));
        tipEl.style.left = x + 'px'; tipEl.style.top = (r.bottom - wr.top) / k + 6 + 'px';
      }, 450);
    });
    win.addEventListener('mouseout', (ev) => { if (!ev.relatedTarget || !ev.relatedTarget.closest || !ev.relatedTarget.closest('[data-tip]')) hideTip(); });
    win.addEventListener('mousedown', hideTip);
    // 键盘
    document.addEventListener('keydown', (ev) => {
      const meta = ev.metaKey || ev.ctrlKey, k = ev.key;
      if (S.overlay && S.overlay.type === 'quick') {
        const n = MPA.quickCount();
        if (k === 'ArrowDown') { ev.preventDefault(); S.qk.sel = (S.qk.sel + 1) % Math.max(n, 1); MPA.renderQuickList(); return; }
        if (k === 'ArrowUp') { ev.preventDefault(); S.qk.sel = (S.qk.sel - 1 + n) % Math.max(n, 1); MPA.renderQuickList(); return; }
        if (k === 'Enter') { ev.preventDefault(); const el = $('#qkl .qk-r.on'); if (el) el.click(); return; }
      }
      if (k === 'Escape') { if (S.pop) { S.pop = null; MPA.renderPop(); } else if (S.overlay) closeOv(); else if (S.view === 'settings') A.closeSettings(); return; }
      if (!meta) return;
      if (k === 'k') { ev.preventDefault(); A.quick(); }
      else if (k === ',') { ev.preventDefault(); A.settings(); }
      else if (k === 'Enter') { ev.preventDefault(); A.send(); }
      else if (k === 'b') { ev.preventDefault(); A.toggleSide(); }
      else if (k === 'j') { ev.preventDefault(); A.toggleCon(); }
      else if (k === 's') { ev.preventDefault(); A.save(); }
      else if (k === 'n') { ev.preventDefault(); A.newReq(); }
      else if (k === 'i') { ev.preventDefault(); A.curlOv(); }
      else if (k === 'w') { ev.preventDefault(); const t = curTab(); if (t) A.closeTab({ dataset: { i: S.active } }, null, { stopPropagation() {} }); }
    });
  };
})();
