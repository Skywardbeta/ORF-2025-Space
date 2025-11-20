package worker

import (
	"context"
	"log"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/gateway"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/repository"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

type RequestHandler struct {
	bprepo     repository.BpRepository
	bpgateway  gateway.BpGateway
	defaultTTL time.Duration // 追加
}

func NewRequestHandler(
	bprepo repository.BpRepository,
	bpgateway gateway.BpGateway,
	defaultTTL time.Duration, // 追加
) *RequestHandler {
	return &RequestHandler{
		bprepo:     bprepo,
		bpgateway:  bpgateway,
		defaultTTL: defaultTTL, // 追加
	}
}

// HandleRequest 予約されたリクエストを処理してキャッシュに保存
func (rh *RequestHandler) HandleRequest(ctx context.Context, req *model.BpRequest, workerID int) error {
	log.Printf("[Worker %d] リクエスト処理開始: %s", workerID, req.URL)

	// Gatewayでリクエストを転送
	resp, err := rh.bpgateway.ProxyRequest(ctx, req)
	if err != nil {
		log.Printf("[Worker %d] リクエストの転送に失敗 (URL: %s): %v", workerID, req.URL, err)
		// エラーが発生しても予約は削除（次回再試行）
		_ = rh.bprepo.RemoveReservedRequest(ctx, req)
		return err
	}

	// レスポンスをキャッシュに保存（URLベースの階層構造で保存）
	ttl := rh.defaultTTL // 設定値を使用
	// SetResponseWithURLを使用してURLベースの階層構造でキャッシュを保存
	err = rh.bprepo.SetResponseWithURL(ctx, req, resp, ttl)
	if err != nil {
		log.Printf("[Worker %d] キャッシュの保存に失敗 (URL: %s): %v", workerID, req.URL, err)
		// キャッシュ保存に失敗しても予約は削除
		_ = rh.bprepo.RemoveReservedRequest(ctx, req)
		return err
	}

	// 予約を削除
	err = rh.bprepo.RemoveReservedRequest(ctx, req)
	if err != nil {
		log.Printf("[Worker %d] 予約の削除に失敗 (URL: %s): %v", workerID, req.URL, err)
	}

	log.Printf("[Worker %d] リクエスト処理完了: %s", workerID, req.URL)
	return nil
}
