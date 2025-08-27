package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/krau/SaveAny-Bot/cmd"
)

//go:generate go run cmd/geni18n/main.go -dir ./common/i18n/locale -out common/i18n/i18nk/keys.go -pkg i18nk

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	cmd.Execute(ctx)
}
