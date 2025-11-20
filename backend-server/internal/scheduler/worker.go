package scheduler

import (
	"context"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

// RequestHandler リクエストを処理するハンドラー（プラグイン可能）
type RequestHandler interface {
	// HandleRequest リクエストを処理する
	// ctx: コンテキスト
	// req: 処理するリクエスト
	// workerID: ワーカーのID（ログ用）
	HandleRequest(ctx context.Context, req *model.BpRequest, workerID int) error
}

// QueueWatcher キューを監視してジョブを取得する（プラグイン可能）
type QueueWatcher interface {
	// WatchQueue キューを監視してジョブを取得する
	// ctx: コンテキスト
	// 戻り値: 取得したリクエスト（タイムアウトの場合はnil）とエラー
	WatchQueue(ctx context.Context) (*model.BpRequest, error)
}

type CacheHandler interface {
	// DeleteExpiredCaches 期限切れのキャッシュを削除する
	DeleteExpiredCaches(ctx context.Context) error
}
