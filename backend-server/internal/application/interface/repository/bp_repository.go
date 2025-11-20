package repository

import (
	"context"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

// BpRepository Redisキャッシュを操作するためのリポジトリインターフェース
type BpRepository interface {
	// GetResponse キャッシュからレスポンスを取得する
	// ctx: コンテキスト（リクエストのキャンセレーションやタイムアウト制御に使用）
	// key: キャッシュキー（通常はURLのPath）
	// 戻り値: キャッシュされたレスポンスと、キャッシュが存在するかどうか
	GetResponse(ctx context.Context, key string) (*model.BpResponse, bool, error)

	// SetResponseWithURL キャッシュにレスポンスを保存する
	// ctx: コンテキスト（リクエストのキャンセレーションやタイムアウト制御に使用）
	// req: リクエスト情報（URLベースの階層構造でキャッシュを保存するために使用）
	// response: 保存するレスポンスデータ
	// ttl: キャッシュの有効期限
	SetResponseWithURL(ctx context.Context, req *model.BpRequest, response *model.BpResponse, ttl time.Duration) error

	DeleteExpiredCaches(ctx context.Context) error

	// ReserveRequest 非同期処理（Worker Pool）で処理するためにリクエストを予約する
	// Redisキューに追加して、RequestProcessorが非同期で処理する
	// req: 予約するリクエスト
	ReserveRequest(ctx context.Context, req *model.BpRequest) error

	// GetReservedRequests 予約されたリクエストのリストを取得する
	// 戻り値: 予約されたリクエストのリスト
	GetReservedRequests(ctx context.Context) ([]*model.BpRequest, error)

	// RemoveReservedRequest 予約されたリクエストを削除する
	// req: 削除するリクエスト
	RemoveReservedRequest(ctx context.Context, req *model.BpRequest) error

	// BLPopReservedRequest 予約されたリクエストをブロッキングで取得する
	// timeout: タイムアウト時間（0の場合は無期限に待機）
	// 戻り値: 取得したリクエスト（タイムアウトの場合はnil）
	BLPopReservedRequest(ctx context.Context, timeout time.Duration) (*model.BpRequest, error)
}
