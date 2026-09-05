package main

func GetMessagePage() string {
	var MessagePage = `<!doctype html>
<html lang="zh-cmn-Hans">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <meta name="theme-color" content="#fafaf9">
  <title>记事板</title>
  <style>
    :root {
      --bg: #fafaf9;
      --card: #ffffff;
      --line: #e7e5e4;
      --text: #1c1917;
      --muted: #78716c;
      --accent: #2563eb;
      --accent-text: #ffffff;
      --danger: #b91c1c;
      --btn-bg: #f5f5f4;
      --btn-line: #e7e5e4;
      --input-bg: #fafaf9;
    }
    * { box-sizing: border-box; }
    html, body { height: 100%; }
    body {
      margin: 0;
      min-height: 100dvh;
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
      background: var(--bg);
    }
    .page {
      max-width: 760px;
      margin: 0 auto;
      padding: max(20px, env(safe-area-inset-top)) 16px max(24px, env(safe-area-inset-bottom));
    }
    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 20px;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 10px;
      color: var(--text);
      font-weight: 650;
      letter-spacing: 0.02em;
    }
    .brand-mark {
      width: 34px;
      height: 34px;
      border-radius: 10px;
      display: grid;
      place-items: center;
      background: var(--accent);
      color: var(--accent-text);
      font-size: 15px;
    }
    .brand-sub {
      display: block;
      font-size: 12px;
      color: var(--muted);
      font-weight: 400;
    }
    .btn {
      appearance: none;
      border: 1px solid var(--btn-line);
      border-radius: 10px;
      background: var(--btn-bg);
      color: var(--text);
      padding: 8px 14px;
      font-size: 14px;
      cursor: pointer;
      text-decoration: none;
      display: inline-flex;
      align-items: center;
      gap: 6px;
      transition: background .15s, border-color .15s;
      min-height: 38px;
    }
    .btn:hover { background: #ebebe9; }
    .btn-primary {
      background: var(--accent);
      color: var(--accent-text);
      border-color: var(--accent);
      font-weight: 600;
    }
    .btn-primary:hover { background: #1d4fd8; }
    .btn-danger { color: var(--danger); }
    .btn-danger:hover { background: #fdecec; }
    .card {
      background: var(--card);
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: 18px;
      box-shadow: 0 1px 2px rgba(0,0,0,0.04);
    }
    .composer { margin-bottom: 22px; }
    .composer textarea {
      width: 100%;
      min-height: 110px;
      resize: vertical;
      border-radius: 12px;
      border: 1px solid var(--line);
      background: var(--input-bg);
      color: var(--text);
      padding: 12px 14px;
      font-size: 15px;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      line-height: 1.5;
      margin-bottom: 12px;
    }
    .composer textarea:focus { outline: 2px solid rgba(37,99,235,0.35); border-color: transparent; }
    .composer-foot {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      flex-wrap: wrap;
    }
    .hint {
      font-size: 12px;
      color: var(--muted);
      word-break: break-all;
    }
    .hint code { color: var(--accent); }
    .list-head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin: 0 2px 12px;
      color: var(--muted);
      font-size: 13px;
    }
    .msg-list { display: flex; flex-direction: column; gap: 14px; }
    .msg-card { padding: 14px 16px; }
    .msg-top {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 10px;
      margin-bottom: 8px;
      font-size: 12px;
      color: var(--muted);
    }
    .msg-time { display: inline-flex; align-items: center; gap: 6px; }
    .msg-preview {
      white-space: pre-wrap;
      word-break: break-word;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 14px;
      line-height: 1.55;
      color: var(--text);
      margin-bottom: 12px;
      overflow: hidden;
      display: -webkit-box;
      -webkit-line-clamp: 8;
      -webkit-box-orient: vertical;
    }
    .msg-actions { display: flex; gap: 8px; flex-wrap: wrap; }
    .msg-actions .btn { padding: 6px 12px; font-size: 13px; min-height: 34px; }
    .empty {
      text-align: center;
      color: var(--muted);
      padding: 40px 0;
      font-size: 14px;
    }
    .toast {
      position: fixed;
      left: 50%;
      bottom: 28px;
      transform: translateX(-50%) translateY(20px);
      background: var(--card);
      border: 1px solid var(--line);
      color: var(--text);
      padding: 10px 18px;
      border-radius: 12px;
      font-size: 14px;
      opacity: 0;
      pointer-events: none;
      transition: opacity .2s, transform .2s;
      z-index: 99;
      max-width: 86vw;
      box-shadow: 0 8px 24px rgba(0,0,0,0.12);
    }
    .toast.show { opacity: 1; transform: translateX(-50%) translateY(0); }
    .toast.ok { border-color: rgba(37,99,235,0.45); }
    .toast.err { border-color: rgba(185,28,28,0.5); }
    @media (max-width: 480px) {
      .page { padding-left: 12px; padding-right: 12px; }
      .card { padding: 14px; border-radius: 13px; }
      .msg-preview { -webkit-line-clamp: 6; }
      .msg-actions .btn { flex: 1; justify-content: center; }
      .composer-foot { flex-direction: column; align-items: stretch; }
      .composer-foot .btn-primary { justify-content: center; }
    }
  </style>
</head>
<body>
<div class="page">
  <div class="topbar">
    <div class="brand">
      <span class="brand-mark">✎</span>
      <span>记事板<span class="brand-sub">/t · 留言与临时记事</span></span>
    </div>
    <button class="btn" id="refresh-btn" type="button">刷新</button>
  </div>

  <form class="card composer" id="composer" action="/t" method="post">
    <textarea id="msg-input" name="message" placeholder="输入任意内容，Ctrl/Cmd+Enter 快速提交"></textarea>
    <div class="composer-foot">
      <span class="hint">curl 也行：curl -X POST -d 'message=内容' https://` + Cfg.Domain + `/t</span>
      <button class="btn btn-primary" type="submit">提交</button>
    </div>
  </form>

  <div class="list-head">
    <span id="list-count"></span>
    <span>按时间倒序</span>
  </div>
  <div class="msg-list" id="msg-list"><div class="empty">加载中…</div></div>
</div>

<div class="toast" id="toast"></div>

<script>
(function () {
  'use strict';

  var listEl = document.getElementById('msg-list');
  var countEl = document.getElementById('list-count');
  var composer = document.getElementById('composer');
  var input = document.getElementById('msg-input');
  var toastEl = document.getElementById('toast');
  var toastTimer = null;

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function toast(msg, ok) {
    toastEl.textContent = msg;
    toastEl.className = 'toast show ' + (ok ? 'ok' : 'err');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { toastEl.className = 'toast'; }, 2200);
  }

  function fmtTime(iso) {
    var d = new Date(iso);
    if (isNaN(d.getTime())) return iso;
    return d.toLocaleString('zh-CN', {
      timeZone: 'Asia/Shanghai', hour12: false,
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit'
    });
  }

  function errMsg(res) {
    if (res.status === 401) return '请先在 /dilfish.html 登录';
    return '请求失败 (HTTP ' + res.status + ')';
  }

  function render(list) {
    countEl.textContent = '共 ' + list.length + ' 条';
    if (!list.length) {
      listEl.innerHTML = '<div class="empty">还没有内容，写一条吧</div>';
      return;
    }
    var html = '';
    for (var i = 0; i < list.length; i++) {
      var m = list[i];
      var id = escapeHtml(m._id);
      var more = m.truncated ? ' …' : '';
      html += '<article class="card msg-card" data-id="' + id + '">'
        + '<div class="msg-top"><span class="msg-time">🕐 ' + escapeHtml(fmtTime(m.createAt)) + '</span></div>'
        + '<div class="msg-preview">' + escapeHtml(m.preview) + more + '</div>'
        + '<div class="msg-actions">'
        + '<button class="btn" type="button" data-act="copy">拷贝</button>'
        + '<a class="btn" href="/t/list/' + id + '" target="_blank" rel="noopener">查看</a>'
        + '<button class="btn btn-danger" type="button" data-act="del">删除</button>'
        + '</div></article>';
    }
    listEl.innerHTML = html;
  }

  function loadList() {
    fetch('/api/t/list', { credentials: 'same-origin' })
      .then(function (res) {
        if (!res.ok) throw new Error(errMsg(res));
        return res.json();
      })
      .then(function (data) { render(data.list || []); })
      .catch(function (e) {
        listEl.innerHTML = '<div class="empty">加载失败：' + escapeHtml(e.message) + '</div>';
      });
  }

  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise(function (resolve, reject) {
      var ta = document.createElement('textarea');
      ta.value = text;
      ta.style.position = 'fixed';
      ta.style.opacity = '0';
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand('copy') ? resolve() : reject(new Error('copy failed'));
      } catch (e) {
        reject(e);
      } finally {
        document.body.removeChild(ta);
      }
    });
  }

  listEl.addEventListener('click', function (ev) {
    var btn = ev.target.closest('button[data-act]');
    if (!btn) return;
    var card = btn.closest('.msg-card');
    var id = card.getAttribute('data-id');

    if (btn.getAttribute('data-act') === 'copy') {
      fetch('/t/list/' + encodeURIComponent(id), { credentials: 'same-origin' })
        .then(function (res) {
          if (!res.ok) throw new Error(errMsg(res));
          return res.text();
        })
        .then(function (text) { return copyText(text); })
        .then(function () { toast('已拷贝到剪贴板', true); })
        .catch(function (e) { toast('拷贝失败：' + e.message, false); });
      return;
    }

    if (btn.getAttribute('data-act') === 'del') {
      if (!confirm('确定删除这条内容？')) return;
      fetch('/api/t/delete', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: id })
      })
        .then(function (res) {
          if (!res.ok) throw new Error(errMsg(res));
          card.remove();
          toast('已删除', true);
          loadList();
        })
        .catch(function (e) { toast('删除失败：' + e.message, false); });
    }
  });

  composer.addEventListener('submit', function (ev) {
    ev.preventDefault();
    var text = input.value;
    if (text.trim().length < 2) {
      toast('内容太短了', false);
      return;
    }
    fetch('/api/t', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ msg: text })
    })
      .then(function (res) {
        if (!res.ok) throw new Error(errMsg(res));
        return res.json();
      })
      .then(function () {
        input.value = '';
        toast('已提交', true);
        loadList();
      })
      .catch(function (e) { toast('提交失败：' + e.message, false); });
  });

  input.addEventListener('keydown', function (ev) {
    if ((ev.ctrlKey || ev.metaKey) && ev.key === 'Enter') {
      composer.requestSubmit();
    }
  });

  document.getElementById('refresh-btn').addEventListener('click', loadList);
  loadList();
})();
</script>
</body>
</html>
`
	return MessagePage
}
