package boot

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

func RegisterFlags(cmd *cobra.Command) {
	flags := cmd.PersistentFlags()
	flags.StringP("config", "c", "", "legacy config file path or URL (imported once)")
	flags.String("data-dir", "", "data directory (default ./data)")
	flags.String("db-path", "", "database file path (default <data-dir>/saveany.db)")
	flags.String("listen", "", "web listen address as host:port (default 127.0.0.1:28080)")
	flags.Bool("no-web", false, "do not start the web server")
}

type Options struct {
	ConfigPath string
	DataDir    string
	DBPath     string
	Listen     string
	NoWeb      bool
	InitWeb    bool
}

func Parse(cmd *cobra.Command) Options {
	flags := cmd.Flags()
	getString := func(name string) string {
		value, err := flags.GetString(name)
		if err != nil {
			log.Warnf("Failed to read --%s: %v", name, err)
		}
		return value
	}
	getBool := func(name string) bool {
		value, err := flags.GetBool(name)
		if err != nil {
			log.Warnf("Failed to read --%s: %v", name, err)
		}
		return value
	}
	configPath := getString("config")
	dataDir := getString("data-dir")
	dbPath := getString("db-path")
	listen := getString("listen")
	noWeb := getBool("no-web")

	if dataDir == "" {
		dataDir = os.Getenv("SAVEANY_DATA_DIR")
	}
	if dataDir == "" {
		dataDir = "./data"
	}
	if configPath == "" {
		configPath = os.Getenv("SAVEANY_CONFIG")
	}
	if dbPath == "" {
		dbPath = os.Getenv("SAVEANY_DB_PATH")
	}
	if dbPath == "" {
		dbPath = filepath.Join(dataDir, "saveany.db")
	}
	if listen == "" {
		listen = os.Getenv("SAVEANY_LISTEN")
	}
	return Options{
		ConfigPath: configPath,
		DataDir:    dataDir,
		DBPath:     dbPath,
		Listen:     listen,
		NoWeb:      noWeb,
	}
}
