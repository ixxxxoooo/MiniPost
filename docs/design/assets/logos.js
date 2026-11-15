/* MiniPost Logo 系统 —— 全部为 64×64 画布的纯 SVG，颜色全部走主题变量：
 *   --lg-a / --lg-b  图标渐变两端（由 --ac 派生）
 *   --lg-fg          渐变底上的前景色
 *   --ac             强调色（glyph 着色用）
 * API：MPLogo.list / glyph / mini / tile / lockup
 */
(function () {
  let uid = 0;

  // ---------- 图形本体：只负责几何，颜色由 paint 传入 ----------
  // w = 描边粗细倍率（mini 时加粗，保证 16px 清晰）
  const SHAPES = {
    // 折纸：纸飞机，两个折面（亮面 + 暗面），圆角相接
    fold(c, w) {
      const sw = 3 * w;
      return '<g class="lg-move" stroke-linejoin="round" stroke-width="' + sw + '" stroke="' + c + '" fill="' + c + '">' +
        '<path d="M9 30 L55 9 L27 36 Z"/>' +
        '<path d="M27 36 L55 9 L36 55 Z" opacity=".62"/>' +
        '</g>';
    },
    // 字母 M：右腿变成向上的发送箭头
    mono(c, w) {
      return '<g fill="none" stroke="' + c + '" stroke-width="' + 6 * w + '" stroke-linecap="round" stroke-linejoin="round">' +
        '<path d="M13 50 V17 L31 37 L50 15"/><path class="lg-move" d="M38 14 H51 V27"/></g>';
    },
    // 信封：封口掀开，信件向上寄出
    post(c, w) {
      return '<g fill="none" stroke="' + c + '" stroke-width="' + 4.5 * w + '" stroke-linecap="round" stroke-linejoin="round">' +
        '<path d="M10 29 V47 a7 7 0 0 0 7 7 H47 a7 7 0 0 0 7 -7 V29"/><path d="M10 29 L32 12 L54 29"/>' +
        '<g class="lg-move"><path d="M32 45 V27"/><path d="M24.5 34 L32 26.5 L39.5 34"/></g></g>';
    },
    // 邮票：点状齿孔边框 + 缩小的纸飞机
    stamp(c, w) {
      const frame = w > 1.2 ? '' :
        '<rect x="11" y="11" width="42" height="42" rx="5" pathLength="28" stroke-dasharray="0 1" stroke-width="3.4" stroke-linecap="round" fill="none" stroke="' + c + '"/>';
      return frame + '<g transform="translate(32 32) scale(' + (w > 1.2 ? .82 : .5) + ') translate(-32 -32)">' + SHAPES.fold(c, w > 1.2 ? 1.4 : 1.7) + '</g>';
    },
  };

  const LIST = [
    { id: 'fold', name: '折纸 Fold', recommended: true, desc: '纸飞机的两个折面，亮面承光、暗面落影；一眼读出「发送」，16px 依然成形。' },
    { id: 'mono', name: '字母 M', desc: '用 M 的右腿长出发送箭头，字母识别度最高，适合水印与极简场景。' },
    { id: 'post', name: '信封 Post', desc: '掀开封口、信件向上寄出，对应 MiniPost 里的 Post，线性风格最贴近系统图标。' },
    { id: 'stamp', name: '邮票 Stamp', desc: '点状齿孔邮票框里放一架小纸飞机，更有品牌故事，Dock 里辨识度强。' },
  ];

  function paintOf(color) {
    if (!color || color === 'accent') return 'var(--ac)';
    if (color === 'fg') return 'currentColor';
    return color;
  }

  function svg(size, inner, cls, extra) {
    return '<svg class="lg ' + (cls || '') + '" width="' + size + '" height="' + size + '" viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" ' + (extra || '') + '>' + inner + '</svg>';
  }

  // 纯图形（无底）
  function glyph(id, o) {
    o = o || {};
    return svg(o.size || 24, (SHAPES[id] || SHAPES.fold)(paintOf(o.color), o.w || 1), 'lg-glyph');
  }
  // 小尺寸：自动加粗
  function mini(id, o) {
    o = o || {};
    return svg(o.size || 16, (SHAPES[id] || SHAPES.fold)(paintOf(o.color), 1.45), 'lg-glyph lg-mini');
  }

  // 应用图标：squircle 底 + 图形，variant 控制底的风格
  function tile(id, o) {
    o = o || {};
    const k = ++uid, size = o.size || 64, v = o.variant || 'color';
    const shape = SHAPES[id] || SHAPES.fold;
    let bg = '', fg = 'var(--lg-fg)', fx = '';
    const R = 14.3;
    if (v === 'color') {
      bg = '<defs><linearGradient id="lgA' + k + '" x1="0" y1="0" x2="0.35" y2="1"><stop offset="0" style="stop-color:var(--lg-a)"/><stop offset="1" style="stop-color:var(--lg-b)"/></linearGradient>' +
        '<linearGradient id="lgG' + k + '" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#fff" stop-opacity=".28"/><stop offset=".55" stop-color="#fff" stop-opacity="0"/></linearGradient></defs>' +
        '<rect width="64" height="64" rx="' + R + '" fill="url(#lgA' + k + ')"/><rect width="64" height="64" rx="' + R + '" fill="url(#lgG' + k + ')"/>' +
        '<rect x=".5" y=".5" width="63" height="63" rx="' + (R - .5) + '" stroke="rgba(255,255,255,.28)"/>';
      fx = 'filter:drop-shadow(0 1.2px 1.4px rgba(0,0,0,.22))';
    } else if (v === 'dark') {
      bg = '<defs><linearGradient id="lgA' + k + '" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#2b2d38"/><stop offset="1" stop-color="#13141b"/></linearGradient></defs>' +
        '<rect width="64" height="64" rx="' + R + '" fill="url(#lgA' + k + ')"/><rect x=".5" y=".5" width="63" height="63" rx="' + (R - .5) + '" stroke="rgba(255,255,255,.12)"/>';
      fg = 'var(--ac)';
      fx = 'filter:drop-shadow(0 0 3px color-mix(in oklab,var(--ac) 60%,transparent))';
    } else if (v === 'tint') {
      bg = '<defs><linearGradient id="lgA' + k + '" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#46464e"/><stop offset="1" stop-color="#2a2a30"/></linearGradient></defs>' +
        '<rect width="64" height="64" rx="' + R + '" fill="url(#lgA' + k + ')"/><rect x=".5" y=".5" width="63" height="63" rx="' + (R - .5) + '" stroke="rgba(255,255,255,.14)"/>';
      fg = '#ececf1';
    } else if (v === 'mono') {
      bg = '<rect width="64" height="64" rx="' + R + '" fill="#fff"/><rect x=".5" y=".5" width="63" height="63" rx="' + (R - .5) + '" stroke="#dcdce3"/>';
      fg = '#16161c';
    } else if (v === 'clear') {
      bg = '<rect width="64" height="64" rx="' + R + '" fill="rgba(255,255,255,.14)"/><rect x=".5" y=".5" width="63" height="63" rx="' + (R - .5) + '" stroke="rgba(255,255,255,.5)"/>' +
        '<path d="M5 22 Q32 6 59 22" stroke="rgba(255,255,255,.35)" stroke-width="1.2" fill="none"/>';
      fg = 'rgba(255,255,255,.96)';
    }
    return svg(size, bg + '<g style="' + fx + '">' + shape(fg, 1) + '</g>', 'lg-tile');
  }

  // 横向组合：图标 + 字标
  function lockup(id, o) {
    o = o || {};
    const s = o.size || 28, gap = o.gap || 8;
    return '<span class="lg-lockup" style="display:inline-flex;align-items:center;gap:' + gap + 'px;line-height:1">' +
      tile(id, { size: s, variant: o.variant || 'color' }) +
      '<span style="font:600 ' + Math.round(s * .62) + 'px/1 -apple-system,BlinkMacSystemFont,\'SF Pro Display\',\'PingFang SC\',sans-serif;letter-spacing:-.01em;color:var(--fg)">' +
      '<span style="font-weight:450;color:var(--fg2)">Mini</span>Post</span></span>';
  }

  // 动效：飞机/箭头「飞出再回来」，点击或悬停触发（.go 由页面脚本控制）
  const st = document.createElement('style');
  st.textContent =
    '.lg .lg-move{transform-box:fill-box;transform-origin:center}' +
    '@keyframes lgFly{0%{transform:translate(0,0);opacity:1}38%{transform:translate(11px,-11px);opacity:0}39%{transform:translate(-11px,11px);opacity:0}100%{transform:translate(0,0);opacity:1}}' +
    '.lg.go .lg-move,.lg-hov:hover .lg-move{animation:lgFly 1.1s cubic-bezier(.3,.7,.2,1) infinite}' +
    '@media (prefers-reduced-motion:reduce){.lg.go .lg-move,.lg-hov:hover .lg-move{animation:none}}';
  document.head.appendChild(st);

  window.MPLogo = { list: LIST, glyph, mini, tile, lockup };
})();
