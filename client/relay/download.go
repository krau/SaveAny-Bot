package relay

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/constant"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/worker"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/mygotg/ext"
)

const minRelayFileSize = 8 << 20

// Prepare wraps a Telegram file with a relay-backed downloader when a relay
// channel and at least two running helpers are available. Any failure leaves
// the original single-client file in place.
func Prepare(ctx context.Context, file tfile.TGFile) (tfile.TGFile, func(context.Context)) {
	relayChatID := config.C().Telegram.RelayChatID
	if relayChatID == 0 {
		return file, nil
	}
	if config.C().Stream {
		log.FromContext(ctx).Debug("Relay download skipped: stream mode is enabled")
		return file, nil
	}
	if file == nil || file.Size() < minRelayFileSize {
		return file, nil
	}
	messageFile, ok := file.(tfile.TGFileMessage)
	if !ok || messageFile.Message() == nil {
		log.FromContext(ctx).Debug("Relay download skipped: file is not a Telegram message")
		return file, nil
	}
	extCtx := tgutil.ExtFromContext(ctx)
	if extCtx == nil {
		return file, nil
	}
	workers := worker.RunningClients()
	if len(workers) < 2 {
		log.FromContext(ctx).Debugf("Relay download skipped: %d running helper bot(s), need at least 2", len(workers))
		return file, nil
	}
	relayPeer, err := extCtx.ResolveInputPeerById(relayChatID)
	if err != nil {
		log.FromContext(ctx).Debugf("Relay channel is unavailable, using primary client: %v", err)
		return file, nil
	}
	if _, ok := relayPeer.(*tg.InputPeerChannel); !ok {
		log.FromContext(ctx).Warn("Relay chat is not a channel, using primary client")
		return file, nil
	}
	message := messageFile.Message()
	sourceChatID := tgutil.ChatIdFromPeer(message.GetPeerID())
	if sourceChatID == 0 || message.GetID() == 0 {
		return file, nil
	}
	updates, err := extCtx.ForwardMessages(sourceChatID, relayChatID, &tg.MessagesForwardMessagesRequest{
		ID: []int{message.GetID()},
	})
	if err != nil {
		log.FromContext(ctx).Debugf("Relay forward failed, using primary client: %v", err)
		return file, nil
	}
	relayMessageID := forwardedMessageID(updates, relayChatID)
	if relayMessageID == 0 {
		log.FromContext(ctx).Warn("Relay forward returned no channel message ID, using primary client")
		return file, nil
	}
	relayed := &relayFile{
		base:           messageFile,
		primary:        extCtx,
		relayChatID:    relayChatID,
		relayMessageID: relayMessageID,
		workers:        workers,
		keepMessages:   config.C().Telegram.RelayKeepMessages,
		locations:      make(map[uint]tg.InputFileLocationClass, len(workers)),
		served:         make(map[uint]int, len(workers)),
	}
	accessible := make([]worker.ClientRef, 0, len(workers))
	for _, ref := range workers {
		if _, err := relayed.workerLocation(ctx, ref); err != nil {
			log.FromContext(ctx).Debugf("Relay worker %s cannot read channel: %v", ref.Name, err)
			continue
		}
		accessible = append(accessible, ref)
	}
	if len(accessible) < 2 {
		log.FromContext(ctx).Warnf("Relay download skipped: only %d helper bot(s) can read the relay channel", len(accessible))
		relayed.cleanup(ctx)
		return file, nil
	}
	relayed.workers = accessible
	log.FromContext(ctx).Infof(
		"Relay download enabled: file=%q size=%d helpers=%s message_id=%d keep_messages=%t",
		file.Name(), file.Size(), workerNames(accessible), relayMessageID, relayed.keepMessages,
	)
	return relayed, relayed.cleanup
}

func workerNames(workers []worker.ClientRef) string {
	names := make([]string, 0, len(workers))
	for _, ref := range workers {
		if ref.Name != "" {
			names = append(names, ref.Name)
		}
	}
	return strings.Join(names, ", ")
}

type relayFile struct {
	base           tfile.TGFileMessage
	primary        *ext.Context
	relayChatID    int64
	relayMessageID int
	workers        []worker.ClientRef
	keepMessages   bool

	mu        sync.Mutex
	next      int
	locations map[uint]tg.InputFileLocationClass
	served    map[uint]int
}

func (f *relayFile) Location() tg.InputFileLocationClass { return f.base.Location() }
func (f *relayFile) Dler() downloader.Client             { return &multiClient{file: f, base: f.base.Dler()} }
func (f *relayFile) Size() int64                         { return f.base.Size() }
func (f *relayFile) Name() string                        { return f.base.Name() }
func (f *relayFile) SetName(name string)                 { f.base.SetName(name) }
func (f *relayFile) Message() *tg.Message                { return f.base.Message() }
func (f *relayFile) WorkerCount() int                    { return len(f.workers) }

func (f *relayFile) cleanup(ctx context.Context) {
	summary := f.servedSummary()
	if summary != "" {
		log.FromContext(ctx).Infof("Relay download finished: %s", summary)
	}
	if f.keepMessages {
		log.FromContext(ctx).Debugf("Relay message %d kept by configuration", f.relayMessageID)
		return
	}
	peer, err := f.primary.ResolveInputPeerById(f.relayChatID)
	if err != nil {
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		_, err = f.primary.Raw.ChannelsDeleteMessages(cleanupCtx, &tg.ChannelsDeleteMessagesRequest{
			Channel: &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash},
			ID:      []int{f.relayMessageID},
		})
	case *tg.InputPeerChat, *tg.InputPeerUser:
		_, err = f.primary.Raw.MessagesDeleteMessages(cleanupCtx, &tg.MessagesDeleteMessagesRequest{
			Revoke: true,
			ID:     []int{f.relayMessageID},
		})
	}
	if err != nil {
		log.FromContext(ctx).Warnf("Relay message %d cleanup failed: %v", f.relayMessageID, err)
	}
}

func (f *relayFile) recordServed(workerID uint) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.served == nil {
		f.served = make(map[uint]int)
	}
	f.served[workerID]++
}

func (f *relayFile) servedSummary() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	parts := make([]string, 0, len(f.workers))
	for _, ref := range f.workers {
		if count := f.served[ref.ID]; count > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d chunk(s)", ref.Name, count))
		}
	}
	return strings.Join(parts, ", ")
}

type multiClient struct {
	file *relayFile
	base downloader.Client
}

func (c *multiClient) UploadGetFile(ctx context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	for range c.file.workers {
		ref := c.file.nextWorker()
		location, err := c.file.workerLocation(ctx, ref)
		if err != nil {
			log.FromContext(ctx).Debugf("Relay worker %s location failed: %v", ref.Name, err)
			continue
		}
		workerReq := *req
		workerReq.Location = location
		result, err := ref.Ext.Raw.UploadGetFile(ctx, &workerReq)
		if err == nil {
			c.file.recordServed(ref.ID)
			log.FromContext(ctx).Debugf("Relay chunk served: worker=%s offset=%d limit=%d", ref.Name, req.Offset, req.Limit)
			return result, nil
		}
		log.FromContext(ctx).Debugf("Relay worker %s download failed: %v", ref.Name, err)
		c.file.invalidate(ref.ID)
	}
	log.FromContext(ctx).Debugf("Relay chunk fallback to primary: offset=%d limit=%d", req.Offset, req.Limit)
	return c.base.UploadGetFile(ctx, req)
}

func (c *multiClient) UploadGetFileHashes(ctx context.Context, req *tg.UploadGetFileHashesRequest) ([]tg.FileHash, error) {
	return c.base.UploadGetFileHashes(ctx, req)
}

func (c *multiClient) UploadReuploadCDNFile(ctx context.Context, req *tg.UploadReuploadCDNFileRequest) ([]tg.FileHash, error) {
	return c.base.UploadReuploadCDNFile(ctx, req)
}

func (c *multiClient) UploadGetCDNFileHashes(ctx context.Context, req *tg.UploadGetCDNFileHashesRequest) ([]tg.FileHash, error) {
	return c.base.UploadGetCDNFileHashes(ctx, req)
}

func (c *multiClient) UploadGetWebFile(ctx context.Context, req *tg.UploadGetWebFileRequest) (*tg.UploadWebFile, error) {
	return c.base.UploadGetWebFile(ctx, req)
}

func (f *relayFile) nextWorker() worker.ClientRef {
	f.mu.Lock()
	defer f.mu.Unlock()
	ref := f.workers[f.next%len(f.workers)]
	f.next++
	return ref
}

func (f *relayFile) workerLocation(ctx context.Context, ref worker.ClientRef) (tg.InputFileLocationClass, error) {
	f.mu.Lock()
	location, ok := f.locations[ref.ID]
	f.mu.Unlock()
	if ok {
		return location, nil
	}
	peer, err := ref.Ext.ResolveInputPeerById(f.relayChatID)
	if err != nil {
		return nil, err
	}
	inputChannel, ok := peer.(*tg.InputPeerChannel)
	if !ok {
		return nil, fmt.Errorf("relay chat is not a channel")
	}
	result, err := ref.Ext.Raw.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: &tg.InputChannel{ChannelID: inputChannel.ChannelID, AccessHash: inputChannel.AccessHash},
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: f.relayMessageID}},
	})
	if err != nil {
		return nil, err
	}
	message := firstMessage(result)
	if message == nil || message.Media == nil {
		return nil, errors.New("relay message has no media")
	}
	relayed, err := tfile.FromMedia(message.Media, ref.Ext.Raw, tfile.WithMessage(message), tfile.WithName(f.base.Name()))
	if err != nil {
		return nil, err
	}
	location = relayed.Location()
	f.mu.Lock()
	f.locations[ref.ID] = location
	f.mu.Unlock()
	return location, nil
}

func (f *relayFile) invalidate(workerID uint) {
	f.mu.Lock()
	delete(f.locations, workerID)
	f.mu.Unlock()
}

func firstMessage(messages tg.MessagesMessagesClass) *tg.Message {
	var list []tg.MessageClass
	switch m := messages.(type) {
	case *tg.MessagesMessages:
		list = m.Messages
	case *tg.MessagesMessagesSlice:
		list = m.Messages
	case *tg.MessagesChannelMessages:
		list = m.Messages
	default:
		return nil
	}
	for _, item := range list {
		if message, ok := item.(*tg.Message); ok {
			return message
		}
	}
	return nil
}

func forwardedMessageID(updates tg.UpdatesClass, relayChatID int64) int {
	var list []tg.UpdateClass
	switch u := updates.(type) {
	case *tg.Updates:
		list = u.Updates
	case *tg.UpdatesCombined:
		list = u.Updates
	default:
		return 0
	}
	for _, update := range list {
		var messageClass tg.MessageClass
		switch u := update.(type) {
		case *tg.UpdateNewChannelMessage:
			messageClass = u.Message
		case *tg.UpdateNewMessage:
			messageClass = u.Message
		default:
			continue
		}
		message, ok := messageClass.(*tg.Message)
		if !ok {
			continue
		}
		if tgutil.ChatIdFromPeer(message.GetPeerID()) == relayChatID ||
			tgutil.ChatIdFromPeer(message.GetPeerID()) == constant.TDLibPeerID(relayChatID).ToPlain() {
			return message.GetID()
		}
	}
	return 0
}
