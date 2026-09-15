package nats

// EventCallEnded is published by the calls service whenever a call reaches a
// terminal state (ended, missed, rejected, cancelled). The chat service
// consumes it to write a system message into the originating chat.
type EventCallEnded struct {
	CallID      int64  `json:"call_id"`
	ChatID      int64  `json:"chat_id"`
	CallerID    int64  `json:"caller_id"`
	CalleeID    int64  `json:"callee_id"`
	CallType    string `json:"call_type"`
	Status      string `json:"status"`
	EndReason   string `json:"end_reason"`
	StartedAtMs int64  `json:"started_at_ms"`
	EndedAtMs   int64  `json:"ended_at_ms"`
	DurationMs  int64  `json:"duration_ms"`
}

type EventUserStatus struct {
	UserID int64 `json:"user_id"`
	Online bool  `json:"online"`
}

type EventMessageCreated struct {
	ChatID       int64      `json:"chat_id"`
	Message      MessageDTO `json:"message"`
	Participants []int64    `json:"participants,omitempty"`
	HasBot       bool       `json:"has_bot,omitempty"`
	BotID        int64      `json:"bot_id,omitempty"`
	SenderOnline bool       `json:"sender_online,omitempty"`
}

type EventMessageEdited struct {
	ChatID  int64      `json:"chat_id"`
	Message MessageDTO `json:"message"`
}

type EventMessageDeleted struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

type EventMessagePinned struct {
	ChatID  int64      `json:"chat_id"`
	Message MessageDTO `json:"message,omitempty"`
}

type EventReactionAdded struct {
	ChatID    int64       `json:"chat_id"`
	MessageID int64       `json:"message_id"`
	Reaction  ReactionDTO `json:"reaction"`
}

type EventReactionRemoved struct {
	ChatID    int64  `json:"chat_id"`
	MessageID int64  `json:"message_id"`
	UserID    int64  `json:"user_id"`
	Reaction  string `json:"reaction"`
}

type EventChatDeleted struct {
	ChatID int64   `json:"chat_id"`
	Users  []int64 `json:"users"`
}

type EventMessagesRead struct {
	ChatID     int64   `json:"chat_id"`
	UserID     int64   `json:"user_id"`
	MessageIDs []int64 `json:"message_ids"`
}

type MessageDTO struct {
	ID                    int64         `json:"id"`
	ChatID                int64         `json:"chat_id"`
	SenderID              int64         `json:"sender_id"`
	Text                  string        `json:"text"`
	IsRead                bool          `json:"is_read"`
	Edited                bool          `json:"edited"`
	SystemType            string        `json:"system_type,omitempty"`
	AttachmentType        string        `json:"attachment_type,omitempty"`
	AttachmentURL         string        `json:"attachment_url,omitempty"`
	AttachmentName        string        `json:"attachment_name,omitempty"`
	AttachmentSize        int64         `json:"attachment_size,omitempty"`
	CreatedAtMs           int64         `json:"created_at_ms"`
	ReadAtMs              int64         `json:"read_at_ms,omitempty"`
	EditedAtMs            int64         `json:"edited_at_ms,omitempty"`
	IsForwarded           bool          `json:"is_forwarded,omitempty"`
	ForwardedFromSenderID int64         `json:"forwarded_from_sender_id,omitempty"`
	ForwardedFromChatID   int64         `json:"forwarded_from_chat_id,omitempty"`
	Reactions             []ReactionDTO `json:"reactions,omitempty"`
}

type ReactionDTO struct {
	MessageID   int64  `json:"message_id"`
	UserID      int64  `json:"user_id"`
	Reaction    string `json:"reaction"`
	Username    string `json:"username,omitempty"`
	CreatedAtMs int64  `json:"created_at_ms"`
}
