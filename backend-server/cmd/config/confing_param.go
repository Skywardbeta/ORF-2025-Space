package config

import "time"

type BpGateway struct {
	TransportMode string
	Host          string
	Port          int
	Timeout       time.Duration
	BpSocket      BpSocketConfig
}

type BpSocketConfig struct {
	LocalNodeNum     uint64
	LocalServiceNum  uint64
	RemoteNodeNum    uint64
	RemoteServiceNum uint64
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
	Dir             string
	DefaultTTL      time.Duration
	CleanupInterval time.Duration
}

type WorkerConfig struct {
	Workers           int
	QueueWatchTimeout time.Duration
}

type MiddlewareConfig struct {
	CertPath      string
	KeyPath       string
	MaxCacheSize  int
	RSABits       int
	CacheDuration int
}
