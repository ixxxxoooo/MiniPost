/* ==========================================================================
   MiniPost 设计稿 · 动效引擎（v2）
   为什么需要它：原型每次交互都会整体重绘 innerHTML，纯 CSS 做不到「元素之间的连续运动」，
   也会让入场动画在每次点击时反复重播。这里用很薄的一层脚本补齐：
     1. 滑动指示器  分段控件的滑块、请求/响应子标签的下划线、标签栏的顶线、设置导航的竖条，
                    在重绘前后用 FLIP 方式从旧位置滑到新位置（Web Animations 作用于伪元素）。
     2. 入场门控    只有「视图真正变化」（切标签、切页、发送完成……）才允许入场动画播放，
                    普通点击引起的重绘不再重播（通过 data-enter / data-con / data-ex 控制）。
     3. 交错序号    给列表项写入 --i，由 CSS 计算延迟，实现逐行淡入。
     4. 光斑跟随    卡片悬停时，高光跟随指针（--mx / --my）。
     5. 主题揭幕    切换明暗/配色时，用 View Transition 做圆形扩散；不支持时直接切换。
     6. 滚动揭示    文档页（总览、设计系统、Logo）的区块随滚动依次浮现。
   全部在「减少动态效果」下自动关闭。没有任何业务逻辑，删掉本文件界面依然完整可用。
   ========================================================================== */
(function () {
  'use strict';
  const reduce = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  const MPA = window.MPA;
  const $ = (s, r) => (r || document).querySelector(s);
  const $$ = (s, r) => Array.from((r || document).querySelectorAll(s));

  /* ---------- 主题揭幕（原型与文档页共用） ---------- */
  let ptr = { x: window.innerWidth / 2, y: 0 };
  window.addEventListener('pointerdown', (e) => { ptr = { x: e.clientX, y: e.clientY }; }, true);
  function vt(change, origin) {
    if (reduce || !document.startViewTransition) { change(); return; }
    const p = origin || ptr;
    let t;
    try { t = document.startViewTransition(() => { change(); }); } catch (e) { change(); return; }
    t.ready.then(() => {
      const r = Math.hypot(Math.max(p.x, window.innerWidth - p.x), Math.max(p.y, window.innerHeight - p.y));
      document.documentElement.animate(
        { clipPath: ['circle(0px at ' + p.x + 'px ' + p.y + 'px)', 'circle(' + r + 'px at ' + p.x + 'px ' + p.y + 'px)'] },
        { duration: 640, easing: 'cubic-bezier(.65,0,.35,1)', pseudoElement: '::view-transition-new(root)' }
      );
    }).catch(() => {});
  }
  window.MPFX = { vt, reduce };

  /* ---------- 原型：重绘包装 ---------- */
  function initApp() {
    if (!MPA || !MPA.S || reduce) return;
    const S = MPA.S;

    // 1) 滑动指示器配置：box 为容器，on 为当前项，ps 为作为指示器的伪元素，w 表示是否同时动画宽度
    const FLIP = [
      { box: '.seg', all: ':scope > b', on: ':scope > b.on', ps: '::before', w: true },
      { box: '.rq-tabs, .resp-h', all: ':scope > .ut', on: ':scope > .ut.on', ps: '::after', w: true },
      { box: '.tabs-sc', all: ':scope > .tab', on: ':scope > .tab.on', ps: '::before', w: true },
      { box: '.set-n', all: ':scope > .ni', on: ':scope > .ni.on', ps: '::before', w: false },
    ];
    const scaleOf = (win) => (win.getBoundingClientRect().width / win.offsetWidth) || 1;
    function eachBox(win, fn) {
      FLIP.forEach((cfg, ci) => $$(cfg.box, win).forEach((box, bi) => {
        const all = $$(cfg.all, box), on = box.querySelector(cfg.on);
        fn(cfg, ci + ':' + bi + ':' + box.className, box, all, on);
      }));
    }
    function capture(win) {
      const snap = {};
      eachBox(win, (cfg, key, box, all, on) => {
        if (!on) return;
        const br = box.getBoundingClientRect(), r = on.getBoundingClientRect();
        snap[key] = { n: all.length, idx: all.indexOf(on), left: r.left - br.left, top: r.top - br.top, width: r.width };
      });
      return snap;
    }
    function play(win, snap) {
      const k = scaleOf(win);
      eachBox(win, (cfg, key, box, all, on) => {
        const old = snap[key];
        if (!old || !on || old.n !== all.length) return;
        const idx = all.indexOf(on);
        if (idx === old.idx) return;
        const br = box.getBoundingClientRect(), r = on.getBoundingClientRect();
        const dx = (old.left - (r.left - br.left)) / k, dy = (old.top - (r.top - br.top)) / k;
        let from = { transform: 'translate(' + dx + 'px,' + dy + 'px)' }, to = { transform: 'translate(0,0)' };
        if (cfg.w) {
          const cs = getComputedStyle(on, cfg.ps), inset = (parseFloat(cs.left) || 0) + (parseFloat(cs.right) || 0);
          from.width = Math.max(old.width / k - inset, 0) + 'px'; to.width = Math.max(r.width / k - inset, 0) + 'px';
        }
        try { on.animate([from, to], { duration: 420, easing: 'cubic-bezier(.34,1.28,.5,1)', pseudoElement: cfg.ps }); } catch (e) { /* 旧浏览器不支持伪元素动画，忽略 */ }
      });
    }

    // 2) 入场门控：把「视图签名」变化映射到 data-enter（以及控制台的 data-con / data-ex）
    const last = {};
    function flag(el, name, sig) { const changed = last[name] !== sig; last[name] = sig; el.dataset[name] = changed ? '1' : '0'; }
    const sigView = () => {
      const t = MPA.curTab && MPA.curTab();
      return [S.view, S.sideTab, S.dir, S.screen, S.active, S.tabs.length, t && t.id, t && t.view, t && t.rtab, t && t.resp, t && t.merged, S.setPage, S.opt && S.opt.home, S.sideCollapsed, JSON.stringify(S.histOpen), S.hist && S.hist.length].join('|');
    };

    // 3) 交错序号
    const GROUPS = [['.code', '.cl'], ['.tree', '.row'], ['.acts-l', '.a'], ['.recent', '.rr'], ['.pop', '.pi'], ['.set-cb', ':scope > *']];
    function stagger(root) {
      GROUPS.forEach((g) => $$(g[0], root).forEach((box) => $$(g[1], box).forEach((el, i) => el.style.setProperty('--i', i))));
    }

    // 4) 退场幽灵：视图切换（响应体 → 响应头、Params → Body、设置分页……）时，旧面板先淡出上移，新面板稍后再入场。
    //    重绘是整块 innerHTML 替换，旧节点已不存在，所以重绘前克隆旧面板，浮在原位置播放 paneOut 后移除；新面板入场延后 --dur-fast。
    const PANES = ['.rb', '.rq-body', '.set-cb', '.home', '.envp', '.welcome'];
    function grabPanes(win) {
      const k = scaleOf(win), out = {};
      PANES.forEach((sel) => {
        const els = $$(sel, $('#app', win) || win);
        if (els.length !== 1) return;
        const el = els[0], r = el.getBoundingClientRect(), pr = (el.offsetParent || win).getBoundingClientRect();
        out[sel] = { node: el.cloneNode(true), left: (r.left - pr.left) / k, top: (r.top - pr.top) / k, w: r.width / k, h: r.height / k };
      });
      return out;
    }
    function ghostPanes(win, old) {
      Object.keys(old).forEach((sel) => {
        const els = $$(sel, $('#app', win) || win);
        if (els.length !== 1) return;
        const el = els[0], g = old[sel], host = el.parentElement;
        if (!host) return;
        if (getComputedStyle(host).position === 'static') host.style.position = 'relative';
        const node = g.node;
        node.removeAttribute('id'); node.classList.add('pane-ghost');
        Object.assign(node.style, { position: 'absolute', left: g.left + 'px', top: g.top + 'px', width: g.w + 'px', height: g.h + 'px', margin: '0', pointerEvents: 'none', zIndex: 6, animation: 'paneOut var(--dur-fast) var(--ease-out) forwards' });
        host.appendChild(node);
        el.style.animationDelay = '110ms'; el.style.animationFillMode = 'both';
        setTimeout(() => { node.remove(); el.style.animationDelay = ''; el.style.animationFillMode = ''; }, 420);
      });
    }

    const wrap = (name, make) => { const orig = MPA[name]; if (typeof orig === 'function') MPA[name] = make(orig); };

    wrap('renderApp', (orig) => function () {
      const win = $('#win');
      if (!win) return orig.apply(this, arguments);
      const snap = capture(win), first = !('boot' in last), panes = first ? null : grabPanes(win);
      const r = orig.apply(this, arguments);
      flag(win, 'enter', sigView());
      if (panes && win.dataset.enter === '1' && !S.static) ghostPanes(win, panes);
      flag(win, 'con', String(S.con && S.con.open));
      flag(win, 'ex', S.con && S.con.open ? S.con.ex + '|' + S.con.filter : 'x');
      if (first) { last.boot = 1; win.dataset.boot = ''; const off = () => { delete win.dataset.boot; }; setTimeout(off, 900); window.addEventListener('pointerdown', off, { once: true }); }
      stagger($('#app', win) || win);
      play(win, snap);
      return r;
    });
    wrap('renderOverlay', (orig) => function () {
      const win = $('#win'), host = $('#ov');
      if (!win || !host) return orig.apply(this, arguments);
      const snap = capture(win);
      const r = orig.apply(this, arguments);
      flag(host, 'enter', [S.overlay && S.overlay.type, S.setPage, S.imp && S.imp.tab, S.imp && S.imp.ok, S.cookieDom].join('|'));
      stagger(host); play(win, snap);
      return r;
    });
    wrap('renderPop', (orig) => function () { const r = orig.apply(this, arguments); const h = $('#pp'); if (h) stagger(h); return r; });
    // 侧栏 / 树的局部重绘（搜索输入、折叠历史）：不播放入场
    ['renderSide', 'renderTree'].forEach((n) => wrap(n, (orig) => function () {
      const win = $('#win'); if (win) { win.dataset.enter = '0'; }
      const snap = win ? capture(win) : null, r = orig.apply(this, arguments);
      if (win) { stagger($('#app', win) || win); play(win, snap); }
      return r;
    }));

    // 主题 / 配色变化时，用圆形揭幕完成整窗重绘
    let theme = null;
    wrap('renderAll', (orig) => function () {
      const key = S.scheme + '|' + S.mode, changed = theme !== null && key !== theme;
      theme = key;
      if (!changed || S.static) return orig.apply(this, arguments);
      const self = this, args = arguments;
      vt(() => orig.apply(self, args));
    });

  }

  /* ---------- 光斑跟随（原型与设计系统页共用） ---------- */
  function initSpot() {
    if (reduce) return;
    document.addEventListener('pointermove', (e) => {
      const t = e.target.closest && e.target.closest('.pcard, .thumb, .dv, .spot');
      if (!t) return;
      const r = t.getBoundingClientRect(), k = r.width / (t.offsetWidth || r.width) || 1;
      t.style.setProperty('--mx', (e.clientX - r.left) / k + 'px');
      t.style.setProperty('--my', (e.clientY - r.top) / k + 'px');
    }, { passive: true });
  }

  /* ---------- 文档页：滚动揭示 ---------- */
  function initDoc() {
    if ($('#win') || reduce || !('IntersectionObserver' in window)) return;
    const sel = 'main > h2, main > h3, main > .sub, main > .card, main > .hero, main > .rec-box, main > table.t, main > section, .dir, .cover > div, .howto > div, .sch a, .grid2 > div, .grid3 > .card, .mo .c, .rules > div, .sw1';
    const els = $$(sel);
    if (!els.length) return;
    document.documentElement.classList.add('fx-reveal');
    els.forEach((e) => e.classList.add('rv'));
    // 揭示完成后立刻清理类名与内联延迟，避免这些过渡拖慢后续的悬停反馈
    const settle = (el) => { el.classList.remove('rv', 'in'); el.style.transitionDelay = ''; };
    const io = new IntersectionObserver((entries) => {
      let i = 0;
      entries.forEach((en) => {
        if (!en.isIntersecting) return;
        const el = en.target, delay = Math.min(i++, 8) * 50;
        el.style.transitionDelay = delay + 'ms';
        el.classList.add('in');
        let done = false;
        const fin = () => { if (!done) { done = true; settle(el); } };
        el.addEventListener('transitionend', (ev) => { if (ev.target === el && ev.propertyName === 'opacity') fin(); });
        setTimeout(fin, delay + 1000);
        io.unobserve(el);
      });
    }, { threshold: 0.06, rootMargin: '0px 0px -3% 0px' });
    els.forEach((e) => io.observe(e));
  }

  initApp();
  initSpot();
  initDoc();
})();
