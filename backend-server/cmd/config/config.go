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

type CacheConfig struct {
	Dir             string        // キャッシュファイルを保存するディレクトリ
	DefaultTTL      time.Duration // デフォルトのキャッシュTTL
	CleanupInterval time.Duration // キャッシュクリーンアップの実行間隔
}

type WorkerConfig struct {
	Workers           int           // Worker Poolのワーカー数
	QueueWatchTimeout time.Duration // キュー監視のタイムアウト
}

type Config struct {
	BPGateway   BpGateway
	RedisClient Redis
	Cache       CacheConfig
	Worker      WorkerConfig
}

func LoadConfig() Config {
	return Config{
		BPGateway: BpGateway{
			Host:    "localhost",
			Port:    8081,
			Timeout: 180 * time.Second,
		},
		RedisClient: Redis{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		Cache: CacheConfig{
			Dir:             "/tmp/bp_cache",
			DefaultTTL:      24 * time.Hour,
			CleanupInterval: 5 * time.Minute,
		},
		Worker: WorkerConfig{
			Workers:           5,
			QueueWatchTimeout: 5 * time.Second,
		},
	}
}
