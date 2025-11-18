package config

type BpGateway struct {
	Host string
	Port int
}

type Redis struct {
	Host string
	Port int
}

type Config struct {
	BPGateway  BpGateway
	CacheStore Redis
}

func LoadConfig() Config {
	return Config{
		BPGateway: BpGateway{
			Host: "localhost",
			Port: 8081,
		},
		CacheStore: Redis{
			Host: "localhost",
			Port: 6379,
		},
	}
}
