package relay

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/worker"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/mygotg/ext"
	tgstorage "github.com/krau/mygotg/storage"
)

type fakeInvoker struct {
	mu    sync.Mutex
	calls map[uint]int
	fail  map[uint]bool
	data  []byte
}

func (f *fakeInvoker) Invoke(_ context.Context, input bin.Encoder, output bin.Decoder) error {
	req, ok := input.(*tg.UploadGetFileRequest)
	if !ok {
		return errors.New("unexpected request")
	}
	id := req.Location.(*tg.InputDocumentFileLocation).ID
	f.mu.Lock()
	f.calls[uint(id)]++
	fail := f.fail[uint(id)]
	f.mu.Unlock()
	if fail {
		return errors.New("mock worker failure")
	}
	box, ok := output.(*tg.UploadFileBox)
	if !ok {
		return errors.New("unexpected response")
	}
	offset := int(req.Offset)
	end := offset + req.Limit
	if end > len(f.data) {
		end = len(f.data)
	}
	box.File = &tg.UploadFile{Bytes: append([]byte(nil), f.data[offset:end]...)}
	return nil
}

func testExt(t *testing.T, inv *fakeInvoker) *ext.Context {
	t.Helper()
	peers := tgstorage.NewPeerStorage(nil, true)
	peers.AddPeer(500, 1, tgstorage.TypeChannel, "")
	return ext.NewContext(context.Background(), tg.NewClient(inv), peers, &tg.User{}, nil, nil, false)
}

func TestResumeStateRoundTrip(t *testing.T) {
	path := ResumeStatePath(t.TempDir() + "/cache.bin")
	identity := tfile.FileIdentity{Kind: "document", ID: 7, AccessHash: 8}
	state := resumeState{
		Version: resumeStateVersion, RelayChatID: 500, RelayMessageID: 42,
		SourceKind: identity.Kind, SourceID: identity.ID, SourceAccessHash: identity.AccessHash,
	}
	if err := saveResumeState(path, state); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadResumeState(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.RelayMessageID != 42 || !loaded.matches(identity) {
		t.Fatalf("loaded state = %+v", loaded)
	}
	if loaded.matches(tfile.FileIdentity{Kind: "document", ID: 9, AccessHash: 8}) {
		t.Fatal("state matched a different file")
	}
	if err := removeResumeState(path); err != nil {
		t.Fatal(err)
	}
	if loaded, err := loadResumeState(path); err != nil || loaded != nil {
		t.Fatalf("state after removal = %+v, %v", loaded, err)
	}
}

func TestMultiClientFailsOverAndFallsBackToPrimary(t *testing.T) {
	ctx := context.Background()
	data := []byte("relay chunk")
	firstInv := &fakeInvoker{calls: map[uint]int{}, fail: map[uint]bool{1: true}, data: data}
	secondInv := &fakeInvoker{calls: map[uint]int{}, fail: map[uint]bool{}, data: data}
	primaryInv := &fakeInvoker{calls: map[uint]int{}, fail: map[uint]bool{}, data: data}
	location := &tg.InputDocumentFileLocation{ID: 1}
	file := &relayFile{
		workers: []worker.ClientRef{
			{ID: 1, Name: "one", Ext: testExt(t, firstInv)},
			{ID: 2, Name: "two", Ext: testExt(t, secondInv)},
		},
		locations: map[uint]tg.InputFileLocationClass{
			1: &tg.InputDocumentFileLocation{ID: 1},
			2: &tg.InputDocumentFileLocation{ID: 2},
		},
		served: make(map[uint]int),
	}
	client := &multiClient{file: file, base: tg.NewClient(primaryInv)}
	res, err := client.UploadGetFile(ctx, &tg.UploadGetFileRequest{Location: location, Offset: 0, Limit: len(data)})
	if err != nil {
		t.Fatal(err)
	}
	uploaded, ok := res.(*tg.UploadFile)
	if !ok || string(uploaded.Bytes) != string(data) {
		t.Fatalf("response = %#v", res)
	}
	if firstInv.calls[1] != 1 || secondInv.calls[2] != 1 {
		t.Fatalf("worker calls: first=%v second=%v", firstInv.calls, secondInv.calls)
	}

	// Both helpers fail; the primary client must serve the chunk.
	firstInv.fail[1] = true
	secondInv.fail[2] = true
	res, err = client.UploadGetFile(ctx, &tg.UploadGetFileRequest{Location: location, Offset: 0, Limit: len(data)})
	if err != nil {
		t.Fatal(err)
	}
	if uploads, ok := res.(*tg.UploadFile); !ok || string(uploads.Bytes) != string(data) {
		t.Fatalf("fallback response = %#v", res)
	}
	if primaryInv.calls[1] != 1 {
		t.Fatalf("primary calls = %v", primaryInv.calls)
	}
}
