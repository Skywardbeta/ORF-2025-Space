package repository

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

type BpRepository struct {
	rclient *redis.Client
}

func NewBpRepository(rclient *redis.Client) *BpRepository {
	return &BpRepository{
		rclient: rclient,
	}
}

// GetResponse キャッシュからレスポンスを取得
func (br *BpRepository) GetResponse(ctx context.Context, cacheKey string) (*model.BpResponse, bool, error) {
	cachedData, err := br.rclient.Get(ctx, cacheKey).Bytes()
	if err == redis.Nil {
		// キャッシュミス
		return nil, false, nil
	} else if err != nil {
		// その他のエラー
		return nil, false, err
	}

	// キャッシュヒット: キャッシュされたデータをBpResponseにデコード
	var cachedResp model.BpResponse
	err = json.Unmarshal(cachedData, &cachedResp)
	if err != nil {
		return nil, false, err
	}

	return &cachedResp, true, nil
}
