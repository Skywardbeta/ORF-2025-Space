package worker

import (
	"context"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/repository"
)

type CacheHandler struct {
	bprepo repository.BpRepository
}

func NewCacheHandler(bprepo repository.BpRepository) *CacheHandler {
	return &CacheHandler{
		bprepo: bprepo,
	}
}

// DeleteExpiredCaches 期限切れのキャッシュを削除する
func (ch *CacheHandler) DeleteExpiredCaches(ctx context.Context) error {
	return ch.bprepo.DeleteExpiredCaches(ctx)
}
