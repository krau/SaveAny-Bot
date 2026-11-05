package tcbdata

import (
	"slices"

	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/parser"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

const (
	TypeAdd        = "add"
	TypeSetDefault = "setdefault"
	TypeConfig     = "config"
	TypeCancel     = "cancel"
	TypeEngine     = "engine"
	TypeTask       = "task"
	TypeRetry      = "retry"
	TypeWatch      = "watch"
	TypeDir        = "dir"
	TypeRule       = "rule"
)

const (
	ConflictStrategyRename    = "rename"
	ConflictStrategyAsk       = "ask"
	ConflictStrategyOverwrite = "overwrite"
	ConflictStrategySkip      = "skip"
)

func ConflictStrategyValues() []string {
	return []string{
		ConflictStrategyRename,
		ConflictStrategyAsk,
		ConflictStrategyOverwrite,
		ConflictStrategySkip,
	}
}

func IsConflictStrategy(strategy string) bool {
	return slices.Contains(ConflictStrategyValues(), strategy)
}

type Add struct {
	TaskType          tasktype.TaskType
	SelectedStorageID uint
	DirID             uint
	SettedDir         bool
	SelectedDirPath   string
	ConflictStrategy  string
	// tfiles
	Files   []tfile.TGFileMessage
	AsBatch bool
	// tphpics
	TphPageNode *telegraph.Page
	TphPics     []string
	TphDirPath  string // unescaped telegraph.Page.Path
	// parseditem
	ParsedItem *parser.Item
	// directlinks
	DirectLinks []string
	// aria2
	Aria2URIs []string
	// ytdlp
	YtdlpURLs  []string
	YtdlpFlags []string
	// transfer
	TransferSourceStorageID uint
	TransferSourcePath      string
	TransferFiles           []string // file paths relative to source storage
}

type SetDefaultStorage struct {
	StorageID uint
	DirID     uint
}
