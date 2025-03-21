package common

import (
	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
	"github.com/gookit/slog/rotatefile"
	"github.com/krau/SaveAny-Bot/config"
)

var Log *slog.Logger

func InitLogger() {
	if Log != nil {
		return
	}
	slog.DefaultChannelName = "SaveAnyBot"
	Log = slog.New()
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
	Log.AddHandlers(consoleH, fileH)
}
