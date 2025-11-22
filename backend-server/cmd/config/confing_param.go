package config

import "time"

// BpGateway BPゲートウェイの設定
type BpGateway struct {
	TransportMode string         `yaml:"transport_mode"` // "http" or "bp"
	Host          string         `yaml:"host"`           // HTTPモード時のホスト
	Port          int            `yaml:"port"`           // HTTPモード時のポート
	Timeout       time.Duration  `yaml:"timeout"`        // タイムアウト
	BpSocket      BpSocketConfig `yaml:"bp_socket"`      // BPモード時の設定
}

// BpSocketConfig BPソケット（dtn-socket）の設定
type BpSocketConfig struct {
	LocalNodeNum     uint64 `yaml:"local_node_num"`
	LocalServiceNum  uint64 `yaml:"local_service_num"`
	RemoteNodeNum    uint64 `yaml:"remote_node_num"`
	RemoteServiceNum uint64 `yaml:"remote_service_num"`
}

type Redis struct {
	// Redisサーバーの接続情報
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type RedisKeys struct {
	// Redis内で使用するキーのパターン
	ReservedRequestsKey string `yaml:"reserved_requests_key"`
	PendingRequestsKey  string `yaml:"pending_requests_key"`
	CacheMetaPattern    string `yaml:"cache_meta_pattern"`
	ScanCount           int    `yaml:"scan_count"` // Redis SCANコマンドのCOUNTパラメータ
}

type CacheConfig struct {
	Dir             string        `yaml:"dir"`              // キャッシュファイルを保存するディレクトリ
	DefaultTTL      time.Duration `yaml:"default_ttl"`      // デフォルトのキャッシュTTL
	CleanupInterval time.Duration `yaml:"cleanup_interval"` // キャッシュクリーンアップの実行間隔
}

type WorkerConfig struct {
	Workers           int           `yaml:"workers"`             // Worker Poolのワーカー数
	QueueWatchTimeout time.Duration `yaml:"queue_watch_timeout"` // キュー監視のタイムアウト
}

type MiddlewareConfig struct {
	CertPath      string `yaml:"cert_path"`      // ルート証明書のパス
	KeyPath       string `yaml:"key_path"`       // ルート秘密鍵のパス
	MaxCacheSize  int    `yaml:"max_cache_size"` // 証明書キャッシュの最大数
	RSABits       int    `yaml:"rsa_bits"`       // RSA鍵のビット長
	CacheDuration int    `yaml:"cache_duration"` // 生成した証明書の有効期間(時間)
}

// Mode サーバーの動作モード
type Mode string

const (
	DebugMode      Mode = "debug"      // デバッグモード（ローカルGatewayを使用）
	ProductionMode Mode = "production" // 本番モード（通常のGatewayを使用）
)

type ServerConfig struct {
	Port            int    `yaml:"port"`              // HTTPサーバーのポート番号
	Mode            Mode   `yaml:"mode"`              // サーバーの動作モード
	DefaultDir      string `yaml:"default_dir"`       // デフォルトページとプレースホルダーファイルのディレクトリ
	DefaultFileName string `yaml:"default_file_name"` // デフォルトHTMLファイル名
}
