package taskstatus

//go:generate go-enum --values --names --noprefix --flag --nocase

/*
ENUM(
pending, succeeded, failed, canceled
)
*/
type TaskStatus string
