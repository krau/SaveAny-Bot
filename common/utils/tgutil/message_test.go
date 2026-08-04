package tgutil

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestSortMessagesByID(t *testing.T) {
	messages := []*tg.Message{{ID: 9}, {ID: 3}, {ID: 7}}
	sortMessagesByID(messages)
	want := []int{3, 7, 9}
	for i := range messages {
		if messages[i].GetID() != want[i] {
			t.Fatalf("message %d has ID %d, want %d", i, messages[i].GetID(), want[i])
		}
	}
}
