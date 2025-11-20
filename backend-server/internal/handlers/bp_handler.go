package handlers

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
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

func (bh *bpHandler) GetContent(c *gin.Context) {
	r := c.Request
	w := c.Writer

	// デバッグログ: リクエストの詳細を出力
	log.Printf("[BpHandler] Received request: Method=%s, Path=%s, Query=%s, Host=%s",
		r.Method, r.URL.Path, r.URL.RawQuery, r.Host)

	// CONNECTメソッドの場合は特別な処理が必要
	if r.Method == http.MethodConnect {
		log.Printf("[BpHandler] Processing CONNECT method")
		bh.handleCONNECT(c)
		return
	}

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

	log.Printf("[BpHandler] Received request: Method=%s, URL=%s", breq.Method, breq.URL)

	// Service層でリクエストを転送（キャッシュ可能な場合はキャッシュもチェック）
	// リクエストのcontextを取得して伝播（キャンセレーションやタイムアウト制御のため）
	ctx := r.Context()
	resp, err := bh.bpService.ProxyRequest(ctx, &breq)
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

// handleCONNECT CONNECTメソッドのリクエストを処理（HTTPトンネリング）
func (bh *bpHandler) handleCONNECT(c *gin.Context) {
	r := c.Request
	w := c.Writer

	// CONNECTメソッドの場合、リクエストのパスが "host:port" 形式
	target := r.URL.Host
	if target == "" {
		target = r.URL.Path
	}

	// "host:port"形式をパース
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		// ポートが指定されていない場合はデフォルトで443（HTTPS）
		host = target
		port = "443"
	}

	// Hijackして双方向のストリーム転送を開始
	// 注意: Hijackする前にヘッダーを書き込んではいけない
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// 転送先に接続
	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
	}
	destConn, err := dialer.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		// エラーレスポンスを送信
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer destConn.Close()

	// CONNECTメソッドのレスポンスを返す（200 Connection Established）
	// Hijack後は直接TCP接続に書き込む
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// 双方向のストリーム転送
	go func() {
		defer destConn.Close()
		defer clientConn.Close()
		io.Copy(destConn, clientConn)
	}()

	io.Copy(clientConn, destConn)
}
