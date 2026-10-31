package boot

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func RegisterFlags(cmd *cobra.Command) {
	flags := cmd.PersistentFlags()
	flags.StringP("config", "c", "", "legacy config file path or URL (imported once)")
	flags.String("data-dir", "", "data directory (default ./data)")
	flags.String("db-path", "", "database file path (default <data-dir>/saveany.db)")
	flags.String("listen", "", "override the web listen address")
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
	configPath, _ := flags.GetString("config")
	dataDir, _ := flags.GetString("data-dir")
	dbPath, _ := flags.GetString("db-path")
	listen, _ := flags.GetString("listen")
	noWeb, _ := flags.GetBool("no-web")

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
