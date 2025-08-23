package rule

type RuleClass[InputType any] interface {
	Type() RuleType
	Match(input InputType) (bool, error)
	StorageName() string
	StoragePath() string
}

type storInfo struct {
	storName string
	storPath string
}
