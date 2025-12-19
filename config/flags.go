package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RegisterFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	// 基础配置
	flags.StringP("config", "c", "", "config file path")
	flags.StringP("lang", "l", "", "language (e.g., zh-Hans, en)")
	flags.IntP("workers", "w", 0, "number of workers")
	flags.Int("retry", 0, "retry times")
	flags.Int("threads", 0, "number of threads")
	flags.Bool("stream", false, "enable stream mode")
	flags.Bool("no-clean-cache", false, "do not clean cache on exit")
	flags.String("proxy", "", "proxy URL (http, https, socks5, socks5h)")

	// Telegram 配置
	flags.String("telegram-token", "", "telegram bot token")
	flags.Int("telegram-app-id", 0, "telegram app id")
	flags.String("telegram-app-hash", "", "telegram app hash")
	flags.Int("telegram-rpc-retry", 0, "telegram rpc retry times")
	flags.Bool("telegram-userbot-enable", false, "enable userbot")
	flags.String("telegram-userbot-session", "", "userbot session path")
	flags.Bool("telegram-proxy-enable", false, "enable telegram proxy")
	flags.String("telegram-proxy-url", "", "telegram proxy URL")

	// 数据库配置
	flags.String("db-path", "", "database path")
	flags.String("db-session", "", "session database path")

	// 临时目录配置
	flags.String("temp-base-path", "", "temp directory base path")

	// Parser 配置
	flags.Bool("parser-plugin-enable", false, "enable parser plugins")
	flags.StringSlice("parser-plugin-dirs", nil, "parser plugin directories")
	flags.String("parser-proxy", "", "parser proxy URL")

	// 绑定到 viper
	bindFlags(cmd)
}

func bindFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	viper.BindPFlag("lang", flags.Lookup("lang"))
	viper.BindPFlag("workers", flags.Lookup("workers"))
	viper.BindPFlag("retry", flags.Lookup("retry"))
	viper.BindPFlag("threads", flags.Lookup("threads"))
	viper.BindPFlag("stream", flags.Lookup("stream"))
	viper.BindPFlag("no_clean_cache", flags.Lookup("no-clean-cache"))
	viper.BindPFlag("proxy", flags.Lookup("proxy"))

	// Telegram
	viper.BindPFlag("telegram.token", flags.Lookup("telegram-token"))
	viper.BindPFlag("telegram.app_id", flags.Lookup("telegram-app-id"))
	viper.BindPFlag("telegram.app_hash", flags.Lookup("telegram-app-hash"))
	viper.BindPFlag("telegram.rpc_retry", flags.Lookup("telegram-rpc-retry"))
	viper.BindPFlag("telegram.userbot.enable", flags.Lookup("telegram-userbot-enable"))
	viper.BindPFlag("telegram.userbot.session", flags.Lookup("telegram-userbot-session"))
	viper.BindPFlag("telegram.proxy.enable", flags.Lookup("telegram-proxy-enable"))
	viper.BindPFlag("telegram.proxy.url", flags.Lookup("telegram-proxy-url"))

	// database
	viper.BindPFlag("db.path", flags.Lookup("db-path"))
	viper.BindPFlag("db.session", flags.Lookup("db-session"))
	// 临时目录
	viper.BindPFlag("temp.base_path", flags.Lookup("temp-base-path"))

	// Parser
	viper.BindPFlag("parser.plugin_enable", flags.Lookup("parser-plugin-enable"))
	viper.BindPFlag("parser.plugin_dirs", flags.Lookup("parser-plugin-dirs"))
	viper.BindPFlag("parser.proxy", flags.Lookup("parser-proxy"))
}

func GetConfigFile(cmd *cobra.Command) string {
	configFile, _ := cmd.Flags().GetString("config")
	return configFile
}
