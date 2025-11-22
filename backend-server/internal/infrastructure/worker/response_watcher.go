package worker

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/gateway"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/repository"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

type ResponseWatcher struct {
	bpgateway gateway.BpGateway
	bprepo    repository.BpRepository
}

func NewResponseWatcher(
	bpgateway gateway.BpGateway,
	bprepo repository.BpRepository,
) *ResponseWatcher {
	return &ResponseWatcher{
		bpgateway: bpgateway,
		bprepo:    bprepo,
	}
}

// Start Unsolicited Response (タイムアウト後に届いたレスポンス) を監視する
func (rw *ResponseWatcher) Start(ctx context.Context) {
	log.Printf("[ResponseWatcher] 監視を開始しました")
	defer log.Printf("[ResponseWatcher] 監視を終了しました")

	ch := rw.bpgateway.GetUnsolicitedResponseCh()

	for {
		select {
		case <-ctx.Done():
			return
		case resp := <-ch:
			if resp == nil {
				continue
			}
			rw.handleResponse(ctx, resp)
		}
	}
}

func (rw *ResponseWatcher) handleResponse(ctx context.Context, resp *model.BpResponse) {
	// X-Original-URL ヘッダーからURLを取得
	urls, ok := resp.Headers["X-Original-URL"]
	if !ok || len(urls) == 0 {
		log.Printf("[ResponseWatcher] X-Original-URL ヘッダーが見つかりません (Status: %d)", resp.StatusCode)
		return
	}
	url := urls[0]

	// エラーレスポンスはキャッシュしない
	if resp.StatusCode != 200 {
		log.Printf("[ResponseWatcher] エラーレスポンスのためキャッシュしません (URL: %s, Status: %d)", url, resp.StatusCode)
		// Pending状態だけ解除しておく
		_ = rw.bprepo.RemovePendingRequest(ctx, url)
		return
	}

	log.Printf("[ResponseWatcher] Unsolicited Responseを受信しました: %s", url)

	// キャッシュに保存
	// Requestオブジェクトを再構築（キャッシュパス生成のため）
	req := &model.BpRequest{
		URL:    url,
		Method: "GET", // 仮定
	}

	// URLが http/https で始まっていない場合は補完（念のため）
	if !strings.HasPrefix(url, "http") {
		// ログだけ出してそのまま処理（GenerateCachePathInfoでエラーになるかも）
		log.Printf("[ResponseWatcher] URLの形式が不正です: %s", url)
	}

	// キャッシュ保存（内部でRemovePendingRequestも呼ばれる）
	// TTLはデフォルト値を使用したいが、ここではハードコードするか、設定から渡す必要がある
	// 簡易的に24時間とする（またはConfigから渡すように修正する）
	// TODO: TTLをConfigから注入する
	ttl := 24 * 60 * 60 * time.Second // 24h

	err := rw.bprepo.SetResponseWithURL(ctx, req, resp, ttl)
	if err != nil {
		log.Printf("[ResponseWatcher] キャッシュ保存エラー (URL: %s): %v", url, err)
		// 失敗してもPendingは解除する（SetResponseWithURL内で呼ばれているはずだが、念のため）
		_ = rw.bprepo.RemovePendingRequest(ctx, url)
	} else {
		log.Printf("[ResponseWatcher] キャッシュを保存しました (URL: %s)", url)
	}
}
