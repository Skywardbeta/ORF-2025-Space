package worker

import (
	"context"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/repository"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

type QueueWatcher struct {
	bprepo       repository.BpRepository
	timeout      time.Duration // 追加
}

func NewQueueWatcher(
	bprepo repository.BpRepository,
	timeout time.Duration, // 追加
) *QueueWatcher {
	return &QueueWatcher{
		bprepo:       bprepo,
		timeout:      timeout,      // 追加
	}
}

// WatchQueue キューを監視してジョブを取得する
func (qw *QueueWatcher) WatchQueue(ctx context.Context) (*model.BpRequest, error) {
	// timeout パラメータは使わず、qw.timeout を使用するように変更
	req, err := qw.bprepo.BLPopReservedRequest(ctx, qw.timeout)
	if err != nil {
		return nil, err
	}
	return req, nil
}
