package tcbdata

import (
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

const (
	TypeAdd        = "add"
	TypeSetDefault = "setdefault"
)

// type TaskDataTGFiles struct {
// 	Files   []tfile.TGFileMessage
// 	AsBatch bool
// }

// type TaskDataTelegraph struct {
// 	Pics     []string
// 	PageNode *telegraph.Page
// }

// type TaskDataType interface {
// 	TaskDataTGFiles | TaskDataTelegraph
// }

type Add struct {
	TaskType         tasktype.TaskType
	SelectedStorName string
	DirID            uint
	SettedDir        bool
	// tfiles
	Files   []tfile.TGFileMessage
	AsBatch bool
	// tphpics
	TphPageNode *telegraph.Page
	TphPics     []string
	TphDirPath  string // unescaped telegraph.Page.Path
}

type SetDefaultStorage struct {
	StorageName string
}
