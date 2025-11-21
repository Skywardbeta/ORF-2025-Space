package worker

import (
	"context"
	"log"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/repository"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

type QueueWatcher struct {
	bprepo  repository.BpRepository
	timeout time.Duration // BLPopのタイムアウト時間(監視時間)
}

func NewQueueWatcher(
	bprepo repository.BpRepository,
	timeout time.Duration,
) *QueueWatcher {
	return &QueueWatcher{
		bprepo:  bprepo,
		timeout: timeout,
	}
}

// WatchQueue キューを監視してジョブを取得する
func (qw *QueueWatcher) WatchQueue(ctx context.Context) (*model.BpRequest, error) {
	req, err := qw.bprepo.BLPopReservedRequest(ctx, qw.timeout)
	if req == nil || err != nil {
		log.Printf("[QueueWatcher] Redisからジョブの取得に失敗: %v", err)

		return nil, err
	}

	log.Printf("[QueueWatcher] Redisからジョブを取得: %s", req.URL)

	return req, nil
}
