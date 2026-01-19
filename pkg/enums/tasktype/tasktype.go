package tasktype

//go:generate go-enum --values --names --flag --nocase
// ENUM(tgfiles,tphpics,parseditem,directlinks,aria2,ytdlp,transfer)
type TaskType string
