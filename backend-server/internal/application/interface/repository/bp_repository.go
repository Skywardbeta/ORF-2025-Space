package repository_interfaces

import (
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

// BpRepository Redisキャッシュを操作するためのリポジトリインターフェース
type BpRepository interface {
	// GetResponse キャッシュからレスポンスを取得する
	// key: キャッシュキー（通常はURLのPath）
	// 戻り値: キャッシュされたレスポンスと、キャッシュが存在するかどうか
	GetResponse(key string) (*model.BpResponse, bool, error)

	// SetResponse レスポンスをキャッシュに保存する
	// key: キャッシュキー（通常はURLのPath）
	// response: 保存するレスポンス
	// ttl: キャッシュの有効期限（0の場合はデフォルトのTTLを使用）
	SetResponse(key string, response *model.BpResponse, ttl time.Duration) error

	// DeleteResponse キャッシュからレスポンスを削除する
	// key: キャッシュキー
	DeleteResponse(key string) error

	// GenerateCacheKey リクエストからキャッシュキーを生成する
	// メソッド、URL、ヘッダーなどから一意のキーを生成
	GenerateCacheKey(req *model.BpRequest) string
}
