package plugins

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/infrastructure/repository"
)

type RedisClientConfig struct {
	ReservedRequestsKey string
	CacheMetaPattern    string
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

func (rc *RedisClient) GetMetaData(ctx context.Context, key string) ([]byte, error) {
	metaData, err := rc.rclient.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return metaData, nil
}

func (rc *RedisClient) ScanExpiredCacheKeys(ctx context.Context) ([]repository.CacheItem, error) {
	// ページネーションを使用してキーをスキャン
	var cursor uint64
	var expiredItems []repository.CacheItem
	pattern := rc.config.CacheMetaPattern

	for {
		keys, nextCursor, err := rc.rclient.Scan(ctx, cursor, pattern, 100).Result()
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

func (rc *RedisClient) SetMetaData(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	err := rc.rclient.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return err
	}
	return nil
}

func (rc *RedisClient) DeleteCache(ctx context.Context, metaKey string, filePath string) error {
	// Redisからメタデータを削除
	if err := rc.rclient.Del(ctx, metaKey).Err(); err != nil {
		return err
	}

	// ファイルシステムからキャッシュファイルを削除
	// ファイルが存在しない場合はエラーを返さない（既に削除済みとみなす）
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
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

func (rc *RedisClient) FlushAll(ctx context.Context) error {
	return rc.rclient.FlushDB(ctx).Err()
}
