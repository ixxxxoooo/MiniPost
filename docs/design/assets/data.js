/* MiniPost 设计稿的演示数据（全部为虚构数据） */
window.MP = window.MP || {};
(function () {
  const METHOD_ABBR = { GET: 'GET', POST: 'POST', PUT: 'PUT', PATCH: 'PATCH', DELETE: 'DEL', HEAD: 'HEAD', OPTIONS: 'OPT' };

  // 项目与主题色（与现有项目主题色预设一致）
  const PROJECT_COLORS = ['#0a84ff', '#30d158', '#ff9f0a', '#ff453a', '#bf5af2', '#64d2ff', '#5e5ce6', '#ff375f', '#34c759', '#ffd60a', '#ff6b35', '#00c7be'];
  const projects = [
    { id: 'p1', name: 'Acme 开放平台', desc: '电商后台 API 联调', color: '#5e5ce6', reqs: 24, folders: 6, envs: 3, updated: '刚刚', cur: true },
    { id: 'p2', name: '支付网关', desc: '三方支付回调与对账', color: '#30d158', reqs: 18, folders: 4, envs: 2, updated: '2 小时前' },
    { id: 'p3', name: 'AI 对话服务', desc: 'SSE 流式接口与鉴权', color: '#ff9f0a', reqs: 11, folders: 3, envs: 2, updated: '昨天' },
    { id: 'p4', name: '内部运营后台', desc: '', color: '#ff375f', reqs: 42, folders: 9, envs: 4, updated: '3 天前' },
    { id: 'p5', name: 'Mock 沙盒', desc: '前端联调用的假数据', color: '#00c7be', reqs: 7, folders: 1, envs: 1, updated: '上周' },
  ];

  // 请求树
  const tree = [
    { t: 'f', id: 'f1', name: '用户服务', open: true, kids: [
      { t: 'r', id: 'r1', m: 'GET', name: '获取用户列表', url: '{{host}}/users?page=1&size=20' },
      { t: 'r', id: 'r2', m: 'POST', name: '创建用户', url: '{{host}}/users' },
      { t: 'r', id: 'r3', m: 'PUT', name: '更新用户资料', url: '{{host}}/users/1024' },
      { t: 'r', id: 'r4', m: 'DELETE', name: '删除用户', url: '{{host}}/users/1024' },
    ] },
    { t: 'f', id: 'f2', name: '订单服务', open: true, kids: [
      { t: 'r', id: 'r5', m: 'GET', name: '订单列表', url: '{{host}}/orders?status=paid' },
      { t: 'r', id: 'r6', m: 'POST', name: '创建订单', url: '{{host}}/orders' },
      { t: 'r', id: 'r9', m: 'PATCH', name: '修改收货地址', url: '{{host}}/orders/8801/address' },
    ] },
    { t: 'f', id: 'f3', name: '流式 / AI', open: false, kids: [
      { t: 'r', id: 'r7', m: 'POST', name: 'Chat 流式补全', url: '{{host}}/v1/chat/stream' },
      { t: 'r', id: 'r10', m: 'GET', name: 'SSE 事件订阅', url: '{{host}}/events?count=5' },
    ] },
    { t: 'f', id: 'f4', name: '认证', open: false, kids: [
      { t: 'r', id: 'r11', m: 'POST', name: '登录获取 Token', url: '{{host}}/auth/login' },
      { t: 'r', id: 'r12', m: 'POST', name: '刷新 Token', url: '{{host}}/auth/refresh' },
    ] },
    { t: 'r', id: 'r8', m: 'GET', name: '健康检查', url: '{{host}}/healthz' },
  ];

  // 请求详情（未列出的请求使用默认值）
  const reqs = {
    r1: {
      params: [['page', '1', '页码，从 1 开始', 1], ['size', '20', '每页条数，最大 100', 1], ['keyword', '', '', 0]],
      headers: [['Accept', 'application/json', '', 1], ['X-Request-Id', '{{$randomUUID}}', '链路追踪', 1], ['Accept-Language', 'zh-CN', '', 0]],
      body: { type: 'none' }, auth: { type: 'bearer', token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.demo' },
    },
    r2: {
      params: [], headers: [['Content-Type', 'application/json', '', 1], ['Accept', 'application/json', '', 1]],
      body: { type: 'json', text: '{\n  "name": "Alice",\n  "email": "alice@example.com",\n  "roles": ["admin", "editor"],\n  "profile": {\n    "age": 28,\n    "active": true,\n    "avatar": null\n  }\n}' },
      auth: { type: 'bearer', token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.demo' },
    },
    r7: {
      params: [], headers: [['Content-Type', 'application/json', '', 1], ['Accept', 'text/event-stream', '', 1]],
      body: { type: 'json', text: '{\n  "model": "mini-1",\n  "stream": true,\n  "messages": [\n    { "role": "user", "content": "用一句话介绍 MiniPost" }\n  ]\n}' },
      auth: { type: 'bearer', token: 'sk-demo-••••••' },
    },
  };
  // 补充：更多请求体 / 认证类型，用于覆盖 form-data、urlencoded、raw 与 Basic、API Key
  Object.assign(reqs, {
    r3: {
      params: [], headers: [['Content-Type', 'multipart/form-data', '', 1]],
      body: { type: 'form', rows: [['name', 'Alice', '', 1, 'text'], ['avatar', 'avatar.png', '头像，最大 2 MB', 1, 'file'], ['note', '', '', 0, 'text']] },
      auth: { type: 'bearer', token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.demo' },
    },
    r11: {
      params: [], headers: [['Content-Type', 'application/x-www-form-urlencoded', '', 1]],
      body: { type: 'urlenc', rows: [['username', 'alice@example.com', '登录名', 1], ['password', '{{password}}', '', 1], ['grant_type', 'password', '', 1]] },
      auth: { type: 'none' },
    },
    r12: {
      params: [], headers: [['Content-Type', 'text/plain', '', 1]],
      body: { type: 'raw', text: 'refresh_token=rt_2f8a91c0b7d4\nclient_id=minipost-desktop' },
      auth: { type: 'none' },
    },
    r4: {
      params: [], headers: [['Accept', '*/*', '', 1]], body: { type: 'none' },
      auth: { type: 'basic', user: 'alice', pass: 'hunter2-demo' },
    },
    r9: {
      params: [], headers: [['Content-Type', 'application/json', '', 1]],
      body: { type: 'json', text: '{\n  "receiver": "Alice",\n  "phone": "13800000000",\n  "address": "上海市浦东新区世纪大道 100 号"\n}' },
      auth: { type: 'apikey', key: 'X-API-Key', value: 'ak_live_9f3c2d71b8e4', where: 'header' },
    },
  });
  const defReq = { params: [['', '', '', 1]], headers: [['Accept', '*/*', '', 1]], body: { type: 'none' }, auth: { type: 'none' } };

  // 示例响应
  const jsonOk = '{\n  "code": 0,\n  "message": "ok",\n  "data": {\n    "total": 128,\n    "page": 1,\n    "users": [\n      {\n        "id": 1024,\n        "name": "Alice",\n        "email": "alice@example.com",\n        "roles": ["admin", "editor"],\n        "active": true,\n        "balance": 1280.5,\n        "lastLogin": "2026-11-15T09:41:07Z"\n      },\n      {\n        "id": 1025,\n        "name": "Bob",\n        "email": "bob@example.com",\n        "roles": ["viewer"],\n        "active": false,\n        "balance": 0,\n        "lastLogin": null\n      }\n    ]\n  }\n}';
  const jsonErr = '{\n  "code": 42201,\n  "message": "参数校验失败",\n  "errors": [\n    { "field": "email", "reason": "格式不正确" },\n    { "field": "roles", "reason": "至少选择一个角色" }\n  ],\n  "requestId": "9f2c-41ab-77de"\n}';
  const respHeaders = [
    ['content-type', 'application/json; charset=utf-8'], ['content-length', '1186'], ['server', 'nginx/1.25.3'],
    ['cache-control', 'no-store'], ['x-request-id', '9f2c41ab-77de-4c0a-b1d5-2e8f3a6c91d0'],
    ['strict-transport-security', 'max-age=63072000; includeSubDomains'], ['date', 'Sun, 15 Nov 2026 01:41:07 GMT'],
  ];
  const cookies = [
    ['sid', 'a1b2c3d4e5f6', '/', '会话', true, true], ['theme', 'dark', '/', '2027-11-15', false, false],
  ];
  const streamEvents = [
    ['0.00', 'ping', '{"id":0,"timestamp":1794652952928}'],
    ['0.50', 'message', '{"delta":"MiniPost"}'],
    ['1.00', 'message', '{"delta":" 是一款"}'],
    ['1.50', 'message', '{"delta":"轻量、快速"}'],
    ['2.00', 'message', '{"delta":"的 HTTP 调试工具，"}'],
    ['2.50', 'message', '{"delta":"支持 SSE 流式响应。"}'],
    ['3.00', 'done', '{"finish_reason":"stop","tokens":38}'],
  ];
  const mergedText = 'MiniPost 是一款轻量、快速的 HTTP 调试工具，支持 SSE 流式响应。';

  const envs = [
    { id: 'e1', name: '开发环境', desc: '本地与测试网关', cur: true, vars: [['host', 'https://api.dev.acme.io', 0, 1], ['token', 'dev-9c1f2a7b-3d4e', 1, 1], ['tenant', 'acme-dev', 0, 1], ['debug', 'true', 0, 0]] },
    { id: 'e2', name: '预发布', desc: '', cur: false, vars: [['host', 'https://api.stg.acme.io', 0, 1], ['token', 'stg-••••', 1, 1]] },
    { id: 'e3', name: '生产环境', desc: '谨慎操作', cur: false, vars: [['host', 'https://api.acme.io', 0, 1], ['token', 'prod-••••', 1, 1]] },
  ];
  const dynVars = [
    ['{{$timestamp}}', '当前时间，秒', '1794708612'], ['{{$timestampMs}}', '当前时间，毫秒', '1794708612623'],
    ['{{$isoTimestamp}}', 'ISO 8601 时间', '2026-11-15T02:10:12Z'], ['{{$randomUUID}}', '随机 UUID v4', '64cbd8bc-2e0c-45d2-80ba-2ccc…'],
    ['{{$randomInt}}', '0–1000 的随机整数', '296'],
  ];
  const history = [
    { day: '今天', items: [['GET', '/users?page=1&size=20', 200, '10:10'], ['POST', '/users', 201, '10:02'], ['POST', '/orders', 422, '09:48'], ['GET', '/healthz', 0, '09:31'], ['GET', '/orders?status=paid', 200, '09:12']] },
    { day: '昨天', items: [['PUT', '/users/1024', 200, '18:40'], ['DELETE', '/users/1024', 204, '18:22'], ['POST', '/auth/login', 200, '11:05']] },
  ];
  const consoleRows = [
    ['10:10:12', 'GET', '{{host}}/users?page=1&size=20', 200, '128 ms', '1.2 KB'],
    ['10:02:44', 'POST', '{{host}}/users', 201, '246 ms', '312 B'],
    ['09:48:03', 'POST', '{{host}}/orders', 422, '91 ms', '208 B'],
    ['09:31:19', 'GET', '{{host}}/healthz', 0, '—', '—'],
  ];
  const cookieDomains = [
    { d: 'api.acme.io', n: 2 }, { d: 'auth.acme.io', n: 1 }, { d: 'example.org', n: 1 },
  ];
  const shortcuts = [
    ['发送请求', '⌘ ↵'], ['保存', '⌘ S'], ['关闭标签', '⌘ W'], ['新建请求', '⌘ N'], ['新建文件夹', '⌘ ⇧ N'],
    ['导入 cURL', '⌘ I'], ['导入集合', '⌘ O'], ['重命名 / 复制 / 删除', '⌘E · ⌘D · ⌫'], ['在响应中查找', '⌘ F'],
    ['下一个 / 上一个标签', '⌃ Tab · ⌃⇧ Tab'], ['复制为 cURL', '⌘ ⇧ C'], ['环境', '⌘ ⌥ E'], ['侧栏 / 控制台', '⌘ B · ⌘ J'],
    ['快速打开', '⌘ K'], ['设置', '⌘ ,'],
  ];

  Object.assign(window.MP, { METHOD_ABBR, PROJECT_COLORS, projects, tree, reqs, defReq, jsonOk, jsonErr, respHeaders, cookies, streamEvents, mergedText, envs, dynVars, history, consoleRows, cookieDomains, shortcuts });
})();
