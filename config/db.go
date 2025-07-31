package config

// dbConfig contains database configuration for both SQLite and Redis
type dbConfig struct {
	// SQLite configuration (existing)
	Path    string `toml:"path" mapstructure:"path"`
	Session string `toml:"session" mapstructure:"session"`
	
	// Redis configuration (new)
	// If RedisAddr is set, Redis will be used instead of SQLite
	RedisAddr     string `toml:"redis_addr" mapstructure:"redis_addr"`         // Redis server address (e.g., "localhost:6379")
	RedisPassword string `toml:"redis_password" mapstructure:"redis_password"` // Redis password (optional)
	RedisDB       int    `toml:"redis_db" mapstructure:"redis_db"`             // Redis database number (default: 0)
}
