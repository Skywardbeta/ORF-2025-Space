package repository

import (
	"context"
	"time"
)

// CacheItem キャッシュのキーとファイルパス（infrastructure層の型）
// domain層のmodel.CacheItemとは別の型として定義
type CacheItem struct {
	Key      string // Redisキー（メタデータキー）
	FilePath string // ファイルシステム上のファイルパス
}

type BpRepoClient interface {
	GetMetaData(ctx context.Context, cachekey string) ([]byte, error)
	ScanExpiredCacheKeys(ctx context.Context) ([]CacheItem, error)
	SetMetaData(ctx context.Context, key string, data []byte, ttl time.Duration) error
	DeleteCache(ctx context.Context, cachekey string, filePath string) error
	ReserveRequest(ctx context.Context, job []byte) error
	GetReservedRequests(ctx context.Context) ([][]byte, error)
	RemoveReservedRequest(ctx context.Context, job []byte) error
	BLPopReservedRequest(ctx context.Context, timeout time.Duration) ([]byte, error)
}
