package rule

type RuleType string

const (
	FileNameRegex RuleType = "FILENAME-REGEX"
	MessageRegex  RuleType = "MESSAGE-REGEX"
)

func (r RuleType) String() string {
	return string(r)
}

func Values() []RuleType {
	return []RuleType{FileNameRegex, MessageRegex}
}
