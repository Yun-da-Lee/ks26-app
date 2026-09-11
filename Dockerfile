# 多階段建置：builder 釘在建置機器的原生架構（BUILDPLATFORM），
# 用 Go 的交叉編譯產生目標架構的靜態執行檔，不啟動 QEMU 模擬器。
FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder

ARG TARGETOS=linux
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /src
COPY app/go.mod ./
RUN go mod download
COPY app/ ./
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/server .

# 最終階段只有一個靜態執行檔與 CA 憑證，沒有 shell、沒有套件管理器、沒有編譯器。
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/server /server

# 非 root。實際的 uid 由 deployment 的 securityContext 再明寫一次。
USER 65532:65532

# 只聽一個 HTTP 埠；真正的埠號由 PORT 環境變數決定，這裡只是宣告預設值。
EXPOSE 8080

ENTRYPOINT ["/server"]
