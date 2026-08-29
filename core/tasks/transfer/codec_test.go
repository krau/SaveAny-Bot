package transfer

import (
	"context"
	"encoding/json"
	"testing"
)

// TestTransferCodecMarshalKeepsDoneList verifies a re-persist keeps the done
// list from the previous payload: losing it would re-transfer finished
// elements on a second restart.
func TestTransferCodecMarshalKeepsDoneList(t *testing.T) {
	task := &Task{
		ID:       "tr-1",
		ctx:      context.Background(),
		doneList: []string{"e-1", "e-2"},
	}
	data, err := taskCodec{}.Marshal(task)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var p taskPayload
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(p.Done) != 2 || p.Done[0] != "e-1" || p.Done[1] != "e-2" {
		t.Fatalf("Done = %v, want [e-1 e-2]", p.Done)
	}
}

// TestTransferCodecUnmarshalUnknownStorage verifies recovery fails cleanly
// when a referenced storage is not configured.
func TestTransferCodecUnmarshalUnknownStorage(t *testing.T) {
	data := []byte(`{"id":"tr-1","elements":[{"id":"e-1","source_storage":"nope","target_storage":"nope"}]}`)
	if _, err := (taskCodec{}).Unmarshal(data); err == nil {
		t.Fatalf("expected error for unknown storage")
	}
}
