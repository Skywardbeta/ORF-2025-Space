package config

import "time"

type Config struct {
	BPGateway   BpGateway
	RedisClient Redis
	RedisKeys   RedisKeys
	Cache       CacheConfig
	Worker      WorkerConfig
	Middlware   MiddlewareConfig
}

func LoadConfig() Config {
	return Config{
		BPGateway: BpGateway{
			TransportMode: "ion_cli", // "ion_cli" or "bp_socket"
			Host:          "localhost",
			Port:          8081,
			Timeout:       5 * time.Second,
			BpSocket: BpSocketConfig{
				LocalNodeNum:     149,
				LocalServiceNum:  1,
				RemoteNodeNum:    150,
				RemoteServiceNum: 1,
			},
		},
		RedisClient: Redis{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		RedisKeys: RedisKeys{
			ReservedRequestsKey: "bp:reserved:requests",
			CacheMetaPattern:    "bp:cache:meta:*",
		},
		Cache: CacheConfig{
			Dir:             "./tmp/bp_cache",
			DefaultTTL:      24 * time.Hour,
			CleanupInterval: 5 * time.Minute,
		},
		Worker: WorkerConfig{
			Workers:           10,
			QueueWatchTimeout: 10 * time.Second,
		},
		Middlware: MiddlewareConfig{
			CertPath:      "./bump.crt",
			KeyPath:       "./bump.key",
			MaxCacheSize:  20,
			RSABits:       2048,
			CacheDuration: 24,
		},
	}
}
