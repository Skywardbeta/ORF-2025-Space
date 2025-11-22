package repository

import (
	"context"
	"time"
)

type CacheItem struct {
	Key      string
	FilePath string
}

type BpRepoClient interface {
	GetMetaData(ctx context.Context, metaKey string) ([]byte, error)
	ScanExpiredKeys(ctx context.Context) ([]CacheItem, error)
	SetMetaData(ctx context.Context, metaKey string, data []byte, ttl time.Duration) error
	DeleteMetaData(ctx context.Context, metaKey string) error
	FlushAllMetaData(ctx context.Context) error
	ReserveRequest(ctx context.Context, job []byte) error
	GetReservedRequests(ctx context.Context) ([][]byte, error)
	RemoveReservedRequest(ctx context.Context, job []byte) error
	BLPopReservedRequest(ctx context.Context, timeout time.Duration) ([]byte, error)
	AddPendingRequest(ctx context.Context, url string) (bool, error)
	RemovePendingRequest(ctx context.Context, url string) error
	FlushAllReservedRequest(ctx context.Context) error
	FlushAllCaches(ctx context.Context) error
}
