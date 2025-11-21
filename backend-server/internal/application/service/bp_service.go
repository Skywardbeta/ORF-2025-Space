package service

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/gateway"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/repository"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/utils"
)

type BpService struct {
	bpgateway    gateway.BpGateway
	bprepository repository.BpRepository
}

func NewBpService(
	bpgateway gateway.BpGateway,
	bprepository repository.BpRepository,
) *BpService {
	s := &BpService{
		bpgateway:    bpgateway,
		bprepository: bprepository,
	}
	s.StartPushListener()
	return s
}

// StartPushListener Push受信したレスポンスを監視してキャッシュに保存する
func (bs *BpService) StartPushListener() {
	go func() {
		ch := bs.bpgateway.GetUnsolicitedResponseCh()
		for resp := range ch {
			log.Printf("[BpService] Push受信: ステータス=%d", resp.StatusCode)

			// URLを復元するロジックが必要
			// 現在の実装では、レスポンス自体にURLが含まれていないため、
			// リモート側がヘッダーに "X-Original-URL" を含めていることを期待する。
			// (ユーザー提供のコードでは bpRes.Headers["X-Original-URL"] = []string{targetURL} となっている)

			urls, ok := resp.Headers["X-Original-URL"]
			if !ok || len(urls) == 0 {
				log.Printf("[BpService] Push受信エラー: X-Original-URL ヘッダーがありません。キャッシュできません。")
				continue
			}
			originalURL := urls[0]

			// ダミーのBpRequestを作成（キャッシュキー生成用）
			req := &model.BpRequest{
				Method: "GET", // Pushは基本的にGETの結果と仮定
				URL:    originalURL,
			}

			// キャッシュに保存
			// TTLはデフォルト値を使用（configから取るべきだが、ここでは簡易的に24時間とする）
			// 理想的にはConfigをServiceに注入する
			ttl := 24 * time.Hour
			ctx := context.Background()

			err := bs.bprepository.SetResponseWithURL(ctx, req, resp, ttl)
			if err != nil {
				log.Printf("[BpService] Pushキャッシュ保存エラー: %v (URL=%s)", err, originalURL)
			} else {
				log.Printf("[BpService] Pushキャッシュ保存成功: URL=%s", originalURL)
			}
		}
	}()
}

// ProxyRequest HTTPリクエストを転送する（キャッシュ可能な場合はキャッシュもチェック）
func (bs *BpService) ProxyRequest(ctx context.Context, breq *model.BpRequest) (*model.BpResponse, error) {
	// キャッシュ不可の場合は直接転送
	if !breq.IsCacheable() {
		log.Printf("[BpService] リクエストはキャッシュ不可: Method=%s, URL=%s", breq.Method, breq.URL)
		return bs.bpgateway.ProxyRequest(ctx, breq)
	}

	log.Printf("[BpService] リクエストはキャッシュ可能: URL=%s", breq.URL)

	// キャッシュ可能な場合はキャッシュから取得
	cacheKey := breq.GenerateCacheKey()
	cachedResp, found, err := bs.bprepository.GetResponse(ctx, cacheKey)
	if err != nil {
		log.Printf("[BpService] キャッシュ取得エラー: %v", err)
		// キャッシュ取得エラー: Gateway層で直接転送
		return bs.bpgateway.ProxyRequest(ctx, breq)
	}

	if found {
		log.Printf("[BpService] キャッシュヒット: URL=%s", breq.URL)
		// キャッシュヒット: キャッシュされたレスポンスを返す
		return cachedResp, nil
	}

	// found == false の場合はキャッシュミス（エラーではない）

	log.Printf("[BpService] キャッシュミス: URL=%s, リクエストを予約します", breq.URL)

	// キャッシュミス: Worker Poolにリクエストを予約してデフォルトページを返す
	if bs.bprepository != nil {
		err := bs.bprepository.ReserveRequest(ctx, breq)
		if err != nil {
			log.Printf("[BpService] ReserveRequest エラー: %v", err)
		} else {
			log.Printf("[BpService] ReserveRequest 成功: URL=%s", breq.URL)
		}
	}

	// デフォルトページを読み込む
	htmlBytes, err := utils.LoadDefaultPage()
	if err != nil {
		// デフォルトページの読み込みに失敗した場合は503 Service Unavailableを返す
		// DTN環境では直接転送は期待できないため、フォールバックとしてエラーを返す
		log.Printf("[BpService] Failed to load default page: %v", err)
		body := []byte("503 Service Unavailable: Failed to load default page and direct proxy is unavailable in DTN environment.")
		return &model.BpResponse{
			StatusCode:    http.StatusServiceUnavailable,
			Headers:       make(map[string][]string),
			Body:          body,
			ContentType:   "text/plain; charset=utf-8",
			ContentLength: int64(len(body)),
		}, nil
	}

	return &model.BpResponse{
		StatusCode:    200,
		Headers:       breq.Headers,
		Body:          htmlBytes,
		ContentType:   "text/html; charset=utf-8",
		ContentLength: int64(len(htmlBytes)),
	}, nil
}
