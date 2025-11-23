package plugins

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/infrastructure/repository"
)

type RedisClientConfig struct {
	ReservedRequestsKey string
	PendingRequestsKey  string // 追加
	CacheMetaPattern    string
	ScanCount           int
}

type RedisClient struct {
	rclient *redis.Client
	config  RedisClientConfig
}

func NewRedisClient(rclient *redis.Client, config RedisClientConfig) *RedisClient {
	return &RedisClient{
		rclient: rclient,
		config:  config,
	}
}

func (rc *RedisClient) GetMetaData(ctx context.Context, metaKey string) ([]byte, error) {
	metaData, err := rc.rclient.Get(ctx, metaKey).Bytes()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return metaData, nil
}

func (rc *RedisClient) ScanExpiredKeys(ctx context.Context) ([]repository.CacheItem, error) {
	// ページネーションを使用してキーをスキャン
	var cursor uint64
	var expiredItems []repository.CacheItem
	pattern := rc.config.CacheMetaPattern

	// ScanCountが0の場合はデフォルト値100を使用
	scanCount := rc.config.ScanCount
	if scanCount == 0 {
		scanCount = 100
	}

	for {
		keys, nextCursor, err := rc.rclient.Scan(ctx, cursor, pattern, int64(scanCount)).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			ttl, err := rc.rclient.TTL(ctx, key).Result()
			if err != nil {
				continue
			}

			if ttl <= 0 {
				// メタデータを取得してJSONデコードしてファイルパスを抽出
				metaData, err := rc.rclient.Get(ctx, key).Bytes()
				if err != nil {
					// メタデータが取得できない場合はキーだけ追加
					expiredItems = append(expiredItems, repository.CacheItem{
						Key:      key,
						FilePath: "",
					})
					continue
				}

				var filePath string
				var metadata model.CacheMetadata
				if err := json.Unmarshal(metaData, &metadata); err == nil {
					filePath = metadata.FilePath
				}

				expiredItems = append(expiredItems, repository.CacheItem{
					Key:      key,
					FilePath: filePath,
				})
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return expiredItems, nil
}

func (rc *RedisClient) SetMetaData(ctx context.Context, metaKey string, data []byte, ttl time.Duration) error {
	err := rc.rclient.Set(ctx, metaKey, data, ttl).Err()
	if err != nil {
		return err
	}
	return nil
}

func (rc *RedisClient) DeleteMetaData(ctx context.Context, metaKey string) error {
	// Redisからメタデータを削除
	if err := rc.rclient.Del(ctx, metaKey).Err(); err != nil {
		return err
	}
	return nil
}

func (rc *RedisClient) FlushAllMetaData(ctx context.Context) error {
	// 1. Redis上の関連キーを削除
	// メタデータをスキャンして削除
	var cursor uint64
	pattern := rc.config.CacheMetaPattern

	// ScanCountが0の場合はデフォルト値100を使用
	scanCount := rc.config.ScanCount
	if scanCount == 0 {
		scanCount = 100
	}

	for {
		keys, nextCursor, err := rc.rclient.Scan(ctx, cursor, pattern, int64(scanCount)).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := rc.rclient.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

func (rc *RedisClient) GetReservedRequests(ctx context.Context) ([][]byte, error) {
	key := rc.config.ReservedRequestsKey

	// Listの全要素を取得
	dataList, err := rc.rclient.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	// 生のバイトデータのリストを返す（JSONデコードはrepository層で行う）
	result := make([][]byte, 0, len(dataList))
	for _, data := range dataList {
		result = append(result, []byte(data))
	}

	return result, nil
}

func (rc *RedisClient) ReserveRequest(ctx context.Context, job []byte) error {
	jobKey := rc.config.ReservedRequestsKey
	err := rc.rclient.LPush(ctx, jobKey, job).Err()
	if err != nil {
		return err
	}

	return nil
}

func (rc *RedisClient) BLPopReservedRequest(ctx context.Context, timeout time.Duration) ([]byte, error) {
	key := rc.config.ReservedRequestsKey

	// BLPOPでブロッキング取得（タイムアウト付き）
	result, err := rc.rclient.BLPop(ctx, timeout, key).Result()
	if err != nil {
		if err == redis.Nil {
			// タイムアウト
			return nil, nil
		}
		return nil, err
	}

	// result[0]はキー名、result[1]は値
	if len(result) < 2 {
		return nil, nil
	}

	// 生のバイトデータを返す（JSONデコードはrepository層で行う）
	return []byte(result[1]), nil
}

func (rc *RedisClient) RemoveReservedRequest(ctx context.Context, job []byte) error {
	// Listから該当する要素を削除
	key := rc.config.ReservedRequestsKey
	err := rc.rclient.LRem(ctx, key, 1, job).Err()
	if err != nil {
		return err
	}

	return nil
}

func (rc *RedisClient) FlushAllReservedRequest(ctx context.Context) error {
	// 予約済みリクエストのキューを削除
	err := rc.rclient.Del(ctx, rc.config.ReservedRequestsKey).Err()
	if err != nil {
		return err
	}
	return nil
}

func (rc *RedisClient) FlushAllCaches(ctx context.Context) error {
	// Redis上の関連キーをすべて削除
	err := rc.FlushAllMetaData(ctx)
	if err != nil {
		return err
	}

	err = rc.FlushAllReservedRequest(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (rc *RedisClient) AddPendingRequest(ctx context.Context, url string) (bool, error) {
	key := rc.config.PendingRequestsKey
	// SAdd returns the number of elements added. If 1, it's new. If 0, it already existed.
	added, err := rc.rclient.SAdd(ctx, key, url).Result()
	if err != nil {
		return false, err
	}
	return added > 0, nil
}

func (rc *RedisClient) RemovePendingRequest(ctx context.Context, url string) error {
	key := rc.config.PendingRequestsKey
	return rc.rclient.SRem(ctx, key, url).Err()
}
