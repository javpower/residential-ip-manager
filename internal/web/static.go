package web

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
  <meta name="theme-color" content="#f5f5f7" />
  <title>住宅出口控制台</title>
  <style>
    :root {
      color-scheme: light;
      --canvas: #f5f5f7;
      --surface: #ffffff;
      --surface-soft: #fafafa;
      --ink: #1d1d1f;
      --secondary: #6e6e73;
      --tertiary: #8e8e93;
      --line: #d2d2d7;
      --line-soft: #e8e8ed;
      --blue: #0071e3;
      --blue-hover: #0077ed;
      --blue-soft: #e8f2ff;
      --green: #248a3d;
      --green-soft: #e9f6ec;
      --orange: #c93400;
      --orange-soft: #fff1e8;
      --red: #d70015;
      --red-soft: #fff0f1;
      --graphite: #2c2c2e;
      --shadow-panel: 0 1px 2px rgba(0, 0, 0, .04), 0 12px 36px rgba(0, 0, 0, .06);
      --shadow-float: 0 18px 60px rgba(0, 0, 0, .16);
      font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
    }
    * { box-sizing: border-box; }
    html { min-width: 320px; background: var(--canvas); }
    body {
      margin: 0;
      min-height: 100dvh;
      background: var(--canvas);
      color: var(--ink);
      font-size: 14px;
      line-height: 1.45;
      letter-spacing: 0;
      -webkit-font-smoothing: antialiased;
    }
    button, input { font: inherit; letter-spacing: 0; }
    button, a { -webkit-tap-highlight-color: transparent; }
    button {
      min-height: 34px;
      border: 1px solid var(--line);
      border-radius: 7px;
      padding: 6px 12px;
      background: var(--surface);
      color: var(--ink);
      font-weight: 550;
      cursor: pointer;
      transition: background-color .16s ease, border-color .16s ease, color .16s ease, transform .16s ease, opacity .16s ease;
    }
    button:hover { background: #f0f0f2; }
    button:active { transform: scale(.98); }
    button:disabled { cursor: default; opacity: .48; transform: none; }
    button.primary { border-color: var(--blue); background: var(--blue); color: #fff; }
    button.primary:hover { border-color: var(--blue-hover); background: var(--blue-hover); }
    button.danger { border-color: transparent; background: transparent; color: var(--red); }
    button.danger:hover { background: var(--red-soft); }
    button.quiet { border-color: transparent; background: transparent; color: var(--secondary); }
    button.quiet:hover { background: #e9e9ec; color: var(--ink); }
    button.icon-button { width: 34px; padding: 0; font-size: 18px; line-height: 1; }
    button.compact { min-height: 30px; padding: 4px 10px; font-size: 12px; }
    :focus-visible { outline: 3px solid rgba(0, 113, 227, .24); outline-offset: 2px; }
    a { color: var(--blue); text-decoration: none; }
    a:hover { text-decoration: underline; }
    h1, h2, h3, p { margin-top: 0; }
    h1 { margin-bottom: 0; font-size: 18px; line-height: 1.2; font-weight: 680; }
    h2 { margin-bottom: 0; font-size: 15px; line-height: 1.3; font-weight: 650; }
    h3 { margin-bottom: 5px; font-size: 13px; font-weight: 650; }
    .muted { color: var(--secondary); }
    .micro { color: var(--tertiary); font-size: 11px; }
    .mono { font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace; font-variant-numeric: tabular-nums; }
    .hidden { display: none !important; }

    .app-mark {
      position: relative;
      width: 38px;
      height: 38px;
      flex: 0 0 38px;
      border-radius: 8px;
      background: var(--graphite);
      box-shadow: inset 0 0 0 1px rgba(255,255,255,.16), 0 4px 12px rgba(0,0,0,.16);
    }
    .app-mark::before {
      content: "";
      position: absolute;
      left: 9px;
      top: 18px;
      width: 20px;
      height: 2px;
      border-radius: 2px;
      background: #fff;
      box-shadow: 0 -7px 0 rgba(255,255,255,.52), 0 7px 0 rgba(255,255,255,.3);
    }
    .app-mark::after {
      content: "";
      position: absolute;
      right: 7px;
      top: 7px;
      width: 7px;
      height: 7px;
      border: 2px solid var(--graphite);
      border-radius: 50%;
      background: #64d26d;
    }

    .login-view {
      min-height: 100dvh;
      display: grid;
      grid-template-columns: minmax(320px, 460px) minmax(0, 1fr);
      background: var(--surface);
    }
    .login-form-side {
      display: grid;
      align-items: center;
      padding: clamp(32px, 6vw, 76px);
      border-right: 1px solid var(--line-soft);
    }
    .login-form-wrap { width: min(336px, 100%); margin: 0 auto; }
    .login-brand { display: flex; align-items: center; gap: 12px; margin-bottom: 48px; }
    .login-copy h2 { margin: 0 0 8px; font-size: 27px; font-weight: 680; }
    .login-copy p { margin-bottom: 28px; color: var(--secondary); }
    .field { display: grid; gap: 7px; margin: 15px 0; }
    .field label { color: var(--secondary); font-size: 12px; font-weight: 550; }
    .field input, .search-input {
      width: 100%;
      min-height: 40px;
      border: 1px solid var(--line);
      border-radius: 7px;
      padding: 8px 11px;
      background: var(--surface);
      color: var(--ink);
      transition: border-color .16s ease, box-shadow .16s ease;
    }
    .field input:focus, .search-input:focus { border-color: var(--blue); box-shadow: 0 0 0 3px rgba(0,113,227,.12); outline: none; }
    .login-submit { width: 100%; margin-top: 9px; min-height: 40px; }
    .form-error { min-height: 21px; margin-top: 10px; color: var(--red); font-size: 12px; }
    .login-scene {
      position: relative;
      display: grid;
      place-items: center;
      overflow: hidden;
      padding: 48px;
      background: #f0f1f3;
    }
    .scene-route {
      position: relative;
      width: min(660px, 88%);
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      align-items: center;
      z-index: 1;
    }
    .scene-route::before { content: ""; position: absolute; left: 11%; right: 11%; top: 20px; height: 2px; background: #b7b7bd; }
    .scene-node { position: relative; display: grid; justify-items: center; gap: 13px; color: var(--secondary); font-size: 12px; text-align: center; }
    .scene-dot { width: 40px; height: 40px; border: 10px solid #fff; border-radius: 50%; background: var(--blue); box-shadow: 0 2px 12px rgba(0,0,0,.12); z-index: 1; }
    .scene-node:nth-child(2) .scene-dot { background: #5e5ce6; }
    .scene-node:nth-child(3) .scene-dot { background: var(--green); }
    .scene-node:nth-child(4) .scene-dot { background: var(--graphite); }

    .app-shell { min-height: 100dvh; }
    .topbar {
      position: sticky;
      top: 0;
      z-index: 20;
      min-height: 64px;
      border-bottom: 1px solid rgba(210,210,215,.88);
      background: rgba(250,250,252,.88);
      backdrop-filter: saturate(180%) blur(18px);
    }
    .topbar-inner {
      width: min(1460px, 100%);
      min-height: 64px;
      margin: 0 auto;
      padding: 10px 24px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 20px;
    }
    .brand { display: flex; align-items: center; gap: 11px; min-width: 224px; }
    .brand-meta { margin-top: 2px; color: var(--secondary); font-size: 11px; }
    .toolbar { display: flex; align-items: center; justify-content: flex-end; gap: 8px; }
    .toolbar-separator { width: 1px; height: 22px; margin: 0 3px; background: var(--line); }
    .button-symbol { display: inline-block; margin-right: 6px; font-size: 15px; line-height: 1; }
    .content { width: min(1460px, 100%); margin: 0 auto; padding: 28px 24px 44px; }

    .route-section { margin-bottom: 20px; }
    .route-head { display: flex; align-items: end; justify-content: space-between; gap: 18px; margin: 0 2px 12px; }
    .state-line { display: flex; align-items: center; gap: 9px; }
    .state-light { width: 9px; height: 9px; border-radius: 50%; background: var(--tertiary); box-shadow: 0 0 0 4px rgba(142,142,147,.12); }
    .state-light.connected { background: var(--green); box-shadow: 0 0 0 4px rgba(36,138,61,.12); }
    .state-light.busy { background: #ff9f0a; box-shadow: 0 0 0 4px rgba(255,159,10,.14); animation: breathe 1.5s ease-in-out infinite; }
    .state-light.error { background: var(--red); box-shadow: 0 0 0 4px rgba(215,0,21,.12); }
    .state-title { font-size: 19px; font-weight: 680; }
    .route-message { margin: 3px 0 0 18px; color: var(--secondary); font-size: 12px; }
    .route-actions { display: flex; align-items: center; justify-content: flex-end; gap: 8px; flex-wrap: wrap; }
    .route-panel {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      border: 1px solid var(--line);
      border-radius: 8px;
      overflow: hidden;
      background: var(--surface);
      box-shadow: var(--shadow-panel);
    }
    .route-step { position: relative; min-height: 104px; padding: 18px 19px; border-right: 1px solid var(--line-soft); }
    .route-step:last-child { border-right: 0; }
    .route-step::after {
      content: "›";
      position: absolute;
      right: -8px;
      top: 36px;
      z-index: 2;
      width: 16px;
      height: 24px;
      display: grid;
      place-items: center;
      color: #b0b0b5;
      background: var(--surface);
      font-size: 21px;
    }
    .route-step:last-child::after { display: none; }
    .route-label { display: flex; align-items: center; gap: 8px; color: var(--secondary); font-size: 11px; font-weight: 650; }
    .route-dot { width: 7px; height: 7px; border-radius: 50%; background: #c7c7cc; }
    .route-step.active .route-dot { background: var(--green); }
    .route-value { display: block; margin-top: 13px; overflow: hidden; color: var(--ink); font-size: 15px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
    .route-detail { display: block; margin-top: 3px; overflow: hidden; color: var(--tertiary); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }

    .metrics {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      margin: 0 0 20px;
      border-top: 1px solid var(--line);
      border-bottom: 1px solid var(--line);
      background: rgba(255,255,255,.42);
    }
    .metric { min-height: 76px; padding: 14px 20px; border-right: 1px solid var(--line); }
    .metric:last-child { border-right: 0; }
    .metric-label { display: block; color: var(--secondary); font-size: 11px; }
    .metric-value { display: block; margin-top: 3px; font-size: 25px; font-weight: 650; font-variant-numeric: tabular-nums; }

    .workspace { display: grid; grid-template-columns: minmax(0, 1fr) 340px; gap: 18px; align-items: start; }
    .panel { border: 1px solid var(--line); border-radius: 8px; background: var(--surface); box-shadow: var(--shadow-panel); overflow: hidden; }
    .panel-head {
      min-height: 58px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 14px;
      padding: 11px 14px 11px 17px;
      border-bottom: 1px solid var(--line-soft);
    }
    .panel-title { display: flex; align-items: baseline; gap: 8px; }
    .panel-count { color: var(--secondary); font-size: 11px; }
    .panel-actions { display: flex; align-items: center; gap: 7px; }
    .search-wrap { position: relative; width: min(230px, 28vw); }
    .search-wrap::before { content: "⌕"; position: absolute; left: 10px; top: 5px; color: var(--tertiary); font-size: 18px; pointer-events: none; }
    .search-input { min-height: 32px; padding: 5px 9px 5px 31px; background: var(--surface-soft); font-size: 12px; }
    .node-tools {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      padding: 9px 14px;
      border-bottom: 1px solid var(--line-soft);
      background: var(--surface-soft);
    }
    .segmented { display: inline-grid; grid-auto-flow: column; grid-auto-columns: 1fr; padding: 2px; border-radius: 7px; background: #e9e9ec; }
    .segmented button { min-height: 26px; border: 0; padding: 3px 10px; background: transparent; color: var(--secondary); font-size: 11px; box-shadow: none; }
    .segmented button:hover { color: var(--ink); }
    .segmented button.active { background: var(--surface); color: var(--ink); box-shadow: 0 1px 3px rgba(0,0,0,.12); }
    .table-wrap { max-height: 560px; overflow: auto; }
    table { width: 100%; border-collapse: separate; border-spacing: 0; font-size: 12px; }
    th, td { height: 46px; padding: 8px 11px; border-bottom: 1px solid var(--line-soft); text-align: left; white-space: nowrap; }
    th { position: sticky; top: 0; z-index: 2; height: 37px; background: rgba(250,250,250,.96); color: var(--secondary); font-size: 10px; font-weight: 650; backdrop-filter: blur(10px); }
    tbody tr { transition: background-color .12s ease; }
    tbody tr:hover { background: #f8f8fa; }
    tbody tr.active-row { background: var(--blue-soft); }
    tbody tr:last-child td { border-bottom: 0; }
    .country-cell { display: flex; align-items: center; gap: 9px; }
    .country-code { width: 29px; color: var(--tertiary); font-size: 10px; font-weight: 650; }
    .ip-cell { color: #3a3a3c; font-family: "SFMono-Regular", Consolas, monospace; font-size: 11px; }
    .isp-cell { max-width: 190px; overflow: hidden; text-overflow: ellipsis; }
    .badge { display: inline-flex; align-items: center; min-height: 22px; border-radius: 999px; padding: 2px 8px; background: #eeeeF0; color: var(--secondary); font-size: 10px; font-weight: 650; }
    .badge.good { background: var(--green-soft); color: var(--green); }
    .badge.info { background: var(--blue-soft); color: var(--blue); }
    .badge.warn { background: var(--orange-soft); color: var(--orange); }
    .badge.bad { background: var(--red-soft); color: var(--red); }
    .latency { color: var(--secondary); font-variant-numeric: tabular-nums; }
    .latency.good { color: var(--green); }
    .latency.warn { color: var(--orange); }
    .latency-cell { display: grid; gap: 2px; }
    .empty-state { padding: 54px 24px; text-align: center; color: var(--secondary); }
    .empty-state strong { display: block; margin-bottom: 5px; color: var(--ink); font-size: 13px; }

    .side-stack { display: grid; gap: 18px; }
    .subscription-body { padding: 17px; }
    .subscription-format { width: 100%; margin-top: 14px; }
    .subscription-address { margin: 14px 0; padding: 10px; border: 1px solid var(--line-soft); border-radius: 6px; background: var(--surface-soft); color: var(--secondary); font-size: 10px; overflow-wrap: anywhere; }
    .subscription-actions { display: grid; grid-template-columns: 1fr auto; gap: 7px; }
    .subscription-open { width: 34px; min-height: 34px; display: grid; place-items: center; border: 1px solid var(--line); border-radius: 7px; background: var(--surface); color: var(--ink); font-size: 17px; font-weight: 650; }
    .subscription-open:hover { background: #f0f0f2; text-decoration: none; }
    .logs { max-height: 388px; overflow: auto; }
    .log { display: grid; grid-template-columns: 48px 7px minmax(0, 1fr); gap: 9px; padding: 10px 14px; border-bottom: 1px solid var(--line-soft); }
    .log:last-child { border-bottom: 0; }
    .log-time { color: var(--tertiary); font-family: "SFMono-Regular", Consolas, monospace; font-size: 10px; }
    .log-dot { width: 7px; height: 7px; margin-top: 5px; border-radius: 50%; background: var(--tertiary); }
    .log-dot.info { background: var(--blue); }
    .log-dot.warning { background: #ff9f0a; }
    .log-dot.error { background: var(--red); }
    .log-message { color: #48484a; font-size: 11px; overflow-wrap: anywhere; }
    .legal-notice { margin-top: 18px; color: var(--tertiary); font-size: 10px; text-align: center; }

    dialog {
      width: min(520px, calc(100% - 30px));
      border: 1px solid rgba(0,0,0,.14);
      border-radius: 8px;
      padding: 0;
      background: var(--surface);
      color: var(--ink);
      box-shadow: var(--shadow-float);
    }
    dialog::backdrop { background: rgba(0,0,0,.24); backdrop-filter: blur(5px); }
    .dialog-head { display: flex; align-items: center; justify-content: space-between; padding: 16px 18px; border-bottom: 1px solid var(--line-soft); }
    .dialog-body { padding: 18px; }
    .dialog-actions { display: flex; justify-content: flex-end; gap: 8px; padding: 12px 18px; border-top: 1px solid var(--line-soft); background: var(--surface-soft); }
    .diagnostic-list { display: grid; gap: 0; border: 1px solid var(--line-soft); border-radius: 7px; overflow: hidden; }
    .diagnostic-row { display: grid; grid-template-columns: 130px minmax(0, 1fr); gap: 12px; padding: 11px 12px; border-bottom: 1px solid var(--line-soft); }
    .diagnostic-row:last-child { border-bottom: 0; }
    .diagnostic-row span:first-child { color: var(--secondary); }
    .diagnostic-row span:last-child { text-align: right; overflow-wrap: anywhere; }
    .confirm-copy { margin: 0; color: var(--secondary); }
    .toast-region { position: fixed; right: 18px; bottom: 18px; z-index: 60; display: grid; gap: 8px; width: min(360px, calc(100% - 36px)); pointer-events: none; }
    .toast { display: grid; grid-template-columns: 8px 1fr; gap: 11px; align-items: start; padding: 12px 14px; border: 1px solid rgba(0,0,0,.12); border-radius: 8px; background: rgba(45,45,47,.96); color: #fff; box-shadow: var(--shadow-float); animation: toast-in .22s ease both; }
    .toast-mark { width: 8px; height: 8px; margin-top: 5px; border-radius: 50%; background: #64d26d; }
    .toast.error .toast-mark { background: #ff453a; }
    .toast-text { font-size: 12px; }
    .spinner { display: inline-block; width: 12px; height: 12px; margin-right: 7px; border: 2px solid currentColor; border-right-color: transparent; border-radius: 50%; vertical-align: -2px; animation: spin .7s linear infinite; }

    @keyframes spin { to { transform: rotate(360deg); } }
    @keyframes breathe { 50% { opacity: .48; } }
    @keyframes toast-in { from { transform: translateY(8px); opacity: 0; } }

    @media (max-width: 1080px) {
      .workspace { grid-template-columns: 1fr; }
      .side-stack { grid-template-columns: minmax(0, .8fr) minmax(0, 1.2fr); }
      .logs { max-height: 280px; }
      .toolbar .label-on-small { display: none; }
      .brand { min-width: auto; }
    }
    @media (max-width: 760px) {
      .login-view { grid-template-columns: 1fr; }
      .login-scene { display: none; }
      .login-form-side { min-height: 100dvh; padding: 28px; border-right: 0; }
      .topbar-inner { min-height: 58px; padding: 9px 14px; }
      .brand-meta { display: none; }
      .toolbar-separator, #envBtn .button-label { display: none; }
      #connectBtn .button-label { display: inline; font-size: 12px; }
      .content { padding: 20px 12px 36px; }
      .route-head { align-items: start; flex-direction: column; }
      .route-actions { width: 100%; justify-content: flex-start; }
      .route-panel { grid-template-columns: repeat(2, minmax(0, 1fr)); }
      .route-step:nth-child(2) { border-right: 0; }
      .route-step:nth-child(-n+2) { border-bottom: 1px solid var(--line-soft); }
      .route-step:nth-child(2)::after { display: none; }
      .route-step { min-height: 91px; padding: 15px; }
      .metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
      .metric:nth-child(2) { border-right: 0; }
      .metric:nth-child(-n+2) { border-bottom: 1px solid var(--line); }
      .panel-head { align-items: flex-start; flex-direction: column; }
      .panel-actions, .search-wrap { width: 100%; }
      .search-wrap { max-width: none; }
      .node-tools { align-items: flex-start; flex-direction: column; }
      .side-stack { grid-template-columns: 1fr; }
      .toolbar button { min-width: 34px; }
      .toolbar .button-symbol { margin-right: 0; }
      .toolbar .button-label { display: none; }
      #connectBtn .button-symbol { margin-right: 5px; }
      .table-wrap { max-height: none; overflow: visible; }
      table, tbody { display: block; }
      table thead { display: none; }
      table tbody tr {
        display: grid;
        grid-template-columns: minmax(0, 1fr) auto;
        grid-template-areas:
          "country action"
          "address action"
          "isp isp"
          "latency purity"
          "status status";
        gap: 1px 12px;
        padding: 12px 14px;
        border-bottom: 1px solid var(--line-soft);
      }
      table tbody tr:last-child { border-bottom: 0; }
      table tbody td { display: block; height: auto; min-height: 0; padding: 0; border: 0; white-space: normal; }
      table tbody td:nth-child(1) { grid-area: country; }
      table tbody td:nth-child(2) { grid-area: address; }
      table tbody td:nth-child(3) { grid-area: isp; max-width: none; margin: 4px 0 5px; color: var(--secondary); font-size: 11px; }
      table tbody td:nth-child(4) { grid-area: latency; align-self: center; }
      table tbody td:nth-child(5) { grid-area: purity; justify-self: end; }
      table tbody td:nth-child(6) { grid-area: status; }
      table tbody td:nth-child(7) { grid-area: action; align-self: center; }
      table tbody .country-cell { gap: 8px; font-weight: 650; }
      table tbody .ip-cell { font-size: 12px; }
    }
    @media (max-width: 430px) {
      .brand h1 { max-width: 122px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      .route-panel { grid-template-columns: 1fr; }
      .route-step { border-right: 0; border-bottom: 1px solid var(--line-soft); }
      .route-step:nth-child(2) { border-bottom: 1px solid var(--line-soft); }
      .route-step::after { display: none; }
      .metric { padding-inline: 14px; }
      .segmented { width: 100%; }
    }
    @media (prefers-reduced-motion: reduce) {
      *, *::before, *::after { scroll-behavior: auto !important; animation-duration: .01ms !important; animation-iteration-count: 1 !important; transition-duration: .01ms !important; }
    }
    @supports not (backdrop-filter: blur(10px)) {
      .topbar, th { background: #fafafc; }
      dialog::backdrop { backdrop-filter: none; }
    }
  </style>
</head>
<body>
  <main id="login" class="login-view">
    <section class="login-form-side">
      <div class="login-form-wrap">
        <div class="login-brand">
          <div class="app-mark" aria-hidden="true"></div>
          <div><h1>住宅出口控制台</h1><div class="brand-meta">Residential IP Manager</div></div>
        </div>
        <div class="login-copy">
          <h2>欢迎回来</h2>
          <p>使用配置文件中的凭据登录。</p>
        </div>
        <form id="loginForm">
          <div class="field"><label for="username">账号</label><input id="username" autocomplete="username" value="admin" /></div>
          <div class="field"><label for="password">密码</label><input id="password" type="password" autocomplete="current-password" /></div>
          <button id="loginSubmit" class="primary login-submit" type="submit">登录</button>
          <div id="loginError" class="form-error" role="alert"></div>
        </form>
      </div>
    </section>
    <section class="login-scene" aria-hidden="true">
      <div class="scene-route">
        <div class="scene-node"><span class="scene-dot"></span><span>本机应用</span></div>
        <div class="scene-node"><span class="scene-dot"></span><span>VMESS</span></div>
        <div class="scene-node"><span class="scene-dot"></span><span>VPNGate</span></div>
        <div class="scene-node"><span class="scene-dot"></span><span>公网出口</span></div>
      </div>
    </section>
  </main>

  <main id="app" class="app-shell hidden">
    <header class="topbar">
      <div class="topbar-inner">
        <div class="brand">
          <div class="app-mark" aria-hidden="true"></div>
          <div><h1>住宅出口控制台</h1><div id="brandStatus" class="brand-meta">正在读取服务状态</div></div>
        </div>
        <div class="toolbar" aria-label="主要操作">
          <button id="connectBtn" class="primary"><span class="button-symbol">↗</span><span class="button-label">自动连接</span></button>
          <button id="envBtn" class="icon-button" title="运行环境" aria-label="运行环境">ⓘ</button>
          <span class="toolbar-separator"></span>
          <button id="logoutBtn" class="quiet"><span class="button-symbol">↪</span><span class="button-label">退出</span></button>
        </div>
      </div>
    </header>

    <div class="content">
      <section class="route-section" aria-labelledby="stateText">
        <div class="route-head">
          <div>
            <div class="state-line"><span id="stateLight" class="state-light"></span><span id="stateText" class="state-title">准备就绪</span></div>
            <p id="messageText" class="route-message">等待服务状态</p>
          </div>
          <div class="route-actions">
            <span id="verificationState" class="badge">尚未验证</span>
            <button id="verifyBtn" class="hidden"><span class="button-symbol">✓</span>立即验证出口</button>
            <button id="disconnectBtn" class="danger hidden">断开出口</button>
          </div>
        </div>
        <div class="route-panel">
          <div id="pathLocal" class="route-step">
            <span class="route-label"><span class="route-dot"></span>本机代理</span>
            <span id="localProxyValue" class="route-value">127.0.0.1:1080</span>
            <span id="localProxyDetail" class="route-detail">等待 VMESS 核心</span>
          </div>
          <div id="pathVMESS" class="route-step">
            <span class="route-label"><span class="route-dot"></span>VMESS 服务</span>
            <span id="vmessValue" class="route-value">端口 10086</span>
            <span id="vmessDetail" class="route-detail">正在读取状态</span>
          </div>
          <div id="pathVPN" class="route-step">
            <span class="route-label"><span class="route-dot"></span>VPNGate 隧道</span>
            <span id="activeNode" class="route-value">未连接</span>
            <span id="tunnelDetail" class="route-detail">等待选择节点</span>
          </div>
          <div id="pathExit" class="route-step">
            <span class="route-label"><span class="route-dot"></span>公网出口</span>
            <span id="exitIP" class="route-value mono">尚未验证</span>
            <span id="exitDetail" class="route-detail">连接后显示出口地址</span>
          </div>
        </div>
      </section>

      <section class="metrics" aria-label="节点统计">
        <div class="metric"><span class="metric-label">节点总数</span><strong id="countNodes" class="metric-value">0</strong></div>
        <div class="metric"><span class="metric-label">当前可用</span><strong id="countAvailable" class="metric-value">0</strong></div>
        <div class="metric"><span class="metric-label">严格家宽</span><strong id="countStrict" class="metric-value">0</strong></div>
        <div class="metric"><span class="metric-label">候选节点</span><strong id="countCandidate" class="metric-value">0</strong></div>
      </section>

      <section class="workspace">
        <div class="panel">
          <div class="panel-head">
            <div class="panel-title"><h2>VPNGate 节点</h2><span id="visibleCount" class="panel-count">0 个结果</span></div>
            <div class="panel-actions">
              <div class="search-wrap"><input id="nodeSearch" class="search-input" type="search" placeholder="搜索国家、IP 或 ISP" aria-label="搜索节点" /></div>
              <button id="refreshBtn" class="icon-button" title="刷新节点" aria-label="刷新节点">↻</button>
            </div>
          </div>
          <div class="node-tools">
            <div id="nodeFilters" class="segmented" aria-label="节点筛选">
              <button class="active" data-filter="all">全部</button>
              <button data-filter="available">可用</button>
              <button data-filter="strict_home">严格家宽</button>
            </div>
            <div class="panel-actions">
              <button id="classifyBtn" class="compact">家宽判定</button>
              <button id="probeBtn" class="compact">TCP 探活</button>
            </div>
          </div>
          <div id="nodeEmpty" class="empty-state hidden"><strong>暂无匹配节点</strong><span>刷新节点或调整筛选条件。</span></div>
          <div id="tableWrap" class="table-wrap">
            <table>
              <thead><tr><th>地区</th><th>节点地址</th><th>ISP / ASN</th><th>连接耗时</th><th>纯度</th><th>状态</th><th></th></tr></thead>
              <tbody id="nodeRows"></tbody>
            </table>
          </div>
        </div>

        <aside class="side-stack">
          <section class="panel">
            <div class="panel-head"><h2>VMESS 订阅</h2><span id="subscriptionState" class="badge">待解析</span></div>
            <div class="subscription-body">
              <div><strong id="subscriptionHost">自动选择地址</strong><div id="subMeta" class="micro">正在读取订阅配置</div></div>
              <div id="subscriptionFormats" class="segmented subscription-format" aria-label="订阅格式">
                <button class="active" data-sub-kind="quantumult-x">Quantumult X</button>
                <button data-sub-kind="vmess">通用 VMESS</button>
              </div>
              <div id="subscriptionURL" class="subscription-address mono">-</div>
              <div class="subscription-actions">
                <button id="copySubBtn" class="primary">复制订阅地址</button>
                <a id="subLink" class="subscription-open" href="#" target="_blank" rel="noopener" title="打开订阅" aria-label="打开订阅">↗</a>
              </div>
            </div>
          </section>
          <section class="panel">
            <div class="panel-head"><h2>运行日志</h2><button id="reloadLogsBtn" class="quiet compact">刷新</button></div>
            <div id="logs" class="logs"><div class="empty-state"><strong>暂无日志</strong></div></div>
          </section>
        </aside>
      </section>
      <footer class="legal-notice">GNU GPL v3 开源软件 · 第三方许可见发布包与源代码</footer>
    </div>
  </main>

  <dialog id="environmentDialog">
    <div class="dialog-head"><h2>运行环境</h2><button class="quiet icon-button" data-close-dialog="environmentDialog" aria-label="关闭">×</button></div>
    <div class="dialog-body"><div id="diagnosticList" class="diagnostic-list"></div></div>
    <div class="dialog-actions">
      <button id="proxyToggleBtn">停止 VMESS</button>
      <button class="primary" data-close-dialog="environmentDialog">完成</button>
    </div>
  </dialog>

  <dialog id="confirmDialog">
    <div class="dialog-head"><h2 id="confirmTitle">确认操作</h2></div>
    <div class="dialog-body"><p id="confirmMessage" class="confirm-copy"></p></div>
    <div class="dialog-actions"><button id="confirmCancel">取消</button><button id="confirmAccept" class="danger">继续</button></div>
  </dialog>

  <div id="toasts" class="toast-region" aria-live="polite"></div>

  <script>
    const $ = (id) => document.getElementById(id);
    const stateLabels = {
      idle: '准备就绪', fetching_nodes: '正在刷新节点', probing_nodes: '正在检测节点',
      connecting: '正在建立隧道', verifying: '正在复核出口', connected: '出口已连接',
      failing_over: '正在切换节点', disconnecting: '正在断开', error: '需要处理'
    };
    const purityLabels = {strict_home: '严格家宽', candidate: '候选', rejected: '已排除'};
    const statusLabels = {available: '可用', unavailable: '不可用', checking: '检测中', cooldown: '冷却中', unknown: '未知'};
    let nodes = [];
    let snapshot = null;
    let environment = null;
    let proxyRunning = false;
    let nodeFilter = 'all';
    let subscriptionURL = '';
    let subscriptionKind = 'quantumult-x';
    let subscriptionURLs = {};
    let loadingAll = false;

    async function api(path, options = {}) {
      const response = await fetch(path, {
        credentials: 'same-origin',
        headers: {'Content-Type': 'application/json'},
        ...options
      });
      if (response.status === 401 && path !== '/api/login') {
        showApp(false);
        throw new Error('登录状态已失效');
      }
      if (!response.ok) {
        const data = await response.json().catch(() => ({error: response.statusText}));
        throw new Error(data.error || response.statusText);
      }
      return response.json();
    }

    function showApp(value) {
      $('login').classList.toggle('hidden', value);
      $('app').classList.toggle('hidden', !value);
      if (!value) {
        $('password').value = '';
        setTimeout(() => $('password').focus(), 60);
      }
    }

    async function loadAll(showFailure) {
      if (loadingAll) return;
      loadingAll = true;
      try {
        const results = await Promise.all([
          api('/api/status'), api('/api/nodes'), api('/api/logs'),
          api('/api/proxy/status'), api('/api/environment'), api('/api/subscription/preview')
        ]);
        nodes = results[1] || [];
        snapshot = results[0].snapshot;
        environment = results[4];
        renderStatus(results[0]);
        renderProxy(results[3]);
        renderEnvironment(results[4]);
        renderNodes();
        renderLogs(results[2] || []);
        renderSubscription(results[5]);
      } catch (error) {
        if (showFailure) toast(error.message, true);
      } finally {
        loadingAll = false;
      }
    }

    function renderStatus(data) {
      const snap = data.snapshot || {};
      const counts = data.counts || {};
      const state = snap.state || 'idle';
      $('stateText').textContent = stateLabels[state] || state;
      $('messageText').textContent = snap.message || '等待操作';
      $('brandStatus').textContent = state === 'connected' ? 'VPNGate 出口在线' : (stateLabels[state] || '服务运行中');
      $('stateLight').className = 'state-light ' + stateTone(state);
      $('activeNode').textContent = snap.active_node_id || '未连接';
      $('exitIP').textContent = snap.exit_ip || '尚未验证';
      $('exitDetail').textContent = snap.metadata && snap.metadata.exit_country
        ? [snap.metadata.exit_country, snap.metadata.exit_asn].filter(Boolean).join(' · ')
        : '连接后显示出口地址';
      $('disconnectBtn').classList.toggle('hidden', state !== 'connected');
      $('verifyBtn').classList.toggle('hidden', state !== 'connected');
      $('connectBtn').disabled = ['fetching_nodes', 'probing_nodes', 'connecting', 'verifying', 'failing_over'].includes(state);
      $('verifyBtn').disabled = state !== 'connected';
      if (snap.last_verified_at) {
        $('verificationState').textContent = '验证通过 · ' + new Date(snap.last_verified_at).toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'});
        $('verificationState').className = 'badge good';
      } else {
        $('verificationState').textContent = state === 'connected' ? '等待出口验证' : '尚未验证';
        $('verificationState').className = 'badge';
      }
      $('countNodes').textContent = counts.nodes || 0;
      $('countAvailable').textContent = counts.available || 0;
      $('countStrict').textContent = counts.strict_home || 0;
      $('countCandidate').textContent = counts.candidate || 0;
    }

    function stateTone(state) {
      if (state === 'connected') return 'connected';
      if (state === 'error') return 'error';
      if (['fetching_nodes', 'probing_nodes', 'connecting', 'verifying', 'failing_over', 'disconnecting'].includes(state)) return 'busy';
      return '';
    }

    function renderProxy(proxy) {
      proxyRunning = Boolean(proxy && proxy.running);
      $('pathLocal').classList.toggle('active', proxyRunning && Boolean(proxy.local_socks));
      $('pathVMESS').classList.toggle('active', proxyRunning);
      $('localProxyValue').textContent = proxy.local_socks || '127.0.0.1:1080';
      $('localProxyDetail').textContent = proxyRunning ? 'SOCKS5 已就绪' : '代理服务未运行';
      $('vmessDetail').textContent = proxyRunning ? 'VMESS 入站已就绪' : (proxy.message || '服务未运行');
      $('proxyToggleBtn').textContent = proxyRunning ? '停止 VMESS' : '启动 VMESS';
      $('proxyToggleBtn').classList.toggle('danger', proxyRunning);
    }

    function renderEnvironment(env) {
      const vpn = env.openvpn || {};
      const proxy = env.proxy || {};
      $('pathVPN').classList.toggle('active', Boolean(vpn.running));
      $('pathExit').classList.toggle('active', Boolean(snapshot && snapshot.exit_ip));
      $('tunnelDetail').textContent = vpn.running
        ? (vpn.local_ip || '隧道在线') + ' · ' + formatBytes(vpn.bytes_in || 0) + ' 下行'
        : (vpn.message || '等待选择节点');
      const rows = [
        ['OpenVPN 引擎', vpn.engine || '内嵌协议引擎'],
        ['隧道状态', vpn.running ? '已连接' : (vpn.connecting ? '连接中' : '未连接')],
        ['隧道地址', vpn.local_ip || '-'],
        ['VPNGate 网关', vpn.gateway || '-'],
        ['收发数据包', formatNumber(vpn.packets_in || 0) + ' / ' + formatNumber(vpn.packets_out || 0)],
        ['收发流量', formatBytes(vpn.bytes_in || 0) + ' / ' + formatBytes(vpn.bytes_out || 0)],
        ['VMESS 服务', proxy.running ? '运行中' : '未运行'],
        ['本机 SOCKS', proxy.local_socks || '-'],
        ['系统权限', vpn.privilege_hint || '无需额外权限']
      ];
      $('diagnosticList').innerHTML = rows.map((row) =>
        '<div class="diagnostic-row"><span>' + escapeHtml(row[0]) + '</span><span class="mono">' + escapeHtml(row[1]) + '</span></div>'
      ).join('');
    }

    function filteredNodes() {
      const query = $('nodeSearch').value.trim().toLowerCase();
      return nodes.filter((node) => {
        if (nodeFilter === 'available' && node.status !== 'available') return false;
        if (nodeFilter === 'strict_home' && node.purity_grade !== 'strict_home') return false;
        if (!query) return true;
        return [node.country, node.country_code, node.ip, node.isp, node.asn, node.city]
          .filter(Boolean).join(' ').toLowerCase().includes(query);
      });
    }

    function renderNodes() {
      const visible = filteredNodes();
      $('visibleCount').textContent = visible.length + ' 个结果';
      $('nodeEmpty').classList.toggle('hidden', visible.length !== 0);
      $('tableWrap').classList.toggle('hidden', visible.length === 0);
      $('nodeRows').innerHTML = visible.map((node) => {
        const active = snapshot && snapshot.active_node_id === node.id;
        const latency = latencyView(node);
        const isp = [node.isp, node.asn].filter(Boolean).join(' · ') || '-';
        return [
          '<tr class="' + (active ? 'active-row' : '') + '">',
          '<td><div class="country-cell"><span class="country-code">' + escapeHtml(node.country_code || '--') + '</span><span>' + escapeHtml(node.country || node.city || '未知') + '</span></div></td>',
          '<td><div class="ip-cell">' + escapeHtml(node.ip) + ':' + escapeHtml(node.remote_port) + '</div><div class="micro">' + escapeHtml((node.protocol || '').toUpperCase()) + '</div></td>',
          '<td class="isp-cell" title="' + escapeHtml(isp) + '">' + escapeHtml(isp) + '</td>',
          '<td><div class="latency-cell" title="' + escapeHtml(node.last_error || '') + '"><span class="latency ' + latency.tone + '">' + escapeHtml(latency.value) + '</span><span class="micro">' + escapeHtml(latency.detail) + '</span></div></td>',
          '<td><span class="badge ' + purityTone(node.purity_grade) + '">' + escapeHtml(purityLabels[node.purity_grade] || node.purity_grade || '未判定') + '</span></td>',
          '<td><span class="badge ' + nodeStatusTone(node.status) + '">' + escapeHtml(statusLabels[node.status] || node.status || '未知') + '</span></td>',
          '<td><button class="compact ' + (active ? 'primary' : '') + '" data-node="' + escapeHtml(node.id) + '" ' + (active ? 'disabled' : '') + '>' + (active ? '当前' : '连接') + '</button></td>',
          '</tr>'
        ].join('');
      }).join('');
      document.querySelectorAll('[data-node]').forEach((button) => {
        button.onclick = () => runAction(button, '正在连接', async () => {
          await api('/api/connect', {method: 'POST', body: JSON.stringify({node_id: button.dataset.node})});
          toast('节点已连接');
          await loadAll(true);
        });
      });
    }

    function renderLogs(logs) {
      if (!logs.length) {
        $('logs').innerHTML = '<div class="empty-state"><strong>暂无日志</strong><span>服务事件会显示在这里。</span></div>';
        return;
      }
      $('logs').innerHTML = logs.slice().reverse().map((item) => {
        const level = String(item.level || 'info').toLowerCase();
        return '<div class="log"><span class="log-time">' + escapeHtml(new Date(item.at).toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'})) + '</span>' +
          '<span class="log-dot ' + escapeHtml(level) + '"></span><span class="log-message">' + escapeHtml(item.message) + '</span></div>';
      }).join('');
    }

    function subscriptionAddress(path, preview) {
      const url = new URL(path, window.location.href);
      const currentIsLoopback = ['127.0.0.1', 'localhost', '::1'].includes(url.hostname);
      const listen = String(preview.subscription_listen || preview.server_listen || '');
      const listensExternally = listen.startsWith('0.0.0.0:') || listen.startsWith('[::]:') || listen.startsWith(':');
      if (currentIsLoopback && listensExternally && preview.resolved_host) url.hostname = preview.resolved_host;
      const portMatch = listen.match(/:(\d+)$/);
      if (portMatch) url.port = portMatch[1];
      return url.href;
    }

    function selectSubscription(kind) {
      subscriptionKind = kind;
      subscriptionURL = subscriptionURLs[kind] || '';
      $('subscriptionURL').textContent = subscriptionURL || '-';
      $('subLink').href = subscriptionURL || '#';
      $('copySubBtn').textContent = kind === 'quantumult-x' ? '复制 Quantumult X 订阅' : '复制 VMESS 订阅';
      document.querySelectorAll('[data-sub-kind]').forEach((button) => button.classList.toggle('active', button.dataset.subKind === kind));
    }

    function renderSubscription(preview) {
      if (!preview) return;
      subscriptionURLs = {
        'quantumult-x': subscriptionAddress(preview.quantumult_x_url, preview),
        vmess: subscriptionAddress(preview.vmess_url, preview)
      };
      selectSubscription(subscriptionKind);
      $('subscriptionHost').textContent = preview.resolved_host || preview.subscription_host || '自动选择地址';
      const listen = String(preview.subscription_listen || preview.server_listen || '');
      const external = listen.startsWith('0.0.0.0:') || listen.startsWith('[::]:');
      $('subMeta').textContent = external ? '局域网客户端可获取订阅' : '仅本机可获取订阅；手机需要开启局域网监听';
      $('subscriptionState').textContent = '可用';
      $('subscriptionState').className = 'badge good';
    }

    async function runAction(button, busyLabel, action) {
      const original = button.innerHTML;
      button.disabled = true;
      button.innerHTML = '<span class="spinner"></span>' + busyLabel;
      try {
        await action();
      } catch (error) {
        toast(error.message, true);
      } finally {
        button.innerHTML = original;
        button.disabled = false;
      }
    }

    function toast(message, isError) {
      const item = document.createElement('div');
      item.className = 'toast' + (isError ? ' error' : '');
      item.innerHTML = '<span class="toast-mark"></span><span class="toast-text">' + escapeHtml(message) + '</span>';
      $('toasts').appendChild(item);
      setTimeout(() => item.remove(), 3600);
    }

    function askConfirm(title, message, confirmLabel) {
      return new Promise((resolve) => {
        const dialog = $('confirmDialog');
        $('confirmTitle').textContent = title;
        $('confirmMessage').textContent = message;
        $('confirmAccept').textContent = confirmLabel || '继续';
        const finish = (value) => {
          dialog.close();
          $('confirmCancel').onclick = null;
          $('confirmAccept').onclick = null;
          resolve(value);
        };
        $('confirmCancel').onclick = () => finish(false);
        $('confirmAccept').onclick = () => finish(true);
        dialog.oncancel = (event) => { event.preventDefault(); finish(false); };
        dialog.showModal();
      });
    }

    function formatBytes(value) {
      const number = Number(value) || 0;
      if (number < 1024) return number + ' B';
      if (number < 1048576) return (number / 1024).toFixed(1) + ' KB';
      if (number < 1073741824) return (number / 1048576).toFixed(1) + ' MB';
      return (number / 1073741824).toFixed(1) + ' GB';
    }
    function formatNumber(value) { return new Intl.NumberFormat('zh-CN').format(Number(value) || 0); }
    function latencyView(node) {
      if (String(node.protocol || '').toLowerCase() !== 'tcp') return {value:'不适用', detail:'UDP', tone:''};
      const samples = node.probe_attempts ? node.probe_successes + '/' + node.probe_attempts + ' 次' : '未采样';
      if (node.latency_suspicious) return {value:'待复测', detail:samples + ' · 疑似本机接管', tone:'warn'};
      if (node.latency_ms != null) return {value:node.latency_ms + ' ms', detail:samples + ' · TCP 中位数', tone:latencyTone(node.latency_ms)};
      if (node.advertised_ping_ms > 0) return {value:node.advertised_ping_ms + ' ms', detail:'VPNGate 参考值', tone:''};
      return {value:'-', detail:samples, tone:''};
    }
    function latencyTone(value) { if (value == null) return ''; return value < 180 ? 'good' : (value > 450 ? 'warn' : ''); }
    function purityTone(value) { return value === 'strict_home' ? 'good' : (value === 'candidate' ? 'info' : (value === 'rejected' ? 'bad' : '')); }
    function nodeStatusTone(value) { return value === 'available' ? 'good' : (value === 'checking' ? 'info' : (value === 'cooldown' ? 'warn' : (value === 'unavailable' ? 'bad' : ''))); }
    function escapeHtml(value) {
      return String(value ?? '').replace(/[&<>"']/g, (character) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[character]));
    }

    $('loginForm').onsubmit = async (event) => {
      event.preventDefault();
      $('loginError').textContent = '';
      await runAction($('loginSubmit'), '正在登录', async () => {
        try {
          await api('/api/login', {method: 'POST', body: JSON.stringify({username: $('username').value, password: $('password').value})});
          showApp(true);
          await loadAll(true);
        } catch (error) {
          $('loginError').textContent = error.message;
          throw error;
        }
      });
    };
    $('connectBtn').onclick = () => runAction($('connectBtn'), '连接中', async () => {
      await api('/api/connect', {method: 'POST', body: '{}'});
      toast('VPNGate 出口已连接');
      await loadAll(true);
    });
    $('verifyBtn').onclick = () => runAction($('verifyBtn'), '复核中', async () => {
      const result = await api('/api/exit/verify', {method: 'POST'});
      toast('出口验证通过：' + result.exit_ip);
      await loadAll(true);
    });
    $('refreshBtn').onclick = () => runAction($('refreshBtn'), '刷新中', async () => {
      await api('/api/nodes/refresh', {method: 'POST'});
      toast('节点列表已刷新');
      await loadAll(true);
    });
    $('classifyBtn').onclick = () => runAction($('classifyBtn'), '判定中', async () => {
      await api('/api/nodes/classify', {method: 'POST'});
      toast('家宽判定已完成');
      await loadAll(true);
    });
    $('probeBtn').onclick = () => runAction($('probeBtn'), '探活中', async () => {
      await api('/api/nodes/probe', {method: 'POST'});
      toast('TCP 探活已完成');
      await loadAll(true);
    });
    $('disconnectBtn').onclick = async () => {
      if (!await askConfirm('断开当前出口', '本机 SOCKS 和远程 VMESS 流量将暂时无法访问公网。', '断开出口')) return;
      await runAction($('disconnectBtn'), '断开中', async () => {
        await api('/api/disconnect', {method: 'POST'});
        toast('VPNGate 出口已断开');
        await loadAll(true);
      });
    };
    $('envBtn').onclick = async () => { await loadAll(true); $('environmentDialog').showModal(); };
    $('proxyToggleBtn').onclick = async () => {
      if (proxyRunning && !await askConfirm('停止 VMESS 服务', '本机 SOCKS 和所有 VMESS 客户端连接将立即中断。', '停止服务')) return;
      const endpoint = proxyRunning ? '/api/proxy/stop' : '/api/proxy/start';
      await runAction($('proxyToggleBtn'), proxyRunning ? '停止中' : '启动中', async () => {
        await api(endpoint, {method: 'POST'});
        toast(proxyRunning ? 'VMESS 服务已停止' : 'VMESS 服务已启动');
        await loadAll(true);
      });
    };
    $('logoutBtn').onclick = async () => { await api('/api/logout', {method: 'POST'}); showApp(false); };
    $('reloadLogsBtn').onclick = () => loadAll(true);
    $('nodeSearch').oninput = renderNodes;
    $('nodeFilters').onclick = (event) => {
      const button = event.target.closest('[data-filter]');
      if (!button) return;
      nodeFilter = button.dataset.filter;
      document.querySelectorAll('[data-filter]').forEach((item) => item.classList.toggle('active', item === button));
      renderNodes();
    };
    $('subscriptionFormats').onclick = (event) => {
      const button = event.target.closest('[data-sub-kind]');
      if (button) selectSubscription(button.dataset.subKind);
    };
    $('copySubBtn').onclick = async () => {
      if (!subscriptionURL) return toast('订阅地址尚未准备好', true);
      try {
        await navigator.clipboard.writeText(subscriptionURL);
        toast('订阅地址已复制');
      } catch (_) {
        toast('浏览器未允许复制，请从地址框手动复制', true);
      }
    };
    document.querySelectorAll('[data-close-dialog]').forEach((button) => {
      button.onclick = () => $(button.dataset.closeDialog).close();
    });
    document.addEventListener('visibilitychange', () => { if (!document.hidden && !$('app').classList.contains('hidden')) loadAll(false); });
    setInterval(() => { if (!document.hidden && !$('app').classList.contains('hidden')) loadAll(false); }, 10000);

    api('/api/me').then(() => { showApp(true); return loadAll(true); }).catch(() => showApp(false));
  </script>
</body>
</html>`
