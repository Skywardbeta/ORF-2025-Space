package config

import "time"

type BpGateway struct {
	Host    string
	Port    int
	Timeout time.Duration // HTTPクライアントのタイムアウト
}

type Redis struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type RedisKeys struct {
	ReservedRequestsKey string
	CacheMetaPattern    string
}

type CacheConfig struct {
	Dir             string        // キャッシュファイルを保存するディレクトリ
	DefaultTTL      time.Duration // デフォルトのキャッシュTTL
	CleanupInterval time.Duration // キャッシュクリーンアップの実行間隔
}

type WorkerConfig struct {
	Workers           int           // Worker Poolのワーカー数
	QueueWatchTimeout time.Duration // キュー監視のタイムアウト
}

type MiddlewareConfig struct {
	CertPath      string // ルート証明書のパス
	KeyPath       string // ルート秘密鍵のパス
	MaxCacheSize  int    // 証明書キャッシュの最大数
	RSABits       int    // RSA鍵のビット長
	CacheDuration int    // 生成した証明書の有効期間(時間)
}
