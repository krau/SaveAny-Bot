package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/krau/SaveAny-Bot/bootstrap"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/spf13/cobra"
)

func Run(_ *cobra.Command, _ []string) {
	bootstrap.InitAll()
	core.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.L.Info(sig, ", exit")
}
