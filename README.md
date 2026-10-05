# Bedrock Bot

這個範例使用 [patyhank/bedrock-library](https://github.com/patyhank/bedrock-library) 建立一個可登入的基岩版 Bot。

## 1. 安裝 Go

請先安裝 Go 1.24 或以上：

- https://go.dev/dl/

## 2. 設定帳號

編輯 `accounts.txt`，格式如下：

```txt
bot1 your_microsoft_email@example.com:your_password
```

這裡的 `bot1` 是帳號別名，之後會透過 `BEDROCK_ACCOUNT_NAME=bot1` 讀取。

## 3. 啟動 Bot

```bash
go mod tidy
go run .
```

也可以覆寫連線伺服器：

```bash
BEDROCK_SERVER=play.example.com:19132 BEDROCK_ACCOUNT_NAME=bot1 go run .
```

## 4. 部署到 Render

Bot 啟動時會同時啟動 HTTP health endpoint，監聽 Render 提供的 `PORT` 環境變數（本機預設 `3000`）。對 `/` 發送 GET 請求會回應 `MC Bot is Alive!`。

1. 在 Render 建立 **Web Service**，連結此程式碼儲存庫，Instance Type 選 **Free**。
2. Build Command 設為 `go build -o bedrock-bot .`，Start Command 設為 `./bedrock-bot`。
3. 在 Environment 加入 `BEDROCK_SERVER`，值為基岩版伺服器的 `host:port`。
4. 若使用已登入的 Microsoft token，透過 Render Secret Files 提供 `token.txt`，並設定 `BEDROCK_TOKEN_FILE` 為該 Secret File 的路徑。不要把帳號密碼或 token 提交到程式碼儲存庫。Render Free 的檔案系統不適合保存執行期間產生的 token；重新部署或重啟後，請確認 token 檔仍可讀取。
5. 部署完成後，在 UptimeRobot 建立 HTTP(s) monitor，監控 Render 提供的 `https://<service-name>.onrender.com/` 網址，檢查間隔設為 **5 分鐘**。

UptimeRobot 的請求會定期存取 health endpoint，有助於避免因閒置而休眠；但 Free 方案的可用性與休眠行為仍受 Render 當前方案限制，不能保證服務永遠在線。

## 5. 常見問題

- 若 `auth.ReadAccount()` 失敗，請確認 `accounts.txt` 格式正確。
- 若登入失敗，請確認 Microsoft 帳號與密碼正確，並確認伺服器支援基岩版登入。
- 若連線錯誤，請確認 server address 是 `host:port` 格式。

## 6. 參考來源

- https://github.com/patyhank/bedrock-library
- https://github.com/patyhank/bedrock-library/blob/master/examples/simple/main.go
