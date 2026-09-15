package nats

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
	ID             int64         `json:"id"`
	ChatID         int64         `json:"chat_id"`
	SenderID       int64         `json:"sender_id"`
	Text           string        `json:"text"`
	IsRead         bool          `json:"is_read"`
	Edited         bool          `json:"edited"`
	SystemType     string        `json:"system_type,omitempty"`
	AttachmentType string        `json:"attachment_type,omitempty"`
	AttachmentURL  string        `json:"attachment_url,omitempty"`
	AttachmentName string        `json:"attachment_name,omitempty"`
	AttachmentSize int64         `json:"attachment_size,omitempty"`
	CreatedAtMs    int64         `json:"created_at_ms"`
	ReadAtMs       int64         `json:"read_at_ms,omitempty"`
	EditedAtMs     int64         `json:"edited_at_ms,omitempty"`
	Reactions      []ReactionDTO `json:"reactions,omitempty"`
}

type ReactionDTO struct {
	MessageID   int64  `json:"message_id"`
	UserID      int64  `json:"user_id"`
	Reaction    string `json:"reaction"`
	Username    string `json:"username,omitempty"`
	CreatedAtMs int64  `json:"created_at_ms"`
}
