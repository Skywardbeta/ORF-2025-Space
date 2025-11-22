package worker

import (
	"context"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

// RequestHandler リクエストを処理するハンドラー（プラグイン可能）
type RequestHandler interface {
	// HandleRequest リクエストを処理する
	HandleRequest(ctx context.Context, req *model.BpRequest, workerID int) error
}

// QueueWatcher キューを監視してジョブを取得する（プラグイン可能）
type QueueWatcher interface {
	// WatchQueue キューを監視してジョブを取得する
	WatchQueue(ctx context.Context) (*model.BpRequest, error)
}

// CacheHandler キャッシュ操作を行うハンドラー
type CacheHandler interface {
	// DeleteExpiredCaches 期限切れのキャッシュを削除する
	DeleteExpiredCaches(ctx context.Context) error

	// DeleteAllCaches すべてのキャッシュを削除する
	DeleteAllCaches(ctx context.Context) error
}

// ResponseWatcher Unsolicited Responseを監視するワーカー
type ResponseWatcher interface {
	Start(ctx context.Context)
}
