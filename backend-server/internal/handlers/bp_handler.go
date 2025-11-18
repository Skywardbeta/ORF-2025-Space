package handlers

import (
	"io"
	"net/http"
	"net/url"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/service"
)

type bpHandler struct {
	bpService *service.BpService
}

func NewBpHandler(bpService *service.BpService) *bpHandler {
	return &bpHandler{
		bpService: bpService,
	}
}

func (bh *bpHandler) GetContent(w http.ResponseWriter, r *http.Request) {
	// 転送されてくるHTTPリクエストを処理（GET、POST、PUT、DELETE、PATCHなどすべてのメソッドに対応）
	// 転送先URLをクエリパラメータから取得
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, "url parameter is required", http.StatusBadRequest)
		return
	}

	// URLの検証
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// リクエストボディを読み込む
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		r.Body.Close()
	}

	breq := model.BpRequest{
		Method:        r.Method,
		URL:           parsedURL.String(),
		Headers:       r.Header,
		Body:          bodyBytes,
		ContentType:   r.Header.Get("Content-Type"),
		ContentLength: r.ContentLength,
	}

	// Service層でリクエストを転送（キャッシュ可能な場合はキャッシュもチェック）
	resp, err := bh.bpService.ProxyRequest(&breq)
	if err != nil {
		http.Error(w, "Failed to proxy request", http.StatusBadGateway)
		return
	}

	// レスポンスヘッダーをコピー
	for key, values := range resp.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// ステータスコードを設定
	w.WriteHeader(resp.StatusCode)

	// レスポンスボディをコピー
	_, err = io.Copy(w, resp.GetBodyReader())
	if err != nil {
		http.Error(w, "Failed to copy response body", http.StatusInternalServerError)
		return
	}
}
