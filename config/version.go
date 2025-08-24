package config

// inject version by '-X' flag
// go build -ldflags "-X github.com/krau/SaveAny-Bot/config.Version=${{ env.VERSION }}"
var (
	Version   string = "dev"
	BuildTime string = "unknown"
	GitCommit string = "unknown"
)

const (
	GitRepo = "krau/SaveAny-Bot"
)