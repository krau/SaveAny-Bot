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
	tem := "[{{datetime}}] [{{level}}] [{{caller}}] {{message}} {{data}} {{extra}}\n"
	consoleH := handler.NewConsoleHandler(logLevels)
	consoleH.Formatter().(*slog.TextFormatter).SetTemplate(tem)
	Log.AddHandler(consoleH)
	if logFilePath != "" && logBackupNum > 0 {
		fileH, err := handler.NewTimeRotateFile(
			logFilePath,
			rotatefile.EveryDay,
			handler.WithLogLevels(slog.AllLevels),
			handler.WithBackupNum(logBackupNum),
		)
		fileH.Formatter().(*slog.TextFormatter).SetTemplate(tem)
		if err != nil {
			panic(err)
		}
		Log.AddHandler(fileH)
	}
}
