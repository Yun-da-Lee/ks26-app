// ks26 內部知識問答服務。
// 所有設定一律由環境變數注入，程式裡不放任何位址、帳號或憑證。
// 沒有推論後端（INFERENCE_URL 未設定或連不上）時，服務降級運作而不是壞掉。
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// config 全部來自環境變數；每一項都有明寫的預設值。
type config struct {
	port         string
	appName      string
	appOwner     string
	imageTag     string
	inferenceURL string
	inferenceKey string
	timeout      time.Duration
	degradedMsg  string
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func loadConfig() config {
	timeoutSec, err := strconv.Atoi(env("INFERENCE_TIMEOUT_SECONDS", "5"))
	if err != nil || timeoutSec <= 0 {
		timeoutSec = 5
	}
	// 埠號一律由環境變數指定，deployment 的 containerPort 與 service 的
	// targetPort 都對著同一個值。
	port := env("PORT", "8080")
	return config{
		port:         port,
		appName:      env("APP_NAME", "ks26 內部知識問答"),
		appOwner:     env("APP_OWNER", "unassigned"),
		imageTag:     env("IMAGE_TAG", "unknown"),
		inferenceURL: env("INFERENCE_URL", ""),
		inferenceKey: env("INFERENCE_API_KEY", ""),
		timeout:      time.Duration(timeoutSec) * time.Second,
		degradedMsg: env("DEGRADED_MESSAGE",
			"目前沒有可用的推論後端，這是降級（degraded）回覆：問題已收到，但無法產生答案。"+
				"請改查內部知識庫，或稍後再試。服務本身仍然健康。"),
	}
}

type askRequest struct {
	Question string `json:"question"`
}

type askResponse struct {
	Answer   string `json:"answer"`
	Degraded bool   `json:"degraded"`
	Reason   string `json:"reason,omitempty"`
	Source   string `json:"source"`
}

type app struct {
	cfg    config
	client *http.Client
}

// ask 是唯一會碰到外部後端的地方。任何失敗都走降級路徑（fallback），
// 呼叫端永遠拿得到一個可以顯示的答案。
func (a *app) ask(ctx context.Context, question string) askResponse {
	if a.cfg.inferenceURL == "" {
		return askResponse{
			Answer:   a.cfg.degradedMsg,
			Degraded: true,
			Reason:   "INFERENCE_URL 未設定",
			Source:   "fallback",
		}
	}
	answer, err := a.callInference(ctx, question)
	if err != nil {
		log.Printf("inference call failed, serving degraded answer: %v", err)
		return askResponse{
			Answer:   a.cfg.degradedMsg,
			Degraded: true,
			Reason:   "推論後端無法連線：" + err.Error(),
			Source:   "fallback",
		}
	}
	return askResponse{Answer: answer, Degraded: false, Source: "inference"}
}

func (a *app) callInference(ctx context.Context, question string) (string, error) {
	body, err := json.Marshal(map[string]string{"question": question, "prompt": question})
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, a.cfg.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.inferenceURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.cfg.inferenceKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.cfg.inferenceKey)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("後端回應 %d", resp.StatusCode)
	}

	var decoded struct {
		Answer string `json:"answer"`
		Text   string `json:"text"`
	}
	if err := json.Unmarshal(raw, &decoded); err == nil {
		if decoded.Answer != "" {
			return decoded.Answer, nil
		}
		if decoded.Text != "" {
			return decoded.Text, nil
		}
	}
	if trimmed := strings.TrimSpace(string(raw)); trimmed != "" {
		return trimmed, nil
	}
	return "", errors.New("後端回了空內容")
}

// healthz 是平台判斷存活的依據：不管有沒有推論後端，一律回 200。
func (a *app) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"degraded": a.cfg.inferenceURL == "",
		"tag":      a.cfg.imageTag,
	})
}

func (a *app) apiAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "只接受 POST", http.StatusMethodNotAllowed)
		return
	}
	var in askRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		http.Error(w, "請求格式不正確", http.StatusBadRequest)
		return
	}
	in.Question = strings.TrimSpace(in.Question)
	if in.Question == "" {
		http.Error(w, "問題不能是空的", http.StatusBadRequest)
		return
	}
	out := a.ask(r.Context(), in.Question)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func (a *app) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	banner := ""
	if a.cfg.inferenceURL == "" {
		banner = `<p class="degraded">降級模式（degraded）：未設定推論後端，回答為罐頭訊息。</p>`
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	name := htmlEscape(a.cfg.appName)
	fmt.Fprintf(w, indexHTML, name, name, banner,
		htmlEscape(a.cfg.appOwner), htmlEscape(a.cfg.imageTag))
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func main() {
	cfg := loadConfig()
	a := &app{cfg: cfg, client: &http.Client{Timeout: cfg.timeout}}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", a.healthz)
	mux.HandleFunc("/api/ask", a.apiAsk)
	mux.HandleFunc("/", a.index)

	addr := net.JoinHostPort("", cfg.port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("listening on %s (owner=%s tag=%s degraded=%t)",
		addr, cfg.appOwner, cfg.imageTag, cfg.inferenceURL == "")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}
