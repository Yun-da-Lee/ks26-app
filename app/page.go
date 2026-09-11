package main

// indexHTML 的四個佔位依序是：標題、降級橫幅、擁有者、映像檔標籤。
// 這裡不放任何外部資源位址，頁面在沒有對外連線的叢集裡也要能顯示。
const indexHTML = `<!doctype html>
<html lang="zh-Hant">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
  :root { color-scheme: light dark; }
  body { margin: 0; padding: 2rem 1rem; font: 16px/1.6 system-ui, sans-serif; }
  main { max-width: 44rem; margin: 0 auto; }
  h1 { font-size: 1.4rem; margin: 0 0 1rem; }
  textarea { width: 100%%; box-sizing: border-box; padding: .6rem; font: inherit;
             border: 1px solid #8888; border-radius: .4rem; min-height: 5rem; }
  button { margin-top: .75rem; padding: .5rem 1.2rem; font: inherit;
           border: 0; border-radius: .4rem; background: #2563eb; color: #fff; cursor: pointer; }
  button[disabled] { opacity: .55; cursor: progress; }
  .answer { margin-top: 1.5rem; padding: 1rem; border-radius: .5rem;
            background: #8881; white-space: pre-wrap; }
  .degraded { padding: .6rem .8rem; border-radius: .4rem;
              background: #f59e0b22; border: 1px solid #f59e0b88; }
  footer { margin-top: 2.5rem; font-size: .8rem; opacity: .7; }
</style>
</head>
<body>
<main>
  <h1>%s</h1>
  %s
  <form id="ask">
    <label for="q">你的問題</label>
    <textarea id="q" name="q" required placeholder="例如：請假流程要找誰簽核？"></textarea>
    <button type="submit" id="go">送出</button>
  </form>
  <div class="answer" id="out" hidden></div>
  <footer>擁有者：%s ・ 映像檔標籤：%s</footer>
</main>
<script>
const form = document.getElementById('ask');
const out = document.getElementById('out');
const btn = document.getElementById('go');
form.addEventListener('submit', async (e) => {
  e.preventDefault();
  const question = document.getElementById('q').value.trim();
  if (!question) return;
  btn.disabled = true;
  out.hidden = false;
  out.textContent = '查詢中…';
  try {
    const res = await fetch('/api/ask', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ question })
    });
    if (!res.ok) throw new Error('伺服器回應 ' + res.status);
    const data = await res.json();
    out.textContent = (data.degraded ? '[降級回覆] ' : '') + data.answer;
  } catch (err) {
    out.textContent = '[降級回覆] 前端無法取得回答：' + err.message;
  } finally {
    btn.disabled = false;
  }
});
</script>
</body>
</html>
`
