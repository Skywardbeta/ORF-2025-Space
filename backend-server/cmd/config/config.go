package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BPGateway   BpGateway        `yaml:"bp_gateway"`
	RedisClient Redis            `yaml:"redis_client"`
	RedisKeys   RedisKeys        `yaml:"redis_keys"`
	Cache       CacheConfig      `yaml:"cache"`
	Worker      WorkerConfig     `yaml:"worker"`
	Middlware   MiddlewareConfig `yaml:"middleware"`
	Server      ServerConfig     `yaml:"server"`
}

func LoadConfig() Config {
	// デフォルト設定
	defaultConfig := Config{
		BPGateway: BpGateway{
			TransportMode: "bp_socket", // "ion_cli" or "bp_socket"
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
			PendingRequestsKey:  "bp:pending:requests",
			CacheMetaPattern:    "bp:cache:meta:*",
			// ScanCount は省略可能（デフォルト値100が使用される）
			// ScanCount:           100,
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
			CertPath:      "./my_crt/bump.crt",
			KeyPath:       "./my_crt/bump.key",
			MaxCacheSize:  20,
			RSABits:       2048,
			CacheDuration: 24,
		},
		Server: ServerConfig{
			Port:            8082,
			Mode:            ProductionMode, // デフォルトはproductionモード
			DefaultDir:      "pages",        // デフォルトページとプレースホルダーファイルのディレクトリ
			DefaultFileName: "default.txt",  // デフォルトHTMLファイル名
		},
	}

	// YAMLファイルから設定を読み込む（存在する場合）
	configPath := getConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		var yamlConfig yamlConfig
		if err := yaml.Unmarshal(data, &yamlConfig); err == nil {
			// YAMLから読み込んだ設定でデフォルト値をマージ
			return mergeConfig(defaultConfig, yamlConfig.toConfig())
		}
		// YAMLのパースエラーは無視してデフォルト値を使用
		fmt.Printf("Warning: Failed to parse config file %s: %v, using defaults\n", configPath, err)
	}

	return defaultConfig
}

// getConfigPath 設定ファイルのパスを取得
// 環境変数 CONFIG_PATH が設定されている場合はそれを使用
// それ以外は config.yaml を探す
func getConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return "config.yaml"
}

// yamlConfig YAMLファイル用の一時的な構造体（time.Durationを文字列として読み込む）
type yamlConfig struct {
	BPGateway struct {
		TransportMode string `yaml:"transport_mode"`
		Host          string `yaml:"host"`
		Port          int    `yaml:"port"`
		Timeout       string `yaml:"timeout"`
		BpSocket      struct {
			LocalNodeNum     uint64 `yaml:"local_node_num"`
			LocalServiceNum  uint64 `yaml:"local_service_num"`
			RemoteNodeNum    uint64 `yaml:"remote_node_num"`
			RemoteServiceNum uint64 `yaml:"remote_service_num"`
		} `yaml:"bp_socket"`
	} `yaml:"bp_gateway"`
	RedisClient struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis_client"`
	RedisKeys struct {
		ReservedRequestsKey string `yaml:"reserved_requests_key"`
		PendingRequestsKey  string `yaml:"pending_requests_key"`
		CacheMetaPattern    string `yaml:"cache_meta_pattern"`
		ScanCount           int    `yaml:"scan_count"`
	} `yaml:"redis_keys"`
	Cache struct {
		Dir             string `yaml:"dir"`
		DefaultTTL      string `yaml:"default_ttl"`
		CleanupInterval string `yaml:"cleanup_interval"`
	} `yaml:"cache"`
	Worker struct {
		Workers           int    `yaml:"workers"`
		QueueWatchTimeout string `yaml:"queue_watch_timeout"`
	} `yaml:"worker"`
	Middlware struct {
		CertPath      string `yaml:"cert_path"`
		KeyPath       string `yaml:"key_path"`
		MaxCacheSize  int    `yaml:"max_cache_size"`
		RSABits       int    `yaml:"rsa_bits"`
		CacheDuration int    `yaml:"cache_duration"`
	} `yaml:"middleware"`
	Server struct {
		Port            int    `yaml:"port"`
		Mode            string `yaml:"mode"`
		DefaultDir      string `yaml:"default_dir"`
		DefaultFileName string `yaml:"default_file_name"`
	} `yaml:"server"`
}

// toConfig yamlConfigをConfigに変換（time.Durationの文字列をパース）
func (yc yamlConfig) toConfig() Config {
	parseDuration := func(s string) time.Duration {
		if s == "" {
			return 0
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0
		}
		return d
	}

	var mode Mode
	if yc.Server.Mode == "debug" {
		mode = DebugMode
	} else {
		mode = ProductionMode
	}

	return Config{
		BPGateway: BpGateway{
			TransportMode: yc.BPGateway.TransportMode,
			Host:          yc.BPGateway.Host,
			Port:          yc.BPGateway.Port,
			Timeout:       parseDuration(yc.BPGateway.Timeout),
			BpSocket: BpSocketConfig{
				LocalNodeNum:     yc.BPGateway.BpSocket.LocalNodeNum,
				LocalServiceNum:  yc.BPGateway.BpSocket.LocalServiceNum,
				RemoteNodeNum:    yc.BPGateway.BpSocket.RemoteNodeNum,
				RemoteServiceNum: yc.BPGateway.BpSocket.RemoteServiceNum,
			},
		},
		RedisClient: Redis{
			Host:     yc.RedisClient.Host,
			Port:     yc.RedisClient.Port,
			Password: yc.RedisClient.Password,
			DB:       yc.RedisClient.DB,
		},
		RedisKeys: RedisKeys{
			ReservedRequestsKey: yc.RedisKeys.ReservedRequestsKey,
			CacheMetaPattern:    yc.RedisKeys.CacheMetaPattern,
			ScanCount:           yc.RedisKeys.ScanCount,
		},
		Cache: CacheConfig{
			Dir:             yc.Cache.Dir,
			DefaultTTL:      parseDuration(yc.Cache.DefaultTTL),
			CleanupInterval: parseDuration(yc.Cache.CleanupInterval),
		},
		Worker: WorkerConfig{
			Workers:           yc.Worker.Workers,
			QueueWatchTimeout: parseDuration(yc.Worker.QueueWatchTimeout),
		},
		Middlware: MiddlewareConfig{
			CertPath:      yc.Middlware.CertPath,
			KeyPath:       yc.Middlware.KeyPath,
			MaxCacheSize:  yc.Middlware.MaxCacheSize,
			RSABits:       yc.Middlware.RSABits,
			CacheDuration: yc.Middlware.CacheDuration,
		},
		Server: ServerConfig{
			Port:            yc.Server.Port,
			Mode:            mode,
			DefaultDir:      yc.Server.DefaultDir,
			DefaultFileName: yc.Server.DefaultFileName,
		},
	}
}

// mergeConfig YAMLから読み込んだ設定でデフォルト設定をマージ
// YAMLで設定されていない項目はデフォルト値を使用
func mergeConfig(defaultConfig, yamlConfig Config) Config {
	merged := defaultConfig

	// BPGateway
	if yamlConfig.BPGateway.TransportMode != "" {
		merged.BPGateway.TransportMode = yamlConfig.BPGateway.TransportMode
	}
	if yamlConfig.BPGateway.Host != "" {
		merged.BPGateway.Host = yamlConfig.BPGateway.Host
	}
	if yamlConfig.BPGateway.Port != 0 {
		merged.BPGateway.Port = yamlConfig.BPGateway.Port
	}
	if yamlConfig.BPGateway.Timeout != 0 {
		merged.BPGateway.Timeout = yamlConfig.BPGateway.Timeout
	}
	if yamlConfig.BPGateway.BpSocket.LocalNodeNum != 0 {
		merged.BPGateway.BpSocket.LocalNodeNum = yamlConfig.BPGateway.BpSocket.LocalNodeNum
	}
	if yamlConfig.BPGateway.BpSocket.LocalServiceNum != 0 {
		merged.BPGateway.BpSocket.LocalServiceNum = yamlConfig.BPGateway.BpSocket.LocalServiceNum
	}
	if yamlConfig.BPGateway.BpSocket.RemoteNodeNum != 0 {
		merged.BPGateway.BpSocket.RemoteNodeNum = yamlConfig.BPGateway.BpSocket.RemoteNodeNum
	}
	if yamlConfig.BPGateway.BpSocket.RemoteServiceNum != 0 {
		merged.BPGateway.BpSocket.RemoteServiceNum = yamlConfig.BPGateway.BpSocket.RemoteServiceNum
	}

	// RedisClient
	if yamlConfig.RedisClient.Host != "" {
		merged.RedisClient.Host = yamlConfig.RedisClient.Host
	}
	if yamlConfig.RedisClient.Port != 0 {
		merged.RedisClient.Port = yamlConfig.RedisClient.Port
	}
	if yamlConfig.RedisClient.Password != "" {
		merged.RedisClient.Password = yamlConfig.RedisClient.Password
	}
	if yamlConfig.RedisClient.DB != 0 || yamlConfig.RedisClient.Host != "" {
		merged.RedisClient.DB = yamlConfig.RedisClient.DB
	}

	// RedisKeys
	if yamlConfig.RedisKeys.ReservedRequestsKey != "" {
		merged.RedisKeys.ReservedRequestsKey = yamlConfig.RedisKeys.ReservedRequestsKey
	}
	if yamlConfig.RedisKeys.PendingRequestsKey != "" {
		merged.RedisKeys.PendingRequestsKey = yamlConfig.RedisKeys.PendingRequestsKey
	}
	if yamlConfig.RedisKeys.CacheMetaPattern != "" {
		merged.RedisKeys.CacheMetaPattern = yamlConfig.RedisKeys.CacheMetaPattern
	}
	if yamlConfig.RedisKeys.ScanCount != 0 {
		merged.RedisKeys.ScanCount = yamlConfig.RedisKeys.ScanCount
	}

	// Cache
	if yamlConfig.Cache.Dir != "" {
		merged.Cache.Dir = yamlConfig.Cache.Dir
	}
	if yamlConfig.Cache.DefaultTTL != 0 {
		merged.Cache.DefaultTTL = yamlConfig.Cache.DefaultTTL
	}
	if yamlConfig.Cache.CleanupInterval != 0 {
		merged.Cache.CleanupInterval = yamlConfig.Cache.CleanupInterval
	}

	// Worker
	if yamlConfig.Worker.Workers != 0 {
		merged.Worker.Workers = yamlConfig.Worker.Workers
	}
	if yamlConfig.Worker.QueueWatchTimeout != 0 {
		merged.Worker.QueueWatchTimeout = yamlConfig.Worker.QueueWatchTimeout
	}

	// Middleware
	if yamlConfig.Middlware.CertPath != "" {
		merged.Middlware.CertPath = yamlConfig.Middlware.CertPath
	}
	if yamlConfig.Middlware.KeyPath != "" {
		merged.Middlware.KeyPath = yamlConfig.Middlware.KeyPath
	}
	if yamlConfig.Middlware.MaxCacheSize != 0 {
		merged.Middlware.MaxCacheSize = yamlConfig.Middlware.MaxCacheSize
	}
	if yamlConfig.Middlware.RSABits != 0 {
		merged.Middlware.RSABits = yamlConfig.Middlware.RSABits
	}
	if yamlConfig.Middlware.CacheDuration != 0 {
		merged.Middlware.CacheDuration = yamlConfig.Middlware.CacheDuration
	}

	// Server
	if yamlConfig.Server.Port != 0 {
		merged.Server.Port = yamlConfig.Server.Port
	}
	if yamlConfig.Server.Mode != "" {
		merged.Server.Mode = yamlConfig.Server.Mode
	}
	if yamlConfig.Server.DefaultDir != "" {
		merged.Server.DefaultDir = yamlConfig.Server.DefaultDir
	}
	if yamlConfig.Server.DefaultFileName != "" {
		merged.Server.DefaultFileName = yamlConfig.Server.DefaultFileName
	}

	return merged
}
