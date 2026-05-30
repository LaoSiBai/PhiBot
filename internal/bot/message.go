package bot

import "time"

type MessageType string

const (
	MessageTypeText MessageType = "text"
)

type Message struct {
	ID        string
	SessionID string
	SenderID  string
	SenderName string
	Content   string
	Type      MessageType
	Timestamp time.Time
}

type Event struct {
	Type    EventType
	Message *Message
	Data    interface{}
}

type EventType string

const (
	EventMessageReceive EventType = "message_receive"
	EventMessageSend    EventType = "message_send"
	EventStreamChunk    EventType = "stream_chunk"
	EventStreamDone     EventType = "stream_done"
	EventError          EventType = "error"
)
