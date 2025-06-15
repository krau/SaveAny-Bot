package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/krau/SaveAny-Bot/cmd"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	cmd.Execute(ctx)
}
