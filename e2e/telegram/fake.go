//go:build e2e

package telegrame2e

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

// SentMessage records a message produced by messages.sendMessage/sendMedia.
type SentMessage struct {
	Peer tg.PeerClass
	Text string
	ID   int
}

// Edit records a messages.editMessage request.
type Edit struct {
	Peer tg.PeerClass
	ID   int
	Text string
}

// Callback records a messages.setBotCallbackAnswer request.
type Callback struct {
	Text  string
	Alert bool
}

// Server is an in-memory Telegram RPC responder. It implements tg.Invoker, so
// the real gotd client, sender and downloader code run against it unchanged.
type Server struct {
	mu sync.Mutex

	self       *tg.User
	messages   map[int]*tg.Message
	byPeer     map[int64][]*tg.Message
	users      map[int64]*tg.User
	chats      map[int64]*tg.Chat
	channels   map[int64]*tg.Channel
	files      map[int64][]byte
	uploads    map[int64]map[int][]byte
	lastUpload int64

	nextMessageID int
	nextFileID    int64

	sent      []SentMessage
	edits     []Edit
	callbacks []Callback
}

func NewServer(self *tg.User) *Server {
	return &Server{
		self:     self,
		messages: make(map[int]*tg.Message),
		byPeer:   make(map[int64][]*tg.Message),
		users:    map[int64]*tg.User{self.ID: self},
		chats:    make(map[int64]*tg.Chat),
		channels: make(map[int64]*tg.Channel),
		files:    make(map[int64][]byte),
		uploads:  make(map[int64]map[int][]byte),
	}
}

func (s *Server) AddUser(user *tg.User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.ID] = user
}

func (s *Server) AddChat(chat *tg.Chat) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chats[chat.ID] = chat
}

func (s *Server) AddChannel(channel *tg.Channel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channels[channel.ID] = channel
}

func (s *Server) AddDocumentMessage(peer, from tg.PeerClass, name string, data []byte) *tg.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	fileID := s.newFileLocked()
	s.files[fileID] = append([]byte(nil), data...)
	document := &tg.Document{
		ID:            fileID,
		AccessHash:    1,
		FileReference: []byte{1},
		Date:          int(time.Now().Unix()),
		MimeType:      "text/plain",
		Size:          int64(len(data)),
		DCID:          2,
		Attributes: []tg.DocumentAttributeClass{
			&tg.DocumentAttributeFilename{FileName: name},
		},
	}
	message := &tg.Message{
		ID:     s.newMessageIDLocked(),
		PeerID: peer,
		Date:   int(time.Now().Unix()),
	}
	message.SetFromID(from)
	message.SetMedia(&tg.MessageMediaDocument{Document: document})
	s.storeLocked(message)
	return message
}

func (s *Server) AddTextMessage(peer, from tg.PeerClass, text string) *tg.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	message := &tg.Message{
		ID:      s.newMessageIDLocked(),
		PeerID:  peer,
		Message: text,
		Date:    int(time.Now().Unix()),
	}
	message.SetFromID(from)
	s.storeLocked(message)
	return message
}

func (s *Server) Sent() []SentMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]SentMessage(nil), s.sent...)
}

func (s *Server) Edits() []Edit {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Edit(nil), s.edits...)
}

func (s *Server) Callbacks() []Callback {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Callback(nil), s.callbacks...)
}

func (s *Server) LastUpload() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.partsLocked(s.lastUpload)
}

func (s *Server) Uploaded(fileID int64) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.partsLocked(fileID)
}

func (s *Server) ResetRecords() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = nil
	s.edits = nil
	s.callbacks = nil
	s.uploads = make(map[int64]map[int][]byte)
	s.lastUpload = 0
}

func (s *Server) newMessageIDLocked() int {
	s.nextMessageID++
	return s.nextMessageID
}

func (s *Server) newFileLocked() int64 {
	s.nextFileID++
	return -1000 - s.nextFileID
}

func (s *Server) storeLocked(message *tg.Message) {
	s.messages[message.ID] = message
	key := peerID(message.PeerID)
	s.byPeer[key] = append(s.byPeer[key], message)
}

func (s *Server) partsLocked(fileID int64) []byte {
	parts := s.uploads[fileID]
	if len(parts) == 0 {
		return nil
	}
	indices := make([]int, 0, len(parts))
	for index := range parts {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	var out []byte
	for _, index := range indices {
		out = append(out, parts[index]...)
	}
	return out
}

// Invoke implements tg.Invoker.
func (s *Server) Invoke(_ context.Context, input bin.Encoder, output bin.Decoder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch request := input.(type) {
	case *tg.MessagesSendMessageRequest:
		message := s.newOutgoingLocked(request.Peer, request.Message, nil)
		if reply, ok := request.ReplyTo.(*tg.InputReplyToMessage); ok && reply.ReplyToMsgID != 0 {
			message.SetReplyTo(&tg.MessageReplyHeader{ReplyToMsgID: reply.ReplyToMsgID})
		}
		s.sent = append(s.sent, SentMessage{Peer: message.PeerID, Text: message.Message, ID: message.ID})
		return setUpdates(output, message, false, s.chatEntitiesLocked())
	case *tg.MessagesEditMessageRequest:
		peer := peerFromInput(request.Peer)
		stored := s.messages[request.ID]
		var message tg.Message
		if stored != nil {
			message = *stored
		} else {
			message = tg.Message{ID: request.ID, PeerID: peer, Date: int(time.Now().Unix())}
			message.SetFromID(&tg.PeerUser{UserID: s.self.ID})
			s.storeLocked(&message)
		}
		message.Message = request.Message
		if request.Entities != nil {
			message.Entities = request.Entities
		}
		s.messages[request.ID] = &message
		s.edits = append(s.edits, Edit{Peer: message.PeerID, ID: message.ID, Text: message.Message})
		updateMessage := message
		return setUpdates(output, &updateMessage, true, s.chatEntitiesLocked())
	case *tg.MessagesSetBotCallbackAnswerRequest:
		s.callbacks = append(s.callbacks, Callback{Text: request.Message, Alert: request.Alert})
		return setBool(output, true)
	case *tg.UploadSaveFilePartRequest:
		if s.uploads[request.FileID] == nil {
			s.uploads[request.FileID] = make(map[int][]byte)
		}
		s.uploads[request.FileID][request.FilePart] = append([]byte(nil), request.Bytes...)
		s.lastUpload = request.FileID
		return setBool(output, true)
	case *tg.UploadSaveBigFilePartRequest:
		if s.uploads[request.FileID] == nil {
			s.uploads[request.FileID] = make(map[int][]byte)
		}
		s.uploads[request.FileID][request.FilePart] = append([]byte(nil), request.Bytes...)
		s.lastUpload = request.FileID
		return setBool(output, true)
	case *tg.UploadGetFileRequest:
		return s.getFileLocked(request, output)
	case *tg.MessagesGetMessagesRequest:
		return s.getMessagesLocked(request.ID, output)
	case *tg.ChannelsGetMessagesRequest:
		return s.getMessagesLocked(request.ID, output)
	case *tg.MessagesGetHistoryRequest:
		return s.getHistoryLocked(request, output)
	case *tg.UsersGetUsersRequest:
		if vector, ok := output.(*tg.UserClassVector); ok {
			for _, id := range request.ID {
				if user, ok := id.(*tg.InputUser); ok {
					if found := s.users[user.UserID]; found != nil {
						vector.Elems = append(vector.Elems, found)
					}
				}
			}
		}
		return nil
	case *tg.ChannelsGetChannelsRequest:
		if box, ok := output.(*tg.MessagesChatsBox); ok {
			chats := make([]tg.ChatClass, 0, len(request.ID))
			for _, id := range request.ID {
				if input, ok := id.(*tg.InputChannel); ok {
					if channel := s.channels[input.ChannelID]; channel != nil {
						chats = append(chats, channel)
					}
				}
			}
			box.Chats = &tg.MessagesChats{Chats: chats}
		}
		return nil
	case *tg.MessagesSendMediaRequest:
		message := s.newOutgoingLocked(request.Peer, request.Message, request.Media)
		s.sent = append(s.sent, SentMessage{Peer: message.PeerID, Text: message.Message, ID: message.ID})
		return setUpdates(output, message, false, s.chatEntitiesLocked())
	case *tg.MessagesSendMultiMediaRequest:
		messages := make([]*tg.Message, 0, len(request.MultiMedia))
		for _, item := range request.MultiMedia {
			message := s.newOutgoingLocked(request.Peer, item.Message, item.Media)
			s.sent = append(s.sent, SentMessage{Peer: message.PeerID, Text: message.Message, ID: message.ID})
			messages = append(messages, message)
		}
		if box, ok := output.(*tg.UpdatesBox); ok {
			updates := make([]tg.UpdateClass, 0, len(messages))
			for _, message := range messages {
				updates = append(updates, &tg.UpdateNewMessage{Message: message, Pts: message.ID, PtsCount: 1})
			}
			box.Updates = &tg.Updates{Updates: updates, Users: []tg.UserClass{s.self}, Chats: s.chatEntitiesLocked(), Date: int(time.Now().Unix())}
		}
		return nil
	case *tg.MessagesUploadMediaRequest:
		media := s.mediaFromInputLocked(request.Media)
		if box, ok := output.(*tg.MessageMediaBox); ok {
			box.MessageMedia = media
		}
		return nil
	}
	return nil
}

func (s *Server) newOutgoingLocked(peerInput tg.InputPeerClass, text string, media tg.InputMediaClass) *tg.Message {
	peer := peerFromInput(peerInput)
	message := &tg.Message{
		ID:      s.newMessageIDLocked(),
		PeerID:  peer,
		Message: text,
		Date:    int(time.Now().Unix()),
		Out:     true,
	}
	message.SetFromID(&tg.PeerUser{UserID: s.self.ID})
	if media != nil {
		if converted := s.mediaFromInputLocked(media); converted != nil {
			message.SetMedia(converted)
		}
	}
	s.storeLocked(message)
	return message
}

func (s *Server) mediaFromInputLocked(input tg.InputMediaClass) tg.MessageMediaClass {
	switch media := input.(type) {
	case *tg.InputMediaUploadedDocument:
		data := s.uploadedDataLocked(media.File)
		fileID := inputFileID(media.File)
		document := &tg.Document{
			ID:            fileID,
			AccessHash:    1,
			FileReference: []byte{1},
			Date:          int(time.Now().Unix()),
			MimeType:      media.MimeType,
			Size:          int64(len(data)),
			DCID:          2,
			Attributes:    media.Attributes,
		}
		return &tg.MessageMediaDocument{Document: document}
	case *tg.InputMediaDocument:
		input, ok := media.ID.(*tg.InputDocument)
		if !ok {
			return nil
		}
		data := s.files[input.ID]
		return &tg.MessageMediaDocument{Document: &tg.Document{
			ID:            input.ID,
			AccessHash:    input.AccessHash,
			FileReference: append([]byte(nil), input.FileReference...),
			Date:          int(time.Now().Unix()),
			Size:          int64(len(data)),
			DCID:          2,
		}}
	default:
		return nil
	}
}

func (s *Server) uploadedDataLocked(file tg.InputFileClass) []byte {
	switch f := file.(type) {
	case *tg.InputFile:
		return s.partsLocked(f.ID)
	case *tg.InputFileBig:
		return s.partsLocked(f.ID)
	default:
		return nil
	}
}

func (s *Server) getFileLocked(request *tg.UploadGetFileRequest, output bin.Decoder) error {
	var data []byte
	switch location := request.Location.(type) {
	case *tg.InputDocumentFileLocation:
		data = s.files[location.ID]
	case *tg.InputPhotoFileLocation:
		data = s.files[location.ID]
	}
	offset := int(request.Offset)
	if offset > len(data) {
		offset = len(data)
	}
	end := len(data)
	if request.Limit > 0 && offset+request.Limit < end {
		end = offset + request.Limit
	}
	chunk := append([]byte(nil), data[offset:end]...)
	if box, ok := output.(*tg.UploadFileBox); ok {
		box.File = &tg.UploadFile{Bytes: chunk}
	}
	return nil
}

func (s *Server) getMessagesLocked(ids []tg.InputMessageClass, output bin.Decoder) error {
	messages := make([]tg.MessageClass, 0, len(ids))
	for _, id := range ids {
		input, ok := id.(*tg.InputMessageID)
		if !ok {
			continue
		}
		if message := s.messages[input.ID]; message != nil {
			messages = append(messages, message)
		}
	}
	return setMessages(output, messages, s.chatEntitiesLocked())
}

func (s *Server) getHistoryLocked(request *tg.MessagesGetHistoryRequest, output bin.Decoder) error {
	var messages []tg.MessageClass
	for _, message := range s.byPeer[peerID(peerFromInput(request.Peer))] {
		messages = append(messages, message)
	}
	return setMessages(output, messages, s.chatEntitiesLocked())
}

func setUpdates(output bin.Decoder, message *tg.Message, edit bool, chats []tg.ChatClass) error {
	box, ok := output.(*tg.UpdatesBox)
	if !ok {
		return nil
	}
	var update tg.UpdateClass
	if edit {
		if _, ok := message.PeerID.(*tg.PeerChannel); ok {
			update = &tg.UpdateEditChannelMessage{Message: message, Pts: message.ID, PtsCount: 1}
		} else {
			update = &tg.UpdateEditMessage{Message: message, Pts: message.ID, PtsCount: 1}
		}
	} else {
		update = &tg.UpdateNewMessage{Message: message, Pts: message.ID, PtsCount: 1}
	}
	box.Updates = &tg.Updates{
		Updates: []tg.UpdateClass{update},
		Users:   []tg.UserClass{},
		Chats:   chats,
		Date:    int(time.Now().Unix()),
	}
	return nil
}

func setMessages(output bin.Decoder, messages []tg.MessageClass, chats []tg.ChatClass) error {
	box, ok := output.(*tg.MessagesMessagesBox)
	if !ok {
		return nil
	}
	box.Messages = &tg.MessagesMessages{Messages: messages, Chats: chats}
	return nil
}

func setBool(output bin.Decoder, value bool) error {
	box, ok := output.(*tg.BoolBox)
	if !ok {
		return nil
	}
	if value {
		box.Bool = &tg.BoolTrue{}
	} else {
		box.Bool = &tg.BoolFalse{}
	}
	return nil
}

func (s *Server) chatEntitiesLocked() []tg.ChatClass {
	out := make([]tg.ChatClass, 0, len(s.chats)+len(s.channels))
	for _, chat := range s.chats {
		out = append(out, chat)
	}
	for _, channel := range s.channels {
		out = append(out, channel)
	}
	return out
}

func peerID(peer tg.PeerClass) int64 {
	switch value := peer.(type) {
	case *tg.PeerUser:
		return value.UserID
	case *tg.PeerChat:
		return value.ChatID
	case *tg.PeerChannel:
		return value.ChannelID
	default:
		return 0
	}
}

func peerFromInput(peer tg.InputPeerClass) tg.PeerClass {
	switch value := peer.(type) {
	case *tg.InputPeerUser:
		return &tg.PeerUser{UserID: value.UserID}
	case *tg.InputPeerChat:
		return &tg.PeerChat{ChatID: value.ChatID}
	case *tg.InputPeerChannel:
		return &tg.PeerChannel{ChannelID: value.ChannelID}
	case *tg.InputPeerSelf:
		return &tg.PeerUser{UserID: 0}
	default:
		return &tg.PeerUser{}
	}
}

func inputFileID(file tg.InputFileClass) int64 {
	switch value := file.(type) {
	case *tg.InputFile:
		return value.ID
	case *tg.InputFileBig:
		return value.ID
	default:
		return 0
	}
}

var _ tg.Invoker = (*Server)(nil)

func (s *Server) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("fake telegram: %d sent, %d edits", len(s.sent), len(s.edits))
}
