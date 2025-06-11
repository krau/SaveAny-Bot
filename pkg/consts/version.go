package consts

// inject version by '-X' flag
// go build -ldflags "-X github.com/krau/SaveAny-Bot/pkg/consts.Version=${{ env.VERSION }}"
var (
	Version   string = "dev"
	BuildTime string = "unknown"
	GitCommit string = "unknown"
)
