package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

type RequestProcessor struct {
	workers         int
	jobQueue        chan *model.BpRequest
	reqhandler      RequestHandler
	queueWatcher    QueueWatcher
	cacheHandler    CacheHandler
	cleanupInterval time.Duration // 追加
}

func NewRequestProcessor(
	workers int,
	reqhandler RequestHandler,
	queueWatcher QueueWatcher,
	cacheHandler CacheHandler,
	cleanupInterval time.Duration, // 追加
) *RequestProcessor {
	return &RequestProcessor{
		workers:         workers,
		jobQueue:        make(chan *model.BpRequest, workers*2),
		reqhandler:      reqhandler,
		queueWatcher:    queueWatcher,
		cacheHandler:    cacheHandler,
		cleanupInterval: cleanupInterval, // 追加
	}
}

func (rp *RequestProcessor) Start(ctx context.Context) {
	// 1. Worker Poolを起動(リクエスト処理)
	log.Printf("[RequestProcessor] Worker Poolを起動します (workers: %d)", rp.workers)
	for i := 0; i < rp.workers; i++ {
		go rp.worker(ctx, i)
	}
	// 2. Redisキュー監視を起動
	go rp.watchQueue(ctx)
	log.Printf("[RequestProcessor] Worker Poolを起動しました")

	// 3. キャッシュクリーンアップcronを起動
	go rp.startCacheCleanup(ctx)
	log.Printf("[RequestProcessor] キャッシュクリーンアップを起動しました")
}

func (rp *RequestProcessor) worker(ctx context.Context, id int) {
	log.Printf("[Worker %d] 起動しました", id)
	defer log.Printf("[Worker %d] 終了しました", id)

	for req := range rp.jobQueue {
		// プラグイン可能なハンドラーを使用
		if err := rp.reqhandler.HandleRequest(ctx, req, id); err != nil {
			log.Printf("[Worker %d] リクエスト処理エラー (URL: %s): %v", id, req.URL, err)
		}
	}
}

func (rp *RequestProcessor) watchQueue(ctx context.Context) {
	log.Printf("[Queue Watcher] Redisキュー監視を開始しました")
	defer log.Printf("[Queue Watcher] Redisキュー監視を終了しました")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// プラグイン可能なキュー監視を使用
			req, err := rp.queueWatcher.WatchQueue(ctx) // timeoutはQueueWatcher内部で管理
			if err != nil {
				log.Printf("[Queue Watcher] キュー取得エラー: %v", err)
				// エラー時は少し待機して続行
				time.Sleep(1 * time.Second)
				continue
			}

			// リクエストが取得できた場合、ジョブキューに投入
			if req != nil {
				select {
				case rp.jobQueue <- req:
					log.Printf("[Queue Watcher] リクエストをキューに投入: %s", req.URL)
				case <-ctx.Done():
					return
				}
			}
			// req == nil の場合はタイムアウトなので、ループを継続
		}
	}
}

func (rp *RequestProcessor) startCacheCleanup(ctx context.Context) {
	log.Printf("[Cache Cleanup] キャッシュクリーンアップを開始しました")
	defer log.Printf("[Cache Cleanup] キャッシュクリーンアップを終了しました")

	ticker := time.NewTicker(rp.cleanupInterval) // 設定値を使用
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// プラグイン可能なキャッシュハンドラーを使用
			if err := rp.cacheHandler.DeleteExpiredCaches(ctx); err != nil {
				log.Printf("[Cache Cleanup] 期限切れキャッシュ削除エラー: %v", err)
			} else {
				log.Printf("[Cache Cleanup] 期限切れキャッシュを削除しました")
			}
		}
	}
}
