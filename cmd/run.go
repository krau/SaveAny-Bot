package cmd

import (
	"github.com/krau/SaveAny-Bot/bootstrap"
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/core"
)

func Run() {
	bootstrap.InitAll()
	go core.Run()
	bot.Run()
}
