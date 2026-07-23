package config

type cacheConfig struct {
	TTL              int64 `toml:"ttl" mapstructure:"ttl" json:"ttl"`
	FileSelectionTTL int64 `toml:"file_selection_ttl" mapstructure:"file_selection_ttl" json:"file_selection_ttl"`
	NumCounters      int64 `toml:"num_counters" mapstructure:"num_counters" json:"num_counters"`
	MaxCost          int64 `toml:"max_cost" mapstructure:"max_cost" json:"max_cost"`
}
