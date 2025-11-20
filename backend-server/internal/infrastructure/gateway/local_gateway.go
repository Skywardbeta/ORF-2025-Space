package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

type LocalGateway struct {
	client *http.Client
}

func NewLocalGateway(timeout time.Duration) *LocalGateway {
	return &LocalGateway{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (g *LocalGateway) ProxyRequest(ctx context.Context, breq *model.BpRequest) (*model.BpResponse, error) {
	// breq.URL（クエリパラメータのurl）に直接HTTPリクエストを送る
	targetURL := breq.URL

	// HTTPリクエストを作成（contextを設定）
	httpReq, err := http.NewRequestWithContext(ctx, breq.Method, targetURL, bytes.NewReader(breq.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	breq.SetHeaders(httpReq)

	// HTTPリクエストを送信
	httpResp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to forward HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	// レスポンスボディを読み込む
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// BpResponseを作成
	return &model.BpResponse{
		StatusCode:    httpResp.StatusCode,
		Headers:       httpResp.Header,
		Body:          bodyBytes,
		ContentType:   httpResp.Header.Get("Content-Type"),
		ContentLength: httpResp.ContentLength,
	}, nil
}
