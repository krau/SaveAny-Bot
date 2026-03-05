package tasktype

// ENUM(tgfiles,tphpics,parseditem,directlinks,aria2,ytdlp,transfer)
//
//go:generate go-enum --values --names --flag --nocase
type TaskType string
