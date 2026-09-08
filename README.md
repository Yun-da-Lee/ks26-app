# ks26-app

KubeSummit 2026 工作坊用的骨架。**這裡只有契約，沒有程式。**

| 檔案 | 誰讀 | 內容 |
|---|---|---|
| `REQUEST.md` | agent | 那「一句需求」 |
| `hard-constraints.md` | 人 | 十一條硬性限制，每條對應一個檢查代號 |
| `AGENTS.md` | agent | 產出規則與交件條件（模型無關） |
| `verify.sh` | 機器 | 二十四項檢查。全過才算交件 |

## 當天你這一組要做的

1. **Fork 這個 repo**——右上角 Fork → Create fork。
   **「Copy the DEFAULT branch only」不要勾**，漏了後面沒東西可審。
2. 掃 QR，把你們 fork 的網址填進表單。
3. 在自己的 fork 開 PR：`main` ← `ai-output`。確認 base repository 是**你們自己的帳號**。
4. 拿 `hard-constraints.md` 逐條對 diff，組內決定誰署名，按 Merge。
5. 到建置看板複製映像檔標籤，填進 `deploy/deployment.yaml` 的 `image` 與 `IMAGE_TAG`，
   `deploy/ingress.yaml` 的 host 改成自己的組號，commit 到 `main`。
6. 在 Rancher 建 GitRepo 指向自己的 fork（`Continuous Delivery → Resources → Git Repos →
   Add Repository`；**`Branch Name` 預設是 `master`，要改成 `main`**），等 Fleet 收斂，
   開網址對**頁尾**那行的映像檔標籤。

## 分支

- `main`：契約四件。這是 fork 的起點。
- `ai-output`：依這份契約產生、`verify.sh` 24/24 的產出物。第 3 步要審的就是它。

## 自己跑驗證

```bash
./verify.sh              # 對當前目錄
./verify.sh <某個目錄>    # 對指定目錄
```
本機有 `yq` 就直接用，沒有的話它會自動改用容器跑同一套檢查
（`docker` / `podman` / `nerdctl` 擇一）。全過回傳 `0`，有任何一條沒過回傳 `1`。
