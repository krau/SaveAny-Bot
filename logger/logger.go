package logger

import (
	"github.com/krau/SaveAny-Bot/config"

	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
	"github.com/gookit/slog/rotatefile"
)

var L *slog.Logger

func InitLogger() {
	if L != nil {
		return
	}
	slog.DefaultChannelName = "SaveAnyBot"
	L = slog.New()
	logLevel := slog.LevelByName(config.Cfg.Log.Level)
	logFilePath := config.Cfg.Log.File
	logBackupNum := config.Cfg.Log.BackupCount
	var logLevels []slog.Level
	for _, level := range slog.AllLevels {
		if level <= logLevel {
			logLevels = append(logLevels, level)
		}
	}
	consoleH := handler.NewConsoleHandler(logLevels)
	fileH, err := handler.NewTimeRotateFile(
		logFilePath,
		rotatefile.EveryDay,
		handler.WithLogLevels(slog.AllLevels),
		handler.WithBackupNum(logBackupNum),
		handler.WithBuffSize(0),
	)
	if err != nil {
		panic(err)
	}
	L.AddHandlers(consoleH, fileH)
}
