package config

type hookConfig struct {
	Exec hookExecConfig `toml:"exec" mapstructure:"exec" json:"exec"`
}

type hookExecConfig struct {
	// command to execute, for all task types
	TaskBeforeStart string `toml:"task_before_start" mapstructure:"task_before_start" json:"task_before_start"`
	TaskSuccess     string `toml:"task_success" mapstructure:"task_success" json:"task_success"`
	TaskFail        string `toml:"task_fail" mapstructure:"task_fail" json:"task_fail"`
	TaskCancel      string `toml:"task_cancel" mapstructure:"task_cancel" json:"task_cancel"`
}
